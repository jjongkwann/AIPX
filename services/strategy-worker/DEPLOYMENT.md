# Strategy Worker - Deployment Guide

## Pre-Deployment Checklist

### 1. Code Preparation

- [ ] Generate gRPC stubs from proto files
  ```bash
  cd /Users/jk/workspace/AIPX/services/strategy-worker
  ./scripts/generate_protos.sh
  ```

- [ ] Uncomment gRPC implementation in `src/grpc_client/oms_client.py`
  - Import statements (lines ~7-8)
  - `submit_order` method implementation
  - `stream_orders` method implementation

- [ ] Run all tests
  ```bash
  make test
  pytest --cov=src --cov-report=term
  ```

- [ ] Code quality checks
  ```bash
  make format    # Format code
  make lint      # Check linting
  make type-check # Type checking
  ```

### 2. Infrastructure Setup

#### Database

- [ ] Create PostgreSQL database
  ```bash
  createdb -U postgres aipx
  ```

- [ ] Run migrations
  ```bash
  make migrate
  # OR
  psql -U postgres -d aipx -f migrations/001_strategy_worker.sql
  ```

- [ ] Verify schema
  ```bash
  psql -U postgres -d aipx -c "\dt"
  ```

- [ ] Create database user (production)
  ```sql
  CREATE USER strategy_worker WITH PASSWORD 'secure_password';
  GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO strategy_worker;
  GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO strategy_worker;
  ```

#### Kafka

- [ ] Create topics
  ```bash
  kafka-topics --bootstrap-server localhost:9092 --create \
    --topic strategy.approved --partitions 3 --replication-factor 3

  kafka-topics --bootstrap-server localhost:9092 --create \
    --topic strategy.execution --partitions 3 --replication-factor 3

  kafka-topics --bootstrap-server localhost:9092 --create \
    --topic strategy.positions --partitions 3 --replication-factor 3

  kafka-topics --bootstrap-server localhost:9092 --create \
    --topic strategy.errors --partitions 3 --replication-factor 3
  ```

- [ ] Verify topics
  ```bash
  kafka-topics --bootstrap-server localhost:9092 --list
  ```

- [ ] Configure retention
  ```bash
  kafka-configs --bootstrap-server localhost:9092 --alter \
    --entity-type topics --entity-name strategy.approved \
    --add-config retention.ms=604800000  # 7 days
  ```

#### Redis

- [ ] Install and start Redis
  ```bash
  # Docker
  docker run -d --name redis -p 6379:6379 redis:7-alpine

  # Or native
  redis-server --daemonize yes
  ```

- [ ] Set password (production)
  ```bash
  redis-cli CONFIG SET requirepass "secure_password"
  ```

- [ ] Test connection
  ```bash
  redis-cli ping
  ```

### 3. Configuration

- [ ] Copy and customize configuration
  ```bash
  cp .env.example .env
  # Edit .env with production values
  ```

- [ ] Update `config/config.yaml` for environment
  - Set correct hostnames
  - Configure production credentials
  - Adjust pool sizes
  - Set appropriate risk limits

- [ ] Verify configuration loads
  ```bash
  python -c "from src.config import get_config; print(get_config().service.environment)"
  ```

### 4. Security

- [ ] Rotate all passwords and secrets
  - Database password
  - Redis password
  - Kafka credentials (if SASL enabled)

- [ ] Enable TLS/SSL
  - [ ] Database connection SSL
  - [ ] Kafka SSL/SASL
  - [ ] gRPC TLS

- [ ] Setup secret management
  - [ ] Use HashiCorp Vault or AWS Secrets Manager
  - [ ] Remove hardcoded credentials
  - [ ] Inject secrets at runtime

- [ ] Review RBAC permissions
  - [ ] Database user permissions
  - [ ] Kafka ACLs
  - [ ] Container security context

### 5. Docker Build

#### Development

```bash
# Build image
docker-compose build strategy-worker

# Start all services
docker-compose up -d

# Verify
docker-compose ps
docker-compose logs -f strategy-worker
```

#### Production

```bash
# Build production image
docker build -t strategy-worker:latest .

# Tag for registry
docker tag strategy-worker:latest registry.example.com/strategy-worker:v0.1.0

# Push to registry
docker push registry.example.com/strategy-worker:v0.1.0
```

### 6. Integration Testing

- [ ] Test Kafka consumer
  ```bash
  # Produce test message
  python tests/integration/produce_test_strategy.py
  ```

- [ ] Test OMS gRPC connection
  ```bash
  # Requires OMS service running
  grpcurl -plaintext localhost:50051 list
  ```

- [ ] Test database operations
  ```bash
  # Run integration tests
  pytest tests/integration/ -v -m integration
  ```

- [ ] End-to-end test
  - [ ] Produce strategy to Kafka
  - [ ] Verify execution created in database
  - [ ] Check orders submitted to OMS
  - [ ] Validate position tracking
  - [ ] Test graceful shutdown

### 7. Monitoring Setup

#### Prometheus

- [ ] Configure Prometheus scraping
  ```yaml
  # prometheus.yml
  scrape_configs:
    - job_name: 'strategy-worker'
      static_configs:
        - targets: ['strategy-worker:9091']
  ```

- [ ] Verify metrics endpoint
  ```bash
  curl http://localhost:9091/metrics
  ```

#### Grafana

- [ ] Import dashboard
  - Create dashboard for strategy metrics
  - Add panels for:
    - Active executions
    - Orders per second
    - Risk violations
    - P&L metrics
    - Error rates

#### Alerting

- [ ] Configure alerts
  - High error rate
  - Risk violations
  - Large losses
  - Service down
  - Database connection issues

### 8. Logging

- [ ] Configure log aggregation
  - [ ] ELK Stack / Splunk / Datadog
  - [ ] Structured JSON logs
  - [ ] Log retention policy

- [ ] Setup log levels
  - Production: INFO
  - Debug: DEBUG
  - Test: WARNING

- [ ] Verify log output
  ```bash
  docker-compose logs -f strategy-worker | grep -i error
  ```

## Deployment Steps

### Local Development

```bash
# 1. Install dependencies
make dev-install

# 2. Generate protos
make proto

# 3. Start infrastructure
docker-compose up -d postgres kafka redis

# 4. Run migrations
make migrate

# 5. Run service
make run
```

### Docker Deployment

```bash
# 1. Build and start
docker-compose up -d

# 2. Check health
docker-compose ps
curl http://localhost:8080/health

# 3. View logs
docker-compose logs -f strategy-worker

# 4. Monitor
open http://localhost:9091/metrics
```

### Kubernetes Deployment

#### 1. Create ConfigMap

```bash
kubectl create configmap strategy-worker-config \
  --from-file=config/config.yaml
```

#### 2. Create Secrets

```bash
kubectl create secret generic strategy-worker-secrets \
  --from-literal=db-password='secure_password' \
  --from-literal=redis-password='secure_password'
```

#### 3. Deploy

```bash
# Apply manifests
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# Verify
kubectl get pods -l app=strategy-worker
kubectl logs -f deployment/strategy-worker
```

#### 4. Example Deployment YAML

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: strategy-worker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: strategy-worker
  template:
    metadata:
      labels:
        app: strategy-worker
    spec:
      containers:
      - name: strategy-worker
        image: registry.example.com/strategy-worker:v0.1.0
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: strategy-worker-secrets
              key: db-password
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: strategy-worker-secrets
              key: redis-password
        - name: CONFIG_PATH
          value: /app/config/config.yaml
        volumeMounts:
        - name: config
          mountPath: /app/config
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: strategy-worker-config
```

## Post-Deployment

### 1. Smoke Tests

- [ ] Service starts successfully
  ```bash
  kubectl logs -f deployment/strategy-worker
  ```

- [ ] Kafka consumer connected
  ```bash
  kafka-consumer-groups --bootstrap-server localhost:9092 \
    --group strategy-worker --describe
  ```

- [ ] Database connection healthy
  ```bash
  psql -U postgres -d aipx -c "SELECT COUNT(*) FROM strategy_executions;"
  ```

- [ ] OMS gRPC connection established
  ```bash
  # Check logs for "OMS gRPC client connected"
  ```

### 2. Load Testing

```bash
# Generate test load
python tests/load/generate_strategies.py --count 100 --rate 10

# Monitor metrics
watch -n 1 'curl -s http://localhost:9091/metrics | grep strategy'
```

### 3. Monitoring Verification

- [ ] Metrics being collected
- [ ] Dashboards showing data
- [ ] Alerts configured
- [ ] On-call rotation setup

### 4. Documentation

- [ ] Update runbook
- [ ] Document procedures
- [ ] Train team members
- [ ] Create incident response plan

## Rollback Plan

### Quick Rollback

```bash
# Kubernetes
kubectl rollout undo deployment/strategy-worker

# Docker Compose
docker-compose down
docker-compose up -d --force-recreate
```

### Database Rollback

```bash
# Run rollback migration (create if needed)
psql -U postgres -d aipx -f migrations/001_strategy_worker_rollback.sql
```

### Kafka Consumer Reset

```bash
# Reset consumer group offset if needed
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group strategy-worker --reset-offsets --to-earliest --execute \
  --topic strategy.approved
```

## Production Checklist

Before going live:

- [ ] All tests passing (make ci)
- [ ] gRPC stubs generated
- [ ] Database migrations applied
- [ ] Kafka topics created
- [ ] Configuration reviewed
- [ ] Secrets rotated
- [ ] TLS enabled
- [ ] Monitoring configured
- [ ] Alerts setup
- [ ] Logs aggregated
- [ ] Backup strategy defined
- [ ] Disaster recovery plan
- [ ] Team trained
- [ ] Documentation complete
- [ ] Performance tested
- [ ] Security audited

## Maintenance

### Regular Tasks

**Daily:**
- Monitor error logs
- Check resource usage
- Review risk events
- Validate P&L accuracy

**Weekly:**
- Review performance metrics
- Check database size
- Analyze slow queries
- Update dependencies

**Monthly:**
- Security updates
- Configuration review
- Capacity planning
- Disaster recovery drill

### Troubleshooting

See [README.md](./README.md#troubleshooting) for common issues.

## Support

- **Documentation**: [README.md](./README.md)
- **Quick Start**: [QUICKSTART.md](./QUICKSTART.md)
- **Implementation**: [IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)
- **Issues**: GitHub Issues
- **On-call**: PagerDuty rotation

---

**Last Updated**: 2025-11-20
**Version**: 0.1.0
