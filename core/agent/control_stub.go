//go:build windows

package agent

import (
	"context"
	"fmt"
)

// ControlServer is unsupported on Windows.
type ControlServer struct{}

func NewControlServer(path string, handler func(context.Context, ControlCommand) ControlResponse) *ControlServer {
	_ = path
	_ = handler
	return &ControlServer{}
}

func (s *ControlServer) Run(ctx context.Context) error {
	_ = ctx
	return fmt.Errorf("control server unsupported on windows")
}
