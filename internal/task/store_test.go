package task

import (
	"sync"
	"testing"
	"time"
)

func TestMemStore(t *testing.T) {
	s := NewMemStore()
	testStore(t, s)
}

func TestShardedStore(t *testing.T) {
	s := NewShardedStore(8)
	testStore(t, s)
}

func testStore(t *testing.T, s Store) {
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
		var wg sync.WaitGroup

		// Concurrent adds
		wg.Add(n)
		for i := range n {
			go func() {
				defer wg.Done()
				id := "concurrent-task-" + string(rune(i))
				s.Add(NewTask(id, nil))
			}()
		}
		wg.Wait()

		// Concurrent updates to same task
		taskID := "single-task"
		s.Add(NewTask(taskID, nil))
		wg.Add(n)
		for range n {
			go func() {
				defer wg.Done()
				_ = s.UpdateStatus(taskID, StatusRunning)
			}()
		}
		wg.Wait()
	})

	t.Run("Cleanup", func(t *testing.T) {
		task1 := NewTask("t1", nil) // Completed, old
		task2 := NewTask("t2", nil) // Completed, new
		task3 := NewTask("t3", nil) // Failed, old
		task4 := NewTask("t4", nil) // Running, old (should NOT be cleaned)

		s.Add(task1)
		s.Add(task2)
		s.Add(task3)
		s.Add(task4)

		_ = s.UpdateStatus("t1", StatusCompleted)
		_ = s.UpdateStatus("t2", StatusCompleted)
		_ = s.UpdateStatus("t3", StatusFailed)
		_ = s.UpdateStatus("t4", StatusRunning)

		// Manually backdate UpdatedAt for "old" tasks
		// Since we can't easily reach into the store's private map for ShardedStore without reflecting,
		// we'll rely on the fact that MemStore is used by ShardedStore and we can test MemStore specifically
		// or just use a very short TTL and sleep.
		// However, for a robust test, let's just use a short TTL.

		time.Sleep(100 * time.Millisecond)
		ttl := 50 * time.Millisecond

		// task2 is "new" because we update it just before cleanup
		_ = s.UpdateStatus("t2", StatusCompleted)

		s.Cleanup(ttl)

		// t1 and t3 should be gone. t2 and t4 should remain.
		if _, err := s.Get("t1"); err == nil {
			t.Error("expected t1 to be cleaned up")
		}
		if _, err := s.Get("t3"); err == nil {
			t.Error("expected t3 to be cleaned up")
		}
		if _, err := s.Get("t2"); err != nil {
			t.Error("expected t2 to remain (newly updated)")
		}
		if _, err := s.Get("t4"); err != nil {
			t.Error("expected t4 to remain (not in terminal state)")
		}
	})
}
