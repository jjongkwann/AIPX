#!/bin/bash

# Phase 6 Verification Script
set -e

echo "========================================="
echo "AIPX Phase 6: Deployment Verification"
echo "========================================="
echo

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

SUCCESS=0
FAILURE=0

check() {
  if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} $1"
    ((SUCCESS++))
  else
    echo -e "${RED}✗${NC} $1"
    ((FAILURE++))
  fi
}

echo "1. Checking Docker Files..."
[ -f "docker-compose.yml" ] && check "docker-compose.yml exists"
[ -f "services/order-management-service/Dockerfile" ] && check "OMS Dockerfile exists"
[ -f "services/user-service/Dockerfile" ] && check "User Service Dockerfile exists"
[ -f "services/cognitive-service/Dockerfile" ] && check "Cognitive Service Dockerfile exists"
[ -f "services/dashboard-service/Dockerfile" ] && check "Dashboard Dockerfile exists"

echo
echo "2. Checking Kubernetes Base Resources..."
[ -f "k8s/base/namespace.yaml" ] && check "namespace.yaml exists"
[ -f "k8s/base/configmap.yaml" ] && check "configmap.yaml exists"
[ -f "k8s/base/secrets.yaml" ] && check "secrets.yaml exists"
[ -f "k8s/base/networkpolicy.yaml" ] && check "networkpolicy.yaml exists"
[ -f "k8s/base/ingress.yaml" ] && check "ingress.yaml exists"

echo
echo "3. Checking Service Manifests..."
[ -f "k8s/services/oms/deployment.yaml" ] && check "OMS deployment.yaml exists"
[ -f "k8s/services/oms/service.yaml" ] && check "OMS service.yaml exists"
[ -f "k8s/services/oms/hpa.yaml" ] && check "OMS hpa.yaml exists"
[ -f "k8s/services/user-service/deployment.yaml" ] && check "User Service deployment exists"
[ -f "k8s/services/cognitive/deployment.yaml" ] && check "Cognitive Service deployment exists"

echo
echo "4. Checking Infrastructure..."
[ -f "k8s/infrastructure/kafka/statefulset.yaml" ] && check "Kafka StatefulSet exists"
[ -f "k8s/infrastructure/redis/statefulset.yaml" ] && check "Redis StatefulSet exists"
[ -f "k8s/infrastructure/postgresql/statefulset.yaml" ] && check "PostgreSQL StatefulSet exists"

echo
echo "5. Checking Monitoring Stack..."
[ -f "k8s/monitoring/prometheus/prometheus.yml" ] && check "Prometheus config exists"
[ -f "k8s/monitoring/prometheus/alerts.yml" ] && check "Prometheus alerts exist"
[ -f "k8s/monitoring/prometheus/values.yaml" ] && check "Prometheus values exist"
[ -f "k8s/monitoring/grafana/dashboards/system-metrics.json" ] && check "System metrics dashboard exists"
[ -f "k8s/monitoring/grafana/dashboards/application-metrics.json" ] && check "Application metrics dashboard exists"
[ -f "k8s/monitoring/grafana/dashboards/business-metrics.json" ] && check "Business metrics dashboard exists"
[ -f "k8s/monitoring/loki/values.yaml" ] && check "Loki config exists"
[ -f "k8s/monitoring/jaeger/deployment.yaml" ] && check "Jaeger deployment exists"

echo
echo "6. Checking CI/CD..."
[ -f ".github/workflows/test.yml" ] && check "Test workflow exists"
[ -f ".github/workflows/build.yml" ] && check "Build workflow exists"
[ -f ".github/workflows/deploy.yml" ] && check "Deploy workflow exists"
[ -f "k8s/argocd/projects/aipx.yaml" ] && check "ArgoCD project exists"
[ -f "k8s/argocd/applications/oms.yaml" ] && check "ArgoCD OMS app exists"

echo
echo "7. Checking Documentation..."
[ -f "DEPLOYMENT.md" ] && check "DEPLOYMENT.md exists"
[ -f "OPERATIONS.md" ] && check "OPERATIONS.md exists"
[ -f "DISASTER_RECOVERY.md" ] && check "DISASTER_RECOVERY.md exists"
[ -f "k8s/README.md" ] && check "k8s/README.md exists"
[ -f "PHASE6_DEPLOYMENT_SUMMARY.md" ] && check "PHASE6_DEPLOYMENT_SUMMARY.md exists"

echo
echo "========================================="
echo "Verification Summary"
echo "========================================="
echo -e "${GREEN}Passed: $SUCCESS${NC}"
echo -e "${RED}Failed: $FAILURE${NC}"
echo

if [ $FAILURE -eq 0 ]; then
  echo -e "${GREEN}✓ All Phase 6 deliverables verified!${NC}"
  exit 0
else
  echo -e "${RED}✗ Some deliverables missing!${NC}"
  exit 1
fi
