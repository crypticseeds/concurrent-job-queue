package task

import (
	"time"
)

// Status represents the current state of a task.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusRunning   Status = "RUNNING"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
)

// Task represents a unit of work to be processed.
type Task struct {
	ID        string    `json:"id"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Payload can be used to pass data to the worker if needed later
	Payload interface{} `json:"payload,omitempty"`
}

// NewTask creates a new task instance with Pending status.
func NewTask(id string, payload interface{}) *Task {
	now := time.Now()
	return &Task{
		ID:        id,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		Payload:   payload,
	}
}

// Clone returns a deep copy of the task.
func (t *Task) Clone() *Task {
	if t == nil {
		return nil
	}
	return &Task{
		ID:        t.ID,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
		Payload:   t.Payload,
	}
}
