package firecracker

import (
	"context"
	"errors"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

var ErrFirecrackerNotImplemented = errors.New("firecracker backend not implemented")

type Backend struct {
	cfg Config
}

func New(cfg Config) *Backend {
	return &Backend{cfg: cfg}
}

func (b *Backend) Name() string { return "firecracker" }

func (b *Backend) Prepare(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return b.cfg.Validate()
}

func (b *Backend) Start(spec execution.ExecutionSpec) (execution.ExecutionHandle, error) {
	return execution.ExecutionHandle{}, ErrFirecrackerNotImplemented
}

func (b *Backend) Wait(h execution.ExecutionHandle) (execution.ExecutionResult, error) {
	return execution.ExecutionResult{Handle: h, ExitCode: 0}, nil
}

func (b *Backend) Kill(h execution.ExecutionHandle) error { return nil }

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
			profiling.ProfilingHost,
			profiling.ProfilingGuest,
			profiling.ProfilingCombined,
		},
		SupportsProfile: false,
	}
}

func (b *Backend) Metadata() receipt.ExecutionInfo {
	return receipt.ExecutionInfo{Backend: b.Name(), Isolation: "vm"}
}

var _ execution.ExecutionBackend = (*Backend)(nil)
var _ execution.MetadataProvider = (*Backend)(nil)
