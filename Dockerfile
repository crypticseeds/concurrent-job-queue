# Multi-stage Dockerfile for concurrent-job-queue

# Build Stage
FROM golang:1.26-alpine3.22 AS builder

WORKDIR /app

# Copy dependency files and download
COPY go.mod ./
# go.sum will be generated if not present, but for now only go.mod exists
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/job-queue ./cmd/server/main.go

# Production Stage
FROM alpine:3.22.3

# Install CA certificates for potential external requests
RUN apk --no-cache add ca-certificates

# Create non-root user/group for running the application
RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/job-queue /app/job-queue
RUN chown -R app:app /app

# Expose server port
EXPOSE 8080

# Run the application
USER app
ENTRYPOINT ["/app/job-queue"]
