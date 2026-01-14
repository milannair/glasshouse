package receipt

import (
	"testing"
	"time"
)

func TestReceiptMasking(t *testing.T) {
	r := Receipt{
		Filesystem: &FilesystemInfo{
			Reads:   []string{"/tmp/keep", "/secret/input"},
			Writes:  []string{"/secret/out", "/var/log/app"},
			Deletes: []string{"/secret/delete"},
		},
	}
	r.MaskPaths([]string{"/secret"})
	if len(r.Filesystem.Reads) != 1 || len(r.Filesystem.Writes) != 1 {
		t.Fatalf("unexpected filesystem entries %+v", r.Filesystem)
	}
	if len(r.Redactions) != 3 {
		t.Fatalf("expected 3 redactions, got %d", len(r.Redactions))
	}
}

func TestExecutionIDDeterministic(t *testing.T) {
	start := time.Unix(1700000000, 1234)
	args := []string{"/bin/echo", "hello"}
	first := executionID(start, 100, args)
	second := executionID(start, 100, args)
	if first != second {
		t.Fatalf("execution id mismatch: %s vs %s", first, second)
	}
	third := executionID(start, 101, args)
	if first == third {
		t.Fatalf("execution id should differ for different pid")
	}
}

func TestAggregatorReceiptContainsEssentials(t *testing.T) {
	agg := NewAggregator("host")
	agg.SetRoot(123, "/bin/true")
	rec := agg.Receipt(0, time.Second)
	if rec.Version == "" {
		t.Fatal("missing version")
	}
	if rec.Filesystem == nil || rec.Network == nil {
		t.Fatal("missing filesystem or network")
	}
}
