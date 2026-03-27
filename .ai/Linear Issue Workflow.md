---
apply: manually
---

---

## trigger: manual

# Linear Issue Workflow — Go Development

## Purpose

This rule defines how work is performed on issues tracked in Linear while developing Go services.

The goal is to ensure:

* clear progress tracking
* strong Go code quality
* well documented implementation decisions
* reproducible development steps

---

# Starting Work on an Issue

### 1. Move Issue Status

When beginning work:

* Set the Linear issue status to **In Progress**

Immediately add a comment documenting work start.

Example comment:

```
Starting implementation.

Plan:
- Review current architecture
- Implement required changes
- Validate with tests and linting
```

---

# Scope Control

Follow strict scope boundaries.

Rules:

* Work **only on the specified issue**
* Do **not automatically begin sub-issues**
* Do **not expand scope without confirmation**
* If additional work is discovered, suggest creating a **new issue**

---

# Implementation Process

Follow this Go development workflow.

### 1. Analyze

Before coding:

* Review package structure
* Identify affected components
* Verify concurrency implications

Document the plan in a Linear comment.

Example:

```
Implementation plan:

- Update worker pool implementation
- Add retry logic
- Update task status transitions
```

---

### 2. Implement

Write **idiomatic Go code** following these rules:

* Use explicit error handling
* Propagate `context.Context` through blocking operations
* Avoid global state
* Prefer small focused functions
* Use interfaces where appropriate

Concurrency requirements:

* All goroutines must have a clear lifecycle
* Goroutines must respect context cancellation
* Channels must have defined ownership
* Avoid goroutine leaks

---

### 3. Code Quality Validation

Before considering the issue complete, verify:

```
go fmt ./...
go vet ./...
golangci-lint run
go test ./... -race
```

All checks must pass.

If fixes were required, update the Linear issue with a comment summarizing what was corrected.

---

# Testing Requirements

Tests must follow Go best practices:

* table-driven tests
* subtests
* clear test naming
* coverage of success and failure paths

Example:

```
go test ./... -race
```

---

# Documentation Requirements

All exported Go symbols must include documentation comments.

Example:

```go
// StartWorker launches a worker goroutine that processes jobs
// until the provided context is cancelled.
func StartWorker(ctx context.Context, jobs <-chan Job) {
```

When implementing significant logic, also document:

* concurrency design
* failure handling
* retry behaviour
* timeout logic

---

# Updating the Linear Issue

After implementation is finished:

Add a **completion comment** summarizing the work.

Example format:

```
Implementation Summary

Changes:
- Implemented worker pool with configurable worker count
- Added task status transitions (Pending → Running → Completed)
- Integrated context-based shutdown handling

Validation:
- go vet passed
- golangci-lint passed
- tests executed with race detector

Notes:
- Worker lifecycle tied to service context
- No goroutine leaks observed
```

---

# Completion Rules

Important constraints:

* **Do NOT automatically mark the issue as Done**
* Wait for **explicit user verification**
* Do not automatically create commits unless requested

This ensures review and learning validation before finalizing work.

---

# Containerization Requirement

All services must run in a containerized environment.

Expectations:

* Dockerfile present
* Application runs using environment-based configuration
* No hardcoded environment values

Example runtime variables:

```
WORKER_COUNT
QUEUE_SIZE
TASK_TIMEOUT
SERVER_PORT
```

---

# Output Expectations

When completing issue work, always provide:

1. Implementation summary
2. Explanation of key Go patterns used
3. Notes about concurrency behaviour
4. Any follow-up work that may require new issues
