package process_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"glasshouse/backend"
)

func requireCommand(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("requires linux")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("missing %s", path)
	}
}

func TestProcessBackendExitCodes(t *testing.T) {
	requireCommand(t, "/bin/true")
	requireCommand(t, "/bin/false")

	ctx := context.Background()
	cases := []struct {
		name     string
		cmd      []string
		wantCode int
	}{
		{name: "true", cmd: []string{"/bin/true"}, wantCode: 0},
		{name: "false", cmd: []string{"/bin/false"}, wantCode: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := backend.NewProcessBackend(backend.ProcessOptions{Stdout: io.Discard, Stderr: io.Discard})
			if err := b.Prepare(ctx); err != nil {
				t.Fatalf("Prepare: %v", err)
			}
			pid, err := b.Start(ctx, tc.cmd)
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			if pid <= 0 {
				t.Fatalf("invalid pid: %d", pid)
			}
			exitCode, err := b.Wait(ctx)
			if err != nil && tc.wantCode == 0 {
				t.Fatalf("Wait: %v", err)
			}
			if exitCode != tc.wantCode {
				t.Fatalf("exit code %d, want %d", exitCode, tc.wantCode)
			}
			if err := b.Cleanup(ctx); err != nil {
				t.Fatalf("Cleanup: %v", err)
			}

			meta := b.Metadata()
			if meta.Backend != "process" {
				t.Fatalf("backend metadata %q", meta.Backend)
			}
			if meta.Isolation != "none" {
				t.Fatalf("isolation metadata %q", meta.Isolation)
			}
		})
	}
}

func TestProcessBackendCapturesOutput(t *testing.T) {
	requireCommand(t, "/bin/echo")

	ctx := context.Background()
	b := backend.NewProcessBackend(backend.ProcessOptions{Stdout: io.Discard, Stderr: io.Discard})
	if err := b.Prepare(ctx); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	_, err := b.Start(ctx, []string{"/bin/echo", "hello"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, err = b.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if err := b.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	outputProvider, ok := b.(backend.OutputProvider)
	if !ok {
		t.Fatal("backend does not implement OutputProvider")
	}
	stdout := string(outputProvider.Stdout())
	if !strings.Contains(stdout, "hello") {
		t.Fatalf("stdout missing hello: %q", stdout)
	}
}

func TestProcessBackendCapturesStderr(t *testing.T) {
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}

	ctx := context.Background()
	b := backend.NewProcessBackend(backend.ProcessOptions{Stdout: io.Discard, Stderr: io.Discard})
	if err := b.Prepare(ctx); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	_, err := b.Start(ctx, []string{"/bin/sh", "-c", "echo err 1>&2"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, err = b.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if err := b.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	outputProvider, ok := b.(backend.OutputProvider)
	if !ok {
		t.Fatal("backend does not implement OutputProvider")
	}
	if len(outputProvider.Stderr()) == 0 {
		t.Fatal("expected stderr output")
	}
}
