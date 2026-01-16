package identity

import (
	"fmt"
	"strconv"
	"strings"
)

// ExecutionID identifies a logical execution for attribution and policy.
// Prefer cgroup-based identity, with a pid+start-time fallback.
type ExecutionID struct {
	CgroupID      uint64
	RootPID       uint32
	RootStartTime uint64
}

func FromCgroup(id uint64) ExecutionID {
	return ExecutionID{CgroupID: id}
}

func FromRoot(pid uint32, startTime uint64) ExecutionID {
	return ExecutionID{RootPID: pid, RootStartTime: startTime}
}

func (id ExecutionID) IsZero() bool {
	return id.CgroupID == 0 && id.RootPID == 0
}

func (id ExecutionID) String() string {
	if id.CgroupID != 0 {
		return fmt.Sprintf("cgroup:%d", id.CgroupID)
	}
	if id.RootPID == 0 {
		return ""
	}
	return fmt.Sprintf("pid:%d:start:%d", id.RootPID, id.RootStartTime)
}

// ParseExecutionID decodes an execution id string (cgroup:<id> or pid:<pid>:start:<start>).
func ParseExecutionID(value string) (ExecutionID, error) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "cgroup:") {
		raw := strings.TrimPrefix(trimmed, "cgroup:")
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return ExecutionID{}, err
		}
		return FromCgroup(id), nil
	}
	if strings.HasPrefix(trimmed, "pid:") {
		parts := strings.Split(trimmed, ":")
		if len(parts) != 4 || parts[0] != "pid" || parts[2] != "start" {
			return ExecutionID{}, fmt.Errorf("invalid pid execution id format")
		}
		pid, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return ExecutionID{}, err
		}
		start, err := strconv.ParseUint(parts[3], 10, 64)
		if err != nil {
			return ExecutionID{}, err
		}
		return FromRoot(uint32(pid), start), nil
	}
	return ExecutionID{}, fmt.Errorf("unknown execution id format")
}
