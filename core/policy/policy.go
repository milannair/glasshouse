package policy

import (
	"context"
	"sort"

	"glasshouse/core/receipt"
)

// Verdict is deterministic given the policy and receipt inputs.
type Verdict struct {
	Allowed bool
	Reasons []string
}

// Policy is a declarative, substrate-agnostic rule set.
type Policy struct {
	Name  string
	Rules []Rule
}

// Rule models a simple predicate over the receipt.
type Rule struct {
	Name        string
	Match       func(r receipt.Receipt) bool
	Enforcement string
}

// Evaluator deterministically evaluates a policy against a receipt.
type Evaluator struct {
	Policy Policy
}

func (e Evaluator) Evaluate(ctx context.Context, r receipt.Receipt) Verdict {
	reasons := []string{}
	for _, rule := range e.Policy.Rules {
		if rule.Match == nil {
			continue
		}
		if !rule.Match(r) {
			reasons = append(reasons, rule.Name)
		}
	}
	sort.Strings(reasons)
	return Verdict{
		Allowed: len(reasons) == 0,
		Reasons: reasons,
	}
}
