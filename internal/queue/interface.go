package queue

import (
	"context"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/task"
)

// Queue defines the interface for a task queue.
type Queue interface {
	Enqueue(t *task.Task) error
	Dequeue(ctx context.Context) (*task.Task, error)
	Ack(t *task.Task) error
	Fail(t *task.Task, retryAfter time.Duration) error
	Depth() (int64, error)
}
