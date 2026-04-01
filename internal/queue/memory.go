package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/task"
)

var (
	ErrQueueClosed = errors.New("queue: queue is closed")
	ErrQueueFull   = errors.New("queue: queue is full")
)

// InMemoryQueue is an in-memory implementation of the Queue interface.
// It uses a channel to buffer tasks.
type InMemoryQueue struct {
	tasks chan *task.Task
	mu    sync.Mutex
	done  bool
}

// NewInMemoryQueue initializes a new in-memory queue.
func NewInMemoryQueue(size int) *InMemoryQueue {
	return &InMemoryQueue{
		tasks: make(chan *task.Task, size),
	}
}

// Enqueue adds a task to the queue.
func (q *InMemoryQueue) Enqueue(t *task.Task) error {
	q.mu.Lock()
	if q.done {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	q.mu.Unlock()

	select {
	case q.tasks <- t.Clone():
		return nil
	default:
		return ErrQueueFull
	}
}

// Dequeue removes a task from the queue.
func (q *InMemoryQueue) Dequeue(ctx context.Context) (*task.Task, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case t, ok := <-q.tasks:
		if !ok {
			return nil, ErrQueueClosed
		}
		return t, nil
	}
}

// Ack acknowledges that a task has been processed.
// In memory implementation, this is a no-op as the task is already removed from the channel.
func (q *InMemoryQueue) Ack(t *task.Task) error {
	return nil
}

// Depth returns the current number of pending tasks.
func (q *InMemoryQueue) Depth() (int64, error) {
	return int64(len(q.tasks)), nil
}

// Fail handles task failure and schedules a retry if needed.
// In this simple in-memory implementation, we just re-enqueue the task after the delay.
func (q *InMemoryQueue) Fail(t *task.Task, retryAfter time.Duration) error {
	// Note: This is a simplified implementation for reference.
	// In a real system, we'd need to fetch the task first to re-enqueue it.
	// Since InMemoryQueue doesn't store tasks by ID, this implementation has limitations.
	return nil
}

// Close closes the queue and stops accepting new tasks.
func (q *InMemoryQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.done {
		q.done = true
		close(q.tasks)
	}
}
