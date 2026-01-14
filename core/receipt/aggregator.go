package receipt

import (
	"net"
	"sort"
	"strconv"
	"syscall"
	"time"

	"glasshouse/core/profiling"
	"glasshouse/core/version"
)

// Aggregator consumes profiling events and builds a deterministic receipt.
type Aggregator struct {
	provenance string
	rootPID    uint32
	pids       map[uint32]struct{}
	processes  map[uint32]ProcessEntry
	fsRead     map[string]struct{}
	fsWrite    map[string]struct{}
	netConns   map[string]Connection
	syscalls   map[string]int
}

func NewAggregator(provenance string) *Aggregator {
	return &Aggregator{
		provenance: provenance,
		pids:       make(map[uint32]struct{}),
		processes:  make(map[uint32]ProcessEntry),
		fsRead:     make(map[string]struct{}),
		fsWrite:    make(map[string]struct{}),
		netConns:   make(map[string]Connection),
		syscalls:   make(map[string]int),
	}
}

func (a *Aggregator) SetRoot(pid uint32, cmd string) {
	a.rootPID = pid
	a.pids[pid] = struct{}{}
	a.processes[pid] = ProcessEntry{PID: pid, PPID: 0, Cmd: cmd}
}

func (a *Aggregator) HandleEvent(ev profiling.Event) {
	if a.rootPID == 0 {
		return
	}

	tracked := ev.PID == a.rootPID
	if !tracked {
		_, tracked = a.pids[ev.PPID]
	}
	if !tracked {
		return
	}

	a.pids[ev.PID] = struct{}{}

	entry, ok := a.processes[ev.PID]
	if !ok {
		entry = ProcessEntry{PID: ev.PID, PPID: ev.PPID}
		a.processes[ev.PID] = entry
	} else if ev.PPID != 0 && entry.PPID == 0 {
		entry.PPID = ev.PPID
		a.processes[ev.PID] = entry
	}

	switch ev.Type {
	case profiling.EventExec:
		a.syscalls["execve"]++
		cmd := ev.Path
		if cmd == "" {
			cmd = ev.Comm
		}
		entry, ok := a.processes[ev.PID]
		if !ok {
			entry = ProcessEntry{PID: ev.PID}
		}
		if ev.PPID != 0 && entry.PPID == 0 {
			entry.PPID = ev.PPID
		}
		if cmd != "" && (entry.Cmd == "" || len(cmd) > len(entry.Cmd)) {
			entry.Cmd = cmd
		}
		a.processes[ev.PID] = entry
	case profiling.EventOpen:
		a.syscalls["open"]++
		path := ev.Path
		if path == "" {
			return
		}
		if isWriteOpen(ev.Flags) {
			a.fsWrite[path] = struct{}{}
		} else {
			a.fsRead[path] = struct{}{}
		}
	case profiling.EventConnect:
		a.syscalls["connect"]++
		dst := formatAddr(ev)
		if dst != "" {
			proto := protoString(ev.Proto)
			key := dst + "|" + proto
			a.netConns[key] = Connection{Dst: dst, Protocol: proto, Attempted: true}
		}
	}
}

func (a *Aggregator) Receipt(exitCode int, duration time.Duration) Receipt {
	processes := make([]ProcessEntry, 0, len(a.processes))
	for _, entry := range a.processes {
		processes = append(processes, entry)
	}
	sort.Slice(processes, func(i, j int) bool { return processes[i].PID < processes[j].PID })

	read := setToSortedSlice(a.fsRead)
	written := setToSortedSlice(a.fsWrite)
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

	connections := make([]Connection, 0, len(a.netConns))
	attempts := make([]NetworkAttempt, 0, len(a.netConns))
	for _, conn := range a.netConns {
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

	return Receipt{
		Version:    version.ReceiptVersion,
		Provenance: a.provenance,
		ExitCode:   exitCode,
		DurationMs: duration.Milliseconds(),
		Processes:  processes,
		Filesystem: fs,
		Network:    netInfo,
		Syscalls: &SyscallInfo{
			Counts: copyCounts(a.syscalls),
			Denied: []string{},
		},
	}
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
