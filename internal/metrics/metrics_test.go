package metrics

import (
	"sync"
	"testing"
)

func TestMemCollector(t *testing.T) {
	c := NewMemCollector()

	t.Run("incremental updates", func(t *testing.T) {
		c.IncTasksCreated()
		c.IncTasksCompleted()
		c.IncTasksFailed()

		m := c.GetMetrics()
		if m.TasksCreated != 1 {
			t.Errorf("expected 1 created task, got %d", m.TasksCreated)
		}
		if m.TasksCompleted != 1 {
			t.Errorf("expected 1 completed task, got %d", m.TasksCompleted)
		}
		if m.TasksFailed != 1 {
			t.Errorf("expected 1 failed task, got %d", m.TasksFailed)
		}
	})

	t.Run("concurrent increments", func(t *testing.T) {
		const goroutines = 100
		const incrementsPerGoroutine = 1000
		c := NewMemCollector()
		var wg sync.WaitGroup

		wg.Add(goroutines * 3)
		for range goroutines {
			go func() {
				defer wg.Done()
				for range incrementsPerGoroutine {
					c.IncTasksCreated()
				}
			}()
			go func() {
				defer wg.Done()
				for range incrementsPerGoroutine {
					c.IncTasksCompleted()
				}
			}()
			go func() {
				defer wg.Done()
				for range incrementsPerGoroutine {
					c.IncTasksFailed()
				}
			}()
		}
		wg.Wait()

		m := c.GetMetrics()
		expected := uint64(goroutines * incrementsPerGoroutine)
		if m.TasksCreated != expected {
			t.Errorf("expected %d created tasks, got %d", expected, m.TasksCreated)
		}
		if m.TasksCompleted != expected {
			t.Errorf("expected %d completed tasks, got %d", expected, m.TasksCompleted)
		}
		if m.TasksFailed != expected {
			t.Errorf("expected %d failed tasks, got %d", expected, m.TasksFailed)
		}
	})
}
