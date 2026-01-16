package firecracker

import (
	"context"
	"errors"
	"testing"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
)

func TestFirecrackerBackendSkeleton(t *testing.T) {
	cfg := Config{KernelImagePath: "kernel", RootFSPath: "rootfs", BinaryPath: "firecracker"}
	b := New(cfg)
	if err := b.Prepare(context.Background()); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	handle, err := b.Start(execution.ExecutionSpec{Args: []string{"/bin/true"}})
	if !errors.Is(err, ErrFirecrackerNotImplemented) {
		t.Fatalf("Start error %v", err)
	}
	if handle.ID != "" {
		t.Fatalf("expected empty handle id, got %q", handle.ID)
	}
	if res, err := b.Wait(handle); err != nil || res.ExitCode != 0 {
		t.Fatalf("Wait exitCode=%d err=%v", res.ExitCode, err)
	}
	if err := b.Cleanup(handle); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	meta := b.Metadata()
	if meta.Backend != "firecracker" {
		t.Fatalf("backend metadata %q", meta.Backend)
	}
	if meta.Isolation != "vm" {
		t.Fatalf("isolation metadata %q", meta.Isolation)
	}

	info := b.ProfilingInfo(handle)
	if info.SupportsProfile {
		t.Fatalf("profiling should be disabled by default until implemented")
	}
	if len(info.SupportedModes) == 0 || info.SupportedModes[0] != profiling.ProfilingDisabled {
		t.Fatalf("unexpected supported modes %#v", info.SupportedModes)
	}
}
