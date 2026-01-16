package backend

import "glasshouse/core/execution"

// Re-export core execution contracts for compatibility with existing callers.
type (
	ExecutionBackend     = execution.ExecutionBackend
	ExecutionSpec        = execution.ExecutionSpec
	ExecutionHandle      = execution.ExecutionHandle
	ExecutionResult      = execution.ExecutionResult
	BackendProfilingInfo = execution.BackendProfilingInfo
	ExtraErrorProvider   = execution.ExtraErrorProvider
	OutputProvider       = execution.OutputProvider
	ProcessStateProvider = execution.ProcessStateProvider
	MetadataProvider     = execution.MetadataProvider
)
