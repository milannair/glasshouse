package enforcement

import (
	"context"

	"glasshouse/core/receipt"
)

// Enforcer optionally applies runtime policy decisions; enforcement can be a no-op.
type Enforcer interface {
	Enforce(ctx context.Context, rec *receipt.Receipt) error
}

// NoopEnforcer performs no enforcement and always returns nil.
type NoopEnforcer struct{}

func (NoopEnforcer) Enforce(ctx context.Context, rec *receipt.Receipt) error {
	_ = ctx
	_ = rec
	return nil
}
