package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/server"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
	"github.com/crypticseeds/concurrent-job-queue/internal/worker"
)

func main() {
	log.Println("Starting Concurrent Job Queue Server...")

	// 1. Initialize dependencies
	store := task.NewMemStore()
	pool := worker.NewPool(store, 3, 10) // 3 workers, queue size 10 (configurable later)

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
		log.Printf("Server listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	// 5. Shutdown sequence
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	pool.Shutdown()
	log.Println("Service exited gracefully")
}
