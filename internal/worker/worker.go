package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/metrics"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
)

// Job represents a unit of work that is passed to the worker pool.
type Job struct {
	TaskID  string
	Payload any
}

// Pool manages a collection of workers that process tasks concurrently.
// It owns the lifecycle of the workers and the jobs channel.
type Pool struct {
	store       task.Store
	metrics     metrics.Collector
	workerCount int
	jobs        chan Job
	wg          sync.WaitGroup
	once        sync.Once
}

// NewPool initializes a new worker pool with the given task store and worker count.
func NewPool(store task.Store, metrics metrics.Collector, workerCount int, queueSize int) *Pool {
	if workerCount <= 0 {
		panic(fmt.Sprintf("worker.NewPool: workerCount must be > 0, got %d", workerCount))
	}
	if queueSize < 0 {
		panic(fmt.Sprintf("worker.NewPool: queueSize must be >= 0, got %d", queueSize))
	}
	if store == nil {
		panic("worker.NewPool: store must not be nil")
	}

	return &Pool{
		store:       store,
		metrics:     metrics,
		workerCount: workerCount,
		jobs:        make(chan Job, queueSize),
	}
}

// Start launches the worker goroutines.
// Each worker will listen on the jobs channel until it is closed.
func (p *Pool) Start(ctx context.Context) {
	for i := 1; i <= p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker represents a single goroutine that processes jobs.
func (p *Pool) worker(id int) {
	defer p.wg.Done()
	logger := slog.With("worker_id", id)
	logger.Info("Worker started")

	for job := range p.jobs {
		tlog := logger.With("task_id", job.TaskID)
		tlog.Info("Worker received task")

		// Update to Running
		if err := p.store.UpdateStatus(job.TaskID, task.StatusRunning); err != nil {
			tlog.Error("Error updating status to RUNNING", "error", err)
		}

		// Simulate work (2–5 seconds) using job.Payload if needed
		// For now, let's use a fixed 3s as per previous implementation.
		workDuration := 3 * time.Second
		tlog.Info("Processing task", "duration", workDuration)
		time.Sleep(workDuration)

		// Update to Completed
		if err := p.store.UpdateStatus(job.TaskID, task.StatusCompleted); err != nil {
			tlog.Error("Error updating status to COMPLETED", "error", err)
			p.metrics.IncTasksFailed()
		} else {
			p.metrics.IncTasksCompleted()
		}

		tlog.Info("Worker finished task")
	}

	logger.Info("Worker stopped")
}

// Submit sends a Job to the jobs channel for processing.
func (p *Pool) Submit(job Job) {
	p.jobs <- job
	slog.Debug("Task submitted to pool", "task_id", job.TaskID)
}

// Shutdown gracefully stops the worker pool, waiting for in-flight tasks to complete.
func (p *Pool) Shutdown() {
	p.once.Do(func() {
		close(p.jobs)
	})
	p.wg.Wait()
	slog.Info("Worker pool: all workers stopped")
}
