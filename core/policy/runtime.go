package policy

import (
	"context"
	"sort"
	"time"

	"glasshouse/core/profiling"
)

// Phase describes when a policy rule is evaluated.
type Phase string

const (
	PhasePreExecution  Phase = "pre_execution"
	PhaseRuntime       Phase = "runtime"
	PhasePostExecution Phase = "post_execution"
)

// EnforcementAction captures what the runtime enforcer should do on violation.
type EnforcementAction string

const (
	EnforcementNone          EnforcementAction = "none"
	EnforcementKillProcess   EnforcementAction = "kill_process"
	EnforcementKillExecution EnforcementAction = "kill_execution"
)

// Violation records a policy failure.
type Violation struct {
	Rule    string
	Phase   Phase
	Action  EnforcementAction
	Message string
}

// PreExecutionContext holds static metadata used before execution.
type PreExecutionContext struct {
	ExecutionID string
	Labels      map[string]string
	StartedAt   time.Time
}

// PreRule models a static constraint evaluated before execution.
type PreRule struct {
	Name   string
	Match  func(ctx PreExecutionContext) bool
	Action EnforcementAction
}

// RuntimeContext provides execution state during event evaluation.
type RuntimeContext struct {
	ExecutionID  string
	StartedAt    time.Time
	Now          time.Time
	ProcessCount int
	Duration     time.Duration
}

// RuntimeRule evaluates a kernel event against policy constraints.
type RuntimeRule struct {
	Name   string
	Match  func(ev profiling.Event, ctx RuntimeContext) bool
	Action EnforcementAction
}

// PreEvaluator applies static constraints deterministically.
type PreEvaluator struct {
	Policy Policy
}

func (e PreEvaluator) Evaluate(ctx context.Context, state PreExecutionContext) []Violation {
	_ = ctx
	violations := []Violation{}
	for _, rule := range e.Policy.PreRules {
		if rule.Match == nil {
			continue
		}
		if !rule.Match(state) {
			violations = append(violations, Violation{
				Rule:   rule.Name,
				Phase:  PhasePreExecution,
				Action: rule.Action,
			})
		}
	}
	sort.Slice(violations, func(i, j int) bool { return violations[i].Rule < violations[j].Rule })
	return violations
}

// RuntimeEvaluator applies runtime policy rules to kernel events.
type RuntimeEvaluator struct {
	Policy Policy
}

func (e RuntimeEvaluator) Evaluate(ctx context.Context, ev profiling.Event, state RuntimeContext) []Violation {
	_ = ctx
	violations := []Violation{}
	for _, rule := range e.Policy.RuntimeRules {
		if rule.Match == nil {
			continue
		}
		if !rule.Match(ev, state) {
			violations = append(violations, Violation{
				Rule:   rule.Name,
				Phase:  PhaseRuntime,
				Action: rule.Action,
			})
		}
	}
	sort.Slice(violations, func(i, j int) bool { return violations[i].Rule < violations[j].Rule })
	return violations
}
