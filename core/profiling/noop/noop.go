package noop

import (
	"context"

	"glasshouse/core/profiling"
)

// Controller is a no-op profiling provider that satisfies the interface while
// making it explicit that profiling is disabled.
type Controller struct{}

// Session is an inert profiling session.
type Session struct {
	events chan profiling.Event
	errors chan error
	closed bool
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Start(ctx context.Context, target profiling.Target) (profiling.Session, error) {
	s := &Session{
		events: make(chan profiling.Event),
		errors: make(chan error),
	}
	close(s.events)
	close(s.errors)
	return s, nil
}

func (c *Controller) Capabilities() profiling.Capabilities {
	return profiling.Capabilities{}
}

func (s *Session) Events() <-chan profiling.Event { return s.events }
func (s *Session) Errors() <-chan error           { return s.errors }
func (s *Session) Close() error                   { return nil }

var _ profiling.Controller = (*Controller)(nil)
var _ profiling.Session = (*Session)(nil)
