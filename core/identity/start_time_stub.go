//go:build !linux

package identity

import "fmt"

// ProcessStartTime is not available on non-Linux platforms.
func ProcessStartTime(pid uint32) (uint64, error) {
	return 0, fmt.Errorf("process start time unavailable on this platform")
}
