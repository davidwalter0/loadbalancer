#!/bin/bash
set -e

# Help function
show_help() {
  echo "Usage: $0 [OPTIONS]"
  echo ""
  echo "Run the loadbalancer container with host networking integrated with k3d"
  echo ""
  echo "Options:"
  echo "  -c, --cluster NAME     Specify k3d cluster name (default: loadbalancer-cluster)"
  echo "  -i, --interface NAME   Specify network interface to use (default: auto-detect)"
  echo "  -r, --registry URL     Use a specific registry (e.g., kdc1:5000)"
  echo "  -d, --debug            Enable debug mode"
  echo "  -h, --help             Show this help message"
  echo ""
}

# Parse arguments
CLUSTER_NAME="loadbalancer-cluster"
INTERFACE=""
DEBUG="false"
REGISTRY=""
IMAGE_NAME="loadbalancer:latest"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -c|--cluster)
      CLUSTER_NAME="$2"
      shift 2
      ;;
    -i|--interface)
      INTERFACE="$2"
      shift 2
      ;;
    -r|--registry)
      REGISTRY="$2"
      shift 2
      ;;
    -d|--debug)
      DEBUG="true"
      shift
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      show_help
      exit 1
      ;;
  esac
done

# Set image name based on registry
if [ -n "$REGISTRY" ]; then
  IMAGE_NAME="${REGISTRY}/loadbalancer:latest"
  echo "Using registry: $REGISTRY"
fi

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

# Check if the cluster exists
if ! k3d cluster list | grep -q "$CLUSTER_NAME"; then
  echo "Cluster $CLUSTER_NAME not found. Creating it..."
  k3d cluster create "$CLUSTER_NAME" \
    --agents 2 \
    --k3s-arg "--disable=traefik@server:0" \
    --port "80:80@loadbalancer" \
    --port "443:443@loadbalancer"
else
  echo "Using existing cluster: $CLUSTER_NAME"
fi

# Set kubectl context to the k3d cluster
k3d kubeconfig get "$CLUSTER_NAME" > /tmp/kubeconfig-"$CLUSTER_NAME"
export KUBECONFIG=/tmp/kubeconfig-"$CLUSTER_NAME"

# Apply RBAC resources
echo "Applying RBAC resources..."
kubectl apply -f k3d/rbac.yaml

# Auto-detect interface if not specified
if [ -z "$INTERFACE" ]; then
  echo "Auto-detecting network interface..."
  # Get the first interface that's up and has an IPv4 address (excluding loopback)
  INTERFACE=$(ip -o -4 addr show | grep -v ' lo ' | head -n1 | awk '{print $2}')
  if [ -z "$INTERFACE" ]; then
    echo "Error: Could not auto-detect a valid network interface"
    exit 1
  fi
  echo "Auto-detected interface: $INTERFACE"
fi

# Get interface information
IP_ADDR=$(ip -o -4 addr show dev "$INTERFACE" | awk '{print $4}' | cut -d'/' -f1)
if [ -z "$IP_ADDR" ]; then
  echo "Error: Interface $INTERFACE does not have an IPv4 address"
  exit 1
fi

echo "Using interface: $INTERFACE with IP: $IP_ADDR"

# Create ConfigMap for loadbalancer configuration
echo "Creating ConfigMap..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: loadbalancer-config
  namespace: kube-system
data:
  linkdevice: "$INTERFACE"
  debug: "$DEBUG"
  kubernetes: "true"
EOF

# Build the Docker image if not using registry
if [ -z "$REGISTRY" ]; then
  echo "Building loadbalancer Docker image..."
  docker build -t $IMAGE_NAME .
else
  echo "Using image from registry: $IMAGE_NAME"
  # Pull the image from the registry
  docker pull $IMAGE_NAME
fi

# Run the container with host networking
echo "Starting loadbalancer container..."
docker run \
  --name loadbalancer-k3d \
  --rm \
  --network host \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --volume /var/log:/var/log:ro \
  -e DEBUG="$DEBUG" \
  -e KUBERNETES="true" \
  -e LINK_DEVICE="$INTERFACE" \
  -e KUBECONFIG="/kubeconfig" \
  -v /tmp/kubeconfig-"$CLUSTER_NAME":/kubeconfig \
  $IMAGE_NAME

echo "Container stopped"