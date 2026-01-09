package audit

import (
	"encoding/json"
	"testing"
)

func TestReceiptSchemaFields(t *testing.T) {
	agg := NewAggregator()
	receipt := agg.Receipt(3, 0)
	receipt.Execution = &ExecutionInfo{Backend: "process", Isolation: "none"}

	if receipt.ExitCode != 3 {
		t.Fatalf("exit code %d", receipt.ExitCode)
	}
	if receipt.Filesystem == nil || receipt.Network == nil {
		t.Fatal("missing filesystem or network")
	}
	if len(receipt.Filesystem.Reads) != 0 || len(receipt.Network.Attempts) != 0 {
		t.Fatal("expected empty activity")
	}

	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"execution", "exit_code", "duration_ms", "processes", "filesystem", "network"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing key %s", key)
		}
	}
}
