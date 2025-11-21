# Deployment Guide

Complete guide for deploying ML Inference Service to production.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Infrastructure Setup](#infrastructure-setup)
- [Configuration](#configuration)
- [Deployment Steps](#deployment-steps)
- [Scaling](#scaling)
- [Monitoring](#monitoring)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Hardware Requirements

**Minimum**:
- CPU: 8 cores
- RAM: 32GB
- GPU: NVIDIA T4 (16GB VRAM)
- Storage: 200GB SSD

**Recommended**:
- CPU: 16+ cores
- RAM: 64GB
- GPU: NVIDIA A100 (40GB VRAM)
- Storage: 500GB NVMe SSD

### Software Requirements

- Ubuntu 20.04 LTS or later
- Docker 24.0+
- Docker Compose 2.20+
- NVIDIA Driver 525.60.13+
- NVIDIA Container Toolkit
- Kubernetes 1.27+ (for K8s deployment)

## Infrastructure Setup

### 1. Install NVIDIA Drivers

```bash
# Add NVIDIA repository
sudo add-apt-repository ppa:graphics-drivers/ppa
sudo apt update

# Install driver
sudo apt install nvidia-driver-525

# Verify installation
nvidia-smi
```

### 2. Install Docker with GPU Support

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add NVIDIA Container Toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-docker.list

sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit

# Configure Docker
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker

# Test GPU access
docker run --gpus all nvidia/cuda:12.0-base nvidia-smi
```

### 3. Setup Network

```bash
# Create Docker network
docker network create aipx-network

# Configure firewall
sudo ufw allow 8005/tcp  # API
sudo ufw allow 9090/tcp  # Metrics
sudo ufw allow 8001/tcp  # Triton gRPC
```

## Configuration

### Environment Variables

Create production `.env` file:

```bash
# Service
SERVICE_NAME=ml-inference-service
SERVICE_PORT=8005
ENV=production
LOG_LEVEL=INFO

# Triton
TRITON_URL=triton:8001
TRITON_HTTP_URL=triton:8000

# Database
DB_HOST=production-timescaledb.example.com
DB_PORT=5432
DB_NAME=aipx_trading
DB_USER=aipx_prod
DB_PASSWORD=<strong-password>

# Redis
REDIS_HOST=production-redis.example.com
REDIS_PORT=6379
REDIS_PASSWORD=<redis-password>

# Model Repository
MODEL_REPOSITORY=/models
S3_BUCKET=aipx-ml-models-prod
S3_REGION=ap-northeast-2

# Performance
MAX_BATCH_SIZE=32
BATCH_TIMEOUT_MS=100
GPU_MEMORY_FRACTION=0.8
WORKER_PROCESSES=4
```

### Triton Configuration

Optimize Triton for production in `docker-compose.yml`:

```yaml
triton:
  command: >
    tritonserver
    --model-repository=/models
    --strict-model-config=false
    --log-verbose=0
    --exit-timeout-secs=60
    --backend-config=tensorflow,version=2
    --http-thread-count=8
    --grpc-server-thread-count=8
  deploy:
    resources:
      reservations:
        devices:
          - driver: nvidia
            count: 1
            capabilities: [gpu]
      limits:
        memory: 16G
```

## Deployment Steps

### Option 1: Docker Compose (Single Server)

1. **Prepare Environment**:
```bash
cd services/ml-inference-service
cp .env.example .env
# Edit .env with production values
```

2. **Build Images**:
```bash
docker-compose build
```

3. **Start Services**:
```bash
docker-compose up -d
```

4. **Verify Health**:
```bash
curl http://localhost:8005/health
```

### Option 2: Kubernetes (Recommended for Production)

1. **Create Namespace**:
```bash
kubectl create namespace aipx-ml
```

2. **Deploy ConfigMap**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ml-inference-config
  namespace: aipx-ml
data:
  TRITON_URL: "triton-service:8001"
  LOG_LEVEL: "INFO"
```

3. **Deploy Triton Server**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: triton-server
  namespace: aipx-ml
spec:
  replicas: 2
  selector:
    matchLabels:
      app: triton-server
  template:
    metadata:
      labels:
        app: triton-server
    spec:
      containers:
      - name: triton
        image: nvcr.io/nvidia/tritonserver:23.10-py3
        command:
          - tritonserver
          - --model-repository=/models
        resources:
          limits:
            nvidia.com/gpu: 1
            memory: "16Gi"
          requests:
            nvidia.com/gpu: 1
            memory: "8Gi"
        volumeMounts:
        - name: model-repository
          mountPath: /models
      volumes:
      - name: model-repository
        persistentVolumeClaim:
          claimName: model-pvc
```

4. **Deploy FastAPI Gateway**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ml-inference-api
  namespace: aipx-ml
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ml-inference-api
  template:
    metadata:
      labels:
        app: ml-inference-api
    spec:
      containers:
      - name: api
        image: aipx/ml-inference-service:latest
        ports:
        - containerPort: 8005
        env:
        - name: TRITON_URL
          value: "triton-service:8001"
        resources:
          limits:
            memory: "4Gi"
          requests:
            memory: "2Gi"
```

5. **Create Services**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: ml-inference-service
  namespace: aipx-ml
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8005
  selector:
    app: ml-inference-api
```

6. **Deploy**:
```bash
kubectl apply -f k8s/
kubectl -n aipx-ml get pods
```

## Scaling

### Horizontal Scaling

**Docker Compose**:
```bash
docker-compose up -d --scale ml-inference-service=3
```

**Kubernetes HPA**:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ml-inference-hpa
  namespace: aipx-ml
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ml-inference-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: inference_latency_p99
      target:
        type: AverageValue
        averageValue: "100m"
```

### Vertical Scaling

Adjust resource limits based on load:

```yaml
resources:
  limits:
    nvidia.com/gpu: 2  # Multi-GPU
    memory: "32Gi"
  requests:
    nvidia.com/gpu: 2
    memory: "16Gi"
```

## Monitoring

### Prometheus Setup

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'ml-inference'
    static_configs:
      - targets: ['ml-inference-service:9090']

  - job_name: 'triton'
    static_configs:
      - targets: ['triton:8002']
```

### Grafana Dashboards

Import pre-built dashboards:
- Triton Inference Server: Dashboard ID 18447
- Custom ML Inference: `/config/grafana/dashboards/ml-inference.json`

### Alerting Rules

```yaml
groups:
- name: ml_inference_alerts
  interval: 30s
  rules:
  - alert: HighInferenceLatency
    expr: inference_latency_p99 > 1000
    for: 5m
    annotations:
      summary: "High inference latency detected"

  - alert: HighErrorRate
    expr: rate(inference_errors_total[5m]) > 0.05
    for: 5m
    annotations:
      summary: "Error rate above 5%"

  - alert: GPUMemoryHigh
    expr: gpu_memory_used / gpu_memory_total > 0.9
    for: 5m
    annotations:
      summary: "GPU memory usage above 90%"
```

## Security

### 1. API Authentication

Add JWT authentication:

```python
from fastapi import Security, HTTPException
from fastapi.security import HTTPBearer

security = HTTPBearer()

@app.post("/api/v1/predict/price")
async def predict_price(
    request: PredictPriceRequest,
    token: str = Security(security)
):
    # Verify token
    if not verify_token(token.credentials):
        raise HTTPException(status_code=401)
    # ... rest of endpoint
```

### 2. Rate Limiting

```python
from slowapi import Limiter
from slowapi.util import get_remote_address

limiter = Limiter(key_func=get_remote_address)

@app.post("/api/v1/predict/price")
@limiter.limit("100/minute")
async def predict_price(request: Request, ...):
    # ... endpoint logic
```

### 3. Network Security

```bash
# Use private network
docker network create --internal aipx-internal

# TLS for external endpoints
# Add SSL certificates to nginx/traefik
```

### 4. Secrets Management

Use Kubernetes secrets or AWS Secrets Manager:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ml-inference-secrets
type: Opaque
data:
  db-password: <base64-encoded>
  redis-password: <base64-encoded>
```

## Troubleshooting

### Issue: Triton Server Not Starting

**Check logs**:
```bash
docker logs aipx-triton-server
```

**Common causes**:
- Missing model files
- Incorrect config.pbtxt
- GPU not accessible

**Solution**:
```bash
# Verify model repository
ls -la models/lstm_price_predictor/

# Check GPU
nvidia-smi

# Validate config
tritonserver --model-repository=/models --strict-model-config=true
```

### Issue: High Inference Latency

**Check metrics**:
```bash
curl http://localhost:8005/api/v1/metrics
```

**Optimize**:
1. Enable dynamic batching
2. Increase batch size
3. Use TensorRT optimization
4. Add more GPU instances

### Issue: Out of Memory

**Check GPU memory**:
```bash
nvidia-smi
```

**Solutions**:
1. Reduce batch size
2. Lower GPU_MEMORY_FRACTION
3. Enable model sharing
4. Add more GPUs

### Issue: Connection Refused

**Check network**:
```bash
docker network inspect aipx-network
```

**Verify services**:
```bash
docker ps
curl http://localhost:8005/health
```

## Rollback Procedure

```bash
# Docker Compose
docker-compose down
docker-compose up -d --force-recreate

# Kubernetes
kubectl rollout undo deployment/ml-inference-api -n aipx-ml
kubectl rollout status deployment/ml-inference-api -n aipx-ml
```

## Backup and Recovery

### Model Backup

```bash
# Backup to S3
aws s3 sync models/ s3://aipx-ml-models-backup/$(date +%Y%m%d)/

# Restore from S3
aws s3 sync s3://aipx-ml-models-backup/20240115/ models/
```

### Database Backup

```bash
# Metrics database
pg_dump -h $DB_HOST -U $DB_USER aipx_trading > backup.sql
```

## Performance Tuning

### Triton Optimization

```bash
# Use TensorRT
trtexec --onnx=model.onnx --saveEngine=model.plan

# Profile model
tritonserver --model-repository=/models --model-control-mode=explicit
```

### System Tuning

```bash
# Increase file descriptors
ulimit -n 65535

# Optimize network
sudo sysctl -w net.core.rmem_max=16777216
sudo sysctl -w net.core.wmem_max=16777216
```

## Contact

For deployment support:
- Email: devops@aipx.com
- Slack: #ml-inference-support
- On-call: +82-xxx-xxxx
