package runner

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"glasshouse/audit"
	"glasshouse/backend"
)

type fakeBackend struct {
	calls       []string
	pid         int
	exitCode    int
	prepareErr  error
	startErr    error
	waitErr     error
	cleanupErr  error
	metadata    backend.BackendMetadata
	stdoutBytes []byte
	stderrBytes []byte
	extraErrors []string
}

func (f *fakeBackend) Prepare(ctx context.Context) error {
	f.calls = append(f.calls, "prepare")
	return f.prepareErr
}

func (f *fakeBackend) Start(ctx context.Context, cmd []string) (int, error) {
	f.calls = append(f.calls, "start")
	if f.startErr != nil {
		return 0, f.startErr
	}
	return f.pid, nil
}

func (f *fakeBackend) Wait(ctx context.Context) (int, error) {
	f.calls = append(f.calls, "wait")
	return f.exitCode, f.waitErr
}

func (f *fakeBackend) Cleanup(ctx context.Context) error {
	f.calls = append(f.calls, "cleanup")
	return f.cleanupErr
}

func (f *fakeBackend) Metadata() backend.BackendMetadata {
	return f.metadata
}

func (f *fakeBackend) ExtraErrors() []string {
	return f.extraErrors
}

func (f *fakeBackend) Stdout() []byte {
	return f.stdoutBytes
}

func (f *fakeBackend) Stderr() []byte {
	return f.stderrBytes
}

var _ backend.Backend = (*fakeBackend)(nil)
var _ backend.ExtraErrorProvider = (*fakeBackend)(nil)
var _ backend.OutputProvider = (*fakeBackend)(nil)

func TestRunCallsBackendInOrder(t *testing.T) {
	fb := &fakeBackend{
		pid:         1234,
		exitCode:    0,
		metadata:    backend.BackendMetadata{Backend: "fake", Isolation: "none"},
		stdoutBytes: []byte("out"),
		stderrBytes: []byte("err"),
	}

	result, err := Run(context.Background(), []string{"/bin/true"}, fb)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	wantCalls := []string{"prepare", "start", "wait", "cleanup"}
	if !reflect.DeepEqual(fb.calls, wantCalls) {
		t.Fatalf("calls %v, want %v", fb.calls, wantCalls)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d", result.ExitCode)
	}
	if result.Receipt.Execution == nil {
		t.Fatal("missing execution metadata")
	}
	if result.Receipt.Execution.Backend != "fake" || result.Receipt.Execution.Isolation != "none" {
		t.Fatalf("execution metadata %+v", result.Receipt.Execution)
	}
}

func TestRunCapturesBackendErrors(t *testing.T) {
	fb := &fakeBackend{
		pid:         1,
		exitCode:    0,
		prepareErr:  backend.ErrorList{Errors: []string{"prep failed"}},
		cleanupErr:  errors.New("cleanup failed"),
		extraErrors: []string{"extra error"},
		metadata:    backend.BackendMetadata{Backend: "fake", Isolation: "none"},
	}

	result, err := Run(context.Background(), []string{"/bin/true"}, fb)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Receipt.Outcome == nil || result.Receipt.Outcome.Error == nil {
		t.Fatal("missing outcome error")
	}
	errStr := *result.Receipt.Outcome.Error
	for _, want := range []string{"prep failed", "backend: cleanup failed", "extra error"} {
		if !strings.Contains(errStr, want) {
			t.Fatalf("error %q missing %q", errStr, want)
		}
	}
}

func TestRunExitCodeFromBackend(t *testing.T) {
	fb := &fakeBackend{
		pid:      42,
		exitCode: 7,
		metadata: backend.BackendMetadata{Backend: "fake", Isolation: "none"},
	}

	result, err := Run(context.Background(), []string{"/bin/true"}, fb)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("exit code %d", result.ExitCode)
	}
}

func TestRunReturnsReceiptOnStartError(t *testing.T) {
	fb := &fakeBackend{
		startErr: errors.New("boom"),
		metadata: backend.BackendMetadata{Backend: "fake", Isolation: "none"},
	}

	result, err := Run(context.Background(), []string{"/bin/true"}, fb)
	if err == nil {
		t.Fatal("expected error")
	}
	if result.Receipt.Execution == nil {
		t.Fatal("missing execution metadata")
	}
	wantCalls := []string{"prepare", "start", "cleanup"}
	if !reflect.DeepEqual(fb.calls, wantCalls) {
		t.Fatalf("calls %v, want %v", fb.calls, wantCalls)
	}
}

func TestExecutionIDDeterministic(t *testing.T) {
	start := time.Unix(1700000000, 1234)
	args := []string{"/bin/echo", "hello"}
	first := executionID(start, 100, args)
	second := executionID(start, 100, args)
	if first != second {
		t.Fatalf("execution id mismatch: %s vs %s", first, second)
	}
	third := executionID(start, 101, args)
	if first == third {
		t.Fatalf("execution id should differ for different pid")
	}
}

func TestReceiptFieldsPresent(t *testing.T) {
	agg := audit.NewAggregator()
	receipt := agg.Receipt(0, 0)
	receipt.Execution = &audit.ExecutionInfo{Backend: "fake", Isolation: "none"}
	if receipt.Execution == nil {
		t.Fatal("missing execution")
	}
	if receipt.Filesystem == nil || receipt.Network == nil {
		t.Fatal("missing filesystem or network")
	}
}
