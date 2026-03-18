package task

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// Store defines the interface for task storage.
type Store interface {
	Add(t *Task)
	Get(id string) (*Task, error)
	UpdateStatus(id string, status Status) error
}

// MemStore is an in-memory implementation of the Store interface.
type MemStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

// NewMemStore initializes a new in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{
		tasks: make(map[string]*Task),
	}
}

// Add saves a new task to the store.
func (s *MemStore) Add(t *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t.Clone()
}

// Get retrieves a task by ID.
func (s *MemStore) Get(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t.Clone(), nil
}

// UpdateStatus changes the status of an existing task and updates the timestamp.
func (s *MemStore) UpdateStatus(id string, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	return nil
}
