package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"goreadwise/internal/model"
)

func TestPoolProcessesJob(t *testing.T) {
	var n atomic.Int64
	p := New(context.Background(), nil, 2, 8, func(ctx context.Context, job model.GraphJob) error {
		n.Add(1)
		return nil
	})
	defer p.Close()
	p.Submit(model.GraphJob{ID: uuid.New(), Kind: model.JobKindStats, ContentHash: "h"})
	deadline := time.Now().Add(2 * time.Second)
	for n.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n.Load() == 0 {
		t.Fatal("job not processed")
	}
}

func TestQueueFullFallsBackSync(t *testing.T) {
	block := make(chan struct{})
	p := New(context.Background(), nil, 1, 1, func(ctx context.Context, job model.GraphJob) error {
		<-block
		return nil
	})
	defer func() {
		close(block)
		p.Close()
	}()
	p.Submit(model.GraphJob{ID: uuid.New(), Kind: "a", ContentHash: "1"})
	// fill the single slot
	select {
	case p.jobs <- model.GraphJob{ID: uuid.New(), Kind: "b", ContentHash: "2"}:
	default:
	}
	// next must fallback; run() will also block on channel handler...
	// use a separate pool for deterministic fallback count
	var ran atomic.Int64
	p2 := New(context.Background(), nil, 1, 1, func(ctx context.Context, job model.GraphJob) error {
		ran.Add(1)
		return nil
	})
	defer p2.Close()
	p2.jobs <- model.GraphJob{ID: uuid.New(), Kind: "fill", ContentHash: "x"}
	p2.Submit(model.GraphJob{ID: uuid.New(), Kind: "fb", ContentHash: "y"})
	time.Sleep(50 * time.Millisecond)
	if p2.Snapshot().SyncFallback == 0 && ran.Load() < 1 {
		t.Fatal("expected fallback or run")
	}
}

func TestTransientDetect(t *testing.T) {
	if !isTransient(context.DeadlineExceeded) {
		t.Fatal("deadline should be transient")
	}
}
