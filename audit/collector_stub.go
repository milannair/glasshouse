//go:build !linux

package audit

import (
	"context"
	"fmt"
)

type stubCollector struct{}

func NewCollector(cfg Config) (Collector, error) {
	return nil, fmt.Errorf("eBPF collector is only supported on Linux")
}

func (s *stubCollector) Start(ctx context.Context) error { return nil }
func (s *stubCollector) Events() <-chan Event            { return nil }
func (s *stubCollector) Errors() <-chan error            { return nil }
func (s *stubCollector) Close() error                    { return nil }
