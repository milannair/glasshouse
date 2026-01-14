package fake

import (
	"context"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

// Backend is a configurable fake backend useful for contract tests.
type Backend struct {
	ExitCode   int
	StartErr   error
	WaitErr    error
	CleanupErr error
	Extra      []string
}

func New(exitCode int) *Backend {
	return &Backend{ExitCode: exitCode}
}

func (b *Backend) Name() string { return "fake" }

func (b *Backend) Prepare(ctx context.Context) error {
	return ctx.Err()
}

func (b *Backend) Start(spec execution.ExecutionSpec) (execution.ExecutionHandle, error) {
	_ = spec
	if b.StartErr != nil {
		return execution.ExecutionHandle{}, b.StartErr
	}
	return execution.ExecutionHandle{ID: "fake"}, nil
}

func (b *Backend) Wait(h execution.ExecutionHandle) (execution.ExecutionResult, error) {
	if b.WaitErr != nil {
		return execution.ExecutionResult{Handle: h, ExitCode: b.ExitCode, Err: b.WaitErr}, b.WaitErr
	}
	return execution.ExecutionResult{Handle: h, ExitCode: b.ExitCode}, nil
}

func (b *Backend) Kill(h execution.ExecutionHandle) error {
	_ = h
	return nil
}

func (b *Backend) Cleanup(h execution.ExecutionHandle) error {
	_ = h
	if b.CleanupErr != nil {
		return b.CleanupErr
	}
	return nil
}

func (b *Backend) ProfilingInfo(h execution.ExecutionHandle) execution.BackendProfilingInfo {
	_ = h
	return execution.BackendProfilingInfo{
		Identity: execution.ExecutionIdentity{
			RootPID:    4242,
			CgroupPath: "/glasshouse/fake",
			Namespaces: map[string]string{},
		},
		SupportedModes:  []profiling.Mode{profiling.ProfilingDisabled, profiling.ProfilingHost},
		SupportsProfile: true,
	}
}

func (b *Backend) Metadata() receipt.ExecutionInfo {
	return receipt.ExecutionInfo{Backend: b.Name(), Isolation: "none"}
}

func (b *Backend) ExtraErrors() []string {
	if len(b.Extra) == 0 {
		return nil
	}
	out := make([]string, len(b.Extra))
	copy(out, b.Extra)
	return out
}

var _ execution.ExecutionBackend = (*Backend)(nil)
var _ execution.ExtraErrorProvider = (*Backend)(nil)
var _ execution.MetadataProvider = (*Backend)(nil)
