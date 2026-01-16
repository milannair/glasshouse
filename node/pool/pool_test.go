package pool

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolLimitsConcurrency(t *testing.T) {
	p := New(2)
	var concurrent int32
	var maxConcurrent int32

	work := func(ctx context.Context) error {
		cur := atomic.AddInt32(&concurrent, 1)
		defer atomic.AddInt32(&concurrent, -1)
		for {
			if curMax := atomic.LoadInt32(&maxConcurrent); cur > curMax {
				atomic.StoreInt32(&maxConcurrent, cur)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
				return nil
			}
		}
	}

	errCh1 := p.Go(context.Background(), work)
	errCh2 := p.Go(context.Background(), work)
	errCh3 := p.Go(context.Background(), work)

	<-errCh1
	<-errCh2
	<-errCh3
	p.Wait()

	if maxConcurrent > 2 {
		t.Fatalf("expected max concurrency <= 2, got %d", maxConcurrent)
	}
}
