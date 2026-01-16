package kata

import (
	"context"
	"testing"

	"glasshouse/core/execution"
)

func TestKataBackendMetadata(t *testing.T) {
	b := New()
	if err := b.Prepare(context.Background()); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	handle, err := b.Start(execution.ExecutionSpec{Args: []string{"/bin/true"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := b.Wait(handle); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if err := b.Cleanup(handle); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	meta := b.Metadata()
	if meta.Isolation != "vm" {
		t.Fatalf("expected vm isolation, got %s", meta.Isolation)
	}
}
