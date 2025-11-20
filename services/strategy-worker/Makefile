.PHONY: help install dev-install test lint format type-check clean run docker-build docker-up docker-down migrate proto

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install production dependencies
	pip install -e .

dev-install: ## Install development dependencies
	pip install -e ".[dev]"

test: ## Run tests
	pytest -v

test-cov: ## Run tests with coverage
	pytest --cov=src --cov-report=html --cov-report=term

lint: ## Run linting
	ruff check src/ tests/

format: ## Format code
	black src/ tests/
	ruff check --fix src/ tests/

type-check: ## Run type checking
	mypy src/

clean: ## Clean generated files
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} +
	rm -rf build/ dist/ .pytest_cache/ .mypy_cache/ htmlcov/

proto: ## Generate gRPC stubs from proto files
	./scripts/generate_protos.sh

migrate: ## Run database migrations
	psql -h localhost -U postgres -d aipx -f migrations/001_strategy_worker.sql

run: ## Run the service locally
	python -m src.main

docker-build: ## Build Docker image
	docker-compose build strategy-worker

docker-up: ## Start all services with Docker Compose
	docker-compose up -d

docker-down: ## Stop all services
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f strategy-worker

docker-shell: ## Open shell in container
	docker-compose exec strategy-worker sh

docker-restart: ## Restart the service
	docker-compose restart strategy-worker

# Development targets
dev: dev-install proto ## Setup development environment
	@echo "Development environment ready!"

ci: lint type-check test ## Run CI checks
	@echo "CI checks passed!"

all: clean format lint type-check test ## Run all checks
	@echo "All checks passed!"
