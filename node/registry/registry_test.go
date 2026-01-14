package registry

import (
	"testing"

	"glasshouse/backend/fake"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	r := New()
	b := fake.New(0)
	r.Register("fake", b)

	got, err := r.Get("fake")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != b {
		t.Fatalf("unexpected backend reference")
	}
	if _, err := r.Get("missing"); err == nil {
		t.Fatal("expected error for missing backend")
	}
}
