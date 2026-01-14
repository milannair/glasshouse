package process_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"glasshouse/backend/process"
	"glasshouse/core/execution"
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
			b := process.New(process.Options{Stdout: io.Discard, Stderr: io.Discard})
			if err := b.Prepare(ctx); err != nil {
				t.Fatalf("Prepare: %v", err)
			}
			handle, err := b.Start(execution.ExecutionSpec{Args: tc.cmd})
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			if handle.ID == "" {
				t.Fatalf("missing handle id")
			}
			waitRes, err := b.Wait(handle)
			if err != nil && tc.wantCode == 0 {
				t.Fatalf("Wait: %v", err)
			}
			if waitRes.ExitCode != tc.wantCode {
				t.Fatalf("exit code %d, want %d", waitRes.ExitCode, tc.wantCode)
			}
			if err := b.Cleanup(handle); err != nil {
				t.Fatalf("Cleanup: %v", err)
			}

			meta := b.Metadata()
			if meta.Backend != "process" {
				t.Fatalf("backend metadata %q", meta.Backend)
			}
			if meta.Isolation == "" {
				t.Fatalf("isolation metadata %q", meta.Isolation)
			}
		})
	}
}

func TestProcessBackendCapturesOutput(t *testing.T) {
	requireCommand(t, "/bin/echo")

	ctx := context.Background()
	b := process.New(process.Options{Stdout: io.Discard, Stderr: io.Discard})
	if err := b.Prepare(ctx); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	handle, err := b.Start(execution.ExecutionSpec{Args: []string{"/bin/echo", "hello"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, err = b.Wait(handle)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if err := b.Cleanup(handle); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	outputProvider, ok := interface{}(b).(execution.OutputProvider)
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
	b := process.New(process.Options{Stdout: io.Discard, Stderr: io.Discard})
	if err := b.Prepare(ctx); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	handle, err := b.Start(execution.ExecutionSpec{Args: []string{"/bin/sh", "-c", "echo err 1>&2"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, err = b.Wait(handle)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if err := b.Cleanup(handle); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	outputProvider, ok := interface{}(b).(execution.OutputProvider)
	if !ok {
		t.Fatal("backend does not implement OutputProvider")
	}
	if len(outputProvider.Stderr()) == 0 {
		t.Fatal("expected stderr output")
	}
}
