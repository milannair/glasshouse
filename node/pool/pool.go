package pool

import (
	"context"
	"sync"
)

// Pool limits concurrency for execution jobs.
type Pool struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

func New(size int) *Pool {
	if size <= 0 {
		size = 1
	}
	return &Pool{
		sem: make(chan struct{}, size),
	}
}

// Go runs fn respecting the concurrency limit.
func (p *Pool) Go(ctx context.Context, fn func(context.Context) error) <-chan error {
	errCh := make(chan error, 1)
	p.sem <- struct{}{}
	p.wg.Add(1)
	go func() {
		defer func() {
			<-p.sem
			p.wg.Done()
		}()
		if ctx == nil {
			ctx = context.Background()
		}
		errCh <- fn(ctx)
		close(errCh)
	}()
	return errCh
}

// Wait blocks until all queued work completes.
func (p *Pool) Wait() {
	p.wg.Wait()
}
