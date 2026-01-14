package manager

import (
	"context"
	"testing"

	"glasshouse/backend/fake"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
	"glasshouse/node/config"
	"glasshouse/node/enforcement"
	"glasshouse/node/registry"
)

type recordingEnforcer struct {
	calls int
}

func (r *recordingEnforcer) Enforce(ctx context.Context, rec *receipt.Receipt) error {
	_ = ctx
	_ = rec
	r.calls++
	return nil
}

func TestManagerRunsRequest(t *testing.T) {
	reg := registry.New()
	reg.Register("fake", fake.New(0))

	cfg := config.Defaults()
	cfg.DefaultBackend = "fake"
	mgr, err := New(cfg, reg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mgr.Profiler = stubProfiler{}

	resp, err := mgr.Run(context.Background(), Request{
		Args:      []string{"/bin/true"},
		Profiling: profiling.ProfilingDisabled,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Result.ExitCode != 0 {
		t.Fatalf("exit code %d", resp.Result.ExitCode)
	}
}

type stubProfiler struct{}

func (s stubProfiler) Start(ctx context.Context, target profiling.Target) (profiling.Session, error) {
	return stubSession{}, nil
}
func (s stubProfiler) Capabilities() profiling.Capabilities {
	return profiling.Capabilities{Host: true}
}

type stubSession struct{}

func (stubSession) Events() <-chan profiling.Event {
	ch := make(chan profiling.Event)
	close(ch)
	return ch
}
func (stubSession) Errors() <-chan error { ch := make(chan error); close(ch); return ch }
func (stubSession) Close() error         { return nil }

var _ enforcement.Enforcer = (*recordingEnforcer)(nil)
