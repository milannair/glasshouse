//go:build !linux

package process

import (
	"context"
	"os"
	"sync"
	"syscall"
)

func setupGuestEnvironment() []string { return []string{"guest_setup: unsupported_os"} }
func startGuestSignalHandler(ctx context.Context, proc *os.Process, reapMu *sync.Mutex, mainReaped *bool, mainStatus *syscall.WaitStatus, shutdownSign *os.Signal) {
	// no-op on non-Linux platforms
}
func reapChildren(proc *os.Process, reapMu *sync.Mutex, mainReaped *bool, mainStatus *syscall.WaitStatus) {
}
