package firecracker

import (
	"context"
	"errors"

	"glasshouse/backend"
)

var ErrFirecrackerNotImplemented = errors.New("firecracker backend not implemented")

type Backend struct {
	cfg Config
}

func New(cfg Config) *Backend {
	return &Backend{cfg: cfg}
}

func (b *Backend) Prepare(ctx context.Context) error {
	return b.cfg.Validate()
}

func (b *Backend) Start(ctx context.Context, cmd []string) (int, error) {
	return 0, ErrFirecrackerNotImplemented
}

func (b *Backend) Wait(ctx context.Context) (int, error) {
	return 0, nil
}

func (b *Backend) Cleanup(ctx context.Context) error {
	return nil
}

func (b *Backend) Metadata() backend.BackendMetadata {
	return backend.BackendMetadata{Backend: "firecracker", Isolation: "vm"}
}

var _ backend.Backend = (*Backend)(nil)
