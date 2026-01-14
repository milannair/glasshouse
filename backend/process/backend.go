package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

type Options struct {
	Guest  bool
	Stdout io.Writer
	Stderr io.Writer
}

type Backend struct {
	opts          Options
	cmd           *exec.Cmd
	signalCancel  context.CancelFunc
	handleSignals bool
	stdoutBuf     bytes.Buffer
	stderrBuf     bytes.Buffer

	reapMu       sync.Mutex
	mainReaped   bool
	mainStatus   syscall.WaitStatus
	shutdownSign os.Signal
}

func New(opts Options) *Backend {
	return &Backend{opts: opts}
}

func (b *Backend) Name() string { return "process" }

func (b *Backend) Prepare(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !(b.opts.Guest) {
		return nil
	}
	errs := setupGuestEnvironment()
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errs, "; "))
}

func (b *Backend) Start(spec execution.ExecutionSpec) (execution.ExecutionHandle, error) {
	if len(spec.Args) == 0 {
		return execution.ExecutionHandle{}, fmt.Errorf("no command provided")
	}

	b.handleSignals = b.opts.Guest || spec.Guest || os.Getpid() == 1
	signalCtx := context.Background()
	if b.handleSignals {
		signalCtx, b.signalCancel = context.WithCancel(signalCtx)
	} else {
		b.signalCancel = func() {}
	}

	cmd := exec.CommandContext(signalCtx, spec.Args[0], spec.Args[1:]...)
	if spec.Workdir != "" {
		cmd.Dir = spec.Workdir
	}
	if len(spec.Env) > 0 {
		cmd.Env = spec.Env
	}
	stdout := b.opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := b.opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	cmd.Stdout = io.MultiWriter(stdout, &b.stdoutBuf)
	cmd.Stderr = io.MultiWriter(stderr, &b.stderrBuf)

	if err := cmd.Start(); err != nil {
		b.signalCancel()
		return execution.ExecutionHandle{}, err
	}
	b.cmd = cmd

	if b.handleSignals {
		startGuestSignalHandler(signalCtx, cmd.Process, &b.reapMu, &b.mainReaped, &b.mainStatus, &b.shutdownSign)
	}

	handle := execution.ExecutionHandle{
		ID:            fmt.Sprintf("pid-%d", cmd.Process.Pid),
		BackendHandle: cmd.Process,
	}
	return handle, nil
}

func (b *Backend) Wait(h execution.ExecutionHandle) (execution.ExecutionResult, error) {
	_ = h
	if b.cmd == nil {
		return execution.ExecutionResult{}, fmt.Errorf("backend not started")
	}

	waitErr := b.cmd.Wait()
	exitCode := 0
	if waitErr != nil {
		exitCode = exitCodeForError(waitErr)
	}

	b.reapMu.Lock()
	if b.mainReaped {
		exitCode = exitCodeFromStatus(b.mainStatus)
		if waitErr != nil && isNoChildErr(waitErr) {
			waitErr = nil
		}
	}
	b.reapMu.Unlock()

	return execution.ExecutionResult{
		Handle:      h,
		ExitCode:    exitCode,
		Err:         waitErr,
		StartedAt:   time.Now(), // placeholder; engine stamps authoritative time
		CompletedAt: time.Now(),
	}, waitErr
}

func (b *Backend) Kill(h execution.ExecutionHandle) error {
	_ = h
	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Kill()
	}
	return nil
}

func (b *Backend) Cleanup(h execution.ExecutionHandle) error {
	_ = h
	if b.signalCancel != nil {
		b.signalCancel()
	}
	return nil
}

func (b *Backend) ProfilingInfo(h execution.ExecutionHandle) execution.BackendProfilingInfo {
	rootPID := 0
	if b.cmd != nil && b.cmd.Process != nil {
		rootPID = b.cmd.Process.Pid
	}
	return execution.BackendProfilingInfo{
		Identity: execution.ExecutionIdentity{
			RootPID:    rootPID,
			CgroupPath: "",
			Namespaces: map[string]string{},
		},
		SupportedModes: []profiling.Mode{
			profiling.ProfilingDisabled,
			profiling.ProfilingHost,
		},
		SupportsProfile: true,
	}
}

func (b *Backend) ExtraErrors() []string {
	if b.shutdownSign == nil {
		return nil
	}
	return []string{fmt.Sprintf("signal: %s", b.shutdownSign.String())}
}

func (b *Backend) ProcessState() *os.ProcessState {
	if b.cmd == nil {
		return nil
	}
	return b.cmd.ProcessState
}

func (b *Backend) Stdout() []byte { return b.stdoutBuf.Bytes() }
func (b *Backend) Stderr() []byte { return b.stderrBuf.Bytes() }

func (b *Backend) Metadata() receipt.ExecutionInfo {
	isolation := "none"
	if b.opts.Guest {
		isolation = "namespace"
	}
	return receipt.ExecutionInfo{
		Backend:   b.Name(),
		Isolation: isolation,
	}
}

func exitCodeForError(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 1
}

func exitCodeFromStatus(status syscall.WaitStatus) int {
	if status.Exited() {
		return status.ExitStatus()
	}
	if status.Signaled() {
		return 128 + int(status.Signal())
	}
	return 1
}

func isNoChildErr(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == syscall.ECHILD {
		return true
	}
	var sysErr *os.SyscallError
	if errors.As(err, &sysErr) && sysErr.Err == syscall.ECHILD {
		return true
	}
	return false
}

var _ execution.ExecutionBackend = (*Backend)(nil)
var _ execution.OutputProvider = (*Backend)(nil)
var _ execution.ExtraErrorProvider = (*Backend)(nil)
var _ execution.ProcessStateProvider = (*Backend)(nil)
var _ execution.MetadataProvider = (*Backend)(nil)
