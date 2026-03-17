package worker

import (
	"context"
	"sync"

	"github.com/crypticseeds/concurrent-job-queue/internal/task"
)

// Pool manages a collection of workers that process tasks concurrently.
// It owns the lifecycle of the workers and the jobs channel.
type Pool struct {
	store       task.Store
	workerCount int
	jobs        chan string
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

// NewPool initializes a new worker pool with the given task store and worker count.
// We pass task IDs over the channel instead of task pointers because:
// 1. It ensures workers always fetch the most up-to-date state from the store.
// 2. It prevents race conditions or stale data issues from passing pointers around.
func NewPool(store task.Store, workerCount int, queueSize int) *Pool {
	return &Pool{
		store:       store,
		workerCount: workerCount,
		jobs:        make(chan string, queueSize),
	}
}

// Start launches the worker goroutines.
// Each worker will listen on the jobs channel until it is closed or the context is cancelled.
func (p *Pool) Start(ctx context.Context) {
	// Implementation will follow in Step 3.2
}

// Submit sends a task ID to the jobs channel for processing.
// The pool owns the channel to ensure controlled submission and backpressure.
func (p *Pool) Submit(taskID string) {
	// Implementation will follow in Step 3.2
}

// Shutdown gracefully stops the worker pool, waiting for in-flight tasks to complete.
func (p *Pool) Shutdown() {
	// Implementation will follow in Step 3.2
}
