---
apply: always
---

# Go Project Rule — Concurrency & Production Practices

This project is a learning-focused Go service that implements a concurrent task processing API.
The goal is to practice idiomatic Go, concurrency safety, and production-style service design.

## Core Behaviour

When assisting with this project:

* Prefer **idiomatic Go patterns** over clever abstractions.
* Encourage **incremental implementation** and learning rather than generating full solutions.
* Provide **design feedback and hints** instead of complete implementations unless explicitly requested.
* Prioritize **clarity, correctness, and maintainability**.

---

# Development Workflow

Follow this process when reviewing or suggesting code.

1. **Analyze architecture**

    * Review module layout, package boundaries, and interfaces.
    * Check concurrency design (goroutines, channels, mutex usage).

2. **Design interfaces**

    * Prefer small focused interfaces.
    * Use composition over inheritance-style abstraction.

3. **Implement**

    * Write idiomatic Go with explicit error handling.
    * Propagate `context.Context` through blocking operations.

4. **Validate**

    * Ensure code passes:

   ```
   go vet ./...
   golangci-lint run
   ```

5. **Test**

    * Table-driven tests
    * Run tests with race detector:

   ```
   go test ./... -race
   ```

6. **Optimize**

    * Use benchmarks for performance work.
    * Profile with `pprof` when needed.

---

# Concurrency Guidelines

When working with goroutines or workers:

* Every goroutine must have a **clear lifecycle**.
* Goroutines must exit on **context cancellation**.
* Avoid goroutine leaks.
* Prefer **worker pools with bounded queues**.
* Use `select` when handling cancellation or multiple channels.
* Never block indefinitely without context awareness.

Example pattern:

```go
func worker(ctx context.Context, jobs <-chan Job, errCh chan<- error) {
    for {
        select {
        case <-ctx.Done():
            errCh <- fmt.Errorf("worker cancelled: %w", ctx.Err())
            return

        case job, ok := <-jobs:
            if !ok {
                return
            }

            if err := process(ctx, job); err != nil {
                errCh <- fmt.Errorf("process job %v: %w", job.ID, err)
                return
            }
        }
    }
}
```

Properties of this pattern:

* bounded goroutine lifetime
* proper cancellation
* explicit error propagation

---

# Code Quality Rules

Always enforce:

* `gofmt` formatting
* `golangci-lint`
* explicit error handling
* context propagation
* documented exported symbols

Use:

```
fmt.Errorf("context: %w", err)
```

for error wrapping.

---

# Testing Practices

Tests should include:

* table-driven tests
* subtests
* race detector
* benchmarks when appropriate

Example execution:

```
go test ./... -race
```

Target high test coverage where practical.

---

# Project Design Expectations

Follow standard Go project structure:

```
cmd/
internal/
pkg/
```

Internal packages should encapsulate implementation details.

For this project specifically:

* `internal/task` → task model and store
* `internal/worker` → worker pool and job processing
* `internal/server` → HTTP API

---

# Configuration

Avoid hardcoded values.

Configuration should come from:

* environment variables
* functional options
* configuration structs

Example:

```
WORKER_COUNT
QUEUE_SIZE
TASK_TIMEOUT
```

---

# Prohibited Practices

Do NOT:

* ignore errors
* use panic for normal control flow
* create unmanaged goroutines
* ignore context cancellation
* mix synchronous and asynchronous patterns incorrectly
* use reflection without clear justification

---

# Output Expectations

When generating code suggestions:

Provide:

1. interface definitions (if relevant)
2. implementation guidance
3. test structure
4. explanation of concurrency patterns used
