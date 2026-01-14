package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type ProcessOptions struct {
	Guest  bool
	Stdout io.Writer
	Stderr io.Writer
}

type processBackend struct {
	opts          ProcessOptions
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

func NewProcessBackend(opts ProcessOptions) Backend {
	return &processBackend{opts: opts}
}

func (b *processBackend) Prepare(ctx context.Context) error {
	if !b.opts.Guest {
		return nil
	}
	errs := setupGuestEnvironment()
	if len(errs) == 0 {
		return nil
	}
	return ErrorList{Errors: errs}
}

func (b *processBackend) Start(ctx context.Context, cmdArgs []string) (int, error) {
	b.handleSignals = b.opts.Guest || os.Getpid() == 1
	signalCtx := ctx
	if b.handleSignals {
		signalCtx, b.signalCancel = context.WithCancel(ctx)
	} else {
		b.signalCancel = func() {}
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
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
		return 0, err
	}
	b.cmd = cmd

	if b.handleSignals {
		startGuestSignalHandler(signalCtx, cmd.Process, &b.reapMu, &b.mainReaped, &b.mainStatus, &b.shutdownSign)
	}

	return cmd.Process.Pid, nil
}

func (b *processBackend) Wait(ctx context.Context) (int, error) {
	if b.cmd == nil {
		return 1, fmt.Errorf("backend not started")
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

	return exitCode, waitErr
}

func (b *processBackend) Cleanup(ctx context.Context) error {
	if b.signalCancel != nil {
		b.signalCancel()
	}
	return nil
}

func (b *processBackend) Metadata() BackendMetadata {
	return BackendMetadata{Backend: "process", Isolation: "none"}
}

func (b *processBackend) ExtraErrors() []string {
	if b.shutdownSign == nil {
		return nil
	}
	return []string{fmt.Sprintf("signal: %s", b.shutdownSign.String())}
}

func (b *processBackend) ProcessState() *os.ProcessState {
	if b.cmd == nil {
		return nil
	}
	return b.cmd.ProcessState
}

func (b *processBackend) Stdout() []byte {
	return b.stdoutBuf.Bytes()
}

func (b *processBackend) Stderr() []byte {
	return b.stderrBuf.Bytes()
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

var _ Backend = (*processBackend)(nil)
