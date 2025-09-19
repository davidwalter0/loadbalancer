#!/bin/bash
set -e

# Variables
CLUSTER_NAME="loadbalancer-cluster"

echo "Setting up k3d cluster for loadbalancer testing..."

# Check if k3d is installed
if ! command -v k3d &> /dev/null; then
  echo "k3d not found. Please install k3d first: https://k3d.io/"
  exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
  echo "kubectl not found. Please install kubectl first."
  exit 1
fi

# Check if the cluster already exists
if k3d cluster list | grep -q "$CLUSTER_NAME"; then
  echo "Cluster $CLUSTER_NAME already exists. Deleting it..."
  k3d cluster delete "$CLUSTER_NAME"
fi

# Create a new k3d cluster
echo "Creating k3d cluster: $CLUSTER_NAME"
k3d cluster create "$CLUSTER_NAME" \
  --agents 2 \
  --k3s-arg "--disable=traefik@server:0" \
  --port "80:80@loadbalancer" \
  --port "443:443@loadbalancer"

# Wait for the cluster to be ready
echo "Waiting for cluster to be ready..."
sleep 10
kubectl wait --for=condition=Ready nodes --all --timeout=60s

# Build and import the loadbalancer image
echo "Building loadbalancer image..."
docker build -t loadbalancer:latest .

echo "Importing loadbalancer image to k3d..."
k3d image import loadbalancer:latest -c "$CLUSTER_NAME"

# Apply RBAC configurations
echo "Applying RBAC configurations..."
kubectl apply -f k3d/rbac.yaml

# Apply ConfigMap
echo "Applying ConfigMap..."
kubectl apply -f k3d/configmap.yaml

# Deploy loadbalancer
echo "Deploying loadbalancer..."
kubectl apply -f k3d/daemonset.yaml

# Wait for the loadbalancer to be ready
echo "Waiting for loadbalancer to be ready..."
kubectl rollout status daemonset/loadbalancer -n kube-system --timeout=120s

# Deploy sample service
echo "Deploying sample service..."
kubectl apply -f k3d/sample-service.yaml

echo "Setup complete! Your k3d cluster with loadbalancer is ready."
echo ""
echo "Current loadbalancer status:"
kubectl get pods -n kube-system -l app=loadbalancer
echo ""
echo "Sample service status:"
kubectl get svc echo-server -o wide
echo ""

# Wait a bit for the IP to be assigned
echo "Waiting for the sample service to get an external IP..."
sleep 10
kubectl get svc echo-server -o wide

echo ""
echo "To check loadbalancer logs:"
echo "  kubectl logs -n kube-system -l app=loadbalancer"
echo ""
echo "To access the sample service (once it has an external IP):"
echo "  curl http://<EXTERNAL-IP>"
echo ""
echo "To update loadbalancer configuration:"
echo "  kubectl edit configmap loadbalancer-config -n kube-system"