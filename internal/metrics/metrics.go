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
	IncTasksRejected()
	ObserveTaskLatency(duration float64)
	ObserveEndToEndLatency(duration float64)
	SetQueueDepth(depth int64)
}

// Snapshot represents a point-in-time view of the metrics.
type Snapshot struct {
	TasksCreated   uint64  `json:"tasks_created_total"`
	TasksCompleted uint64  `json:"tasks_completed_total"`
	TasksFailed    uint64  `json:"tasks_failed_total"`
	TasksRejected  uint64  `json:"tasks_rejected_total"`
	AvgLatency     float64 `json:"avg_latency_seconds,omitempty"`
	QueueDepth     int64   `json:"queue_depth,omitempty"`
}

// MemCollector is an in-memory implementation for tests.
type MemCollector struct {
	created   uint64
	completed uint64
	failed    uint64
	rejected  uint64
	depth     int64
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

func (c *MemCollector) IncTasksRejected() {
	atomic.AddUint64(&c.rejected, 1)
}

func (c *MemCollector) ObserveTaskLatency(duration float64) {}

func (c *MemCollector) ObserveEndToEndLatency(duration float64) {}

func (c *MemCollector) SetQueueDepth(depth int64) {
	atomic.StoreInt64(&c.depth, depth)
}

func (c *MemCollector) GetMetrics() Snapshot {
	return Snapshot{
		TasksCreated:   atomic.LoadUint64(&c.created),
		TasksCompleted: atomic.LoadUint64(&c.completed),
		TasksFailed:    atomic.LoadUint64(&c.failed),
		TasksRejected:  atomic.LoadUint64(&c.rejected),
		QueueDepth:     atomic.LoadInt64(&c.depth),
	}
}

// OTELCollector implements the Collector interface using OpenTelemetry.
type OTELCollector struct {
	created    metric.Int64Counter
	completed  metric.Int64Counter
	failed     metric.Int64Counter
	rejected   metric.Int64Counter
	latency    metric.Float64Histogram
	e2eLatency metric.Float64Histogram
	depth      metric.Int64Gauge
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
	rejected, err := meter.Int64Counter("tasks_rejected_total", metric.WithDescription("Total number of tasks rejected"))
	if err != nil {
		return nil, nil, fmt.Errorf("create tasks_rejected_total counter: %w", err)
	}
	latency, err := meter.Float64Histogram("task_processing_duration_seconds", metric.WithDescription("Time taken to process tasks"))
	if err != nil {
		return nil, nil, fmt.Errorf("create task_processing_duration_seconds histogram: %w", err)
	}
	e2eLatency, err := meter.Float64Histogram("task_e2e_latency_seconds", metric.WithDescription("Total time from task creation to completion"))
	if err != nil {
		return nil, nil, fmt.Errorf("create task_e2e_latency_seconds histogram: %w", err)
	}
	depth, err := meter.Int64Gauge("queue_depth", metric.WithDescription("Current number of pending tasks in the queue"))
	if err != nil {
		return nil, nil, fmt.Errorf("create queue_depth gauge: %w", err)
	}

	return &OTELCollector{
		created:    created,
		completed:  completed,
		failed:     failed,
		rejected:   rejected,
		latency:    latency,
		e2eLatency: e2eLatency,
		depth:      depth,
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

// IncTasksRejected increments the count of rejected tasks.
func (c *OTELCollector) IncTasksRejected() {
	c.rejected.Add(context.Background(), 1)
}

func (c *OTELCollector) ObserveTaskLatency(duration float64) {
	c.latency.Record(context.Background(), duration)
}

func (c *OTELCollector) ObserveEndToEndLatency(duration float64) {
	c.e2eLatency.Record(context.Background(), duration)
}

func (c *OTELCollector) SetQueueDepth(depth int64) {
	c.depth.Record(context.Background(), depth)
}
