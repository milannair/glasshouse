package transport

import (
	"context"
	"strings"
	"testing"
)

func TestLoopbackSend(t *testing.T) {
	var buf strings.Builder
	loop := NewLoopback(&buf)
	err := loop.Send(context.Background(), Event{
		Type: "heartbeat",
		Payload: map[string]interface{}{
			"status": "ok",
		},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if !strings.Contains(buf.String(), `"heartbeat"`) {
		t.Fatalf("missing event type in output: %q", buf.String())
	}
}
