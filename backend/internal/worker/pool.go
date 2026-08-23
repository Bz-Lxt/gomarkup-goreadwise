package worker

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goreadwise/internal/logger"
	"goreadwise/internal/model"
	"goreadwise/internal/store"
)

type Handler func(ctx context.Context, job model.GraphJob) error

type Pool struct {
	db      *store.DB
	jobs    chan model.GraphJob
	handle  Handler
	workers int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc

	syncFallback atomic.Int64
	done         atomic.Int64
	failed       atomic.Int64
}

func New(parent context.Context, db *store.DB, workers, queue int, handle Handler) *Pool {
	ctx, cancel := context.WithCancel(parent)
	p := &Pool{
		db:      db,
		jobs:    make(chan model.GraphJob, queue),
		handle:  handle,
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.loop(i)
	}
	return p
}

func (p *Pool) loop(id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			p.run(job)
		}
	}
}

func (p *Pool) run(job model.GraphJob) {
	ctx, cancel := context.WithTimeout(p.ctx, 20*time.Second)
	defer cancel()
	if p.db != nil {
		_ = p.db.MarkJob(ctx, job.ID, model.JobRunning, "")
	}
	err := p.handle(ctx, job)
	if err != nil {
		p.failed.Add(1)
		msg := err.Error()
		status := model.JobFailed
		if isTransient(err) && job.Attempts < 3 {
			status = model.JobPending
			select {
			case p.jobs <- job:
			case <-p.ctx.Done():
			}
		}
		if p.db != nil {
			_ = p.db.MarkJob(ctx, job.ID, status, msg)
		}
		logger.L().Warn("job failed", slog.String("kind", job.Kind), slog.String("err", msg))
		return
	}
	p.done.Add(1)
	if p.db != nil {
		_ = p.db.MarkJob(ctx, job.ID, model.JobDone, "")
	}
}

func (p *Pool) Submit(job model.GraphJob) {
	select {
	case p.jobs <- job:
	default:
		p.syncFallback.Add(1)
		logger.L().Warn("queue full, sync fallback", slog.String("kind", job.Kind))
		p.run(job)
	}
}

func (p *Pool) SubmitOrRun(job model.GraphJob, syncFn func(context.Context) error) {
	select {
	case p.jobs <- job:
	default:
		p.syncFallback.Add(1)
		if syncFn != nil {
			_ = syncFn(p.ctx)
		} else {
			p.run(job)
		}
	}
}

func (p *Pool) Recover(ctx context.Context) {
	if p.db == nil {
		return
	}
	jobs, err := p.db.RecoverPendingJobs(ctx)
	if err != nil {
		logger.L().Error("recover jobs", slog.String("err", err.Error()))
		return
	}
	for _, j := range jobs {
		p.Submit(j)
	}
}

func (p *Pool) Close() {
	p.cancel()
	p.wg.Wait()
}

func (p *Pool) Depth() int { return len(p.jobs) }
func (p *Pool) Cap() int   { return cap(p.jobs) }
func (p *Pool) Snapshot() model.MetricsSnapshot {
	return model.MetricsSnapshot{
		QueueDepth:   p.Depth(),
		QueueCap:     p.Cap(),
		SyncFallback: p.syncFallback.Load(),
		JobsDone:     p.done.Load(),
		JobsFailed:   p.failed.Load(),
	}
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	s := strings.ToLower(err.Error())
	needles := []string{"connection", "timeout", "reset", "broken pipe", "eof", "i/o"}
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func IsBusinessError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "validation") || strings.Contains(s, "not_found") || strings.Contains(s, "conflict")
}

func ShouldRetry(err error, attempts int) bool {
	if attempts >= 3 {
		return false
	}
	if IsBusinessError(err) {
		return false
	}
	return isTransient(err)
}
