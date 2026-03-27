package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strconv"

	"github.com/crypticseeds/concurrent-job-queue/internal/metrics"
	"github.com/crypticseeds/concurrent-job-queue/internal/server"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
	"github.com/crypticseeds/concurrent-job-queue/internal/worker"
)

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting Concurrent Job Queue Server")

	// 1. Initialize dependencies
	workerCount := getEnvInt("WORKER_COUNT", 3)
	queueSize := getEnvInt("QUEUE_SIZE", 10)
	taskTTL := getEnvDuration("TASK_TTL", 1*time.Hour)
	if workerCount <= 0 {
		slog.Error("Invalid configuration: WORKER_COUNT must be greater than 0", "worker_count", workerCount)
		os.Exit(1)
	}
	if queueSize <= 0 {
		slog.Error("Invalid configuration: QUEUE_SIZE must be greater than 0", "queue_size", queueSize)
		os.Exit(1)
	}

	slog.Info("Configuration", "worker_count", workerCount, "queue_size", queueSize, "task_ttl", taskTTL)

	store := task.NewShardedStore(32)

	var metricsCollector metrics.Collector
	var metricsHandler http.Handler

	// Initialize OTEL metrics
	otelCollector, metricsExporter, err := metrics.NewOTELCollector()
	if err != nil {
		slog.Error("Failed to initialize OTEL metrics, falling back to in-memory", "error", err)
		metricsCollector = metrics.NewMemCollector()
	} else {
		metricsCollector = otelCollector
		metricsHandler = metricsExporter
	}

	pool := worker.NewPool(store, metricsCollector, workerCount, queueSize)

	// 2. Start worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	// 3. Start background cleanup
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				slog.Debug("Running background task cleanup", "ttl", taskTTL)
				store.Cleanup(taskTTL)
			case <-ctx.Done():
				return
			}
		}
	}()

	srv := server.NewServer(store, pool, metricsCollector, metricsHandler)
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: srv,
	}

	// 4. Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("Shutting down server")

	// 5. Shutdown sequence
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	pool.Shutdown()
	slog.Info("Service exited gracefully")
}

func getEnvInt(key string, defaultValue int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		slog.Warn("Invalid environment variable value, using default", "key", key, "value", val, "default", defaultValue)
		return defaultValue
	}
	return i
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		slog.Warn("Invalid environment variable duration, using default", "key", key, "value", val, "default", defaultValue)
		return defaultValue
	}
	return d
}
