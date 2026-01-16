//go:build linux

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Enforcer applies observe+kill actions; it is best-effort and never blocks syscalls.
type Enforcer struct{}

func (Enforcer) KillProcess(pid uint32) error {
	if pid == 0 {
		return fmt.Errorf("missing pid")
	}
	return syscall.Kill(int(pid), syscall.SIGKILL)
}

func (Enforcer) KillExecution(pid uint32) (string, error) {
	if pid == 0 {
		return "", fmt.Errorf("missing pid")
	}
	if err := killCgroup(pid); err == nil {
		return "cgroup", nil
	}
	if err := syscall.Kill(int(pid), syscall.SIGKILL); err != nil {
		return "pid", err
	}
	return "pid", nil
}

func killCgroup(pid uint32) error {
	path, err := cgroupKillPath(pid)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte("1"), 0644)
}

func cgroupKillPath(pid uint32) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "0::") {
			path := strings.TrimPrefix(line, "0::")
			path = strings.TrimPrefix(path, "/")
			return filepath.Join("/sys/fs/cgroup", path, "cgroup.kill"), nil
		}
	}
	return "", fmt.Errorf("cgroup v2 path not found")
}
