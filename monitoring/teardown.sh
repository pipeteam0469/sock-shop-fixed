#!/bin/bash
# ============================================================
# PipeGuard Monitoring Stack - Teardown Script
# ============================================================

set -e

NAMESPACE="monitoring"

echo "Tearing down PipeGuard Monitoring Stack..."

echo "[1/5] Deleting AI Service..."
kubectl delete -f "$(dirname "$0")/ai-service/01-deployment.yaml" 2>/dev/null || true

echo "[2/5] Deleting Grafana..."
kubectl delete -f "$(dirname "$0")/grafana/03-deployment.yaml" 2>/dev/null || true
kubectl delete -f "$(dirname "$0")/grafana/02-provisioning-configmap.yaml" 2>/dev/null || true
kubectl delete -f "$(dirname "$0")/grafana/01-dashboards-configmap.yaml" 2>/dev/null || true

echo "[3/5] Deleting Prometheus..."
kubectl delete -f "$(dirname "$0")/prometheus/03-deployment.yaml" 2>/dev/null || true
kubectl delete -f "$(dirname "$0")/prometheus/02-rules.yaml" 2>/dev/null || true
kubectl delete -f "$(dirname "$0")/prometheus/01-configmap.yaml" 2>/dev/null || true

echo "[4/5] Deleting Alertmanager..."
kubectl delete -f "$(dirname "$0")/alertmanager/02-deployment.yaml" 2>/dev/null || true
kubectl delete -f "$(dirname "$0")/alertmanager/01-configmap.yaml" 2>/dev/null || true

echo "[5/5] Deleting RBAC and Namespace..."
kubectl delete -f "$(dirname "$0")/prometheus/00-namespace-rbac.yaml" 2>/dev/null || true
kubectl delete namespace "$NAMESPACE" 2>/dev/null || true

echo "Done. All monitoring resources removed."
