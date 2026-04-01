package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements the Store interface using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore initializes a new PostgreSQL-based task store.
func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Ensure schema exists (In production, use migrations)
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		retries INT DEFAULT 0,
		payload JSONB,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_error TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	`
	if _, err := pool.Exec(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

// Add saves a new task to the database.
func (s *PostgresStore) Add(t *Task) {
	ctx := context.Background()
	payload, _ := json.Marshal(t.Payload)

	query := `
	INSERT INTO tasks (id, status, retries, payload, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (id) DO UPDATE SET
		status = EXCLUDED.status,
		updated_at = EXCLUDED.updated_at;
	`
	_, err := s.pool.Exec(ctx, query, t.ID, t.Status, t.Retries, payload, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		fmt.Printf("Error adding task to Postgres: %v\n", err)
	}
}

// Get retrieves a task by ID.
func (s *PostgresStore) Get(id string) (*Task, error) {
	ctx := context.Background()
	var t Task
	var payload []byte

	query := `SELECT id, status, retries, payload, created_at, updated_at, last_error FROM tasks WHERE id = $1`
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Status, &t.Retries, &payload, &t.CreatedAt, &t.UpdatedAt, &t.LastError,
	)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	if len(payload) > 0 && string(payload) != "null" {
		_ = json.Unmarshal(payload, &t.Payload)
	}

	return &t, nil
}

// UpdateStatus changes the status of an existing task and updates the timestamp.
func (s *PostgresStore) UpdateStatus(id string, status Status) error {
	ctx := context.Background()
	query := `UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3`
	tag, err := s.pool.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// Cleanup removes tasks that have been in a terminal state for longer than the specified TTL.
func (s *PostgresStore) Cleanup(ttl time.Duration) {
	ctx := context.Background()
	query := `DELETE FROM tasks WHERE (status = 'COMPLETED' OR status = 'FAILED') AND updated_at < $1`
	_, err := s.pool.Exec(ctx, query, time.Now().Add(-ttl))
	if err != nil {
		fmt.Printf("Error during Postgres cleanup: %v\n", err)
	}
}

// Close closes the database connection pool.
func (s *PostgresStore) Close() {
	s.pool.Close()
}
