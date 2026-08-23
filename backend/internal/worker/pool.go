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

	// mu guards overflow. overflow parks retried jobs that could not be
	// pushed back into the bounded jobs channel because it was momentarily
	// full. A dedicated drainer goroutine feeds them back into jobs as soon
	// as a worker frees a slot.
	mu       sync.Mutex
	overflow []model.GraphJob
	wake     chan struct{}
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
		wake:    make(chan struct{}, 1),
	}
	p.wg.Add(1)
	go p.drain()
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.loop(i)
	}
	return p
}

// drain is a dedicated goroutine that feeds overflowed retried jobs back into
// the bounded jobs channel. Unlike a worker, it is NOT a consumer of jobs, so
// doing a blocking send here can never self-deadlock: it simply parks until a
// worker frees a slot. Without this, a worker that retries a job back into a
// full queue blocks forever (the classic single-worker / all-workers-retry
// self-deadlock), stalling the pipeline until the process is signalled.
//
// Overflow entries are popped before the send: if shutdown interrupts the
// in-flight send the job is already persisted as 'pending' in the DB, so
// Recover() will re-enqueue it on the next boot.
func (p *Pool) drain() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.wake:
			for {
				p.mu.Lock()
				if len(p.overflow) == 0 {
					p.mu.Unlock()
					break
				}
				job := p.overflow[0]
				p.overflow = p.overflow[1:]
				p.mu.Unlock()
				select {
				case p.jobs <- job:
				case <-p.ctx.Done():
					return
				}
			}
		}
	}
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
		}
		if p.db != nil {
			_ = p.db.MarkJob(ctx, job.ID, status, msg)
		}
		logger.L().Warn("job failed", slog.String("kind", job.Kind), slog.String("err", msg))
		if status == model.JobPending {
			// Retry re-enqueue must NEVER block a worker: a worker that
			// blocks on a full p.jobs would self-deadlock (it is the very
			// consumer that must drain the queue), pinning queue_depth at
			// queue_cap until a stop signal arrives. Park on overflow
			// instead; the dedicated drainer feeds it back as soon as a
			// slot frees up.
			job.Attempts++
			p.requeue(job)
		}
		return
	}
	p.done.Add(1)
	if p.db != nil {
		_ = p.db.MarkJob(ctx, job.ID, model.JobDone, "")
	}
}

// requeue puts a retried job back in front of the line. It first tries a
// non-blocking send into p.jobs; if the bounded channel is full, it parks the
// job on the overflow list and nudges the drainer. This guarantees forward
// progress even when every worker is simultaneously retrying.
func (p *Pool) requeue(job model.GraphJob) {
	select {
	case p.jobs <- job:
		return
	default:
	}
	p.mu.Lock()
	p.overflow = append(p.overflow, job)
	p.mu.Unlock()
	select {
	case p.wake <- struct{}{}:
	default:
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

func (p *Pool) Depth() int {
	n := len(p.jobs)
	p.mu.Lock()
	n += len(p.overflow)
	p.mu.Unlock()
	return n
}
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
