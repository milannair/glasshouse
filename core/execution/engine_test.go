package execution

import (
	"context"
	"testing"

	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

type stubProfiler struct{}

type stubSession struct {
	events chan profiling.Event
	errs   chan error
}

func (s stubProfiler) Start(ctx context.Context, target profiling.Target) (profiling.Session, error) {
	_ = ctx
	_ = target
	ev := make(chan profiling.Event)
	errs := make(chan error)
	close(ev)
	close(errs)
	return stubSession{events: ev, errs: errs}, nil
}

func (s stubProfiler) Capabilities() profiling.Capabilities {
	return profiling.Capabilities{Host: true}
}

func (s stubSession) Events() <-chan profiling.Event { return s.events }
func (s stubSession) Errors() <-chan error           { return s.errs }
func (s stubSession) Close() error                   { return nil }

func TestEngineBuildsReceiptWhenProfilingEnabled(t *testing.T) {
	engine := Engine{
		Backend:  &testBackend{exitCode: 0},
		Profiler: stubProfiler{},
	}
	spec := ExecutionSpec{
		Args:      []string{"/bin/true"},
		Profiling: profiling.ProfilingHost,
	}

	result, err := engine.Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !result.ProfilingAttached {
		t.Fatalf("expected profiling attachment")
	}
	if result.Receipt == nil {
		t.Fatalf("expected receipt when profiling enabled")
	}
	if result.Receipt.Execution == nil || result.Receipt.Execution.Backend != "test" {
		t.Fatalf("missing execution metadata: %+v", result.Receipt.Execution)
	}
}

func TestEngineSkipsReceiptWhenProfilingDisabled(t *testing.T) {
	engine := Engine{
		Backend:  &testBackend{exitCode: 0},
		Profiler: stubProfiler{},
	}
	spec := ExecutionSpec{
		Args:      []string{"/bin/true"},
		Profiling: profiling.ProfilingDisabled,
	}

	result, err := engine.Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if result.Receipt != nil {
		t.Fatalf("expected no receipt when profiling disabled")
	}
	if result.ProfilingAttached {
		t.Fatalf("profiling should not attach when disabled")
	}
}

func TestEnginePropagatesStartError(t *testing.T) {
	b := &testBackend{exitCode: 0, startErr: context.Canceled}
	engine := Engine{
		Backend:  b,
		Profiler: stubProfiler{},
	}
	spec := ExecutionSpec{
		Args:      []string{"/bin/true"},
		Profiling: profiling.ProfilingHost,
	}
	_, err := engine.Run(context.Background(), spec)
	if err == nil {
		t.Fatal("expected error")
	}
}

type testBackend struct {
	exitCode int
	startErr error
	waitErr  error
}

func (t *testBackend) Name() string { return "test" }
func (t *testBackend) Prepare(ctx context.Context) error {
	return ctx.Err()
}
func (t *testBackend) Start(spec ExecutionSpec) (ExecutionHandle, error) {
	_ = spec
	return ExecutionHandle{ID: "test"}, t.startErr
}
func (t *testBackend) Wait(h ExecutionHandle) (ExecutionResult, error) {
	res := ExecutionResult{Handle: h, ExitCode: t.exitCode, Err: t.waitErr}
	return res, t.waitErr
}
func (t *testBackend) Kill(h ExecutionHandle) error {
	_ = h
	return nil
}
func (t *testBackend) Cleanup(h ExecutionHandle) error {
	_ = h
	return nil
}
func (t *testBackend) ProfilingInfo(h ExecutionHandle) BackendProfilingInfo {
	_ = h
	return BackendProfilingInfo{
		Identity: ExecutionIdentity{
			RootPID:    4242,
			CgroupPath: "/glasshouse/test",
			Namespaces: map[string]string{},
		},
		SupportedModes:  []profiling.Mode{profiling.ProfilingHost, profiling.ProfilingDisabled},
		SupportsProfile: true,
	}
}
func (t *testBackend) Metadata() receipt.ExecutionInfo {
	return receipt.ExecutionInfo{Backend: "test", Isolation: "none"}
}

var _ ExecutionBackend = (*testBackend)(nil)
