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
	"github.com/crypticseeds/concurrent-job-queue/internal/queue"
	"github.com/crypticseeds/concurrent-job-queue/internal/server"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
	"github.com/crypticseeds/concurrent-job-queue/internal/worker"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting Concurrent Job Queue Server")

	// 1. Initialize dependencies
	workerCount := getEnvInt("WORKER_COUNT", 3)
	taskTTL := getEnvDuration("TASK_TTL", 1*time.Hour)
	readTimeout := getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second)
	writeTimeout := getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second)
	idleTimeout := getEnvDuration("HTTP_IDLE_TIMEOUT", 120*time.Second)

	redisAddr := os.Getenv("REDIS_ADDR")
	pgConnString := os.Getenv("DATABASE_URL")

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

	var store task.Store
	var q queue.Queue
	var pool *worker.Pool

	if pgConnString != "" {
		slog.Info("Using PostgreSQL store")
		pgStore, err := task.NewPostgresStore(ctx, pgConnString)
		if err != nil {
			slog.Error("Failed to initialize Postgres store", "error", err)
			os.Exit(1)
		}
		store = pgStore
	} else {
		slog.Info("Using in-memory sharded store")
		store = task.NewShardedStore(32)
	}

	if redisAddr != "" {
		slog.Info("Using Redis queue", "addr", redisAddr)
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		hostname, _ := os.Hostname()
		redisQueue, err := queue.NewRedisQueue(rdb, "task_stream", "worker_group", hostname)
		if err != nil {
			slog.Error("Failed to initialize Redis queue", "error", err)
			os.Exit(1)
		}
		q = redisQueue

		pool = worker.NewPool(store, q, metricsCollector, workerCount)
		pool.SetRedisClient(rdb) // Reuse the same client pool
	} else {
		slog.Info("Using in-memory queue")
		q = queue.NewInMemoryQueue(100) // Default to 100 for in-memory
		pool = worker.NewPool(store, q, metricsCollector, workerCount)
	}

	// 2. Start worker pool
	pool.Start(ctx)

	// 3. Start background cleanup and metrics update
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		metricsTicker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		defer metricsTicker.Stop()
		for {
			select {
			case <-ticker.C:
				slog.Debug("Running background task cleanup", "ttl", taskTTL)
				store.Cleanup(taskTTL)
			case <-metricsTicker.C:
				if depth, err := q.Depth(); err == nil {
					metricsCollector.SetQueueDepth(depth)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	srv := server.NewServer(store, q, pool, metricsCollector, metricsHandler)
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      srv,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	go func() {
		slog.Info("Server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
		}
	}()

	<-ctx.Done()
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
	if d <= 0 {
		slog.Warn("Environment variable duration must be positive, using default", "key", key, "value", val, "default", defaultValue)
		return defaultValue
	}
	return d
}
