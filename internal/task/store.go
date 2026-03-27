package task

import (
	"errors"
	"hash/fnv"
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
	Cleanup(ttl time.Duration)
}

// MemStore is an in-memory implementation of the Store interface.
// It uses a single RWMutex, which can become a bottleneck under high concurrency.
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

// Cleanup removes tasks that have been in a terminal state for longer than the specified TTL.
func (s *MemStore) Cleanup(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, t := range s.tasks {
		if (t.Status == StatusCompleted || t.Status == StatusFailed) && now.Sub(t.UpdatedAt) > ttl {
			delete(s.tasks, id)
		}
	}
}

// ShardedStore is a highly concurrent implementation of the Store interface.
// It distributes tasks across multiple shards to reduce lock contention.
type ShardedStore struct {
	shards []*MemStore
	count  uint32
}

// NewShardedStore initializes a new sharded store with the specified number of shards.
// The shard count should be a power of two for optimal distribution.
func NewShardedStore(shardCount uint32) *ShardedStore {
	if shardCount == 0 {
		shardCount = 32
	}
	shards := make([]*MemStore, shardCount)
	for i := range shardCount {
		shards[i] = NewMemStore()
	}
	return &ShardedStore{
		shards: shards,
		count:  shardCount,
	}
}

// getShard returns the shard responsible for the given task ID.
func (s *ShardedStore) getShard(id string) *MemStore {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return s.shards[h.Sum32()%s.count]
}

// Add saves a new task to the appropriate shard in the sharded store.
func (s *ShardedStore) Add(t *Task) {
	s.getShard(t.ID).Add(t)
}

// Get retrieves a task by ID from the appropriate shard.
func (s *ShardedStore) Get(id string) (*Task, error) {
	return s.getShard(id).Get(id)
}

// UpdateStatus changes the status of an existing task in the appropriate shard.
func (s *ShardedStore) UpdateStatus(id string, status Status) error {
	return s.getShard(id).UpdateStatus(id, status)
}

// Cleanup triggers cleanup on all shards.
func (s *ShardedStore) Cleanup(ttl time.Duration) {
	for _, shard := range s.shards {
		shard.Cleanup(ttl)
	}
}
