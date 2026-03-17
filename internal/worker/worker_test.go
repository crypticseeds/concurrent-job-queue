package worker

import (
	"context"
	"testing"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/metrics"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
)

func TestPool(t *testing.T) {
	store := task.NewMemStore()
	collector := metrics.NewMemCollector()

	t.Run("Task Lifecycle", func(t *testing.T) {
		// Use a small pool
		pool := NewPool(store, collector, 1, 10)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		pool.Start(ctx)

		taskID := "lifecycle-test"
		sTask := task.NewTask(taskID, nil)
		store.Add(sTask)

		pool.Submit(taskID)

		// Poll for completion (max 5s, given 3s simulation)
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
		pool := NewPool(store, collector, 2, 10)
		pool.Start(t.Context())

		// Submit 2 tasks that take 3s each
		store.Add(task.NewTask("s1", nil))
		store.Add(task.NewTask("s2", nil))
		pool.Submit("s1")
		pool.Submit("s2")

		// Wait a tiny bit for workers to pick them up
		time.Sleep(100 * time.Millisecond)

		start := time.Now()
		pool.Shutdown()
		elapsed := time.Since(start)

		// Shutdown should wait for the 3s tasks to finish
		if elapsed < 2*time.Second {
			t.Errorf("pool shut down too fast (%v), didn't wait for tasks", elapsed)
		}

		m1, _ := store.Get("s1")
		m2, _ := store.Get("s2")
		if m1.Status != task.StatusCompleted || m2.Status != task.StatusCompleted {
			t.Errorf("tasks not completed after shutdown: s1=%s, s2=%s", m1.Status, m2.Status)
		}
	})
}
