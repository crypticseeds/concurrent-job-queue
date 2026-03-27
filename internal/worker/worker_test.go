package worker

import (
	"context"
	"errors"
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

		pool.Submit(Job{TaskID: taskID, Payload: nil})

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
		pool.Submit(Job{TaskID: "s1", Payload: nil})
		pool.Submit(Job{TaskID: "s2", Payload: nil})

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

	t.Run("Non-Blocking Submit (Queue Full)", func(t *testing.T) {
		// Pool with 0 queue size and 1 worker
		pool := NewPool(store, collector, 1, 0)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		pool.Start(ctx)

		// Wait a bit to ensure worker is ready
		time.Sleep(50 * time.Millisecond)

		// Submit one job to occupy the worker
		job1 := Job{TaskID: "j1", Payload: nil}
		if err := pool.Submit(job1); err != nil {
			t.Fatalf("failed to submit first job: %v", err)
		}

		// Since queue size is 0, the next submit should fail immediately
		// but we might need a tiny sleep to ensure the worker has NOT yet finished
		// OR we can just rely on the fact that the worker takes 3s.
		job2 := Job{TaskID: "j2", Payload: nil}
		err := pool.Submit(job2)
		if !errors.Is(err, ErrQueueFull) {
			t.Errorf("expected ErrQueueFull, got %v", err)
		}
	})
}
