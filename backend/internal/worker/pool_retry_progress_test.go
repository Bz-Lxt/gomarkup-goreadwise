package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"goreadwise/internal/model"
	"goreadwise/internal/worker"
)

func TestPoolMakesProgressWhenRetryQueueIsFull(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	var firstAttempts atomic.Int32

	parent, cancel := context.WithCancel(context.Background())
	pool := worker.New(parent, nil, 1, 1, func(ctx context.Context, job model.GraphJob) error {
		switch job.ID {
		case firstID:
			if firstAttempts.Add(1) == 1 {
				close(firstStarted)
				select {
				case <-releaseFirst:
				case <-ctx.Done():
					return ctx.Err()
				}
				return context.DeadlineExceeded
			}
		case secondID:
			close(secondStarted)
		}
		return nil
	})
	defer func() {
		cancel()
		pool.Close()
	}()

	pool.Submit(model.GraphJob{ID: firstID, Kind: model.JobKindStats})
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first job did not start")
	}

	pool.Submit(model.GraphJob{ID: secondID, Kind: model.JobKindStats})
	close(releaseFirst)

	select {
	case <-secondStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("queued job did not start after the running job requested a retry")
	}
}
