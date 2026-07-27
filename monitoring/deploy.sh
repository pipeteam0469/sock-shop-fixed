#!/bin/bash
# ============================================================
# PipeGuard Monitoring Stack - Full Deployment Script
# ============================================================
# Deploys: Prometheus, Alertmanager, Grafana, FastAPI AI Service
# Order matters: RBAC → ConfigMaps → Services
# ============================================================

set -e

NAMESPACE="monitoring"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=========================================="
echo "PipeGuard Monitoring Stack Deployment"
echo "=========================================="

# Step 1: Create namespace and RBAC
echo "[1/8] Creating namespace and RBAC..."
kubectl apply -f "$SCRIPT_DIR/prometheus/00-namespace-rbac.yaml"

# Step 2: Create Prometheus ConfigMap and Rules
echo "[2/8] Creating Prometheus configuration..."
kubectl apply -f "$SCRIPT_DIR/prometheus/01-configmap.yaml"
kubectl apply -f "$SCRIPT_DIR/prometheus/02-rules.yaml"

# Step 3: Create Alertmanager ConfigMap and Deployment
echo "[3/8] Creating Alertmanager..."
kubectl apply -f "$SCRIPT_DIR/alertmanager/01-configmap.yaml"
kubectl apply -f "$SCRIPT_DIR/alertmanager/02-deployment.yaml"

# Step 4: Create Grafana Dashboards ConfigMap
echo "[4/8] Creating Grafana dashboards..."
kubectl apply -f "$SCRIPT_DIR/grafana/01-dashboards-configmap.yaml"
kubectl apply -f "$SCRIPT_DIR/grafana/02-provisioning-configmap.yaml"

# Step 5: Deploy Grafana
echo "[5/8] Deploying Grafana..."
kubectl apply -f "$SCRIPT_DIR/grafana/03-deployment.yaml"

# Step 6: Deploy Prometheus (needs alertmanager service to exist first)
echo "[6/8] Deploying Prometheus..."
kubectl apply -f "$SCRIPT_DIR/prometheus/03-deployment.yaml"

# Step 7: Deploy AI Service
echo "[7/8] Deploying AI Alert Service..."
kubectl apply -f "$SCRIPT_DIR/ai-service/01-deployment.yaml"

# Step 8: Wait for everything to be ready
echo "[8/8] Waiting for all pods to be ready..."
sleep 10

echo ""
echo "=========================================="
echo "Checking pod status..."
echo "=========================================="
kubectl get pods -n "$NAMESPACE"

echo ""
echo "=========================================="
echo "Checking services..."
echo "=========================================="
kubectl get svc -n "$NAMESPACE"

echo ""
echo "=========================================="
echo "Deployment complete!"
echo "=========================================="
echo ""
echo "Access URLs:"
echo "  Grafana:     http://<VM2-IP>:3000/grafana  (admin / pipeguard2024)"
echo "  Prometheus:  http://<VM2-IP>:9090/prometheus"
echo "  Alertmanager: http://<VM2-IP>:9093/alertmanager"
echo "  AI Service:  http://<VM2-IP>:8000"
echo ""
echo "To test the alert pipeline:"
echo "  kubectl port-forward -n monitoring svc/grafana 3000:3000 &"
echo "  curl http://localhost:8000/alert/test"
echo ""
