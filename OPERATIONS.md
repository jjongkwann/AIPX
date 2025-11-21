# AIPX Operations Runbook

Operational procedures and troubleshooting guide for the AIPX trading platform.

## Table of Contents

1. [Daily Operations](#daily-operations)
2. [Monitoring](#monitoring)
3. [Common Tasks](#common-tasks)
4. [Incident Response](#incident-response)
5. [Troubleshooting Guide](#troubleshooting-guide)
6. [Maintenance Windows](#maintenance-windows)

---

## Daily Operations

### Morning Checks

```bash
# Check cluster health
kubectl get nodes
kubectl top nodes

# Check pod status
kubectl get pods -n aipx
kubectl top pods -n aipx

# Check recent events
kubectl get events -n aipx --sort-by='.lastTimestamp' | tail -20

# Check deployments
kubectl get deployments -n aipx
```

### Health Check Script

```bash
#!/bin/bash
# health-check.sh

echo "=== AIPX Health Check ==="
echo

echo "1. Cluster Nodes:"
kubectl get nodes

echo
echo "2. Pod Status:"
kubectl get pods -n aipx | grep -v Running | grep -v Completed

echo
echo "3. Failed Pods:"
kubectl get pods -n aipx --field-selector=status.phase=Failed

echo
echo "4. Service Endpoints:"
kubectl get endpoints -n aipx | grep -E "oms|user-service|cognitive"

echo
echo "5. Recent Events:"
kubectl get events -n aipx --sort-by='.lastTimestamp' | tail -10

echo
echo "6. HPA Status:"
kubectl get hpa -n aipx

echo
echo "7. Kafka Consumer Lag:"
kubectl exec -it kafka-0 -n aipx -- kafka-consumer-groups --describe --all-groups --bootstrap-server localhost:9092 | grep LAG
```

---

## Monitoring

### Grafana Dashboards

Access Grafana at https://grafana.aipx.io

Key dashboards:
1. **System Metrics** - CPU, memory, network, disk
2. **Application Metrics** - Request rate, latency, errors
3. **Business Metrics** - Orders/sec, P&L, strategy performance

### Alert Channels

- **Critical:** PagerDuty + Slack #aipx-critical
- **Warning:** Slack #aipx-warnings
- **Info:** Slack #aipx-deployments

### Key Metrics to Watch

```promql
# Error rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# P95 latency
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service))

# Order throughput
sum(rate(orders_total[5m]))

# Kafka consumer lag
kafka_consumer_lag{topic="strategy-signals"}

# Database connections
pg_stat_database_numbackends{datname="aipx"}
```

### Prometheus Alerts

Check active alerts:
```bash
# View all alerts
kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090

# Open http://localhost:9090/alerts
```

---

## Common Tasks

### 1. Scaling Services

#### Manual Scaling

```bash
# Scale up
kubectl scale deployment/oms -n aipx --replicas=5

# Scale down
kubectl scale deployment/oms -n aipx --replicas=2

# Check HPA
kubectl get hpa -n aipx
```

#### Adjust HPA

```bash
# Edit HPA
kubectl edit hpa oms-hpa -n aipx

# Update target CPU
spec:
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60  # Changed from 70
```

### 2. Viewing Logs

```bash
# Recent logs
kubectl logs deployment/oms -n aipx --tail=100

# Follow logs
kubectl logs -f deployment/oms -n aipx

# Logs from specific pod
kubectl logs oms-5d7f8c9b4d-abc12 -n aipx

# Previous container logs (if crashed)
kubectl logs oms-5d7f8c9b4d-abc12 -n aipx --previous

# Multiple pods
kubectl logs -l app=oms -n aipx --tail=50

# Search logs in Loki
kubectl port-forward -n monitoring svc/loki 3100:3100
# Query: {namespace="aipx",app="oms"} |= "error"
```

### 3. Executing Commands in Pods

```bash
# Get shell access
kubectl exec -it oms-5d7f8c9b4d-abc12 -n aipx -- /bin/sh

# Run single command
kubectl exec oms-5d7f8c9b4d-abc12 -n aipx -- ps aux

# PostgreSQL queries
kubectl exec -it postgresql-0 -n aipx -- psql -U postgres -d aipx -c "SELECT COUNT(*) FROM orders;"

# Redis commands
kubectl exec -it redis-0 -n aipx -- redis-cli INFO stats

# Kafka topics
kubectl exec -it kafka-0 -n aipx -- kafka-topics --list --bootstrap-server localhost:9092
```

### 4. Updating Configuration

```bash
# Edit ConfigMap
kubectl edit configmap aipx-config -n aipx

# Restart pods to pick up changes
kubectl rollout restart deployment/oms -n aipx

# Update secret
kubectl create secret generic postgres-credentials -n aipx \
  --from-literal=password=<NEW_PASSWORD> \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 5. Deploying Updates

```bash
# Update image
kubectl set image deployment/oms oms=ghcr.io/aipx/oms:v1.2.0 -n aipx

# Or edit deployment
kubectl edit deployment/oms -n aipx

# Watch rollout
kubectl rollout status deployment/oms -n aipx

# Pause rollout if issues
kubectl rollout pause deployment/oms -n aipx

# Resume rollout
kubectl rollout resume deployment/oms -n aipx

# Rollback
kubectl rollout undo deployment/oms -n aipx
```

### 6. Database Operations

#### Backup

```bash
# Create backup
kubectl exec postgresql-0 -n aipx -- pg_dump -U postgres aipx > backup-$(date +%Y%m%d).sql

# Backup to cloud
kubectl exec postgresql-0 -n aipx -- pg_dump -U postgres aipx | \
  gzip | \
  aws s3 cp - s3://aipx-backups/db-backup-$(date +%Y%m%d).sql.gz
```

#### Restore

```bash
# Restore from backup
kubectl exec -i postgresql-0 -n aipx -- psql -U postgres aipx < backup-20240115.sql

# Restore from cloud
aws s3 cp s3://aipx-backups/db-backup-20240115.sql.gz - | \
  gunzip | \
  kubectl exec -i postgresql-0 -n aipx -- psql -U postgres aipx
```

#### Common Queries

```sql
-- Check order counts
SELECT status, COUNT(*) FROM orders GROUP BY status;

-- Recent orders
SELECT * FROM orders ORDER BY created_at DESC LIMIT 10;

-- Active strategies
SELECT * FROM strategies WHERE status = 'active';

-- Database size
SELECT pg_size_pretty(pg_database_size('aipx'));

-- Table sizes
SELECT
  schemaname || '.' || tablename AS table,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 10;
```

### 7. Kafka Operations

```bash
# List topics
kubectl exec kafka-0 -n aipx -- kafka-topics --list --bootstrap-server localhost:9092

# Describe topic
kubectl exec kafka-0 -n aipx -- kafka-topics --describe --topic strategy-signals --bootstrap-server localhost:9092

# Consumer groups
kubectl exec kafka-0 -n aipx -- kafka-consumer-groups --list --bootstrap-server localhost:9092

# Consumer lag
kubectl exec kafka-0 -n aipx -- kafka-consumer-groups \
  --describe \
  --group aipx-consumer-group \
  --bootstrap-server localhost:9092

# Produce test message
echo "test message" | kubectl exec -i kafka-0 -n aipx -- kafka-console-producer --topic test --bootstrap-server localhost:9092

# Consume messages
kubectl exec -it kafka-0 -n aipx -- kafka-console-consumer --topic strategy-signals --from-beginning --bootstrap-server localhost:9092 --max-messages 10
```

---

## Incident Response

### Severity Levels

- **P0 (Critical):** Complete service outage
- **P1 (High):** Major functionality impaired
- **P2 (Medium):** Partial functionality impaired
- **P3 (Low):** Minor issues, workaround available

### Response Procedures

#### P0/P1 Incident

1. **Acknowledge**
   ```bash
   # Join incident channel
   # Post: "Acknowledged, investigating"
   ```

2. **Assess**
   ```bash
   # Check pod status
   kubectl get pods -n aipx

   # Check recent events
   kubectl get events -n aipx --sort-by='.lastTimestamp' | tail -20

   # Check logs
   kubectl logs -l app=<failing-service> -n aipx --tail=100

   # Check metrics
   # Open Grafana dashboards
   ```

3. **Mitigate**
   ```bash
   # Quick fixes:

   # Restart pods
   kubectl rollout restart deployment/<service> -n aipx

   # Rollback deployment
   kubectl rollout undo deployment/<service> -n aipx

   # Scale up
   kubectl scale deployment/<service> -n aipx --replicas=5

   # Kill problematic pod
   kubectl delete pod <pod-name> -n aipx
   ```

4. **Communicate**
   - Update incident channel every 15 minutes
   - Post status to status page
   - Notify stakeholders

5. **Resolve**
   - Verify metrics returned to normal
   - Post resolution message
   - Schedule postmortem

### Common Incidents

#### Service Not Responding

```bash
# 1. Check pod status
kubectl get pods -n aipx | grep <service>

# 2. Check logs
kubectl logs deployment/<service> -n aipx --tail=100

# 3. Check recent deployments
kubectl rollout history deployment/<service> -n aipx

# 4. Quick fix: restart
kubectl rollout restart deployment/<service> -n aipx

# 5. If that fails: rollback
kubectl rollout undo deployment/<service> -n aipx
```

#### High Error Rate

```bash
# 1. Check which service
# Open Grafana > Application Metrics > Error Rate by Service

# 2. Check logs for errors
kubectl logs -l app=<service> -n aipx | grep -i error | tail -50

# 3. Check dependencies
kubectl get endpoints -n aipx

# 4. Check resource usage
kubectl top pods -n aipx | grep <service>

# 5. Scale up if needed
kubectl scale deployment/<service> -n aipx --replicas=5
```

#### Database Connection Pool Exhausted

```bash
# 1. Check connections
kubectl exec postgresql-0 -n aipx -- psql -U postgres -d aipx -c "SELECT COUNT(*) FROM pg_stat_activity;"

# 2. Check slow queries
kubectl exec postgresql-0 -n aipx -- psql -U postgres -d aipx -c "SELECT pid, now() - pg_stat_activity.query_start AS duration, query FROM pg_stat_activity WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';"

# 3. Kill long-running queries
kubectl exec postgresql-0 -n aipx -- psql -U postgres -d aipx -c "SELECT pg_terminate_backend(<pid>);"

# 4. Restart services to reset connection pools
kubectl rollout restart deployment -n aipx
```

#### Kafka Consumer Lag

```bash
# 1. Check lag
kubectl exec kafka-0 -n aipx -- kafka-consumer-groups \
  --describe --group aipx-consumer-group \
  --bootstrap-server localhost:9092

# 2. Scale up consumers
kubectl scale deployment/strategy-worker -n aipx --replicas=5

# 3. Check consumer logs
kubectl logs -l app=strategy-worker -n aipx --tail=100

# 4. Restart consumers if stuck
kubectl rollout restart deployment/strategy-worker -n aipx
```

---

## Troubleshooting Guide

### Pod CrashLoopBackOff

```bash
# 1. Check logs
kubectl logs <pod-name> -n aipx --previous

# 2. Describe pod
kubectl describe pod <pod-name> -n aipx

# 3. Check events
kubectl get events -n aipx --sort-by='.lastTimestamp' | grep <pod-name>

# Common causes:
# - Missing environment variable
# - Wrong image
# - Application crash on startup
# - Health check failing too early
```

### ImagePullBackOff

```bash
# 1. Check image name
kubectl describe pod <pod-name> -n aipx | grep Image

# 2. Check image exists
docker pull <image-name>

# 3. Check image pull secrets
kubectl get secrets -n aipx | grep regcred

# 4. Fix: Update deployment with correct image
kubectl set image deployment/<service> <container>=<correct-image> -n aipx
```

### High CPU/Memory

```bash
# 1. Identify resource hogs
kubectl top pods -n aipx --sort-by=cpu
kubectl top pods -n aipx --sort-by=memory

# 2. Check limits
kubectl describe pod <pod-name> -n aipx | grep -A 5 Limits

# 3. Increase limits if justified
kubectl edit deployment/<service> -n aipx
# Update resources.limits.memory and resources.limits.cpu

# 4. Or scale horizontally
kubectl scale deployment/<service> -n aipx --replicas=5
```

### Persistent Volume Issues

```bash
# 1. Check PVC status
kubectl get pvc -n aipx

# 2. Check PV
kubectl get pv

# 3. Describe PVC
kubectl describe pvc <pvc-name> -n aipx

# 4. Check storage class
kubectl get storageclass

# 5. If stuck: delete and recreate
kubectl delete pvc <pvc-name> -n aipx
# Then redeploy the statefulset
```

---

## Maintenance Windows

### Planned Maintenance

1. **Schedule** (preferably Sunday 2-4 AM UTC)
2. **Notify** users 1 week in advance
3. **Prepare** rollback plan
4. **Backup** all data
5. **Execute** maintenance
6. **Verify** everything works
7. **Document** what was done

### Kubernetes Upgrade

```bash
# 1. Check current version
kubectl version

# 2. Backup everything
./scripts/backup-all.sh

# 3. Upgrade control plane (EKS example)
eksctl upgrade cluster --name aipx-production --version 1.28

# 4. Upgrade node groups
eksctl upgrade nodegroup --cluster=aipx-production --name=standard-workers

# 5. Verify
kubectl get nodes
kubectl get pods -n aipx

# 6. Test critical paths
./scripts/smoke-test.sh
```

### Database Maintenance

```bash
# 1. Announce maintenance window

# 2. Scale down services
kubectl scale deployment -n aipx --replicas=0 --all

# 3. Backup database
kubectl exec postgresql-0 -n aipx -- pg_dump -U postgres aipx > backup-maintenance.sql

# 4. Run maintenance
kubectl exec -it postgresql-0 -n aipx -- psql -U postgres aipx -c "VACUUM ANALYZE;"
kubectl exec -it postgresql-0 -n aipx -- psql -U postgres aipx -c "REINDEX DATABASE aipx;"

# 5. Scale up services
kubectl scale deployment -n aipx --replicas=3 --all

# 6. Verify
kubectl get pods -n aipx
```

---

## On-Call Rotation

### Responsibilities

- Respond to alerts within 15 minutes
- Investigate and mitigate incidents
- Escalate if needed
- Document issues in runbook
- Hand off to next on-call

### Handoff Checklist

- [ ] Review open incidents
- [ ] Review recent changes
- [ ] Review upcoming maintenance
- [ ] Share access credentials
- [ ] Test alert notifications

---

## Useful Commands Cheat Sheet

```bash
# Quick pod restart
kubectl rollout restart deployment/<service> -n aipx

# Get all resources
kubectl get all -n aipx

# Resource usage
kubectl top nodes
kubectl top pods -n aipx

# Recent events
kubectl get events -n aipx --sort-by='.lastTimestamp' | tail -20

# Pod logs
kubectl logs -f deployment/<service> -n aipx

# Execute command
kubectl exec -it <pod> -n aipx -- <command>

# Port forward
kubectl port-forward svc/<service> -n aipx <local-port>:<remote-port>

# Deployment history
kubectl rollout history deployment/<service> -n aipx

# Rollback
kubectl rollout undo deployment/<service> -n aipx

# Scale
kubectl scale deployment/<service> -n aipx --replicas=<n>

# Check HPA
kubectl get hpa -n aipx

# Describe resource
kubectl describe <resource> <name> -n aipx
```

---

## Contact Information

- **On-Call:** PagerDuty rotation
- **Slack:** #aipx-support
- **Email:** devops@aipx.io
- **Escalation:** CTO (for P0 incidents)

---

## Additional Resources

- [DEPLOYMENT.md](./DEPLOYMENT.md) - Deployment procedures
- [DISASTER_RECOVERY.md](./DISASTER_RECOVERY.md) - Backup and restore
- Grafana: https://grafana.aipx.io
- Prometheus: https://prometheus.aipx.io
- ArgoCD: https://argocd.aipx.io
