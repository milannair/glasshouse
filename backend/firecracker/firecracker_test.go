package firecracker

import (
	"testing"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
)

func TestFirecrackerBackendMetadata(t *testing.T) {
	cfg := Config{KernelImagePath: "kernel", RootFSPath: "rootfs", BinaryPath: "firecracker"}
	b := New(cfg)

	meta := b.Metadata()
	if meta.Backend != "firecracker" {
		t.Fatalf("backend metadata %q", meta.Backend)
	}
	if meta.Isolation != "vm" {
		t.Fatalf("isolation metadata %q", meta.Isolation)
	}

	// Profiling info test with empty handle
	info := b.ProfilingInfo(emptyHandle())
	if info.SupportsProfile {
		t.Fatalf("profiling should be disabled by default")
	}
	if len(info.SupportedModes) == 0 || info.SupportedModes[0] != profiling.ProfilingDisabled {
		t.Fatalf("unexpected supported modes %#v", info.SupportedModes)
	}
}

func TestConfigValidation(t *testing.T) {
	// Missing kernel
	cfg := Config{RootFSPath: "rootfs"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing kernel")
	}

	// Missing rootfs
	cfg = Config{KernelImagePath: "kernel"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing rootfs")
	}

	// Valid config
	cfg = Config{KernelImagePath: "kernel", RootFSPath: "rootfs"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func emptyHandle() execution.ExecutionHandle {
	return execution.ExecutionHandle{}
}
