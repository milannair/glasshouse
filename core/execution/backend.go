package execution

import "context"

// ExecutionBackend is implemented by all execution adapters. It is intentionally
// minimal and substrate-agnostic so backends can be swapped without touching
// core orchestration or policy.
type ExecutionBackend interface {
	Name() string
	Prepare(ctx context.Context) error
	Start(spec ExecutionSpec) (ExecutionHandle, error)
	Wait(h ExecutionHandle) (ExecutionResult, error)
	Kill(h ExecutionHandle) error
	Cleanup(h ExecutionHandle) error
	ProfilingInfo(h ExecutionHandle) BackendProfilingInfo
}
