//go:build !linux

package ebpf

import (
	"context"
	"fmt"

	"glasshouse/audit"
	"glasshouse/core/profiling"
)

// NewController returns a stub on non-Linux platforms.
func NewController(cfg audit.Config) *Controller {
	return &Controller{err: fmt.Errorf("eBPF profiling is only available on Linux")}
}

type Controller struct {
	err error
}

func (c *Controller) Start(ctx context.Context, target profiling.Target) (profiling.Session, error) {
	_ = ctx
	_ = target
	return nil, c.err
}

func (c *Controller) Capabilities() profiling.Capabilities {
	return profiling.Capabilities{}
}

var _ profiling.Controller = (*Controller)(nil)
