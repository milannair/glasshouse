package config

import (
	"fmt"

	"glasshouse/core/profiling"
)

// Config drives the node-agent execution behavior.
type Config struct {
	// DefaultBackend is the backend name to use when a request does not specify one.
	DefaultBackend string
	// ProfilingMode is the default profiling mode for executions (per-request override).
	ProfilingMode profiling.Mode
	// Concurrency limits the number of concurrent executions the node will run.
	Concurrency int
}

// Defaults returns a safe baseline configuration for sandbox-only execution.
func Defaults() Config {
	return Config{
		DefaultBackend: "process",
		ProfilingMode:  profiling.ProfilingDisabled,
		Concurrency:    1,
	}
}

// Validate ensures the config is usable.
func (c Config) Validate() error {
	if c.DefaultBackend == "" {
		return fmt.Errorf("default backend required")
	}
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be > 0")
	}
	return nil
}
