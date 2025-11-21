# AIPX Deployment Guide

Complete guide for deploying the AIPX trading platform to production.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development](#local-development)
3. [Kubernetes Deployment](#kubernetes-deployment)
4. [CI/CD Setup](#cicd-setup)
5. [Monitoring Setup](#monitoring-setup)
6. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Tools

- Docker 24.0+
- Docker Compose 2.20+
- Kubernetes 1.28+
- kubectl 1.28+
- Helm 3.12+
- ArgoCD 2.9+

### Cloud Resources

- Kubernetes cluster (EKS, GKE, or AKS)
- Container registry (GitHub Container Registry, ECR, or GCR)
- DNS management
- SSL certificates (via cert-manager)

### Secrets Required

```bash
# API Keys
ANTHROPIC_API_KEY=sk-ant-...

# Database
POSTGRES_PASSWORD=<strong-password>

# JWT
JWT_SECRET=<random-256-bit-key>

# Redis
REDIS_PASSWORD=<strong-password>
```

---

## Local Development

### 1. Quick Start with Docker Compose

```bash
# Clone repository
git clone https://github.com/your-org/aipx.git
cd aipx

# Copy environment file
cp .env.example .env

# Edit .env with your API keys
vim .env

# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f
```

### 2. Access Services

- Dashboard: http://localhost:3000
- User API: http://localhost:8082
- Cognitive API: http://localhost:8001
- Backtesting API: http://localhost:8002
- Grafana: http://localhost:3001 (admin/admin)
- Prometheus: http://localhost:9099

### 3. Run Tests

```bash
# Go services
cd services/order-management-service
go test ./...

# Python services
cd services/cognitive-service
pytest tests/

# Frontend
cd services/dashboard-service
npm test
```

---

## Kubernetes Deployment

### 1. Cluster Setup

#### Create Kubernetes Cluster

**AWS EKS:**
```bash
eksctl create cluster \
  --name aipx-production \
  --region us-east-1 \
  --nodegroup-name standard-workers \
  --node-type t3.xlarge \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 10 \
  --managed
```

**GKE:**
```bash
gcloud container clusters create aipx-production \
  --zone us-central1-a \
  --num-nodes 3 \
  --machine-type n1-standard-4 \
  --enable-autoscaling \
  --min-nodes 3 \
  --max-nodes 10
```

#### Configure kubectl

```bash
# AWS
aws eks update-kubeconfig --name aipx-production --region us-east-1

# GCP
gcloud container clusters get-credentials aipx-production --zone us-central1-a

# Verify connection
kubectl cluster-info
kubectl get nodes
```

### 2. Install Prerequisites

#### Install cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

#### Install NGINX Ingress

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer
```

#### Get LoadBalancer IP

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller

# Note the EXTERNAL-IP and configure DNS
# api.aipx.io -> EXTERNAL-IP
# dashboard.aipx.io -> EXTERNAL-IP
```

### 3. Create Secrets

```bash
# Create namespace
kubectl apply -f k8s/base/namespace.yaml

# Create secrets
kubectl create secret generic postgres-credentials -n aipx \
  --from-literal=username=postgres \
  --from-literal=password=<STRONG_PASSWORD> \
  --from-literal=dsn=postgresql://postgres:<PASSWORD>@postgresql:5432/aipx?sslmode=require

kubectl create secret generic anthropic-api-key -n aipx \
  --from-literal=api_key=<YOUR_API_KEY>

kubectl create secret generic jwt-secret -n aipx \
  --from-literal=secret=<RANDOM_256_BIT_KEY>

kubectl create secret generic redis-password -n aipx \
  --from-literal=password=<STRONG_PASSWORD>
```

### 4. Deploy Infrastructure

```bash
# Apply base configuration
kubectl apply -k k8s/base/

# Deploy infrastructure (Kafka, Redis, PostgreSQL)
kubectl apply -k k8s/infrastructure/kafka/
kubectl apply -k k8s/infrastructure/redis/
kubectl apply -k k8s/infrastructure/postgresql/

# Wait for infrastructure to be ready
kubectl rollout status statefulset/kafka -n aipx --timeout=10m
kubectl rollout status statefulset/redis -n aipx --timeout=5m
kubectl rollout status statefulset/postgresql -n aipx --timeout=5m
```

### 5. Deploy Services

```bash
# Deploy all services
kubectl apply -k k8s/services/oms/
kubectl apply -k k8s/services/user-service/
kubectl apply -k k8s/services/cognitive/
kubectl apply -k k8s/services/data-ingestion/
kubectl apply -k k8s/services/data-recorder/
kubectl apply -k k8s/services/notification/
kubectl apply -k k8s/services/strategy-worker/
kubectl apply -k k8s/services/backtesting/
kubectl apply -k k8s/services/ml-inference/
kubectl apply -k k8s/services/dashboard/

# Check deployment status
kubectl get pods -n aipx
kubectl get svc -n aipx

# Wait for all deployments
kubectl rollout status deployment -n aipx --timeout=15m
```

### 6. Verify Deployment

```bash
# Check pod health
kubectl get pods -n aipx

# Check service endpoints
kubectl get endpoints -n aipx

# Test internal connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://user-service.aipx.svc.cluster.local:8082/health

# Check logs
kubectl logs -f deployment/oms -n aipx
```

---

## CI/CD Setup

### 1. GitHub Actions Setup

#### Configure Secrets

Go to GitHub Settings > Secrets and add:

```
ANTHROPIC_API_KEY=<your-key>
KUBE_CONFIG=<base64-encoded-kubeconfig>
PAT_TOKEN=<github-personal-access-token>
SLACK_WEBHOOK=<slack-webhook-url>
```

#### Enable Workflows

```bash
# Push code to trigger workflows
git add .
git commit -m "feat: Initial deployment setup"
git push origin main

# Workflows will automatically:
# 1. Run tests on PR
# 2. Build and push images on merge to main
# 3. Deploy to staging
```

### 2. ArgoCD Setup

#### Install ArgoCD

```bash
kubectl create namespace argocd

kubectl apply -n argocd -f \
  https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Get admin password
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d

# Port forward to access UI
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Access at https://localhost:8080
# Username: admin
# Password: <from above command>
```

#### Create ArgoCD Project and Applications

```bash
# Apply ArgoCD project
kubectl apply -f k8s/argocd/projects/aipx.yaml

# Apply applications
kubectl apply -f k8s/argocd/applications/
```

#### Configure Auto-Sync

ArgoCD will now automatically:
- Monitor Git repository for changes
- Sync Kubernetes manifests
- Report sync status
- Auto-heal deployments

---

## Monitoring Setup

### 1. Install Prometheus Stack

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace \
  -f k8s/monitoring/prometheus/values.yaml
```

### 2. Install Loki

```bash
helm repo add grafana https://grafana.github.io/helm-charts

helm install loki grafana/loki \
  -n monitoring \
  -f k8s/monitoring/loki/values.yaml

helm install promtail grafana/promtail \
  -n monitoring \
  -f k8s/monitoring/loki/values.yaml
```

### 3. Deploy Jaeger

```bash
kubectl apply -f k8s/monitoring/jaeger/deployment.yaml
```

### 4. Access Monitoring Tools

```bash
# Grafana
kubectl port-forward -n monitoring svc/kube-prometheus-stack-grafana 3000:80

# Prometheus
kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090

# Jaeger
kubectl port-forward -n aipx svc/jaeger-query 16686:16686
```

**URLs:**
- Grafana: http://localhost:3000 (admin/<password-from-values>)
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686

### 5. Import Dashboards

In Grafana:
1. Go to Dashboards > Import
2. Upload the dashboard JSON files from `k8s/monitoring/grafana/dashboards/`
3. Select Prometheus as data source

---

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n aipx

# Check events
kubectl get events -n aipx --sort-by='.lastTimestamp' | tail -20

# Check logs
kubectl logs <pod-name> -n aipx --previous
```

### Service Not Accessible

```bash
# Check service
kubectl get svc <service-name> -n aipx

# Check endpoints
kubectl get endpoints <service-name> -n aipx

# Test from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://<service-name>.aipx.svc.cluster.local:<port>/health
```

### Database Connection Issues

```bash
# Check PostgreSQL pod
kubectl logs statefulset/postgresql -n aipx

# Connect to PostgreSQL
kubectl exec -it postgresql-0 -n aipx -- psql -U postgres -d aipx

# Check connections
SELECT * FROM pg_stat_activity;
```

### Kafka Issues

```bash
# Check Kafka pods
kubectl logs kafka-0 -n aipx

# List topics
kubectl exec -it kafka-0 -n aipx -- kafka-topics --list --bootstrap-server localhost:9092

# Check consumer lag
kubectl exec -it kafka-0 -n aipx -- kafka-consumer-groups --describe --group aipx-consumer-group --bootstrap-server localhost:9092
```

### High Memory/CPU Usage

```bash
# Check resource usage
kubectl top pods -n aipx
kubectl top nodes

# Scale down if needed
kubectl scale deployment/<deployment-name> -n aipx --replicas=1

# Check HPA status
kubectl get hpa -n aipx
```

### Rollback Deployment

```bash
# View rollout history
kubectl rollout history deployment/<deployment-name> -n aipx

# Rollback to previous version
kubectl rollout undo deployment/<deployment-name> -n aipx

# Rollback to specific revision
kubectl rollout undo deployment/<deployment-name> -n aipx --to-revision=2
```

---

## Production Checklist

Before going to production, verify:

- [ ] All secrets configured securely
- [ ] Database backups configured
- [ ] Monitoring and alerting set up
- [ ] SSL certificates installed
- [ ] DNS records configured
- [ ] Resource limits set appropriately
- [ ] HPA configured and tested
- [ ] PDB (Pod Disruption Budget) in place
- [ ] Network policies applied
- [ ] Backup and restore tested
- [ ] Disaster recovery plan documented
- [ ] On-call rotation configured
- [ ] Runbooks created
- [ ] Load testing performed
- [ ] Security scan completed
- [ ] Compliance requirements met

---

## Next Steps

- Read [OPERATIONS.md](./OPERATIONS.md) for day-to-day operations
- Read [DISASTER_RECOVERY.md](./DISASTER_RECOVERY.md) for backup/restore procedures
- Set up monitoring alerts in Prometheus
- Configure log aggregation queries in Loki
- Create custom Grafana dashboards for your metrics
- Set up automated backups
- Perform load testing
- Configure autoscaling thresholds

---

## Support

For issues and questions:
- GitHub Issues: https://github.com/your-org/aipx/issues
- Slack: #aipx-support
- Email: devops@aipx.io
