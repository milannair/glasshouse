package receipt

// Receipt is a deterministic, versioned artifact emitted when profiling is enabled.
type Receipt struct {
	Version     string `json:"version"`
	ExecutionID string `json:"execution_id,omitempty"`
	Provenance  string `json:"provenance,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	StartTime   string `json:"start_time,omitempty"`
	EndTime     string `json:"end_time,omitempty"`
	// ObservationMode is guest|host|host+guest depending on the attachment scope.
	ObservationMode string          `json:"observation_mode,omitempty"`
	Completeness    string          `json:"completeness,omitempty"`
	Outcome         *Outcome        `json:"outcome,omitempty"`
	Timing          *Timing         `json:"timing,omitempty"`
	ProcessTree     []ProcessV2     `json:"process_tree,omitempty"`
	Syscalls        *SyscallInfo    `json:"syscalls,omitempty"`
	Environment     *Environment    `json:"environment,omitempty"`
	Execution       *ExecutionInfo  `json:"execution,omitempty"`
	Artifacts       *Artifacts      `json:"artifacts,omitempty"`
	ExitCode        int             `json:"exit_code"`
	DurationMs      int64           `json:"duration_ms"`
	Processes       []ProcessEntry  `json:"processes"`
	Filesystem      *FilesystemInfo `json:"filesystem"`
	Network         *NetworkInfo    `json:"network"`
	Resources       *Resources      `json:"resources,omitempty"`
	Redactions      []string        `json:"redactions,omitempty"`
	Policy          *PolicyInfo     `json:"policy,omitempty"`
}

type ProcessEntry struct {
	PID  uint32 `json:"pid"`
	PPID uint32 `json:"ppid"`
	Cmd  string `json:"cmd"`
}

type FilesystemInfo struct {
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

type ExecutionInfo struct {
	Backend   string `json:"backend"`
	Isolation string `json:"isolation"`
}

type Sandbox struct {
	Network string `json:"network"`
}

type Artifacts struct {
	StdoutHash string `json:"stdout_hash"`
	StderrHash string `json:"stderr_hash"`
}

// PolicyInfo captures policy violations and enforcement decisions.
type PolicyInfo struct {
	Violations   []PolicyViolation   `json:"violations,omitempty"`
	Enforcements []PolicyEnforcement `json:"enforcements,omitempty"`
	Trusted      bool                `json:"trusted,omitempty"`
	Failed       bool                `json:"failed,omitempty"`
}

type PolicyViolation struct {
	Phase   string `json:"phase,omitempty"`
	Rule    string `json:"rule,omitempty"`
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
}

type PolicyEnforcement struct {
	Action  string `json:"action,omitempty"`
	Target  string `json:"target,omitempty"`
	Rule    string `json:"rule,omitempty"`
	Message string `json:"message,omitempty"`
}
