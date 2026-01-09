package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"glasshouse/audit"
	"glasshouse/backend"
)

type RunResult struct {
	Receipt  audit.Receipt
	ExitCode int
}

func Run(ctx context.Context, cmdArgs []string, execBackend backend.Backend) (RunResult, error) {
	if len(cmdArgs) == 0 {
		return RunResult{}, fmt.Errorf("no command provided")
	}

	if execBackend == nil {
		return RunResult{}, fmt.Errorf("no backend provided")
	}
	extraErrors := []string{}
	if err := execBackend.Prepare(ctx); err != nil {
		extraErrors = appendBackendErrors(extraErrors, err)
	}

	start := time.Now()
	workingDir, _ := os.Getwd()

	collector, collectErr := audit.NewCollector(audit.Config{})
	if collectErr != nil {
		fmt.Fprintln(os.Stderr, "glasshouse:", collectErr)
		extraErrors = append(extraErrors, fmt.Sprintf("collector: %v", collectErr))
		collector = nil
	}

	agg := audit.NewAggregator()
	if collector != nil {
		if err := collector.Start(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "glasshouse:", err)
			extraErrors = append(extraErrors, fmt.Sprintf("collector: %v", err))
			_ = collector.Close()
			collector = nil
		}
	}

	rootPID, err := execBackend.Start(ctx, cmdArgs)
	if err != nil {
		if collector != nil {
			_ = collector.Close()
		}
		if cleanupErr := execBackend.Cleanup(ctx); cleanupErr != nil {
			extraErrors = append(extraErrors, fmt.Sprintf("backend: %v", cleanupErr))
		}
		receipt := agg.Receipt(1, time.Since(start))
		stdoutBytes, stderrBytes := backendOutput(execBackend)
		fillReceiptMeta(&receipt, start, 0, cmdArgs, workingDir, stdoutBytes, stderrBytes, err, extraErrors, audit.Resources{}, execBackend.Metadata())
		return RunResult{Receipt: receipt, ExitCode: 1}, err
	}

	rootCmd := strings.Join(cmdArgs, " ")
	agg.SetRoot(uint32(rootPID), rootCmd)

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

	exitCode, waitErr := execBackend.Wait(ctx)
	duration := time.Since(start)

	if collector != nil {
		_ = collector.Close()
	}

	if cleanupErr := execBackend.Cleanup(ctx); cleanupErr != nil {
		extraErrors = append(extraErrors, fmt.Sprintf("backend: %v", cleanupErr))
	}
	if extraProvider, ok := execBackend.(backend.ExtraErrorProvider); ok {
		extraErrors = append(extraErrors, extraProvider.ExtraErrors()...)
	}

	resources := audit.Resources{}
	resourcesAvailable := false
	if psProvider, ok := execBackend.(backend.ProcessStateProvider); ok {
		if ps := psProvider.ProcessState(); ps != nil {
			cpu := ps.UserTime() + ps.SystemTime()
			resources.CPUTimeMs = cpu.Milliseconds()
			resourcesAvailable = true

			if runtime.GOOS == "linux" {
				if usage, ok := ps.SysUsage().(*syscall.Rusage); ok {
					resources.MaxRSSKB = int64(usage.Maxrss)
				}
			}
		}
	}

	receipt := agg.Receipt(exitCode, duration)
	if resourcesAvailable {
		receipt.Resources = &resources
	}
	stdoutBytes, stderrBytes := backendOutput(execBackend)
	fillReceiptMeta(&receipt, start, uint32(rootPID), cmdArgs, workingDir, stdoutBytes, stderrBytes, waitErr, extraErrors, resources, execBackend.Metadata())
	debugReceipt(&receipt)

	return RunResult{Receipt: receipt, ExitCode: exitCode}, waitErr
}

func appendBackendErrors(extraErrors []string, err error) []string {
	if err == nil {
		return extraErrors
	}
	if list, ok := err.(backend.ErrorList); ok {
		return append(extraErrors, list.Errors...)
	}
	if list, ok := err.(*backend.ErrorList); ok {
		return append(extraErrors, list.Errors...)
	}
	return append(extraErrors, err.Error())
}

func backendOutput(execBackend backend.Backend) ([]byte, []byte) {
	if execBackend == nil {
		return nil, nil
	}
	if outputProvider, ok := execBackend.(backend.OutputProvider); ok {
		return outputProvider.Stdout(), outputProvider.Stderr()
	}
	return nil, nil
}

func fillReceiptMeta(receipt *audit.Receipt, start time.Time, pid uint32, cmdArgs []string, workingDir string, stdoutBytes, stderrBytes []byte, runErr error, extraErrors []string, resources audit.Resources, metadata backend.BackendMetadata) {
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

	receipt.Execution = &audit.ExecutionInfo{
		Backend:   metadata.Backend,
		Isolation: metadata.Isolation,
	}

	receipt.Artifacts = &audit.Artifacts{
		StdoutHash: hashBytes(stdoutBytes),
		StderrHash: hashBytes(stderrBytes),
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
