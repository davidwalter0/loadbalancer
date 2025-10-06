#!/usr/bin/env bash
#
# Helper script to run loadbalancer container on macOS (Darwin)
# with Rancher Desktop, Docker Desktop, or similar
#
# Usage:
#   ./scripts/run-darwin.sh [kubeconfig-path] [additional-args...]
#
# Examples:
#   ./scripts/run-darwin.sh ~/.kube/config.k3d-dev
#   ./scripts/run-darwin.sh ~/.kube/config.k3d-dev --debug
#   ./scripts/run-darwin.sh  # uses default ~/.kube/config

set -e

# Default values
KUBECONFIG_HOST="${1:-$HOME/.kube/config}"
KUBECONFIG_CONTAINER="/root/.kube/config"
IMAGE="${LOADBALANCER_IMAGE:-davidwalter0/loadbalancer:latest}"
CONTAINER_NAME="${LOADBALANCER_CONTAINER_NAME:-loadbalancer-dev}"

# Shift to get additional args
shift || true
EXTRA_ARGS="$@"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}LoadBalancer Container Runner for macOS${NC}"
echo "=========================================="
echo ""

# Check if kubeconfig exists
if [ ! -f "$KUBECONFIG_HOST" ]; then
  echo -e "${RED}Error: Kubeconfig not found at: $KUBECONFIG_HOST${NC}"
  echo ""
  echo "Usage: $0 [kubeconfig-path] [additional-args...]"
  echo ""
  echo "Available kubeconfigs:"
  ls -1 ~/.kube/config* 2>/dev/null || echo "  (none found)"
  exit 1
fi

echo -e "${GREEN}Configuration:${NC}"
echo "  Image: $IMAGE"
echo "  Container: $CONTAINER_NAME"
echo "  Kubeconfig (host): $KUBECONFIG_HOST"
echo "  Kubeconfig (container): $KUBECONFIG_CONTAINER"
echo "  Extra args: ${EXTRA_ARGS:-none}"
echo ""

# Check if image exists
if ! docker image inspect "$IMAGE" &>/dev/null; then
  echo -e "${YELLOW}Warning: Image $IMAGE not found locally${NC}"
  echo "Building image..."
  cd "$(dirname "$0")/.."
  make image
  echo ""
fi

# Stop and remove existing container if it exists
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo -e "${YELLOW}Stopping and removing existing container: $CONTAINER_NAME${NC}"
  docker stop "$CONTAINER_NAME" 2>/dev/null || true
  docker rm "$CONTAINER_NAME" 2>/dev/null || true
  echo ""
fi

# Prepare kubeconfig directory in container
KUBECONFIG_DIR=$(dirname "$KUBECONFIG_CONTAINER")

echo -e "${GREEN}Starting loadbalancer container...${NC}"
echo ""

# Run container with appropriate settings for macOS
# Note: --privileged is needed on macOS Docker because setcap doesn't work
# and we need network management capabilities
docker run -d \
  --name "$CONTAINER_NAME" \
  --network host \
  --privileged \
  -v "$KUBECONFIG_HOST:$KUBECONFIG_CONTAINER:ro" \
  -e KUBECONFIG="$KUBECONFIG_CONTAINER" \
  -e DEBUG="${DEBUG:-false}" \
  -e KUBERNETES="${KUBERNETES:-true}" \
  "$IMAGE" \
  $EXTRA_ARGS

echo -e "${GREEN}Container started successfully!${NC}"
echo ""
echo "Container ID: $(docker ps -qf name=$CONTAINER_NAME)"
echo ""
echo -e "${GREEN}Commands:${NC}"
echo "  View logs:    docker logs -f $CONTAINER_NAME"
echo "  Stop:         docker stop $CONTAINER_NAME"
echo "  Remove:       docker rm $CONTAINER_NAME"
echo "  Shell:        docker exec -it $CONTAINER_NAME bash"
echo "  Restart:      docker restart $CONTAINER_NAME"
echo ""

# Wait a moment and check if container is still running
sleep 2
if ! docker ps -qf name=$CONTAINER_NAME | grep -q .; then
  echo -e "${RED}Error: Container exited unexpectedly${NC}"
  echo ""
  echo "Last logs:"
  docker logs "$CONTAINER_NAME"
  exit 1
fi

echo -e "${GREEN}Showing initial logs:${NC}"
docker logs "$CONTAINER_NAME"
echo ""
echo -e "${GREEN}Follow logs with: docker logs -f $CONTAINER_NAME${NC}"
