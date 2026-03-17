# Makefile for concurrent-job-queue

BINARY_NAME=job-queue
CMD_DIR=./cmd/server
DOCKER_IMAGE=crypticseeds/concurrent-job-queue

.PHONY: all build run test clean docker-build docker-run

all: build

build:
	go build -o bin/$(BINARY_NAME) $(CMD_DIR)

run: build
	./bin/$(BINARY_NAME)

test:
	go test -v -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/
	go clean

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE)
