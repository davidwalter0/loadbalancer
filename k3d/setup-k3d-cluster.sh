#!/bin/bash
set -e

# Variables
CLUSTER_NAME="loadbalancer"
NETWORK_NAME="k3d-${CLUSTER_NAME}"
LB_CONTAINER_NAME="loadbalancer"

echo "Setting up k3d cluster with external loadbalancer..."

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

# Stop and remove any existing external loadbalancer container
if docker ps -a | grep -q "$LB_CONTAINER_NAME"; then
  echo "Removing existing external loadbalancer container..."
  docker rm -f "$LB_CONTAINER_NAME"
fi

# Create a new k3d cluster WITHOUT built-in load balancer
echo "Creating k3d cluster: $CLUSTER_NAME"
k3d cluster create "$CLUSTER_NAME" \
  --agents 2 \
  --k3s-arg "--disable=traefik@server:0" \
  --no-lb \
  --api-port 0.0.0.0:6443

# Wait for the cluster to be ready
echo "Waiting for cluster to be ready..."
sleep 10
kubectl wait --for=condition=Ready nodes --all --timeout=60s

# Build the loadbalancer image
echo "Building loadbalancer image..."
docker build -t loadbalancer:latest .

# Apply RBAC configurations
echo "Applying RBAC configurations..."
kubectl apply -f k3d/rbac.yaml

# Get the kubeconfig for the external loadbalancer
echo "Extracting kubeconfig for external loadbalancer..."
mkdir -p /tmp/k3d-lb
k3d kubeconfig get "$CLUSTER_NAME" > /tmp/k3d-lb/kubeconfig

# Start the external loadbalancer container
echo "Starting external loadbalancer container..."
docker run -d \
  --name "$LB_CONTAINER_NAME" \
  --network host \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --cap-add NET_BIND_SERVICE \
  -v /tmp/k3d-lb/kubeconfig:/app/kubeconfig:ro \
  -e KUBERNETES=true \
  -e KUBECONFIG=/app/kubeconfig \
  -e DEBUG=true \
  -e RESTRICTED_CIDR="${RESTRICTED_CIDR:-192.168.0.224/28}" \
  loadbalancer:latest \
  --tag-worker-nodes


echo "Waiting for loadbalancer to start..."
sleep 5

# Deploy sample service
echo "Deploying sample service..."
kubectl apply -f k3d/sample-service.yaml

echo "Setup complete! Your k3d cluster with external loadbalancer is ready."
echo ""
echo "K3d API Server: localhost:6443"
echo ""
echo "External loadbalancer container:"
docker ps --filter "name=$LB_CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
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
echo "  docker logs -f $LB_CONTAINER_NAME"
echo ""
echo "To access the sample service (once it has an external IP):"
echo "  curl http://<EXTERNAL-IP>"
echo ""
echo "To stop the loadbalancer:"
echo "  docker stop $LB_CONTAINER_NAME"
echo ""
echo "To restart the loadbalancer:"
echo "  docker start $LB_CONTAINER_NAME"
