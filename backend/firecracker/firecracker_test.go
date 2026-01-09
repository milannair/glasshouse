package firecracker

import (
	"context"
	"errors"
	"testing"
)

func TestFirecrackerBackendSkeleton(t *testing.T) {
	cfg := Config{KernelImagePath: "kernel", RootFSPath: "rootfs", BinaryPath: "firecracker"}
	b := New(cfg)
	if err := b.Prepare(context.Background()); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	pid, err := b.Start(context.Background(), []string{"/bin/true"})
	if !errors.Is(err, ErrFirecrackerNotImplemented) {
		t.Fatalf("Start error %v", err)
	}
	if pid != 0 {
		t.Fatalf("expected pid 0, got %d", pid)
	}
	if exitCode, err := b.Wait(context.Background()); err != nil || exitCode != 0 {
		t.Fatalf("Wait exitCode=%d err=%v", exitCode, err)
	}
	if err := b.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	meta := b.Metadata()
	if meta.Backend != "firecracker" {
		t.Fatalf("backend metadata %q", meta.Backend)
	}
	if meta.Isolation != "vm" {
		t.Fatalf("isolation metadata %q", meta.Isolation)
	}
}
