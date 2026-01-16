package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"glasshouse/core/identity"
	"glasshouse/core/policy"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

// Config configures the glasshouse agent daemon.
type Config struct {
	ReceiptDir    string
	Observation   string
	Policy        policy.Policy
	ControlSocket string
}

// Agent runs in daemon mode: it only observes kernel events and never launches workloads.
// Per-execution runs continue to use execution.Engine with their own profiler session.
type Agent struct {
	cfg         Config
	profiler    profiling.Controller
	aggregator  *receipt.Aggregator
	preEval     policy.PreEvaluator
	runtimeEval policy.RuntimeEvaluator
	postEval    policy.Evaluator
	enforcer    Enforcer

	mu      sync.Mutex
	runtime map[string]*runtimeState
}

type runtimeState struct {
	startedAt time.Time
	pids      map[uint32]struct{}
}

func New(cfg Config, profiler profiling.Controller) *Agent {
	obs := cfg.Observation
	if obs == "" {
		obs = "host"
		cfg.Observation = obs
	}
	agg := receipt.NewStreamAggregator(receipt.AggregatorOptions{
		Provenance: obs,
		AutoCreate: true,
	})
	return &Agent{
		cfg:         cfg,
		profiler:    profiler,
		aggregator:  agg,
		preEval:     policy.PreEvaluator{Policy: cfg.Policy},
		runtimeEval: policy.RuntimeEvaluator{Policy: cfg.Policy},
		postEval:    policy.Evaluator{Policy: cfg.Policy},
		enforcer:    Enforcer{},
		runtime:     make(map[string]*runtimeState),
	}
}

// Run starts the profiler, control plane, and event loop.
func (a *Agent) Run(ctx context.Context) error {
	session, err := a.profiler.Start(ctx, profiling.Target{Mode: profiling.ProfilingHost})
	if err != nil {
		return err
	}
	defer session.Close()

	if a.cfg.ControlSocket != "" {
		srv := NewControlServer(a.cfg.ControlSocket, a.handleControl)
		go func() {
			if err := srv.Run(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "glasshouse-agent: control server error: %v\n", err)
			}
		}()
	}

	for {
		select {
		case ev, ok := <-session.Events():
			if !ok {
				return nil
			}
			a.handleEvent(ctx, ev)
		case err, ok := <-session.Errors():
			if ok && err != nil {
				fmt.Fprintf(os.Stderr, "glasshouse-agent: event error: %v\n", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (a *Agent) handleEvent(ctx context.Context, ev profiling.Event) {
	execID := a.aggregator.HandleEvent(ev)
	if execID.IsZero() {
		return
	}
	idStr := execID.String()

	now := time.Now()
	state := a.ensureRuntimeState(idStr, now)
	state.pids[ev.PID] = struct{}{}
	ctxState := policy.RuntimeContext{
		ExecutionID:  idStr,
		StartedAt:    state.startedAt,
		Now:          now,
		ProcessCount: len(state.pids),
		Duration:     now.Sub(state.startedAt),
	}
	violations := a.runtimeEval.Evaluate(ctx, ev, ctxState)
	if len(violations) == 0 {
		return
	}

	for _, violation := range violations {
		a.aggregator.RecordPolicyViolation(execID, receipt.PolicyViolation{
			Phase:   string(violation.Phase),
			Rule:    violation.Rule,
			Action:  string(violation.Action),
			Message: violation.Message,
		})
		if violation.Action == policy.EnforcementNone {
			continue
		}
		a.aggregator.SetPolicyFailed(execID, true)
		action, target, err := a.enforce(ctx, ev, violation)
		a.aggregator.RecordPolicyEnforcement(execID, receipt.PolicyEnforcement{
			Action:  action,
			Target:  target,
			Rule:    violation.Rule,
			Message: errorMessage(err),
		})
	}
}

func (a *Agent) handleControl(ctx context.Context, cmd ControlCommand) ControlResponse {
	switch strings.ToLower(strings.TrimSpace(cmd.Action)) {
	case "start":
		return a.handleStart(ctx, cmd)
	case "end":
		return a.handleEnd(ctx, cmd, true)
	case "flush":
		return a.handleEnd(ctx, cmd, false)
	default:
		return ControlResponse{OK: false, Error: "unknown action"}
	}
}

func (a *Agent) handleStart(ctx context.Context, cmd ControlCommand) ControlResponse {
	startTime, err := parseTime(cmd.StartedAt)
	if err != nil {
		return ControlResponse{OK: false, Error: fmt.Sprintf("invalid start time: %v", err)}
	}
	if startTime.IsZero() {
		startTime = time.Now()
	}

	execID, err := resolveExecutionID(cmd)
	if err != nil {
		return ControlResponse{OK: false, Error: err.Error()}
	}

	rootStart := cmd.RootStartTime
	if rootStart == 0 && cmd.RootPID != 0 {
		if value, err := identity.ProcessStartTime(cmd.RootPID); err == nil {
			rootStart = value
		}
	}

	id := a.aggregator.StartExecution(receipt.ExecutionStart{
		ID:              execID,
		RootPID:         cmd.RootPID,
		RootStartTime:   rootStart,
		Command:         cmd.Command,
		StartedAt:       startTime,
		ObservationMode: a.cfg.Observation,
	})

	if id.IsZero() {
		return ControlResponse{OK: false, Error: "failed to register execution"}
	}
	a.ensureRuntimeState(id.String(), startTime)

	preViolations := a.preEval.Evaluate(ctx, policy.PreExecutionContext{
		ExecutionID: id.String(),
		Labels:      cmd.Labels,
		StartedAt:   startTime,
	})
	for _, violation := range preViolations {
		a.aggregator.RecordPolicyViolation(id, receipt.PolicyViolation{
			Phase:   string(violation.Phase),
			Rule:    violation.Rule,
			Action:  string(violation.Action),
			Message: violation.Message,
		})
		if violation.Action != policy.EnforcementNone && cmd.RootPID != 0 {
			_, _, err := a.enforce(ctx, profiling.Event{PID: cmd.RootPID, CgroupID: cmd.CgroupID}, violation)
			if err != nil {
				a.aggregator.RecordPolicyEnforcement(id, receipt.PolicyEnforcement{
					Action:  string(violation.Action),
					Target:  "pid",
					Rule:    violation.Rule,
					Message: errorMessage(err),
				})
			}
			a.aggregator.SetPolicyFailed(id, true)
		}
	}

	return ControlResponse{OK: true, ExecutionID: id.String()}
}

func (a *Agent) handleEnd(ctx context.Context, cmd ControlCommand, closed bool) ControlResponse {
	endTime, err := parseTime(cmd.EndedAt)
	if err != nil {
		return ControlResponse{OK: false, Error: fmt.Sprintf("invalid end time: %v", err)}
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	execID, err := resolveExecutionID(cmd)
	if err != nil {
		return ControlResponse{OK: false, Error: err.Error()}
	}

	if closed {
		a.aggregator.EndExecution(execID, endTime)
	}

	startTime := a.lookupStartTime(execID.String())
	duration := time.Duration(0)
	if !startTime.IsZero() {
		duration = endTime.Sub(startTime)
	}

	rec, ok := a.aggregator.FlushExecution(execID, cmd.ExitCode, duration)
	if !ok {
		return ControlResponse{OK: false, Error: "execution not found"}
	}
	if !closed {
		rec.Completeness = "partial"
	}

	verdict := a.postEval.Evaluate(ctx, rec)
	if rec.Policy == nil {
		rec.Policy = &receipt.PolicyInfo{}
	}
	rec.Policy.Trusted = verdict.Allowed
	if !verdict.Allowed {
		for _, reason := range verdict.Reasons {
			rec.Policy.Violations = append(rec.Policy.Violations, receipt.PolicyViolation{
				Phase: string(policy.PhasePostExecution),
				Rule:  reason,
			})
		}
	}

	if err := a.emitReceipt(rec); err != nil {
		return ControlResponse{OK: false, Error: err.Error(), ExecutionID: rec.ExecutionID}
	}

	a.aggregator.ForgetExecution(execID)
	a.clearRuntimeState(execID.String())
	return ControlResponse{OK: true, ExecutionID: rec.ExecutionID}
}

func (a *Agent) ensureRuntimeState(id string, startedAt time.Time) *runtimeState {
	a.mu.Lock()
	defer a.mu.Unlock()
	state, ok := a.runtime[id]
	if !ok {
		state = &runtimeState{startedAt: startedAt, pids: make(map[uint32]struct{})}
		a.runtime[id] = state
	}
	if state.startedAt.IsZero() {
		state.startedAt = startedAt
	}
	return state
}

func (a *Agent) lookupStartTime(id string) time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	if state, ok := a.runtime[id]; ok {
		return state.startedAt
	}
	return time.Time{}
}

func (a *Agent) clearRuntimeState(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.runtime, id)
}

func (a *Agent) emitReceipt(rec receipt.Receipt) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal receipt: %w", err)
	}
	if a.cfg.ReceiptDir == "" {
		_, err := fmt.Fprintln(os.Stdout, string(data))
		return err
	}
	if err := os.MkdirAll(a.cfg.ReceiptDir, 0755); err != nil {
		return fmt.Errorf("create receipt dir: %w", err)
	}
	name := sanitizeExecutionID(rec.ExecutionID)
	path := filepath.Join(a.cfg.ReceiptDir, fmt.Sprintf("receipt-%s.json", name))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write receipt: %w", err)
	}
	return nil
}

func sanitizeExecutionID(value string) string {
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(":", "_", "/", "_", " ", "_")
	return replacer.Replace(value)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
