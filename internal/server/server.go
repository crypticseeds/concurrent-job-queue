package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/crypticseeds/concurrent-job-queue/internal/metrics"
	"github.com/crypticseeds/concurrent-job-queue/internal/task"
	"github.com/crypticseeds/concurrent-job-queue/internal/worker"
	"github.com/google/uuid"
)

// Server represents the HTTP server for the concurrent job queue.
type Server struct {
	store          task.Store
	pool           *worker.Pool
	metrics        metrics.Collector
	metricsHandler http.Handler
	router         *http.ServeMux
}

// NewServer initializes a new Server with dependencies.
func NewServer(store task.Store, pool *worker.Pool, metrics metrics.Collector, metricsHandler http.Handler) *Server {
	s := &Server{
		store:          store,
		pool:           pool,
		metrics:        metrics,
		metricsHandler: metricsHandler,
		router:         http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures the routing for the server using modern Go 1.22+ patterns.
func (s *Server) setupRoutes() {
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("GET /metrics", s.handleMetrics)
	s.router.HandleFunc("POST /tasks", s.handleCreateTask)
	s.router.HandleFunc("GET /tasks/{id}", s.handleGetTask)
}

// ServeHTTP satisfies the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// handleHealth provides a simple service health check.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// handleMetrics returns the current system metrics.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.metricsHandler != nil {
		s.metricsHandler.ServeHTTP(w, r)
		return
	}

	// Fallback to JSON for MemCollector (legacy/tests)
	if mc, ok := s.metrics.(*metrics.MemCollector); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mc.GetMetrics())
		return
	}

	w.WriteHeader(http.StatusNotImplemented)
}

// CreateTaskRequest defines the expected payload for POST /tasks.
type CreateTaskRequest struct {
	Payload any `json:"payload"`
}

// CreateTaskResponse defines the response for successful task creation.
type CreateTaskResponse struct {
	ID     string      `json:"id"`
	Status task.Status `json:"status"`
}

// handleCreateTask handles POST requests to submit new tasks.
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
	}

	// Generate a new unique ID for the task
	taskID := uuid.New().String()
	t := task.NewTask(taskID, req.Payload)

	// Persist to store
	s.store.Add(t)

	// Submit to worker pool
	if err := s.pool.Submit(worker.Job{
		TaskID:  taskID,
		Payload: req.Payload,
	}); err != nil {
		if errors.Is(err, worker.ErrQueueFull) {
			s.metrics.IncTasksRejected()
			http.Error(w, "service unavailable: queue full", http.StatusServiceUnavailable)
			return
		}
		// Fallback for other errors if any
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	s.metrics.IncTasksCreated()

	slog.Info("Task created and submitted", "task_id", taskID)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateTaskResponse{
		ID:     t.ID,
		Status: t.Status,
	})
}

// handleGetTask handles GET requests to retrieve task status by ID.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	t, err := s.store.Get(id)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			slog.Debug("Task not found", "task_id", id)
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		slog.Error("Error fetching task", "task_id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}
