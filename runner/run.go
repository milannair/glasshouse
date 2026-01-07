package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"glasshouse/audit"
)

type RunResult struct {
	Receipt  audit.Receipt
	ExitCode int
}

func Run(ctx context.Context, cmdArgs []string) (RunResult, error) {
	if len(cmdArgs) == 0 {
		return RunResult{}, fmt.Errorf("no command provided")
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	collector, collectErr := audit.NewCollector(audit.Config{})
	if collectErr != nil {
		fmt.Fprintln(os.Stderr, "glasshouse:", collectErr)
		collector = nil
	}

	agg := audit.NewAggregator()
	if collector != nil {
		if err := collector.Start(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "glasshouse:", err)
			_ = collector.Close()
			collector = nil
		}
	}

	if err := cmd.Start(); err != nil {
		if collector != nil {
			_ = collector.Close()
		}
		receipt := agg.Receipt(1, time.Since(start))
		return RunResult{Receipt: receipt, ExitCode: 1}, err
	}

	rootCmd := strings.Join(cmdArgs, " ")
	agg.SetRoot(uint32(cmd.Process.Pid), rootCmd)

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
	duration := time.Since(start)

	if collector != nil {
		_ = collector.Close()
	}

	exitCode := 0
	if waitErr != nil {
		exitCode = exitCodeForError(waitErr)
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
