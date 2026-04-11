// Package worker provides a goroutine-based worker pool for background job processing.
package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Job is a function executed by a worker goroutine.
type Job func(ctx context.Context)

// Pool manages a fixed set of worker goroutines fed by a shared job queue.
type Pool struct {
	jobs    chan Job
	wg      sync.WaitGroup
	once    sync.Once
	logger  *zap.Logger
}

// New creates a Pool with `size` goroutines and a job channel buffer of `queueSize`.
func New(size, queueSize int, logger *zap.Logger) *Pool {
	p := &Pool{
		jobs:   make(chan Job, queueSize),
		logger: logger,
	}
	p.start(size)
	return p
}

// start launches `n` persistent worker goroutines.
func (p *Pool) start(n int) {
	for i := 0; i < n; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			p.logger.Debug("worker started", zap.Int("worker_id", id))
			for job := range p.jobs {
				func() {
					defer func() {
						if r := recover(); r != nil {
							p.logger.Error("worker panic recovered", zap.Any("panic", r))
						}
					}()
					job(context.Background())
				}()
			}
			p.logger.Debug("worker stopped", zap.Int("worker_id", id))
		}(i)
	}
}

// Submit enqueues a job. Returns false if the queue is full (non-blocking).
func (p *Pool) Submit(job Job) bool {
	select {
	case p.jobs <- job:
		return true
	default:
		p.logger.Warn("worker pool queue is full, dropping job")
		return false
	}
}

// Shutdown drains the job channel and waits for all workers to finish.
// Safe to call multiple times.
func (p *Pool) Shutdown() {
	p.once.Do(func() {
		close(p.jobs)
		p.wg.Wait()
		p.logger.Info("worker pool shut down")
	})
}
