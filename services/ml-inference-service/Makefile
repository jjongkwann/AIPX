.PHONY: help install test lint format docker-build docker-up docker-down train clean

help:
	@echo "ML Inference Service - Available Commands"
	@echo "=========================================="
	@echo "install       - Install dependencies"
	@echo "test          - Run tests"
	@echo "test-cov      - Run tests with coverage"
	@echo "lint          - Run linting"
	@echo "format        - Format code"
	@echo "docker-build  - Build Docker images"
	@echo "docker-up     - Start services"
	@echo "docker-down   - Stop services"
	@echo "train         - Train LSTM model"
	@echo "clean         - Clean generated files"

install:
	pip install -r requirements.txt

test:
	pytest tests/ -v

test-cov:
	pytest tests/ --cov=src --cov-report=html --cov-report=term

lint:
	flake8 src/ tests/ --max-line-length=120
	pylint src/ tests/

format:
	black src/ tests/ training/
	isort src/ tests/ training/

docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f ml-inference-service

train:
	python training/train_lstm.py

clean:
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} +
	rm -rf .pytest_cache
	rm -rf htmlcov
	rm -rf .coverage

health:
	curl http://localhost:8005/health

predict-price:
	curl -X POST http://localhost:8005/api/v1/predict/price \
		-H "Content-Type: application/json" \
		-d '{"symbol":"005930","ohlcv_data":[[71000,71500,70500,71200,1000000]]}'

metrics:
	curl http://localhost:8005/api/v1/metrics
