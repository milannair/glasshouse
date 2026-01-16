package manager

import (
	"context"
	"fmt"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
	"glasshouse/core/profiling/noop"
	"glasshouse/node/config"
	"glasshouse/node/enforcement"
	"glasshouse/node/pool"
	"glasshouse/node/registry"
)

// Request describes a single execution request handled by the node agent.
type Request struct {
	Args      []string
	Workdir   string
	Env       []string
	Backend   string
	Profiling profiling.Mode
	Labels    map[string]string
}

// Response captures the result and any policy enforcement error.
type Response struct {
	Result execution.ExecutionResult
	Err    error
}

// Manager orchestrates backend selection, execution, optional profiling, and enforcement.
type Manager struct {
	Config   config.Config
	Registry *registry.Registry
	Pool     *pool.Pool
	Profiler profiling.Controller
	Enforcer enforcement.Enforcer
}

func New(cfg config.Config, reg *registry.Registry) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, fmt.Errorf("registry required")
	}
	return &Manager{
		Config:   cfg,
		Registry: reg,
		Pool:     pool.New(cfg.Concurrency),
		Profiler: noop.NewController(),
		Enforcer: enforcement.NoopEnforcer{},
	}, nil
}

// Run executes a request synchronously via the selected backend.
func (m *Manager) Run(ctx context.Context, req Request) (Response, error) {
	backendName := req.Backend
	if backendName == "" {
		backendName = m.Config.DefaultBackend
	}
	backend, err := m.Registry.Get(backendName)
	if err != nil {
		return Response{}, err
	}

	profileMode := req.Profiling
	if profileMode == "" {
		profileMode = m.Config.ProfilingMode
	}

	spec := execution.ExecutionSpec{
		Args:      req.Args,
		Workdir:   req.Workdir,
		Env:       req.Env,
		Guest:     false,
		Profiling: profileMode,
		Labels:    req.Labels,
	}

	engine := execution.Engine{
		Backend:  backend,
		Profiler: m.Profiler,
	}
	result, runErr := engine.Run(ctx, spec)
	if runErr != nil && result.Err == nil {
		result.Err = runErr
	}

	if m.Enforcer != nil && result.Receipt != nil {
		if err := m.Enforcer.Enforce(ctx, result.Receipt); err != nil && result.Err == nil {
			result.Err = err
		}
	}

	return Response{Result: result, Err: result.Err}, result.Err
}
