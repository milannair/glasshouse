package agent

import (
	"fmt"
	"strings"
	"time"

	"glasshouse/core/identity"
)

// ControlCommand describes a control plane request for the agent.
type ControlCommand struct {
	Action        string            `json:"action"`
	ExecutionID   string            `json:"execution_id,omitempty"`
	CgroupID      uint64            `json:"cgroup_id,omitempty"`
	RootPID       uint32            `json:"root_pid,omitempty"`
	RootStartTime uint64            `json:"root_start_time,omitempty"`
	Command       string            `json:"command,omitempty"`
	StartedAt     string            `json:"started_at,omitempty"`
	EndedAt       string            `json:"ended_at,omitempty"`
	ExitCode      int               `json:"exit_code,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// ControlResponse reports the result of a control command.
type ControlResponse struct {
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
}

func parseTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, trimmed)
}

func resolveExecutionID(cmd ControlCommand) (identity.ExecutionID, error) {
	if cmd.ExecutionID != "" {
		return identity.ParseExecutionID(cmd.ExecutionID)
	}
	if cmd.CgroupID != 0 {
		return identity.FromCgroup(cmd.CgroupID), nil
	}
	if cmd.RootPID != 0 {
		start := cmd.RootStartTime
		if start == 0 {
			if value, err := identity.ProcessStartTime(cmd.RootPID); err == nil {
				start = value
			}
		}
		return identity.FromRoot(cmd.RootPID, start), nil
	}
	return identity.ExecutionID{}, fmt.Errorf("missing execution identifier")
}
