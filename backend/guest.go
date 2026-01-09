package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

func setupGuestEnvironment() []string {
	if runtime.GOOS != "linux" {
		return []string{"guest_setup: unsupported_os"}
	}

	errs := []string{}
	mounts := []struct {
		source string
		target string
		fstype string
	}{
		{source: "proc", target: "/proc", fstype: "proc"},
		{source: "sysfs", target: "/sys", fstype: "sysfs"},
		{source: "bpf", target: "/sys/fs/bpf", fstype: "bpf"},
	}

	for _, m := range mounts {
		if err := mountIfNeeded(m.source, m.target, m.fstype); err != nil {
			errs = append(errs, fmt.Sprintf("guest_setup: mount %s: %v", m.target, err))
		}
	}

	if err := setMemlockUnlimited(); err != nil {
		errs = append(errs, fmt.Sprintf("guest_setup: rlimit_memlock: %v", err))
	}

	return errs
}

func mountIfNeeded(source, target, fstype string) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	logGuest("mount %s -> %s", fstype, target)
	if err := syscall.Mount(source, target, fstype, 0, ""); err != nil {
		if errors.Is(err, syscall.EBUSY) || errors.Is(err, syscall.EEXIST) {
			return nil
		}
		return err
	}
	return nil
}

func setMemlockUnlimited() error {
	limit := &unix.Rlimit{Cur: unix.RLIM_INFINITY, Max: unix.RLIM_INFINITY}
	if err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, limit); err != nil {
		return err
	}
	logGuest("set RLIMIT_MEMLOCK=unlimited")
	return nil
}

func startGuestSignalHandler(ctx context.Context, proc *os.Process, reapMu *sync.Mutex, mainReaped *bool, mainStatus *syscall.WaitStatus, shutdownSign *os.Signal) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGCHLD)
	go func() {
		defer signal.Stop(sigs)
		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-sigs:
				switch sig {
				case syscall.SIGCHLD:
					reapChildren(proc, reapMu, mainReaped, mainStatus)
				case syscall.SIGTERM, syscall.SIGINT:
					if shutdownSign != nil && *shutdownSign == nil {
						*shutdownSign = sig
					}
					logGuest("received %s, shutting down", sig.String())
					if proc != nil {
						_ = proc.Signal(sig)
					}
				}
			}
		}
	}()
}

func reapChildren(proc *os.Process, reapMu *sync.Mutex, mainReaped *bool, mainStatus *syscall.WaitStatus) {
	mainPID := -1
	if proc != nil {
		mainPID = proc.Pid
	}
	for {
		var status syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if pid == 0 || errors.Is(err, syscall.ECHILD) {
			return
		}
		if err != nil {
			return
		}
		if pid == mainPID {
			reapMu.Lock()
			*mainReaped = true
			*mainStatus = status
			reapMu.Unlock()
		}
	}
}

func logGuest(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "glasshouse: guest: "+format+"\n", args...)
}
