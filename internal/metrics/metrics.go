package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Collector defines the interface for collecting system metrics.
type Collector interface {
	IncTasksCreated()
	IncTasksCompleted()
	IncTasksFailed()
}

// Snapshot represents a point-in-time view of the metrics.
type Snapshot struct {
	TasksCreated   uint64 `json:"tasks_created_total"`
	TasksCompleted uint64 `json:"tasks_completed_total"`
	TasksFailed    uint64 `json:"tasks_failed_total"`
}

// MemCollector is an in-memory implementation for tests.
type MemCollector struct {
	created   uint64
	completed uint64
	failed    uint64
}

func NewMemCollector() *MemCollector {
	return &MemCollector{}
}

func (c *MemCollector) IncTasksCreated() {
	atomic.AddUint64(&c.created, 1)
}

func (c *MemCollector) IncTasksCompleted() {
	atomic.AddUint64(&c.completed, 1)
}

func (c *MemCollector) IncTasksFailed() {
	atomic.AddUint64(&c.failed, 1)
}

func (c *MemCollector) GetMetrics() Snapshot {
	return Snapshot{
		TasksCreated:   atomic.LoadUint64(&c.created),
		TasksCompleted: atomic.LoadUint64(&c.completed),
		TasksFailed:    atomic.LoadUint64(&c.failed),
	}
}

// OTELCollector implements the Collector interface using OpenTelemetry.
type OTELCollector struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
	failed    metric.Int64Counter
}

// NewOTELCollector initializes OpenTelemetry metrics and returns a collector.
func NewOTELCollector() (*OTELCollector, http.Handler, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := meterProvider.Meter("job-queue")

	created, err := meter.Int64Counter("tasks_created_total", metric.WithDescription("Total number of tasks created"))
	if err != nil {
		return nil, nil, fmt.Errorf("create tasks_created_total counter: %w", err)
	}
	completed, err := meter.Int64Counter("tasks_completed_total", metric.WithDescription("Total number of tasks completed"))
	if err != nil {
		return nil, nil, fmt.Errorf("create tasks_completed_total counter: %w", err)
	}
	failed, err := meter.Int64Counter("tasks_failed_total", metric.WithDescription("Total number of tasks failed"))
	if err != nil {
		return nil, nil, fmt.Errorf("create tasks_failed_total counter: %w", err)
	}

	return &OTELCollector{
		created:   created,
		completed: completed,
		failed:    failed,
	}, promhttp.Handler(), nil
}

// IncTasksCreated increments the count of created tasks.
func (c *OTELCollector) IncTasksCreated() {
	c.created.Add(context.Background(), 1)
}

// IncTasksCompleted increments the count of completed tasks.
func (c *OTELCollector) IncTasksCompleted() {
	c.completed.Add(context.Background(), 1)
}

// IncTasksFailed increments the count of failed tasks.
func (c *OTELCollector) IncTasksFailed() {
	c.failed.Add(context.Background(), 1)
}
