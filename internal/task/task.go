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
	ID         string        `json:"id"`
	Status     Status        `json:"status"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Retries    int           `json:"retries"`
	LastError  string        `json:"last_error,omitempty"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	// Metadata stores implementation-specific data (e.g., Redis Message ID)
	Metadata map[string]string `json:"metadata,omitempty"`
	// Payload can be used to pass data to the worker if needed later
	Payload any `json:"payload,omitempty"`
}

// NewTask creates a new task instance with Pending status.
func NewTask(id string, payload any) *Task {
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
	var metadata map[string]string
	if t.Metadata != nil {
		metadata = make(map[string]string, len(t.Metadata))
		for k, v := range t.Metadata {
			metadata[k] = v
		}
	}
	return &Task{
		ID:         t.ID,
		Status:     t.Status,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
		Retries:    t.Retries,
		LastError:  t.LastError,
		RetryAfter: t.RetryAfter,
		Metadata:   metadata,
		Payload:    t.Payload,
	}
}
