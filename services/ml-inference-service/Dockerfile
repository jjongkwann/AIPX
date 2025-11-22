# Multi-stage build for ML Inference Service
FROM python:3.11-slim as base

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY services/ml-inference-service/requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY services/ml-inference-service/src/ ./src/
COPY services/ml-inference-service/training/ ./training/

# Create directories
RUN mkdir -p /models /data/training /models/checkpoints

# Expose port
EXPOSE 8005

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8005/health || exit 1

# Run FastAPI service
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8005"]
