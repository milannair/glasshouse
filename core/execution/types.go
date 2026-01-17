package execution

import (
	"time"

	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

// ExecutionSpec describes a single execution request.
// It is substrate-agnostic and used by all backends.
type ExecutionSpec struct {
	Args        []string
	Workdir     string
	Env         []string
	Guest       bool
	Profiling   profiling.Mode
	Labels      map[string]string
	ReceiptMask []string
}

// ExecutionHandle identifies a running execution in a backend.
type ExecutionHandle struct {
	ID            string
	BackendHandle any
}

// ExecutionIdentity provides stable identifiers for the running execution.
type ExecutionIdentity struct {
	RootPID    int
	CgroupPath string
	Namespaces map[string]string
}

// BackendProfilingInfo describes profiling attachment options for a handle.
type BackendProfilingInfo struct {
	Identity        ExecutionIdentity
	SupportedModes  []profiling.Mode
	SupportsProfile bool
}

// ExecutionResult is the backend-reported outcome. Receipt generation is
// layered on top and is nil when profiling is disabled or unavailable.
type ExecutionResult struct {
	Handle            ExecutionHandle
	ExitCode          int
	Err               error
	StartedAt         time.Time
	CompletedAt       time.Time
	Stdout            string // Captured stdout (Firecracker backend)
	Stderr            string // Captured stderr (Firecracker backend)
	ProfilingEnabled  bool
	ProfilingAttached bool
	ProfilingError    error
	Receipt           *receipt.Receipt
}
