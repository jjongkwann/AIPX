# AIPX Kubernetes Manifests

Production-ready Kubernetes configurations for the AIPX trading platform.

## Directory Structure

```
k8s/
├── base/                    # Base resources for all environments
│   ├── namespace.yaml       # Namespace definitions
│   ├── configmap.yaml       # Common configuration
│   ├── secrets.yaml         # Secret templates (DO NOT commit actual secrets)
│   ├── serviceaccount.yaml  # RBAC configuration
│   ├── networkpolicy.yaml   # Network security policies
│   ├── resourcequota.yaml   # Resource quotas
│   ├── limitrange.yaml      # Default resource limits
│   ├── ingress.yaml         # Ingress configuration
│   └── kustomization.yaml   # Kustomize base
│
├── services/                # Application services
│   ├── oms/                 # Order Management Service
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── hpa.yaml
│   │   ├── pdb.yaml
│   │   ├── servicemonitor.yaml
│   │   └── kustomization.yaml
│   ├── user-service/        # User Service
│   ├── cognitive/           # Cognitive Service
│   ├── data-ingestion/      # Data Ingestion Service
│   ├── data-recorder/       # Data Recorder Service
│   ├── notification/        # Notification Service
│   ├── strategy-worker/     # Strategy Worker
│   ├── backtesting/         # Backtesting Service
│   ├── ml-inference/        # ML Inference Service
│   └── dashboard/           # Dashboard (Frontend)
│
├── infrastructure/          # Infrastructure components
│   ├── kafka/               # Kafka StatefulSet (3 replicas)
│   ├── redis/               # Redis StatefulSet (3 replicas)
│   └── postgresql/          # PostgreSQL StatefulSet (TimescaleDB)
│
├── monitoring/              # Monitoring stack
│   ├── prometheus/
│   │   ├── prometheus.yml   # Prometheus configuration
│   │   ├── alerts.yml       # Alert rules
│   │   └── values.yaml      # Helm values for kube-prometheus-stack
│   ├── grafana/
│   │   └── dashboards/      # Custom Grafana dashboards
│   ├── loki/
│   │   └── values.yaml      # Loki + Promtail configuration
│   └── jaeger/
│       └── deployment.yaml  # Jaeger all-in-one
│
├── argocd/                  # GitOps with ArgoCD
│   ├── projects/
│   │   └── aipx.yaml        # ArgoCD AppProject
│   └── applications/        # ArgoCD Applications
│       ├── aipx-infrastructure.yaml
│       ├── oms.yaml
│       ├── user-service.yaml
│       └── monitoring.yaml
│
└── overlays/                # Environment-specific overlays
    ├── dev/                 # Development environment
    ├── staging/             # Staging environment
    └── production/          # Production environment
```

## Quick Start

### Prerequisites

```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# Install kustomize
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash

# Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### Deploy to Development

```bash
# 1. Create namespace
kubectl apply -f base/namespace.yaml

# 2. Create secrets (replace with actual values)
kubectl create secret generic postgres-credentials -n aipx-dev \
  --from-literal=username=postgres \
  --from-literal=password=<PASSWORD> \
  --from-literal=dsn=postgresql://postgres:<PASSWORD>@postgresql:5432/aipx

kubectl create secret generic anthropic-api-key -n aipx-dev \
  --from-literal=api_key=<API_KEY>

# 3. Deploy infrastructure
kubectl apply -k infrastructure/postgresql/
kubectl apply -k infrastructure/redis/
kubectl apply -k infrastructure/kafka/

# 4. Wait for infrastructure
kubectl wait --for=condition=ready pod -l app=postgresql -n aipx-dev --timeout=5m

# 5. Deploy services
kubectl apply -k services/oms/
kubectl apply -k services/user-service/
# ... deploy other services

# 6. Check status
kubectl get pods -n aipx-dev
```

### Deploy with Kustomize

```bash
# Deploy specific environment
kubectl apply -k overlays/dev/
kubectl apply -k overlays/staging/
kubectl apply -k overlays/production/
```

### Deploy with ArgoCD

```bash
# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Create project
kubectl apply -f argocd/projects/aipx.yaml

# Deploy applications
kubectl apply -f argocd/applications/
```

## Service Details

### Order Management Service (OMS)

- **Port:** 50051 (gRPC), 9090 (metrics)
- **Replicas:** 3-10 (HPA)
- **Resources:** 256Mi-512Mi memory, 250m-500m CPU
- **Dependencies:** PostgreSQL, Kafka, Redis

### User Service

- **Port:** 8082 (HTTP), 9091 (metrics)
- **Replicas:** 3-8 (HPA)
- **Resources:** 128Mi-256Mi memory, 100m-250m CPU
- **Dependencies:** PostgreSQL

### Cognitive Service

- **Port:** 8001 (HTTP), 9093 (metrics)
- **Replicas:** 2
- **Resources:** 512Mi-1Gi memory, 500m-1000m CPU
- **Dependencies:** PostgreSQL, Kafka, Anthropic API

### Infrastructure Components

#### PostgreSQL (TimescaleDB)
- **Replicas:** 1 (StatefulSet)
- **Storage:** 200Gi
- **Resources:** 1Gi-2Gi memory, 500m-1000m CPU
- **Backup:** Every 15 minutes to S3

#### Kafka
- **Replicas:** 3 (StatefulSet)
- **Storage:** 100Gi per broker
- **Resources:** 2Gi-4Gi memory, 1000m-2000m CPU
- **Replication Factor:** 3

#### Redis
- **Replicas:** 3 (StatefulSet)
- **Storage:** 20Gi per replica
- **Resources:** 256Mi-512Mi memory, 250m-500m CPU
- **Persistence:** AOF + RDB

## Monitoring

### Prometheus

Scrapes metrics from:
- Kubernetes API server
- Nodes (node-exporter)
- Pods (via annotations)
- Services (via ServiceMonitors)

**Access:**
```bash
kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090
```

### Grafana

Pre-configured dashboards:
1. System Metrics (CPU, Memory, Network, Disk)
2. Application Metrics (Request rate, Latency, Errors)
3. Business Metrics (Orders/sec, P&L, Win rate)

**Access:**
```bash
kubectl port-forward -n monitoring svc/kube-prometheus-stack-grafana 3000:80
# Username: admin
# Password: <from values.yaml>
```

### Loki

Centralized logging with Promtail agents on each node.

**Query examples:**
```logql
{namespace="aipx",app="oms"} |= "error"
{namespace="aipx"} |= "order" | json | status="failed"
```

### Jaeger

Distributed tracing for microservices.

**Access:**
```bash
kubectl port-forward -n aipx svc/jaeger-query 16686:16686
```

## Security

### Network Policies

- Default deny all traffic
- Allow DNS resolution
- Allow services to communicate with infrastructure
- Allow ingress from nginx-ingress
- Allow Prometheus scraping

### RBAC

- ServiceAccount: `aipx-service-account`
- Permissions: Read-only access to ConfigMaps, Secrets, Pods, Services

### Secrets Management

**DO NOT commit secrets to Git!**

Options:
1. **Sealed Secrets:** Encrypt secrets with kubeseal
2. **External Secrets Operator:** Sync from AWS Secrets Manager, Vault
3. **Manual:** Create via kubectl commands

Example with Sealed Secrets:
```bash
# Install sealed-secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml

# Create sealed secret
kubeseal --format=yaml < secret.yaml > sealed-secret.yaml

# Commit sealed-secret.yaml to Git
```

## Resource Management

### ResourceQuota (per namespace)

- CPU requests: 50 cores
- CPU limits: 100 cores
- Memory requests: 100Gi
- Memory limits: 200Gi
- PVCs: 20
- Pods: 100

### LimitRange

- Container default: 256Mi memory, 250m CPU
- Container max: 8Gi memory, 4 cores
- Pod max: 16Gi memory, 8 cores

### HorizontalPodAutoscaler

All services have HPA configured:
- Target CPU: 70%
- Target Memory: 80%
- Scale up: Fast (30s stabilization)
- Scale down: Slow (5m stabilization)

## High Availability

### Pod Anti-Affinity

Services spread across nodes:
```yaml
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values: [oms]
        topologyKey: kubernetes.io/hostname
```

### Topology Spread Constraints

Services spread across zones:
```yaml
topologySpreadConstraints:
- maxSkew: 1
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
  labelSelector:
    matchLabels:
      app: oms
```

### PodDisruptionBudget

Ensures minimum replicas during disruptions:
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: oms-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: oms
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n aipx
kubectl describe pod <pod-name> -n aipx
kubectl logs <pod-name> -n aipx
kubectl logs <pod-name> -n aipx --previous
```

### Check Service Connectivity

```bash
kubectl get svc -n aipx
kubectl get endpoints -n aipx
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://user-service.aipx.svc.cluster.local:8082/health
```

### Check Resource Usage

```bash
kubectl top nodes
kubectl top pods -n aipx
kubectl get hpa -n aipx
```

### Check Events

```bash
kubectl get events -n aipx --sort-by='.lastTimestamp' | tail -20
```

### Validate Manifests

```bash
# Dry run
kubectl apply -k services/oms/ --dry-run=client

# Server-side dry run
kubectl apply -k services/oms/ --dry-run=server

# Diff
kubectl diff -k services/oms/
```

## CI/CD Integration

### GitHub Actions

Workflows automatically:
1. Run tests on PR
2. Build and push Docker images on merge
3. Update image tags in manifests
4. ArgoCD syncs changes

### ArgoCD

- **Sync Policy:** Automated with self-heal
- **Prune:** Enabled (removes deleted resources)
- **Retry:** 5 attempts with exponential backoff
- **Notifications:** Slack integration

## Best Practices

1. **Use Kustomize:** Layer overlays for environments
2. **Version Everything:** Tag images, track manifests in Git
3. **Set Resource Limits:** Prevent resource exhaustion
4. **Use Health Checks:** liveness and readiness probes
5. **Implement HPA:** Auto-scale based on metrics
6. **Configure PDB:** Maintain availability during disruptions
7. **Apply Network Policies:** Secure pod-to-pod communication
8. **Monitor Everything:** Metrics, logs, traces
9. **Test Disaster Recovery:** Regular backup/restore drills
10. **Document Changes:** Update this README

## Additional Resources

- [Deployment Guide](../DEPLOYMENT.md)
- [Operations Runbook](../OPERATIONS.md)
- [Disaster Recovery Plan](../DISASTER_RECOVERY.md)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kustomize Documentation](https://kustomize.io/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)

## Support

- GitHub Issues: https://github.com/your-org/aipx/issues
- Slack: #aipx-infrastructure
- Email: devops@aipx.io
