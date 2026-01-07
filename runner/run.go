package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"glasshouse/audit"
)

type RunResult struct {
	Receipt  audit.Receipt
	ExitCode int
}

type RunOptions struct {
	Guest bool
}

func Run(ctx context.Context, cmdArgs []string, opts RunOptions) (RunResult, error) {
	if len(cmdArgs) == 0 {
		return RunResult{}, fmt.Errorf("no command provided")
	}

	extraErrors := []string{}
	if opts.Guest {
		extraErrors = append(extraErrors, setupGuestEnvironment()...)
	}

	start := time.Now()
	runCtx := ctx
	cancel := func() {}
	if opts.Guest {
		runCtx, cancel = context.WithCancel(ctx)
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, cmdArgs[0], cmdArgs[1:]...)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	workingDir, _ := os.Getwd()

	collector, collectErr := audit.NewCollector(audit.Config{})
	if collectErr != nil {
		fmt.Fprintln(os.Stderr, "glasshouse:", collectErr)
		extraErrors = append(extraErrors, fmt.Sprintf("collector: %v", collectErr))
		collector = nil
	}

	agg := audit.NewAggregator()
	if collector != nil {
		if err := collector.Start(runCtx); err != nil {
			fmt.Fprintln(os.Stderr, "glasshouse:", err)
			extraErrors = append(extraErrors, fmt.Sprintf("collector: %v", err))
			_ = collector.Close()
			collector = nil
		}
	}

	if err := cmd.Start(); err != nil {
		if collector != nil {
			_ = collector.Close()
		}
		receipt := agg.Receipt(1, time.Since(start))
		fillReceiptMeta(&receipt, start, 0, cmdArgs, workingDir, &stdoutBuf, &stderrBuf, err, extraErrors, audit.Resources{})
		return RunResult{Receipt: receipt, ExitCode: 1}, err
	}

	rootCmd := strings.Join(cmdArgs, " ")
	agg.SetRoot(uint32(cmd.Process.Pid), rootCmd)

	var (
		reapMu       sync.Mutex
		mainReaped   bool
		mainStatus   syscall.WaitStatus
		shutdownSign os.Signal
	)
	if opts.Guest {
		startGuestSignalHandler(runCtx, cmd.Process, &reapMu, &mainReaped, &mainStatus, &shutdownSign, cancel)
	}

	if collector != nil {
		go func() {
			for {
				select {
				case ev, ok := <-collector.Events():
					if !ok {
						return
					}
					agg.HandleEvent(ev)
				case err, ok := <-collector.Errors():
					if !ok {
						return
					}
					fmt.Fprintln(os.Stderr, "glasshouse:", err)
				}
			}
		}()
	}

	waitErr := cmd.Wait()
	exitCode := 0
	if waitErr != nil {
		exitCode = exitCodeForError(waitErr)
	}
	reapMu.Lock()
	if mainReaped {
		exitCode = exitCodeFromStatus(mainStatus)
		if waitErr != nil && isNoChildErr(waitErr) {
			waitErr = nil
		}
	}
	reapMu.Unlock()
	duration := time.Since(start)

	if collector != nil {
		_ = collector.Close()
	}

	resources := audit.Resources{}
	resourcesAvailable := false
	if ps := cmd.ProcessState; ps != nil {
		cpu := ps.UserTime() + ps.SystemTime()
		resources.CPUTimeMs = cpu.Milliseconds()
		resourcesAvailable = true

		if runtime.GOOS == "linux" {
			if usage, ok := ps.SysUsage().(*syscall.Rusage); ok {
				resources.MaxRSSKB = int64(usage.Maxrss)
			}
		}
	}

	receipt := agg.Receipt(exitCode, duration)
	if resourcesAvailable {
		receipt.Resources = &resources
	}
	if shutdownSign != nil {
		extraErrors = append(extraErrors, fmt.Sprintf("signal: %s", shutdownSign.String()))
	}
	fillReceiptMeta(&receipt, start, uint32(cmd.Process.Pid), cmdArgs, workingDir, &stdoutBuf, &stderrBuf, waitErr, extraErrors, resources)
	debugReceipt(&receipt)

	return RunResult{Receipt: receipt, ExitCode: exitCode}, waitErr
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

func fillReceiptMeta(receipt *audit.Receipt, start time.Time, pid uint32, cmdArgs []string, workingDir string, stdoutBuf, stderrBuf *bytes.Buffer, runErr error, extraErrors []string, resources audit.Resources) {
	receipt.ReceiptVersion = "v0.2"
	receipt.ExecutionID = executionID(start, pid, cmdArgs)
	receipt.Timestamp = start.UTC().Format(time.RFC3339Nano)

	exitCode := receipt.ExitCode
	errorStr := errorString(runErr)
	if len(extraErrors) > 0 {
		extra := strings.Join(extraErrors, "; ")
		if errorStr == nil {
			errorStr = &extra
		} else {
			combined := *errorStr + "; " + extra
			errorStr = &combined
		}
	}
	receipt.Outcome = &audit.Outcome{
		ExitCode: exitCode,
		Signal:   signalForError(runErr),
		Error:    errorStr,
	}

	receipt.Timing = &audit.Timing{
		DurationMs: receipt.DurationMs,
		CPUTimeMs:  resources.CPUTimeMs,
	}

	rootExe := resolveExe(cmdArgs)
	receipt.ProcessTree = buildProcessTree(receipt.Processes, pid, rootExe, cmdArgs, workingDir)

	receipt.Environment = &audit.Environment{
		Runtime: runtimeName(cmdArgs),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Sandbox: audit.Sandbox{Network: "enabled"},
	}

	receipt.Artifacts = &audit.Artifacts{
		StdoutHash: hashBytes(stdoutBuf.Bytes()),
		StderrHash: hashBytes(stderrBuf.Bytes()),
	}

	if receipt.Syscalls == nil {
		receipt.Syscalls = &audit.SyscallInfo{
			Counts: map[string]int{},
			Denied: []string{},
		}
	}
	if receipt.Filesystem == nil {
		receipt.Filesystem = &audit.FilesystemInfo{
			Reads:            []string{},
			Writes:           []string{},
			Deletes:          []string{},
			PolicyViolations: []string{},
		}
	}
	if receipt.Network == nil {
		receipt.Network = &audit.NetworkInfo{
			Attempts:      []audit.NetworkAttempt{},
			BytesSent:     0,
			BytesReceived: 0,
		}
	}
}

func executionID(start time.Time, pid uint32, cmdArgs []string) string {
	base := fmt.Sprintf("%d:%d:%s", start.UnixNano(), pid, strings.Join(cmdArgs, " "))
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func resolveExe(cmdArgs []string) string {
	if len(cmdArgs) == 0 {
		return ""
	}
	exe, err := exec.LookPath(cmdArgs[0])
	if err != nil {
		return cmdArgs[0]
	}
	return exe
}

func buildProcessTree(processes []audit.ProcessEntry, rootPID uint32, rootExe string, rootArgv []string, workingDir string) []audit.ProcessV2 {
	if len(processes) == 0 {
		return []audit.ProcessV2{}
	}
	out := make([]audit.ProcessV2, 0, len(processes))
	for _, proc := range processes {
		argv := argvFromCmd(proc.Cmd)
		exe := ""
		if len(argv) > 0 {
			exe = argv[0]
		}
		wd := ""
		if proc.PID == rootPID {
			if len(rootExe) > 0 {
				exe = rootExe
			}
			if len(rootArgv) > 0 {
				argv = append([]string(nil), rootArgv...)
			}
			wd = workingDir
		}
		out = append(out, audit.ProcessV2{
			PID:        proc.PID,
			PPID:       proc.PPID,
			Exe:        exe,
			Argv:       argv,
			WorkingDir: wd,
		})
	}
	return out
}

func argvFromCmd(cmd string) []string {
	if strings.TrimSpace(cmd) == "" {
		return []string{}
	}
	return strings.Fields(cmd)
}

func runtimeName(cmdArgs []string) string {
	if len(cmdArgs) == 0 {
		return "unknown"
	}
	base := filepath.Base(cmdArgs[0])
	if strings.HasPrefix(base, "python3") {
		return "python3.x"
	}
	if strings.HasPrefix(base, "python") {
		return "pythonx"
	}
	return base
}

func signalForError(err error) *string {
	if err == nil {
		return nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return nil
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return nil
	}
	sig := status.Signal().String()
	return &sig
}

func errorString(err error) *string {
	if err == nil {
		return nil
	}
	msg := err.Error()
	return &msg
}

func debugReceipt(receipt *audit.Receipt) {
	if !isTruthyEnv("GLASSHOUSE_DEBUG_COUNTS") {
		return
	}
	counts := map[string]int{}
	if receipt.Syscalls != nil {
		counts = receipt.Syscalls.Counts
	}
	reads := 0
	writes := 0
	if receipt.Filesystem != nil {
		reads = len(receipt.Filesystem.Reads)
		writes = len(receipt.Filesystem.Writes)
	}
	attempts := 0
	if receipt.Network != nil {
		attempts = len(receipt.Network.Attempts)
	}
	fmt.Fprintf(os.Stderr, "glasshouse: syscalls=%v reads=%d writes=%d network_attempts=%d\n", counts, reads, writes, attempts)
}

func isTruthyEnv(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
