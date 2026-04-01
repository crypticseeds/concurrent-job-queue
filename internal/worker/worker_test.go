package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/metrics"
	"github.com/crypticseeds/concurrent-job-queue/internal/queue"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
)

func TestPool(t *testing.T) {
	store := task.NewMemStore()
	collector := metrics.NewMemCollector()

	t.Run("Task Lifecycle", func(t *testing.T) {
		// Use a small pool
		q := queue.NewInMemoryQueue(10)
		pool := NewPool(store, q, collector, 1)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		pool.Start(ctx)

		taskID := "lifecycle-test"
		sTask := task.NewTask(taskID, "100ms")
		store.Add(sTask)

		q.Enqueue(sTask)

		// Poll for completion (max 5s, given 100ms simulation)
		deadline := time.Now().Add(6 * time.Second)
		var lastTask *task.Task
		for time.Now().Before(deadline) {
			lastTask, _ = store.Get(taskID)
			if lastTask.Status == task.StatusCompleted {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if lastTask.Status != task.StatusCompleted {
			t.Errorf("expected status COMPLETED, got %s", lastTask.Status)
		}

		m := collector.GetMetrics()
		if m.TasksCompleted != 1 {
			t.Errorf("expected 1 completed metric, got %d", m.TasksCompleted)
		}
	})

	t.Run("Shutdown", func(t *testing.T) {
		q := queue.NewInMemoryQueue(10)
		pool := NewPool(store, q, collector, 2)
		ctx, cancel := context.WithCancel(t.Context())
		pool.Start(ctx)

		// Submit 2 tasks that take 300ms each
		task1 := task.NewTask("s1", "300ms")
		task2 := task.NewTask("s2", "300ms")
		store.Add(task1)
		store.Add(task2)
		q.Enqueue(task1)
		q.Enqueue(task2)

		// Wait a tiny bit for workers to pick them up
		time.Sleep(100 * time.Millisecond)

		start := time.Now()
		cancel()
		pool.Shutdown()
		elapsed := time.Since(start)

		// Shutdown should wait for the tasks to finish
		if elapsed < 150*time.Millisecond {
			t.Errorf("pool shut down too fast (%v), didn't wait for tasks", elapsed)
		}

		m1, _ := store.Get("s1")
		m2, _ := store.Get("s2")
		if m1.Status != task.StatusCompleted || m2.Status != task.StatusCompleted {
			t.Errorf("tasks not completed after shutdown: s1=%s, s2=%s", m1.Status, m2.Status)
		}
	})

	t.Run("Non-Blocking Submit (Queue Full)", func(t *testing.T) {
		// Queue with 1 slot and 0 worker (to keep slot occupied)
		q := queue.NewInMemoryQueue(1)

		// Fill the queue
		task1 := task.NewTask("j1", nil)
		if err := q.Enqueue(task1); err != nil {
			t.Fatalf("failed to enqueue first task: %v", err)
		}

		// Next enqueue should fail
		task2 := task.NewTask("j2", nil)
		err := q.Enqueue(task2)
		if !errors.Is(err, queue.ErrQueueFull) {
			t.Errorf("expected ErrQueueFull, got %v", err)
		}
	})
}
