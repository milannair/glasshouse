package kata

import (
	"context"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

// Backend is a placeholder for Kata Containers integration.
type Backend struct{}

func New() *Backend { return &Backend{} }

func (b *Backend) Name() string { return "kata" }

func (b *Backend) Prepare(ctx context.Context) error {
	return ctx.Err()
}

func (b *Backend) Start(spec execution.ExecutionSpec) (execution.ExecutionHandle, error) {
	_ = spec
	return execution.ExecutionHandle{}, nil
}

func (b *Backend) Wait(h execution.ExecutionHandle) (execution.ExecutionResult, error) {
	return execution.ExecutionResult{Handle: h}, nil
}

func (b *Backend) Kill(h execution.ExecutionHandle) error {
	_ = h
	return nil
}

func (b *Backend) Cleanup(h execution.ExecutionHandle) error {
	_ = h
	return nil
}

func (b *Backend) ProfilingInfo(h execution.ExecutionHandle) execution.BackendProfilingInfo {
	_ = h
	return execution.BackendProfilingInfo{
		Identity: execution.ExecutionIdentity{
			RootPID:    0,
			CgroupPath: "",
			Namespaces: map[string]string{},
		},
		SupportedModes: []profiling.Mode{
			profiling.ProfilingDisabled,
		},
		SupportsProfile: false,
	}
}

func (b *Backend) Metadata() receipt.ExecutionInfo {
	return receipt.ExecutionInfo{Backend: b.Name(), Isolation: "vm"}
}

var _ execution.ExecutionBackend = (*Backend)(nil)
var _ execution.MetadataProvider = (*Backend)(nil)
