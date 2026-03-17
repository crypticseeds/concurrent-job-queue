package server

import (
	"fmt"
	"net/http"

	"github.com/crypticseeds/concurrent-job-queue/internal/task"
	"github.com/crypticseeds/concurrent-job-queue/internal/worker"
)

// Server represents the HTTP server for the concurrent job queue.
type Server struct {
	store  task.Store
	pool   *worker.Pool
	router *http.ServeMux
}

// NewServer initializes a new Server with dependencies.
func NewServer(store task.Store, pool *worker.Pool) *Server {
	s := &Server{
		store:  store,
		pool:   pool,
		router: http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures the routing for the server using modern Go 1.22+ patterns.
func (s *Server) setupRoutes() {
	s.router.HandleFunc("GET /health", s.handleHealth)
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

// handleCreateTask handles POST requests to submit new tasks.
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	// Placeholder for Step 4.2
	fmt.Fprintln(w, "POST /tasks: not implemented yet")
}

// handleGetTask handles GET requests to retrieve task status by ID.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	// Placeholder for Step 4.2
	id := r.PathValue("id")
	fmt.Fprintf(w, "GET /tasks/%s: not implemented yet\n", id)
}
