package policy

import (
	"context"
	"testing"

	"glasshouse/core/receipt"
)

func TestEvaluatorDeterministic(t *testing.T) {
	p := Policy{
		Name: "exit-zero",
		Rules: []Rule{
			{
				Name: "require-zero",
				Match: func(r receipt.Receipt) bool {
					return r.ExitCode == 0
				},
				Enforcement: "audit",
			},
		},
	}
	ev := Evaluator{Policy: p}
	r := receipt.Receipt{ExitCode: 1}
	first := ev.Evaluate(context.Background(), r)
	second := ev.Evaluate(context.Background(), r)
	if first.Allowed || second.Allowed {
		t.Fatalf("expected deny for exit code 1")
	}
	if len(first.Reasons) != 1 || first.Reasons[0] != "require-zero" {
		t.Fatalf("unexpected reasons %v", first.Reasons)
	}
	if len(second.Reasons) != 1 || second.Reasons[0] != "require-zero" {
		t.Fatalf("non-deterministic reasons %v", second.Reasons)
	}
}
