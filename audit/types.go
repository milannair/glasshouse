package audit

type EventType uint32

const (
	EventExec EventType = 1
	EventOpen EventType = 2
	EventConnect EventType = 3
)

type Event struct {
	Type       EventType
	PID        uint32
	PPID       uint32
	Flags      uint32
	Comm       string
	Path       string
	AddrFamily uint8
	Proto      uint8
	Addr       [16]byte
	Port       uint16
}

type Receipt struct {
	ExitCode   int             `json:"exit_code"`
	DurationMs int64           `json:"duration_ms"`
	Processes  []ProcessEntry  `json:"processes"`
	Filesystem *FilesystemInfo `json:"filesystem,omitempty"`
	Network    *NetworkInfo    `json:"network,omitempty"`
	Resources  *Resources      `json:"resources,omitempty"`
}

type ProcessEntry struct {
	PID  uint32 `json:"pid"`
	PPID uint32 `json:"ppid"`
	Cmd  string `json:"cmd"`
}

type FilesystemInfo struct {
	Read    []string `json:"read,omitempty"`
	Written []string `json:"written,omitempty"`
}

type NetworkInfo struct {
	Connections []Connection `json:"connections,omitempty"`
}

type Connection struct {
	Dst       string `json:"dst"`
	Protocol  string `json:"protocol,omitempty"`
	Attempted bool   `json:"attempted"`
}

type Resources struct {
	CPUTimeMs int64 `json:"cpu_time_ms,omitempty"`
	MaxRSSKB  int64 `json:"max_rss_kb,omitempty"`
}
