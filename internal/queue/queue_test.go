package queue

import (
	"context"
	"testing"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/task"
)

func TestInMemoryQueue(t *testing.T) {
	q := NewInMemoryQueue(10)
	defer q.Close()

	ctx := context.Background()
	task1 := task.NewTask("task-1", "payload")

	// Test Enqueue
	if err := q.Enqueue(task1); err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Test Depth
	depth, _ := q.Depth()
	if depth != 1 {
		t.Errorf("expected depth 1, got %d", depth)
	}

	// Test Dequeue
	gotTask, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("failed to dequeue: %v", err)
	}
	if gotTask.ID != task1.ID {
		t.Errorf("expected task ID %s, got %s", task1.ID, gotTask.ID)
	}

	// Test Empty Depth
	depth, _ = q.Depth()
	if depth != 0 {
		t.Errorf("expected depth 0, got %d", depth)
	}
}

func TestInMemoryQueue_ContextCancel(t *testing.T) {
	q := NewInMemoryQueue(10)
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := q.Dequeue(ctx)
	if err == nil {
		t.Error("expected error on empty dequeue with timeout, got nil")
	}
}
