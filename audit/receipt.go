package audit

import (
	"net"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type Aggregator struct {
	mu        sync.Mutex
	rootPID   uint32
	pids      map[uint32]struct{}
	processes map[uint32]ProcessEntry
	fsRead    map[string]struct{}
	fsWrite   map[string]struct{}
	netConns  map[string]Connection
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		pids:      make(map[uint32]struct{}),
		processes: make(map[uint32]ProcessEntry),
		fsRead:    make(map[string]struct{}),
		fsWrite:   make(map[string]struct{}),
		netConns:  make(map[string]Connection),
	}
}

func (a *Aggregator) SetRoot(pid uint32, cmd string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.rootPID = pid
	a.pids[pid] = struct{}{}
	a.processes[pid] = ProcessEntry{PID: pid, PPID: 0, Cmd: cmd}
}

func (a *Aggregator) HandleEvent(ev Event) {
	a.mu.Lock()
	defer a.mu.Unlock()

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
	case EventExec:
		cmd := ev.Path
		if cmd == "" {
			cmd = ev.Comm
		}
		entry, ok := a.processes[ev.PID]
		if !ok {
			entry = ProcessEntry{PID: ev.PID}
		}
		if ev.PPID != 0 {
			entry.PPID = ev.PPID
		}
		if cmd != "" && (entry.Cmd == "" || len(cmd) > len(entry.Cmd)) {
			entry.Cmd = cmd
		}
		a.processes[ev.PID] = entry
	case EventOpen:
		path := ev.Path
		if path == "" {
			return
		}
		if isWriteOpen(ev.Flags) {
			a.fsWrite[path] = struct{}{}
		} else {
			a.fsRead[path] = struct{}{}
		}
	case EventConnect:
		dst := formatAddr(ev)
		if dst != "" {
			proto := protoString(ev.Proto)
			key := dst + "|" + proto
			a.netConns[key] = Connection{Dst: dst, Protocol: proto, Attempted: true}
		}
	}
}

func (a *Aggregator) Receipt(exitCode int, duration time.Duration) Receipt {
	a.mu.Lock()
	defer a.mu.Unlock()

	processes := make([]ProcessEntry, 0, len(a.processes))
	for _, entry := range a.processes {
		processes = append(processes, entry)
	}
	sort.Slice(processes, func(i, j int) bool { return processes[i].PID < processes[j].PID })

	read := setToSortedSlice(a.fsRead)
	written := setToSortedSlice(a.fsWrite)
	var fs *FilesystemInfo
	if len(read) > 0 || len(written) > 0 {
		fs = &FilesystemInfo{Read: read, Written: written}
	}

	var netInfo *NetworkInfo
	if len(a.netConns) > 0 {
		connections := make([]Connection, 0, len(a.netConns))
		for _, conn := range a.netConns {
			connections = append(connections, conn)
		}
		sort.Slice(connections, func(i, j int) bool {
			if connections[i].Dst == connections[j].Dst {
				return connections[i].Protocol < connections[j].Protocol
			}
			return connections[i].Dst < connections[j].Dst
		})
		netInfo = &NetworkInfo{Connections: connections}
	}

	return Receipt{
		ExitCode:   exitCode,
		DurationMs: duration.Milliseconds(),
		Processes:  processes,
		Filesystem: fs,
		Network:    netInfo,
	}
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

func formatAddr(ev Event) string {
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
