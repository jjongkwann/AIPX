.PHONY: help proto clean docker-up docker-down test lint format install

# Default target
help:
	@echo "AIPX - Makefile Commands"
	@echo "========================"
	@echo ""
	@echo "Development:"
	@echo "  make proto        - Compile Protocol Buffers"
	@echo "  make docker-up    - Start local development environment"
	@echo "  make docker-down  - Stop local development environment"
	@echo "  make install      - Install dependencies"
	@echo ""
	@echo "Code Quality:"
	@echo "  make test         - Run all tests"
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
	@rm -rf shared/go/gen/*
	@rm -rf shared/python/gen/*
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
	@echo "Installing Python dependencies..."
	@pip install -r shared/python/requirements.txt
	@echo ""
	@echo "✅ Dependencies installed"

# Run tests
test:
	@echo "🧪 Running tests..."
	@echo ""
	@echo "Running Go tests..."
	@cd shared/go && go test ./... -v
	@echo ""
	@echo "Running Python tests..."
	@cd shared/python && python -m pytest tests/ -v
	@echo ""
	@echo "✅ All tests passed"

# Run linters
lint:
	@echo "🔍 Running linters..."
	@echo ""
	@echo "Linting Go code..."
	@cd shared/go && golangci-lint run ./...
	@echo ""
	@echo "Linting Python code..."
	@cd shared/python && pylint --rcfile=.pylintrc aipx/
	@cd shared/python && black --check aipx/
	@echo ""
	@echo "✅ Linting complete"

# Format code
format:
	@echo "💅 Formatting code..."
	@echo ""
	@echo "Formatting Go code..."
	@cd shared/go && go fmt ./...
	@echo ""
	@echo "Formatting Python code..."
	@cd shared/python && black aipx/
	@cd shared/python && isort aipx/
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
