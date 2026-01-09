package backend

import (
	"context"
	"strings"
)

type Backend interface {
	Prepare(ctx context.Context) error
	Start(ctx context.Context, cmd []string) (rootPID int, err error)
	Wait(ctx context.Context) (exitCode int, err error)
	Cleanup(ctx context.Context) error
	Metadata() BackendMetadata
}

type BackendMetadata struct {
	Backend   string `json:"backend"`
	Isolation string `json:"isolation"`
}

type ErrorList struct {
	Errors []string
}

func (e ErrorList) Error() string {
	return strings.Join(e.Errors, "; ")
}

type ExtraErrorProvider interface {
	ExtraErrors() []string
}

type OutputProvider interface {
	Stdout() []byte
	Stderr() []byte
}
