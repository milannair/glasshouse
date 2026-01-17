//go:build linux
// +build linux

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Result is written to /workspace/.pending/result.json
type Result struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

func main() {
	log("guest-init starting")

	// Mount essential filesystems
	mustMount("proc", "/proc", "proc")
	mustMount("sysfs", "/sys", "sysfs")
	mustMount("devtmpfs", "/dev", "devtmpfs")

	// Mount workspace from /dev/vdb
	if err := os.MkdirAll("/workspace", 0755); err != nil {
		fatal("mkdir /workspace: " + err.Error())
	}
	if err := syscall.Mount("/dev/vdb", "/workspace", "ext4", 0, ""); err != nil {
		fatal("mount /workspace: " + err.Error())
	}
	log("workspace mounted")

	// Read code from pending directory
	codePath := "/workspace/.pending/code.py"
	codeBytes, err := os.ReadFile(codePath)
	if err != nil {
		writeResult(Result{Error: "read code: " + err.Error(), ExitCode: 1})
		poweroff()
		return
	}
	log("code: " + string(codeBytes))

	// Find Python - try common paths
	pythonPath := ""
	for _, p := range []string{"/usr/bin/python3", "/usr/bin/python", "/bin/python3", "/bin/python"} {
		if _, err := os.Stat(p); err == nil {
			pythonPath = p
			break
		}
	}
	if pythonPath == "" {
		log("ERROR: Python not found in common paths")
		writeResult(Result{Error: "python not found", ExitCode: 1})
		poweroff()
		return
	}
	log("using python: " + pythonPath)

	// Execute Python
	start := time.Now()
	cmd := exec.Command(pythonPath, "-c", string(codeBytes))
	cmd.Dir = "/workspace"

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	duration := time.Since(start)

	result := Result{
		DurationMs: duration.Milliseconds(),
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			result.Error = err.Error()
		}
		log("python error: " + result.Stderr)
	}

	log(fmt.Sprintf("execution complete, exit_code=%d, duration=%dms", result.ExitCode, result.DurationMs))
	writeResult(result)
	poweroff()
}

func mustMount(source, target, fstype string) {
	if err := os.MkdirAll(target, 0755); err != nil {
		log("mkdir " + target + ": " + err.Error())
		return
	}
	if err := syscall.Mount(source, target, fstype, 0, ""); err != nil {
		log("mount " + target + ": " + err.Error())
	}
}

func writeResult(r Result) {
	data, _ := json.Marshal(r)
	if err := os.WriteFile("/workspace/.pending/result.json", data, 0644); err != nil {
		log("write result: " + err.Error())
	}
}

func poweroff() {
	log("powering off")
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}

func fatal(msg string) {
	log("FATAL: " + msg)
	writeResult(Result{Error: msg, ExitCode: 1})
	poweroff()
}

func log(msg string) {
	fmt.Println("[guest-init]", msg)
}
