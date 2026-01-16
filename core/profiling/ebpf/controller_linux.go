//go:build linux

package ebpf

import (
	"context"
	"fmt"

	"glasshouse/audit"
	"glasshouse/core/profiling"
)

// Controller wraps the legacy audit collector to satisfy the profiling.Controller
// interface using host-side eBPF CO-RE programs.
type Controller struct {
	cfg audit.Config
}

func NewController(cfg audit.Config) *Controller {
	return &Controller{cfg: cfg}
}

func (c *Controller) Start(ctx context.Context, target profiling.Target) (profiling.Session, error) {
	// Target is intentionally not used because current eBPF programs attach
	// system-wide; future guest/combined modes can route via target identity.
	collector, err := audit.NewCollector(c.cfg)
	if err != nil {
		return nil, err
	}
	if err := collector.Start(ctx); err != nil {
		return nil, err
	}
	return &session{collector: collector}, nil
}

func (c *Controller) Capabilities() profiling.Capabilities {
	return profiling.Capabilities{Host: true}
}

type session struct {
	collector audit.Collector
}

func (s *session) Events() <-chan profiling.Event {
	out := make(chan profiling.Event)
	go func() {
		defer close(out)
		for ev := range s.collector.Events() {
			out <- profiling.Event{
				Type:       profiling.EventType(ev.Type),
				PID:        ev.PID,
				PPID:       ev.PPID,
				CgroupID:   ev.CgroupID,
				Flags:      ev.Flags,
				Comm:       ev.Comm,
				Path:       ev.Path,
				AddrFamily: ev.AddrFamily,
				Proto:      ev.Proto,
				Addr:       ev.Addr,
				Port:       ev.Port,
			}
		}
	}()
	return out
}

func (s *session) Errors() <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)
		for err := range s.collector.Errors() {
			if err != nil {
				out <- fmt.Errorf("collector: %w", err)
			}
		}
	}()
	return out
}

func (s *session) Close() error {
	if s.collector == nil {
		return nil
	}
	return s.collector.Close()
}

var _ profiling.Controller = (*Controller)(nil)
var _ profiling.Session = (*session)(nil)
