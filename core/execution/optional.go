package execution

import (
	"os"

	"glasshouse/core/receipt"
)

// ExtraErrorProvider allows backends to surface non-fatal errors collected during execution.
type ExtraErrorProvider interface {
	ExtraErrors() []string
}

// OutputProvider allows backends to expose captured stdout/stderr for hashing.
type OutputProvider interface {
	Stdout() []byte
	Stderr() []byte
}

// ProcessStateProvider is implemented by backends that can expose process resource usage.
type ProcessStateProvider interface {
	ProcessState() *os.ProcessState
}

// MetadataProvider allows backends to override backend/isolation metadata.
type MetadataProvider interface {
	Metadata() receipt.ExecutionInfo
}
