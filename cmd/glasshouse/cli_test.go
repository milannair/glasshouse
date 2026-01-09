package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"glasshouse/audit"
)

var (
	buildOnce sync.Once
	buildErr  error
	cliPath   string
)

func repoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(filepath.Dir(cwd))
}

func buildCLI(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "glasshouse-cli-")
		if err != nil {
			buildErr = err
			return
		}
		cliPath = filepath.Join(tmpDir, "glasshouse")

		cmd := exec.Command("go", "build", "-o", cliPath, "./cmd/glasshouse")
		cmd.Dir = repoRoot(t)
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("build failed: %v: %s", err, strings.TrimSpace(string(out)))
			return
		}
	})
	if buildErr != nil {
		t.Fatalf("build: %v", buildErr)
	}
	return cliPath
}

func requireLinuxCommand(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("requires linux")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("missing %s", path)
	}
}

func runCLI(t *testing.T, args []string) (int, audit.Receipt, string) {
	t.Helper()
	bin := buildCLI(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GLASSHOUSE_BPF_DIR="+filepath.Join(repoRoot(t), "ebpf", "objects"))
	out, err := cmd.CombinedOutput()
	output := string(out)

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if err == nil {
		// ok
	} else if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("exec error: %v", err)
	}

	receiptPath := filepath.Join(dir, "receipt.json")
	data, readErr := os.ReadFile(receiptPath)
	if readErr != nil {
		t.Fatalf("read receipt: %v", readErr)
	}
	var receipt audit.Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("unmarshal receipt: %v", err)
	}
	return exitCode, receipt, output
}

func TestCLIRunTrue(t *testing.T) {
	requireLinuxCommand(t, "/bin/true")
	code, receipt, _ := runCLI(t, []string{"run", "--", "/bin/true"})
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if receipt.ExitCode != 0 {
		t.Fatalf("receipt exit code %d", receipt.ExitCode)
	}
	if receipt.Execution == nil || receipt.Execution.Backend != "process" {
		t.Fatalf("missing execution metadata")
	}
}

func TestCLIRunFalse(t *testing.T) {
	requireLinuxCommand(t, "/bin/false")
	code, receipt, _ := runCLI(t, []string{"run", "--", "/bin/false"})
	if code != 1 {
		t.Fatalf("exit code %d", code)
	}
	if receipt.ExitCode != 1 {
		t.Fatalf("receipt exit code %d", receipt.ExitCode)
	}
	if receipt.Execution == nil || receipt.Execution.Backend != "process" {
		t.Fatalf("missing execution metadata")
	}
}

func TestCLIRunDoesNotExist(t *testing.T) {
	requireLinuxCommand(t, "/bin/true")
	code, receipt, _ := runCLI(t, []string{"run", "--", "/bin/does-not-exist"})
	if code != 1 {
		t.Fatalf("exit code %d", code)
	}
	if receipt.ExitCode != 1 {
		t.Fatalf("receipt exit code %d", receipt.ExitCode)
	}
	if receipt.Outcome == nil || receipt.Outcome.Error == nil || !strings.Contains(*receipt.Outcome.Error, "does-not-exist") {
		t.Fatalf("missing error for does-not-exist")
	}
	if receipt.Execution == nil || receipt.Execution.Backend != "process" {
		t.Fatalf("missing execution metadata")
	}
}
