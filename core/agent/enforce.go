package agent

import (
	"context"
	"fmt"

	"glasshouse/core/policy"
	"glasshouse/core/profiling"
)

func (a *Agent) enforce(ctx context.Context, ev profiling.Event, violation policy.Violation) (string, string, error) {
	_ = ctx
	switch violation.Action {
	case policy.EnforcementKillExecution:
		kind, err := a.enforcer.KillExecution(ev.PID)
		return string(violation.Action), fmt.Sprintf("%s:%d", kind, ev.PID), err
	case policy.EnforcementKillProcess:
		err := a.enforcer.KillProcess(ev.PID)
		return string(violation.Action), fmt.Sprintf("pid:%d", ev.PID), err
	default:
		return string(violation.Action), "", nil
	}
}
