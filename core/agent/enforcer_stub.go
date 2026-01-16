//go:build !linux

package agent

import "fmt"

// Enforcer is a stub on non-Linux platforms.
type Enforcer struct{}

func (Enforcer) KillProcess(pid uint32) error {
	_ = pid
	return fmt.Errorf("enforcement unsupported on this platform")
}

func (Enforcer) KillExecution(pid uint32) (string, error) {
	_ = pid
	return "", fmt.Errorf("enforcement unsupported on this platform")
}
