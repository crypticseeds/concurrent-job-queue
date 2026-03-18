package task

import (
	"sync"
	"testing"
)

func TestMemStore(t *testing.T) {
	s := NewMemStore()

	t.Run("Add and Get", func(t *testing.T) {
		taskID := "test-task-1"
		task := NewTask(taskID, "payload")
		s.Add(task)

		got, err := s.Get(taskID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != taskID {
			t.Errorf("expected ID %s, got %s", taskID, got.ID)
		}
	})

	t.Run("Get Non-existent", func(t *testing.T) {
		_, err := s.Get("missing")
		if err == nil {
			t.Error("expected error for non-existent task, got nil")
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		taskID := "test-task-2"
		task := NewTask(taskID, nil)
		s.Add(task)

		err := s.UpdateStatus(taskID, StatusRunning)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _ := s.Get(taskID)
		if got.Status != StatusRunning {
			t.Errorf("expected status %s, got %s", StatusRunning, got.Status)
		}
	})

	t.Run("UpdateStatus Non-existent", func(t *testing.T) {
		err := s.UpdateStatus("missing", StatusRunning)
		if err == nil {
			t.Error("expected error for non-existent task status update, got nil")
		}
	})

	t.Run("Concurrent Add and UpdateStatus", func(t *testing.T) {
		const n = 100
		store := NewMemStore()
		var wg sync.WaitGroup

		// Concurrent adds
		wg.Add(n)
		for i := range n {
			go func() {
				defer wg.Done()
				id := "concurrent-task-" + string(rune(i))
				store.Add(NewTask(id, nil))
			}()
		}
		wg.Wait()

		// Concurrent updates to same task
		taskID := "single-task"
		store.Add(NewTask(taskID, nil))
		wg.Add(n)
		for range n {
			go func() {
				defer wg.Done()
				_ = store.UpdateStatus(taskID, StatusRunning)
			}()
		}
		wg.Wait()
	})
}
