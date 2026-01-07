package audit

type EventType uint32

const (
	EventExec    EventType = 1
	EventOpen    EventType = 2
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
	ReceiptVersion string `json:"receipt_version,omitempty"`
	ExecutionID    string `json:"execution_id,omitempty"`
	Timestamp      string `json:"timestamp,omitempty"`

	Outcome     *Outcome     `json:"outcome,omitempty"`
	Timing      *Timing      `json:"timing,omitempty"`
	ProcessTree []ProcessV2  `json:"process_tree,omitempty"`
	Syscalls    *SyscallInfo `json:"syscalls,omitempty"`
	Environment *Environment `json:"environment,omitempty"`
	Artifacts   *Artifacts   `json:"artifacts,omitempty"`

	ExitCode   int             `json:"exit_code"`
	DurationMs int64           `json:"duration_ms"`
	Processes  []ProcessEntry  `json:"processes"`
	Filesystem *FilesystemInfo `json:"filesystem"`
	Network    *NetworkInfo    `json:"network"`
	Resources  *Resources      `json:"resources,omitempty"`
}

type ProcessEntry struct {
	PID  uint32 `json:"pid"`
	PPID uint32 `json:"ppid"`
	Cmd  string `json:"cmd"`
}

type FilesystemInfo struct {
	Read             []string `json:"read,omitempty"`
	Written          []string `json:"written,omitempty"`
	Reads            []string `json:"reads"`
	Writes           []string `json:"writes"`
	Deletes          []string `json:"deletes"`
	PolicyViolations []string `json:"policy_violations"`
}

type NetworkInfo struct {
	Connections   []Connection     `json:"connections,omitempty"`
	Attempts      []NetworkAttempt `json:"attempts"`
	BytesSent     int64            `json:"bytes_sent"`
	BytesReceived int64            `json:"bytes_received"`
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

type Outcome struct {
	ExitCode int     `json:"exit_code"`
	Signal   *string `json:"signal"`
	Error    *string `json:"error"`
}

type Timing struct {
	DurationMs int64 `json:"duration_ms"`
	CPUTimeMs  int64 `json:"cpu_time_ms"`
}

type ProcessV2 struct {
	PID        uint32   `json:"pid"`
	PPID       uint32   `json:"ppid"`
	Exe        string   `json:"exe"`
	Argv       []string `json:"argv"`
	WorkingDir string   `json:"working_dir"`
}

type NetworkAttempt struct {
	Dst      string `json:"dst"`
	Protocol string `json:"protocol,omitempty"`
	Result   string `json:"result,omitempty"`
	Policy   string `json:"policy,omitempty"`
}

type SyscallInfo struct {
	Counts map[string]int `json:"counts"`
	Denied []string       `json:"denied"`
}

type Environment struct {
	Runtime string  `json:"runtime"`
	OS      string  `json:"os"`
	Arch    string  `json:"arch"`
	Sandbox Sandbox `json:"sandbox"`
}

type Sandbox struct {
	Network string `json:"network"`
}

type Artifacts struct {
	StdoutHash string `json:"stdout_hash"`
	StderrHash string `json:"stderr_hash"`
}
