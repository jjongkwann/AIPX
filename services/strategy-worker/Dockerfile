# Strategy Worker Dockerfile
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy dependency files
COPY pyproject.toml ./

# Install Python dependencies
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -e .

# Copy application code
COPY src/ ./src/
COPY config/ ./config/
COPY migrations/ ./migrations/
COPY scripts/ ./scripts/

# Generate gRPC stubs (will be run during build)
# Uncomment after proto files are available
# RUN chmod +x ./scripts/generate_protos.sh && ./scripts/generate_protos.sh

# Create non-root user
RUN useradd -m -u 1000 worker && \
    chown -R worker:worker /app

USER worker

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV CONFIG_PATH=/app/config/config.yaml

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD python -c "import asyncio; print('healthy')" || exit 1

# Run the application
CMD ["python", "-m", "src.main"]
