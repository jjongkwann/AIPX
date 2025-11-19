.PHONY: help build run test clean docker-build docker-run deps lint migrate

# Variables
SERVICE_NAME := data-recorder-service
DOCKER_IMAGE := $(SERVICE_NAME):latest
GO_FILES := $(shell find . -type f -name '*.go')

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the binary
	CGO_ENABLED=0 go build -o bin/$(SERVICE_NAME) cmd/server/main.go

run: ## Run the service locally
	go run cmd/server/main.go

test: ## Run tests
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

lint: ## Run linter
	golangci-lint run

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.txt coverage.html

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container
	docker run -d \
		--name $(SERVICE_NAME) \
		--env-file .env \
		-p 8080:8080 \
		$(DOCKER_IMAGE)

docker-stop: ## Stop Docker container
	docker stop $(SERVICE_NAME) || true
	docker rm $(SERVICE_NAME) || true

migrate: ## Run database migrations
	@echo "Running migrations..."
	psql -h $(TIMESCALEDB_HOST) -p $(TIMESCALEDB_PORT) -U $(TIMESCALEDB_USER) -d $(TIMESCALEDB_DB) -f migrations/001_create_tables.sql

fmt: ## Format code
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	go vet ./...

dev: ## Run in development mode with hot reload
	air

.DEFAULT_GOAL := help
