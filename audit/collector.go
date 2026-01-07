package audit

import "context"

type Config struct {
	BPFObjectDir string
}

type Collector interface {
	Start(ctx context.Context) error
	Events() <-chan Event
	Errors() <-chan error
	Close() error
}
