# AIPX Disaster Recovery Plan

Comprehensive backup and restore procedures for the AIPX trading platform.

## Table of Contents

1. [Overview](#overview)
2. [Backup Strategy](#backup-strategy)
3. [Backup Procedures](#backup-procedures)
4. [Restore Procedures](#restore-procedures)
5. [Disaster Scenarios](#disaster-scenarios)
6. [Testing](#testing)

---

## Overview

### Recovery Objectives

- **RTO (Recovery Time Objective):** 4 hours
- **RPO (Recovery Point Objective):** 15 minutes

### Backup Schedule

| Component | Frequency | Retention | Storage |
|-----------|-----------|-----------|---------|
| PostgreSQL | Every 15 min | 30 days | S3 + GCS |
| Kafka topics | Hourly | 7 days | S3 |
| Redis snapshots | Every 6 hours | 7 days | S3 |
| Kubernetes configs | On change | Forever | Git |
| Application logs | Real-time | 90 days | Loki |

### Critical Data

1. **Database (PostgreSQL)**
   - User accounts
   - Trading orders
   - Strategy configurations
   - Historical market data

2. **Message Queue (Kafka)**
   - Strategy signals
   - Order events
   - Market data streams

3. **Cache (Redis)**
   - Session data
   - Real-time market data

4. **Application State**
   - Kubernetes manifests
   - ConfigMaps
   - Secrets (encrypted)

---

## Backup Strategy

### Automated Backups

#### PostgreSQL Continuous Archiving

```yaml
# k8s/infrastructure/postgresql/backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgresql-backup
  namespace: aipx
spec:
  schedule: "*/15 * * * *"  # Every 15 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16
            command:
            - /bin/sh
            - -c
            - |
              TIMESTAMP=$(date +%Y%m%d_%H%M%S)
              pg_dump -U postgres -h postgresql aipx | \
              gzip | \
              aws s3 cp - s3://aipx-backups/postgres/backup_${TIMESTAMP}.sql.gz

              # Also backup to GCS for redundancy
              pg_dump -U postgres -h postgresql aipx | \
              gzip | \
              gsutil cp - gs://aipx-backups/postgres/backup_${TIMESTAMP}.sql.gz

              # Keep last 30 days
              aws s3 ls s3://aipx-backups/postgres/ | \
              awk '{print $4}' | \
              sort -r | \
              tail -n +2880 | \
              xargs -I {} aws s3 rm s3://aipx-backups/postgres/{}
            env:
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: access_key_id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: secret_access_key
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: password
          restartPolicy: OnFailure
```

#### Kafka Backup

```yaml
# k8s/infrastructure/kafka/backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: kafka-backup
  namespace: aipx
spec:
  schedule: "0 * * * *"  # Hourly
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: confluentinc/cp-kafka:7.5.0
            command:
            - /bin/sh
            - -c
            - |
              TIMESTAMP=$(date +%Y%m%d_%H%M%S)

              # Export topic data
              for topic in strategy-signals order-events market-data; do
                kafka-console-consumer \
                  --bootstrap-server kafka:9092 \
                  --topic $topic \
                  --from-beginning \
                  --max-messages 1000000 \
                  --timeout-ms 30000 > /tmp/${topic}.json

                gzip /tmp/${topic}.json
                aws s3 cp /tmp/${topic}.json.gz s3://aipx-backups/kafka/${topic}_${TIMESTAMP}.json.gz
              done
            env:
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: access_key_id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: secret_access_key
          restartPolicy: OnFailure
```

#### Redis Backup

```bash
# Redis AOF and RDB are already configured in StatefulSet
# Additional script for cloud backup

#!/bin/bash
# backup-redis.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Trigger BGSAVE
kubectl exec redis-0 -n aipx -- redis-cli BGSAVE

# Wait for save to complete
sleep 30

# Copy RDB file
kubectl cp aipx/redis-0:/data/dump.rdb /tmp/redis_${TIMESTAMP}.rdb

# Upload to S3
gzip /tmp/redis_${TIMESTAMP}.rdb
aws s3 cp /tmp/redis_${TIMESTAMP}.rdb.gz s3://aipx-backups/redis/

# Cleanup
rm /tmp/redis_${TIMESTAMP}.rdb.gz
```

### Kubernetes State Backup

```bash
#!/bin/bash
# backup-k8s-state.sh

BACKUP_DIR="/tmp/k8s-backup-$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# Backup all resources in aipx namespace
kubectl get all -n aipx -o yaml > $BACKUP_DIR/all-resources.yaml
kubectl get configmap -n aipx -o yaml > $BACKUP_DIR/configmaps.yaml
kubectl get secret -n aipx -o yaml > $BACKUP_DIR/secrets.yaml
kubectl get pvc -n aipx -o yaml > $BACKUP_DIR/pvcs.yaml
kubectl get ingress -n aipx -o yaml > $BACKUP_DIR/ingress.yaml

# Backup RBAC
kubectl get serviceaccount -n aipx -o yaml > $BACKUP_DIR/serviceaccounts.yaml
kubectl get role -n aipx -o yaml > $BACKUP_DIR/roles.yaml
kubectl get rolebinding -n aipx -o yaml > $BACKUP_DIR/rolebindings.yaml

# Create tarball
tar -czf $BACKUP_DIR.tar.gz -C /tmp $(basename $BACKUP_DIR)

# Upload to S3
aws s3 cp $BACKUP_DIR.tar.gz s3://aipx-backups/kubernetes/

# Cleanup
rm -rf $BACKUP_DIR $BACKUP_DIR.tar.gz
```

---

## Backup Procedures

### Manual Backup

```bash
# Full system backup before major changes

# 1. Backup database
./scripts/backup-postgres.sh

# 2. Backup Kafka
./scripts/backup-kafka.sh

# 3. Backup Redis
./scripts/backup-redis.sh

# 4. Backup Kubernetes state
./scripts/backup-k8s-state.sh

# 5. Verify backups exist
aws s3 ls s3://aipx-backups/postgres/ | tail -5
aws s3 ls s3://aipx-backups/kafka/ | tail -5
aws s3 ls s3://aipx-backups/redis/ | tail -5
aws s3 ls s3://aipx-backups/kubernetes/ | tail -5
```

### Verify Backup Integrity

```bash
#!/bin/bash
# verify-backups.sh

echo "Verifying PostgreSQL backup..."
LATEST_PG=$(aws s3 ls s3://aipx-backups/postgres/ | sort | tail -n 1 | awk '{print $4}')
aws s3 cp s3://aipx-backups/postgres/$LATEST_PG - | gunzip | head -20

echo "Verifying Kafka backup..."
LATEST_KAFKA=$(aws s3 ls s3://aipx-backups/kafka/ | sort | tail -n 1 | awk '{print $4}')
aws s3 cp s3://aipx-backups/kafka/$LATEST_KAFKA - | gunzip | head -20

echo "Verifying Redis backup..."
LATEST_REDIS=$(aws s3 ls s3://aipx-backups/redis/ | sort | tail -n 1 | awk '{print $4}')
aws s3 cp s3://aipx-backups/redis/$LATEST_REDIS /tmp/test.rdb.gz
gunzip /tmp/test.rdb.gz
redis-server --port 6380 &
sleep 2
redis-cli -p 6380 DEBUG RELOAD /tmp/test.rdb
redis-cli -p 6380 DBSIZE
redis-cli -p 6380 SHUTDOWN
rm /tmp/test.rdb

echo "All backups verified successfully!"
```

---

## Restore Procedures

### Full System Restore

```bash
#!/bin/bash
# restore-full-system.sh

set -e

echo "=== AIPX Full System Restore ==="
echo "WARNING: This will overwrite existing data!"
read -p "Continue? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
  echo "Aborted."
  exit 1
fi

# 1. Restore Kubernetes infrastructure
echo "Restoring infrastructure..."
kubectl apply -f k8s/base/
kubectl apply -k k8s/infrastructure/

# Wait for infrastructure
kubectl wait --for=condition=ready pod -l app=postgresql -n aipx --timeout=10m
kubectl wait --for=condition=ready pod -l app=kafka -n aipx --timeout=10m
kubectl wait --for=condition=ready pod -l app=redis -n aipx --timeout=10m

# 2. Restore PostgreSQL
echo "Restoring PostgreSQL..."
LATEST_PG=$(aws s3 ls s3://aipx-backups/postgres/ | sort | tail -n 1 | awk '{print $4}')
aws s3 cp s3://aipx-backups/postgres/$LATEST_PG - | \
  gunzip | \
  kubectl exec -i postgresql-0 -n aipx -- psql -U postgres aipx

# 3. Restore Redis
echo "Restoring Redis..."
LATEST_REDIS=$(aws s3 ls s3://aipx-backups/redis/ | sort | tail -n 1 | awk '{print $4}')
aws s3 cp s3://aipx-backups/redis/$LATEST_REDIS - | \
  gunzip | \
  kubectl exec -i redis-0 -n aipx -- tee /data/dump.rdb > /dev/null
kubectl rollout restart statefulset/redis -n aipx

# 4. Restore Kafka topics (if needed)
echo "Restoring Kafka topics..."
for topic in strategy-signals order-events market-data; do
  LATEST_TOPIC=$(aws s3 ls s3://aipx-backups/kafka/ | grep $topic | sort | tail -n 1 | awk '{print $4}')
  aws s3 cp s3://aipx-backups/kafka/$LATEST_TOPIC - | \
    gunzip | \
    kubectl exec -i kafka-0 -n aipx -- kafka-console-producer --topic $topic --bootstrap-server localhost:9092
done

# 5. Deploy services
echo "Deploying services..."
kubectl apply -k k8s/services/

# 6. Verify
echo "Verifying restore..."
kubectl get pods -n aipx
kubectl exec postgresql-0 -n aipx -- psql -U postgres aipx -c "SELECT COUNT(*) FROM orders;"
kubectl exec redis-0 -n aipx -- redis-cli DBSIZE
kubectl exec kafka-0 -n aipx -- kafka-topics --list --bootstrap-server localhost:9092

echo "Restore completed successfully!"
```

### Restore Individual Components

#### PostgreSQL Only

```bash
#!/bin/bash
# restore-postgres.sh

BACKUP_FILE=${1:-latest}

if [ "$BACKUP_FILE" == "latest" ]; then
  BACKUP_FILE=$(aws s3 ls s3://aipx-backups/postgres/ | sort | tail -n 1 | awk '{print $4}')
fi

echo "Restoring PostgreSQL from: $BACKUP_FILE"
read -p "This will drop and recreate the database. Continue? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
  exit 1
fi

# Scale down services
kubectl scale deployment -n aipx --replicas=0 --all

# Drop and recreate database
kubectl exec postgresql-0 -n aipx -- psql -U postgres -c "DROP DATABASE IF EXISTS aipx;"
kubectl exec postgresql-0 -n aipx -- psql -U postgres -c "CREATE DATABASE aipx;"

# Restore backup
aws s3 cp s3://aipx-backups/postgres/$BACKUP_FILE - | \
  gunzip | \
  kubectl exec -i postgresql-0 -n aipx -- psql -U postgres aipx

# Scale up services
kubectl scale deployment -n aipx --replicas=3 --all

echo "PostgreSQL restored successfully!"
```

#### Kafka Topics Only

```bash
#!/bin/bash
# restore-kafka-topics.sh

TOPIC=$1
BACKUP_FILE=$2

if [ -z "$TOPIC" ] || [ -z "$BACKUP_FILE" ]; then
  echo "Usage: $0 <topic> <backup-file>"
  exit 1
fi

echo "Restoring Kafka topic: $TOPIC from $BACKUP_FILE"

# Download and restore
aws s3 cp s3://aipx-backups/kafka/$BACKUP_FILE - | \
  gunzip | \
  kubectl exec -i kafka-0 -n aipx -- kafka-console-producer \
    --topic $TOPIC \
    --bootstrap-server localhost:9092

echo "Topic restored successfully!"
```

#### Redis Only

```bash
#!/bin/bash
# restore-redis.sh

BACKUP_FILE=${1:-latest}

if [ "$BACKUP_FILE" == "latest" ]; then
  BACKUP_FILE=$(aws s3 ls s3://aipx-backups/redis/ | sort | tail -n 1 | awk '{print $4}')
fi

echo "Restoring Redis from: $BACKUP_FILE"

# Download backup
aws s3 cp s3://aipx-backups/redis/$BACKUP_FILE /tmp/redis.rdb.gz
gunzip /tmp/redis.rdb.gz

# Copy to Redis pod and restart
kubectl cp /tmp/redis.rdb aipx/redis-0:/data/dump.rdb
kubectl rollout restart statefulset/redis -n aipx

# Wait for restart
kubectl wait --for=condition=ready pod/redis-0 -n aipx --timeout=5m

# Verify
kubectl exec redis-0 -n aipx -- redis-cli DBSIZE

rm /tmp/redis.rdb
echo "Redis restored successfully!"
```

---

## Disaster Scenarios

### Scenario 1: Complete Cluster Failure

**Situation:** Entire Kubernetes cluster is lost

**Recovery Steps:**

1. Create new cluster
```bash
eksctl create cluster -f k8s/cluster-config.yaml
```

2. Install prerequisites
```bash
./scripts/install-prerequisites.sh
```

3. Restore secrets
```bash
./scripts/restore-secrets.sh
```

4. Full system restore
```bash
./scripts/restore-full-system.sh
```

5. Update DNS
```bash
# Point DNS to new LoadBalancer IP
kubectl get svc -n ingress-nginx
```

6. Verify and test
```bash
./scripts/smoke-test.sh
```

**Estimated Time:** 4 hours

### Scenario 2: Database Corruption

**Situation:** PostgreSQL data is corrupted

**Recovery Steps:**

1. Identify corruption
```bash
kubectl exec postgresql-0 -n aipx -- pg_dump -U postgres aipx > /dev/null
# If fails, corruption confirmed
```

2. Scale down services
```bash
kubectl scale deployment -n aipx --replicas=0 --all
```

3. Find latest good backup
```bash
aws s3 ls s3://aipx-backups/postgres/ | tail -20
```

4. Restore database
```bash
./scripts/restore-postgres.sh <backup-file>
```

5. Verify data integrity
```bash
kubectl exec postgresql-0 -n aipx -- psql -U postgres aipx -c "SELECT COUNT(*) FROM orders;"
```

6. Scale up services
```bash
kubectl scale deployment -n aipx --replicas=3 --all
```

**Estimated Time:** 1 hour

### Scenario 3: Kafka Data Loss

**Situation:** Kafka topics are lost or corrupted

**Recovery Steps:**

1. Check Kafka status
```bash
kubectl exec kafka-0 -n aipx -- kafka-topics --list --bootstrap-server localhost:9092
```

2. Recreate topics if needed
```bash
kubectl exec kafka-0 -n aipx -- kafka-topics --create --topic strategy-signals --bootstrap-server localhost:9092 --partitions 3 --replication-factor 3
```

3. Restore from backup
```bash
./scripts/restore-kafka-topics.sh strategy-signals <backup-file>
```

4. Verify
```bash
kubectl exec kafka-0 -n aipx -- kafka-console-consumer --topic strategy-signals --from-beginning --max-messages 10 --bootstrap-server localhost:9092
```

**Estimated Time:** 30 minutes

### Scenario 4: Region Outage

**Situation:** Entire AWS/GCP region is unavailable

**Recovery Steps:**

1. Activate DR region cluster (pre-provisioned)
```bash
kubectl config use-context aipx-dr
```

2. Restore from cross-region backups
```bash
./scripts/restore-from-dr.sh
```

3. Update DNS to DR region
```bash
# Update Route53/Cloud DNS
```

4. Verify services
```bash
./scripts/health-check.sh
```

**Estimated Time:** 2 hours (with pre-provisioned infrastructure)

---

## Testing

### Monthly DR Drill

```bash
#!/bin/bash
# dr-drill.sh

echo "=== Monthly Disaster Recovery Drill ==="

# 1. Create test namespace
kubectl create namespace aipx-dr-test

# 2. Restore to test namespace
./scripts/restore-to-namespace.sh aipx-dr-test

# 3. Run tests
./scripts/smoke-test.sh aipx-dr-test

# 4. Verify data
kubectl exec postgresql-0 -n aipx-dr-test -- psql -U postgres aipx -c "SELECT COUNT(*) FROM orders;"

# 5. Cleanup
kubectl delete namespace aipx-dr-test

echo "DR drill completed successfully!"
```

### Backup Verification

Run weekly:

```bash
#!/bin/bash
# weekly-backup-verification.sh

# Test latest backups can be restored
./scripts/verify-backups.sh

# Send report
./scripts/send-backup-report.sh
```

---

## Backup Monitoring

### Prometheus Alerts

```yaml
# Alert if backup hasn't run
- alert: BackupMissing
  expr: time() - backup_last_success_timestamp_seconds > 3600
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Backup hasn't run for {{ $labels.component }}"

# Alert if backup failed
- alert: BackupFailed
  expr: backup_last_result == 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Last backup failed for {{ $labels.component }}"
```

### Backup Dashboard

Create Grafana dashboard showing:
- Last successful backup time
- Backup size trends
- Backup duration
- Failed backup count
- Recovery test results

---

## Contact Information

- **DR Lead:** devops@aipx.io
- **On-Call:** PagerDuty
- **Escalation:** CTO

---

## Document Maintenance

This document should be:
- Reviewed quarterly
- Updated after any DR test
- Updated when infrastructure changes
- Verified annually with full restore test

Last Updated: 2024-01-15
Next Review: 2024-04-15
