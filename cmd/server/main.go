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

	slog.Info("Configuration", "worker_count", workerCount, "queue_size", queueSize)

	store := task.NewMemStore()
	pool := worker.NewPool(store, workerCount, queueSize)

	// 2. Start worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	// 3. Initialize HTTP server
	srv := server.NewServer(store, pool)
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
