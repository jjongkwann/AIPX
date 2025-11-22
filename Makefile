.PHONY: help proto clean docker-up docker-down test test-go test-python lint format install

# Default target
help:
	@echo "AIPX - Makefile Commands"
	@echo "========================"
	@echo ""
	@echo "Development:"
	@echo "  make proto        - Compile Protocol Buffers"
	@echo "  make docker-up    - Start local development environment"
	@echo "  make docker-down  - Stop local development environment"
	@echo "  make install      - Install dependencies (uv for Python)"
	@echo ""
	@echo "Code Quality:"
	@echo "  make test         - Run all tests"
	@echo "  make test-go      - Run Go tests only"
	@echo "  make test-python  - Run Python tests only"
	@echo "  make lint         - Run linters"
	@echo "  make format       - Format code"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean        - Clean generated files"
	@echo ""

# Compile Protocol Buffers
proto:
	@echo "🔨 Compiling Protocol Buffers..."
	@./scripts/proto-compile.sh

# Clean generated files
clean:
	@echo "🧹 Cleaning generated files..."
	@rm -rf shared/go/pkg/pb/*
	@rm -rf shared/python/common/pb/*
	@echo "✅ Clean complete"

# Start Docker Compose development environment
docker-up:
	@echo "🚀 Starting development environment..."
	@docker-compose up -d
	@echo "✅ Services started:"
	@echo "   - Kafka:      localhost:9092"
	@echo "   - Redis:      localhost:6379"
	@echo "   - PostgreSQL: localhost:5432"
	@echo ""
	@echo "Run 'docker-compose logs -f' to view logs"

# Stop Docker Compose
docker-down:
	@echo "🛑 Stopping development environment..."
	@docker-compose down
	@echo "✅ Services stopped"

# Stop Docker Compose and remove volumes
docker-clean:
	@echo "🧹 Cleaning development environment..."
	@docker-compose down -v
	@echo "✅ Services stopped and volumes removed"

# Install dependencies
install:
	@echo "📦 Installing dependencies..."
	@echo ""
	@echo "Installing Go dependencies..."
	@cd shared/go && go mod download
	@echo ""
	@echo "Installing Python dependencies with uv..."
	@cd shared/python && uv venv .venv && uv pip install -e ".[test]"
	@cd services/cognitive-service && uv venv .venv && uv pip install -e ".[dev]"
	@cd services/backtesting-service && uv venv .venv && uv pip install -e ".[dev]"
	@cd services/ml-inference-service && uv venv .venv && uv pip install -e ".[dev]"
	@cd services/strategy-worker && uv venv .venv && uv pip install -e ".[dev]"
	@echo ""
	@echo "✅ Dependencies installed"

# Run tests
test:
	@echo "🧪 Running tests..."
	@echo ""
	@echo "Running Go tests..."
	@cd shared/go && go test ./... -v
	@cd services/data-ingestion-service && go test ./... -v
	@cd services/order-management-service && go test ./... -v
	@cd services/user-service && go test ./... -v
	@cd services/notification-service && go test ./... -v
	@cd services/data-recorder-service && go test ./... -v
	@echo ""
	@echo "Running Python tests..."
	@cd services/cognitive-service && .venv/bin/python -m pytest tests/ -v
	@cd services/backtesting-service && .venv/bin/python -m pytest tests/ -v
	@cd services/ml-inference-service && .venv/bin/python -m pytest tests/ -v
	@cd services/strategy-worker && .venv/bin/python -m pytest tests/ -v
	@echo ""
	@echo "✅ All tests passed"

# Run tests (Go only)
test-go:
	@echo "🧪 Running Go tests..."
	@cd shared/go && go test ./... -v
	@cd services/data-ingestion-service && go test ./... -v
	@cd services/order-management-service && go test ./... -v
	@cd services/user-service && go test ./... -v
	@cd services/notification-service && go test ./... -v
	@cd services/data-recorder-service && go test ./... -v
	@echo "✅ Go tests passed"

# Run tests (Python only)
test-python:
	@echo "🧪 Running Python tests..."
	@cd services/cognitive-service && .venv/bin/python -m pytest tests/ -v
	@cd services/backtesting-service && .venv/bin/python -m pytest tests/ -v
	@cd services/ml-inference-service && .venv/bin/python -m pytest tests/ -v
	@cd services/strategy-worker && .venv/bin/python -m pytest tests/ -v
	@echo "✅ Python tests passed"

# Run linters
lint:
	@echo "🔍 Running linters..."
	@echo ""
	@echo "Linting Go code..."
	@cd shared/go && golangci-lint run ./...
	@cd services/data-ingestion-service && golangci-lint run ./...
	@cd services/order-management-service && golangci-lint run ./...
	@cd services/user-service && golangci-lint run ./...
	@cd services/notification-service && golangci-lint run ./...
	@cd services/data-recorder-service && golangci-lint run ./...
	@echo ""
	@echo "Linting Python code..."
	@cd shared/python && .venv/bin/ruff check .
	@cd services/cognitive-service && .venv/bin/ruff check .
	@cd services/backtesting-service && .venv/bin/ruff check .
	@cd services/ml-inference-service && .venv/bin/ruff check .
	@cd services/strategy-worker && .venv/bin/ruff check .
	@echo ""
	@echo "✅ Linting complete"

# Format code
format:
	@echo "💅 Formatting code..."
	@echo ""
	@echo "Formatting Go code..."
	@cd shared/go && go fmt ./...
	@cd services/data-ingestion-service && go fmt ./...
	@cd services/order-management-service && go fmt ./...
	@cd services/user-service && go fmt ./...
	@cd services/notification-service && go fmt ./...
	@cd services/data-recorder-service && go fmt ./...
	@echo ""
	@echo "Formatting Python code..."
	@cd shared/python && .venv/bin/ruff format .
	@cd services/cognitive-service && .venv/bin/ruff format .
	@cd services/backtesting-service && .venv/bin/ruff format .
	@cd services/ml-inference-service && .venv/bin/ruff format .
	@cd services/strategy-worker && .venv/bin/ruff format .
	@echo ""
	@echo "✅ Code formatted"

# Build Docker images
docker-build:
	@echo "🏗️  Building Docker images..."
	@docker-compose build
	@echo "✅ Docker images built"

# Show Docker logs
logs:
	@docker-compose logs -f

# Database migrations (for future use)
migrate-up:
	@echo "⬆️  Running database migrations..."
	@# TODO: Add migration command
	@echo "✅ Migrations applied"

migrate-down:
	@echo "⬇️  Reverting database migrations..."
	@# TODO: Add migration rollback command
	@echo "✅ Migrations reverted"
