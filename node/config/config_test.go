package config

import (
	"testing"

	"glasshouse/core/profiling"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.DefaultBackend != "process" {
		t.Fatalf("default backend %q", cfg.DefaultBackend)
	}
	if cfg.ProfilingMode != profiling.ProfilingDisabled {
		t.Fatalf("default profiling %q", cfg.ProfilingMode)
	}
	if cfg.Concurrency != 1 {
		t.Fatalf("default concurrency %d", cfg.Concurrency)
	}
}

func TestValidate(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatalf("defaults should validate: %v", err)
	}
	cfg := Defaults()
	cfg.DefaultBackend = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing backend")
	}
	cfg = Defaults()
	cfg.Concurrency = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for bad concurrency")
	}
}
