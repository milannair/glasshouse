//go:build linux

package identity

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ProcessStartTime returns the kernel start time (in clock ticks since boot)
// for the given pid. It is used to disambiguate pid reuse.
func ProcessStartTime(pid uint32) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	// The stat format is: pid (comm) state ... starttime ...
	// We trim the comm field (which can contain spaces) by locating ") ".
	payload := string(data)
	idx := strings.LastIndex(payload, ") ")
	if idx == -1 {
		return 0, fmt.Errorf("invalid stat format")
	}
	fields := strings.Fields(payload[idx+2:])
	// starttime is the 22nd field overall; in the post-comm fields it's index 19 (0-based).
	if len(fields) < 20 {
		return 0, fmt.Errorf("short stat payload")
	}
	value, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}
