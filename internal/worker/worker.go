package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/metrics"
	"github.com/crypticseeds/concurrent-job-queue/internal/queue"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
	"github.com/redis/go-redis/v9"
)

// Pool manages a collection of workers that process tasks concurrently.
// It owns the lifecycle of the workers and the queue.
type Pool struct {
	store       task.Store
	queue       queue.Queue
	metrics     metrics.Collector
	workerCount int
	redisClient *redis.Client // Keep client to create unique consumer names
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

// NewPool initializes a new worker pool with the given task store, queue and worker count.
func NewPool(store task.Store, q queue.Queue, metrics metrics.Collector, workerCount int) *Pool {
	if workerCount <= 0 {
		panic(fmt.Sprintf("worker.NewPool: workerCount must be > 0, got %d", workerCount))
	}
	if store == nil {
		panic("worker.NewPool: store must not be nil")
	}
	if q == nil {
		panic("worker.NewPool: queue must not be nil")
	}

	return &Pool{
		store:       store,
		queue:       q,
		metrics:     metrics,
		workerCount: workerCount,
	}
}

// SetRedisClient allows passing the redis client for specialized consumer naming.
func (p *Pool) SetRedisClient(client *redis.Client) {
	p.redisClient = client
}

// Start launches the worker goroutines.
// Each worker will listen on the queue until the context is cancelled.
func (p *Pool) Start(ctx context.Context) {
	hostname, _ := os.Hostname()
	for i := 1; i <= p.workerCount; i++ {
		p.wg.Add(1)
		// If using Redis, we might want unique consumer names per worker goroutine
		workerQueue := p.queue
		if p.redisClient != nil {
			// Create a specialized RedisQueue for each worker with a unique consumer name
			consumerName := fmt.Sprintf("%s-worker-%d", hostname, i)
			if rq, err := queue.NewRedisQueue(p.redisClient, "task_stream", "worker_group", consumerName); err == nil {
				workerQueue = rq
			}
		}
		go p.worker(ctx, i, workerQueue)
	}
}

// worker represents a single goroutine that processes tasks from the queue.
func (p *Pool) worker(ctx context.Context, id int, q queue.Queue) {
	defer p.wg.Done()
	logger := slog.With("worker_id", id)
	logger.Info("Worker started")

	for {
		t, err := q.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			logger.Error("Error dequeuing task", "error", err)
			time.Sleep(1 * time.Second) // Backoff
			continue
		}

		tlog := logger.With("task_id", t.ID)
		tlog.Info("Worker received task")

		// Update to Running
		if err := p.store.UpdateStatus(t.ID, task.StatusRunning); err != nil {
			tlog.Error("Error updating status to RUNNING", "error", err)
		}

		// Simulate work (duration can be passed in payload for testing)
		workDuration := 100 * time.Millisecond
		if dStr, ok := t.Payload.(string); ok {
			if d, err := time.ParseDuration(dStr); err == nil {
				workDuration = d
			}
		} else if d, ok := t.Payload.(time.Duration); ok {
			workDuration = d
		} else if dFloat, ok := t.Payload.(float64); ok {
			// JSON numbers are often float64
			workDuration = time.Duration(dFloat * float64(time.Millisecond))
		}

		tlog.Info("Processing task", "duration", workDuration)

		startTime := time.Now()
		// Handle processing
		time.Sleep(workDuration)

		// Success
		p.metrics.ObserveTaskLatency(time.Since(startTime).Seconds())
		p.metrics.ObserveEndToEndLatency(time.Since(t.CreatedAt).Seconds())
		if err := p.store.UpdateStatus(t.ID, task.StatusCompleted); err != nil {
			tlog.Error("Error updating status to COMPLETED", "error", err)
			p.metrics.IncTasksFailed()
		} else {
			p.metrics.IncTasksCompleted()
			if err := p.queue.Ack(t); err != nil {
				tlog.Error("Error acknowledging task", "error", err)
			}
		}

		tlog.Info("Worker finished task")
	}

	logger.Info("Worker stopped")
}

// Shutdown gracefully stops the worker pool, waiting for in-flight tasks to complete.
func (p *Pool) Shutdown() {
	p.wg.Wait()
	slog.Info("Worker pool: all workers stopped")
}
