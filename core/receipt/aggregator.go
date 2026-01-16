package receipt

import (
	"net"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"glasshouse/core/identity"
	"glasshouse/core/profiling"
	"glasshouse/core/version"
)

// ExecutionState represents the lifecycle stage of an execution.
type ExecutionState string

const (
	ExecutionCreated    ExecutionState = "CREATED"
	ExecutionRunning    ExecutionState = "RUNNING"
	ExecutionTerminated ExecutionState = "TERMINATED"
	ExecutionFlushed    ExecutionState = "FLUSHED"
)

// AggregatorOptions configures a streaming aggregator.
type AggregatorOptions struct {
	Provenance string
	AutoCreate bool
}

// ExecutionStart describes execution metadata for aggregation.
type ExecutionStart struct {
	ID              identity.ExecutionID
	RootPID         uint32
	RootStartTime   uint64
	Command         string
	StartedAt       time.Time
	ObservationMode string
}

// Aggregator consumes profiling events and builds deterministic receipts.
type Aggregator struct {
	mu         sync.Mutex
	provenance string
	autoCreate bool

	executions map[string]*executionAggregate
	byCgroup   map[uint64]string
	byPID      map[uint32]string
	pidStart   map[uint32]uint64
	defaultID  string
}

type executionAggregate struct {
	id              identity.ExecutionID
	idString        string
	provenance      string
	observationMode string
	state           ExecutionState
	startTime       time.Time
	endTime         time.Time
	rootPID         uint32
	rootStartTime   uint64
	command         string

	pids      map[uint32]struct{}
	processes map[uint32]ProcessEntry
	fsRead    map[string]struct{}
	fsWrite   map[string]struct{}
	netConns  map[string]Connection
	syscalls  map[string]int
	policy    *PolicyInfo
}

// NewAggregator preserves legacy single-execution behavior.
func NewAggregator(provenance string) *Aggregator {
	return NewStreamAggregator(AggregatorOptions{Provenance: provenance})
}

// NewStreamAggregator constructs an aggregator that can track multiple executions.
func NewStreamAggregator(opts AggregatorOptions) *Aggregator {
	return &Aggregator{
		provenance: opts.Provenance,
		autoCreate: opts.AutoCreate,
		executions: make(map[string]*executionAggregate),
		byCgroup:   make(map[uint64]string),
		byPID:      make(map[uint32]string),
		pidStart:   make(map[uint32]uint64),
	}
}

// StartExecution registers a new execution for tracking.
func (a *Aggregator) StartExecution(start ExecutionStart) identity.ExecutionID {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.startExecutionLocked(start, false)
}

// SetRoot keeps the legacy single-execution interface.
func (a *Aggregator) SetRoot(pid uint32, cmd string) {
	start := ExecutionStart{
		RootPID:         pid,
		Command:         cmd,
		StartedAt:       time.Now(),
		ObservationMode: observationModeFromProvenance(a.provenance),
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	id := a.startExecutionLocked(start, true)
	a.defaultID = id.String()
}

// HandleEvent assigns an event to an execution and aggregates it.
func (a *Aggregator) HandleEvent(ev profiling.Event) identity.ExecutionID {
	a.mu.Lock()
	defer a.mu.Unlock()

	if exec := a.matchExecutionLocked(ev); exec != nil {
		exec.handleEvent(ev)
		a.indexPIDLocked(exec, ev.PID)
		return exec.id
	}
	if !a.autoCreate {
		return identity.ExecutionID{}
	}

	start := ExecutionStart{
		ID:              identity.FromCgroup(ev.CgroupID),
		RootPID:         ev.PID,
		RootStartTime:   a.resolveStartTimeLocked(ev.PID),
		StartedAt:       time.Now(),
		ObservationMode: observationModeFromProvenance(a.provenance),
	}
	if ev.CgroupID == 0 {
		start.ID = identity.FromRoot(ev.PID, start.RootStartTime)
	}
	exec := a.startExecutionLocked(start, true)
	if exec == (identity.ExecutionID{}) {
		return identity.ExecutionID{}
	}
	if agg := a.executions[exec.String()]; agg != nil {
		agg.handleEvent(ev)
		a.indexPIDLocked(agg, ev.PID)
	}
	return exec
}

// EndExecution marks an execution as terminated.
func (a *Aggregator) EndExecution(id identity.ExecutionID, end time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	exec := a.executions[id.String()]
	if exec == nil {
		return
	}
	exec.state = ExecutionTerminated
	exec.endTime = end
}

// Receipt emits a receipt for the legacy default execution.
func (a *Aggregator) Receipt(exitCode int, duration time.Duration) Receipt {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.defaultID == "" {
		return Receipt{Version: version.ReceiptVersion, Provenance: a.provenance}
	}
	return a.flushLocked(a.defaultID, exitCode, duration)
}

// FlushExecution emits a receipt for a specific execution.
func (a *Aggregator) FlushExecution(id identity.ExecutionID, exitCode int, duration time.Duration) (Receipt, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := id.String()
	if key == "" {
		return Receipt{}, false
	}
	if _, ok := a.executions[key]; !ok {
		return Receipt{}, false
	}
	return a.flushLocked(key, exitCode, duration), true
}

// ForgetExecution removes execution state from the aggregator.
func (a *Aggregator) ForgetExecution(id identity.ExecutionID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := id.String()
	if key == "" {
		return
	}
	if _, ok := a.executions[key]; !ok {
		return
	}
	delete(a.executions, key)
	if id.CgroupID != 0 {
		delete(a.byCgroup, id.CgroupID)
	}
	for pid, execKey := range a.byPID {
		if execKey == key {
			delete(a.byPID, pid)
		}
	}
	if a.defaultID == key {
		a.defaultID = ""
	}
}

// RecordPolicyViolation adds a policy violation to an execution receipt.
func (a *Aggregator) RecordPolicyViolation(id identity.ExecutionID, violation PolicyViolation) {
	a.mu.Lock()
	defer a.mu.Unlock()
	exec := a.executions[id.String()]
	if exec == nil {
		return
	}
	if exec.policy == nil {
		exec.policy = &PolicyInfo{}
	}
	exec.policy.Violations = append(exec.policy.Violations, violation)
}

// RecordPolicyEnforcement records an enforcement action for the receipt.
func (a *Aggregator) RecordPolicyEnforcement(id identity.ExecutionID, enforcement PolicyEnforcement) {
	a.mu.Lock()
	defer a.mu.Unlock()
	exec := a.executions[id.String()]
	if exec == nil {
		return
	}
	if exec.policy == nil {
		exec.policy = &PolicyInfo{}
	}
	exec.policy.Enforcements = append(exec.policy.Enforcements, enforcement)
}

// SetPolicyTrusted sets the post-execution trust decision.
func (a *Aggregator) SetPolicyTrusted(id identity.ExecutionID, trusted bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	exec := a.executions[id.String()]
	if exec == nil {
		return
	}
	if exec.policy == nil {
		exec.policy = &PolicyInfo{}
	}
	exec.policy.Trusted = trusted
}

// SetPolicyFailed marks the execution as policy failed.
func (a *Aggregator) SetPolicyFailed(id identity.ExecutionID, failed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	exec := a.executions[id.String()]
	if exec == nil {
		return
	}
	if exec.policy == nil {
		exec.policy = &PolicyInfo{}
	}
	exec.policy.Failed = failed
}

func (a *Aggregator) startExecutionLocked(start ExecutionStart, setDefault bool) identity.ExecutionID {
	if start.StartedAt.IsZero() {
		start.StartedAt = time.Now()
	}
	id := start.ID
	if id.IsZero() {
		if start.RootStartTime == 0 && start.RootPID != 0 {
			start.RootStartTime = a.resolveStartTimeLocked(start.RootPID)
		}
		id = identity.FromRoot(start.RootPID, start.RootStartTime)
	}
	key := id.String()
	if key == "" {
		return identity.ExecutionID{}
	}
	if _, exists := a.executions[key]; exists {
		return id
	}

	exec := newExecutionAggregate(a.provenance, start, id, key)
	a.executions[key] = exec
	if id.CgroupID != 0 {
		a.byCgroup[id.CgroupID] = key
	}
	if start.RootPID != 0 {
		a.byPID[start.RootPID] = key
	}
	if setDefault && a.defaultID == "" {
		a.defaultID = key
	}
	return id
}

func newExecutionAggregate(provenance string, start ExecutionStart, id identity.ExecutionID, key string) *executionAggregate {
	mode := start.ObservationMode
	if mode == "" {
		mode = observationModeFromProvenance(provenance)
	}
	exec := &executionAggregate{
		id:              id,
		idString:        key,
		provenance:      provenance,
		observationMode: mode,
		state:           ExecutionCreated,
		startTime:       start.StartedAt,
		rootPID:         start.RootPID,
		rootStartTime:   start.RootStartTime,
		command:         start.Command,
		pids:            make(map[uint32]struct{}),
		processes:       make(map[uint32]ProcessEntry),
		fsRead:          make(map[string]struct{}),
		fsWrite:         make(map[string]struct{}),
		netConns:        make(map[string]Connection),
		syscalls:        make(map[string]int),
	}
	if start.RootPID != 0 {
		exec.pids[start.RootPID] = struct{}{}
		exec.processes[start.RootPID] = ProcessEntry{PID: start.RootPID, PPID: 0, Cmd: start.Command}
	}
	return exec
}

func (a *Aggregator) resolveStartTimeLocked(pid uint32) uint64 {
	if pid == 0 {
		return 0
	}
	if value, ok := a.pidStart[pid]; ok {
		return value
	}
	value, err := identity.ProcessStartTime(pid)
	if err != nil {
		a.pidStart[pid] = 0
		return 0
	}
	a.pidStart[pid] = value
	return value
}

func (a *Aggregator) matchExecutionLocked(ev profiling.Event) *executionAggregate {
	if ev.CgroupID != 0 {
		if key, ok := a.byCgroup[ev.CgroupID]; ok {
			return a.executions[key]
		}
	}
	if key, ok := a.byPID[ev.PID]; ok {
		return a.executions[key]
	}
	if key, ok := a.byPID[ev.PPID]; ok {
		return a.executions[key]
	}
	return nil
}

func (a *Aggregator) indexPIDLocked(exec *executionAggregate, pid uint32) {
	if pid == 0 || exec == nil {
		return
	}
	a.byPID[pid] = exec.idString
}

func (a *Aggregator) flushLocked(key string, exitCode int, duration time.Duration) Receipt {
	exec := a.executions[key]
	if exec == nil {
		return Receipt{Version: version.ReceiptVersion, Provenance: a.provenance}
	}
	completeness := "partial"
	if exec.state == ExecutionTerminated {
		completeness = "closed"
	}
	if exec.endTime.IsZero() {
		exec.endTime = exec.startTime.Add(duration)
	}
	if exec.state == ExecutionCreated {
		exec.state = ExecutionRunning
	}
	rec := exec.receipt(exitCode, duration, completeness)
	exec.state = ExecutionFlushed
	return rec
}

func (e *executionAggregate) handleEvent(ev profiling.Event) {
	if e.state == ExecutionCreated {
		e.state = ExecutionRunning
	}

	e.pids[ev.PID] = struct{}{}

	entry, ok := e.processes[ev.PID]
	if !ok {
		entry = ProcessEntry{PID: ev.PID, PPID: ev.PPID}
		e.processes[ev.PID] = entry
	} else if ev.PPID != 0 && entry.PPID == 0 {
		entry.PPID = ev.PPID
		e.processes[ev.PID] = entry
	}

	switch ev.Type {
	case profiling.EventExec:
		e.syscalls["execve"]++
		cmd := ev.Path
		if cmd == "" {
			cmd = ev.Comm
		}
		entry, ok := e.processes[ev.PID]
		if !ok {
			entry = ProcessEntry{PID: ev.PID}
		}
		if ev.PPID != 0 && entry.PPID == 0 {
			entry.PPID = ev.PPID
		}
		if cmd != "" && (entry.Cmd == "" || len(cmd) > len(entry.Cmd)) {
			entry.Cmd = cmd
		}
		e.processes[ev.PID] = entry
	case profiling.EventOpen:
		e.syscalls["open"]++
		path := ev.Path
		if path == "" {
			return
		}
		if isWriteOpen(ev.Flags) {
			e.fsWrite[path] = struct{}{}
		} else {
			e.fsRead[path] = struct{}{}
		}
	case profiling.EventConnect:
		e.syscalls["connect"]++
		dst := formatAddr(ev)
		if dst != "" {
			proto := protoString(ev.Proto)
			key := dst + "|" + proto
			e.netConns[key] = Connection{Dst: dst, Protocol: proto, Attempted: true}
		}
	}
}

func (e *executionAggregate) receipt(exitCode int, duration time.Duration, completeness string) Receipt {
	processes := make([]ProcessEntry, 0, len(e.processes))
	for _, entry := range e.processes {
		processes = append(processes, entry)
	}
	sort.Slice(processes, func(i, j int) bool { return processes[i].PID < processes[j].PID })

	read := setToSortedSlice(e.fsRead)
	written := setToSortedSlice(e.fsWrite)
	if read == nil {
		read = []string{}
	}
	if written == nil {
		written = []string{}
	}
	fs := &FilesystemInfo{
		Reads:            read,
		Writes:           written,
		Deletes:          []string{},
		PolicyViolations: []string{},
	}

	connections := make([]Connection, 0, len(e.netConns))
	attempts := make([]NetworkAttempt, 0, len(e.netConns))
	for _, conn := range e.netConns {
		connections = append(connections, conn)
		attempts = append(attempts, NetworkAttempt{
			Dst:      conn.Dst,
			Protocol: conn.Protocol,
			Result:   "attempted",
		})
	}
	sort.Slice(connections, func(i, j int) bool {
		if connections[i].Dst == connections[j].Dst {
			return connections[i].Protocol < connections[j].Protocol
		}
		return connections[i].Dst < connections[j].Dst
	})
	sort.Slice(attempts, func(i, j int) bool {
		if attempts[i].Dst == attempts[j].Dst {
			return attempts[i].Protocol < attempts[j].Protocol
		}
		return attempts[i].Dst < attempts[j].Dst
	})
	netInfo := &NetworkInfo{
		Connections:   connections,
		Attempts:      attempts,
		BytesSent:     0,
		BytesReceived: 0,
	}

	rec := Receipt{
		Version:         version.ReceiptVersion,
		ExecutionID:     e.idString,
		Provenance:      e.provenance,
		StartTime:       formatTime(e.startTime),
		EndTime:         formatTime(e.endTime),
		ObservationMode: e.observationMode,
		Completeness:    completeness,
		ExitCode:        exitCode,
		DurationMs:      duration.Milliseconds(),
		Processes:       processes,
		Filesystem:      fs,
		Network:         netInfo,
		Syscalls: &SyscallInfo{
			Counts: copyCounts(e.syscalls),
			Denied: []string{},
		},
	}
	if e.policy != nil {
		copy := *e.policy
		rec.Policy = &copy
	}
	return rec
}

func copyCounts(counts map[string]int) map[string]int {
	out := make(map[string]int, len(counts))
	for key, value := range counts {
		out[key] = value
	}
	return out
}

func setToSortedSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func isWriteOpen(flags uint32) bool {
	writeMask := uint32(syscall.O_WRONLY | syscall.O_RDWR | syscall.O_CREAT | syscall.O_TRUNC | syscall.O_APPEND)
	return flags&writeMask != 0
}

func formatAddr(ev profiling.Event) string {
	if ev.Port == 0 {
		return ""
	}

	switch ev.AddrFamily {
	case syscall.AF_INET:
		ip := net.IPv4(ev.Addr[0], ev.Addr[1], ev.Addr[2], ev.Addr[3])
		return net.JoinHostPort(ip.String(), itoa16(ev.Port))
	case syscall.AF_INET6:
		ip := net.IP(ev.Addr[:])
		return net.JoinHostPort(ip.String(), itoa16(ev.Port))
	default:
		return ""
	}
}

func protoString(proto uint8) string {
	switch proto {
	case syscall.IPPROTO_TCP:
		return "tcp"
	case syscall.IPPROTO_UDP:
		return "udp"
	default:
		return ""
	}
}

func itoa16(v uint16) string {
	return strconv.Itoa(int(v))
}
