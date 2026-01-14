package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Event is a generic message emitted by the guest probe.
type Event struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// Transport delivers guest events to a host-visible channel.
type Transport interface {
	Send(ctx context.Context, ev Event) error
	Close() error
}

// Loopback writes events as JSON lines to the provided writer (stdout by default).
type Loopback struct {
	w io.Writer
}

func NewLoopback(w io.Writer) *Loopback {
	if w == nil {
		w = os.Stdout
	}
	return &Loopback{w: w}
}

func (l *Loopback) Send(ctx context.Context, ev Event) error {
	_ = ctx
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if _, err := l.w.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

func (l *Loopback) Close() error { return nil }

var _ Transport = (*Loopback)(nil)
