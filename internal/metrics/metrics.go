package metrics

import (
	"sync/atomic"
)

// Collector defines the interface for collecting system metrics.
type Collector interface {
	IncTasksCreated()
	IncTasksCompleted()
	IncTasksFailed()
	GetMetrics() Snapshot
}

// Snapshot represents a point-in-time view of the metrics.
type Snapshot struct {
	TasksCreated   uint64 `json:"tasks_created_total"`
	TasksCompleted uint64 `json:"tasks_completed_total"`
	TasksFailed    uint64 `json:"tasks_failed_total"`
}

// MemCollector is an in-memory implementation of the Collector interface using atomic operations.
type MemCollector struct {
	created   uint64
	completed uint64
	failed    uint64
}

// NewMemCollector initializes a new in-memory metrics collector.
func NewMemCollector() *MemCollector {
	return &MemCollector{}
}

// IncTasksCreated increments the count of created tasks.
func (c *MemCollector) IncTasksCreated() {
	atomic.AddUint64(&c.created, 1)
}

// IncTasksCompleted increments the count of completed tasks.
func (c *MemCollector) IncTasksCompleted() {
	atomic.AddUint64(&c.completed, 1)
}

// IncTasksFailed increments the count of failed tasks.
func (c *MemCollector) IncTasksFailed() {
	atomic.AddUint64(&c.failed, 1)
}

// GetMetrics returns a snapshot of current metrics.
func (c *MemCollector) GetMetrics() Snapshot {
	return Snapshot{
		TasksCreated:   atomic.LoadUint64(&c.created),
		TasksCompleted: atomic.LoadUint64(&c.completed),
		TasksFailed:    atomic.LoadUint64(&c.failed),
	}
}
