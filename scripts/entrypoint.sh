#!/bin/bash
set -e

# Function to detect available network interfaces (informational only)
detect_interfaces() {
  echo "Available network interfaces:"
  echo "----------------------------"

  # Check if ip command exists
  if ! command -v ip &> /dev/null; then
    echo "Warning: 'ip' command not available, skipping interface detection"
    return
  fi

  # Get list of interfaces excluding loopback
  interfaces=$(ip -o link show 2>/dev/null | grep -v 'lo:' | awk -F': ' '{print $2}') || true

  if [ -z "$interfaces" ]; then
    echo "No network interfaces detected"
    return
  fi

  i=1
  for iface in $interfaces; do
    ip_addr=$(ip -o -4 addr show dev $iface 2>/dev/null | awk '{print $4}' | cut -d'/' -f1) || true
    if [ -n "$ip_addr" ]; then
      echo "$i) $iface - $ip_addr"
      i=$((i+1))
    fi
  done
}

# Show interfaces for informational purposes
detect_interfaces

# Check if running in Kubernetes environment
if [ -n "${KUBERNETES_SERVICE_HOST}" ]; then
  echo "Running inside Kubernetes environment"
  # Set Kubernetes flag if not explicitly set
  if [ -z "${KUBERNETES}" ]; then
    export KUBERNETES=true
  fi
else
  echo "Running outside Kubernetes environment"

  # Try to detect if we can connect to Kubernetes with kubectl (if available)
  if command -v kubectl &> /dev/null; then
    if kubectl get nodes > /dev/null 2>&1; then
      echo "Kubernetes connection detected"
      # Set Kubernetes flag if not explicitly overridden to false
      if [ "${KUBERNETES}" != "false" ]; then
        export KUBERNETES=true
      fi
    fi
  fi
fi

# Start loadbalancer with appropriate arguments
echo "Starting loadbalancer..."

# Build command line arguments
ARGS=()

# Check for kubeconfig in standard locations if not explicitly set
if [ -n "${KUBECONFIG}" ]; then
  echo "Using explicit kubeconfig: ${KUBECONFIG}"
  ARGS+=("--kubeconfig=${KUBECONFIG}")
else
  # Check common kubeconfig locations
  if [ -f "/root/.kube/config" ]; then
    echo "Found kubeconfig at /root/.kube/config"
    ARGS+=("--kubeconfig=/root/.kube/config")
  elif [ -f "$HOME/.kube/config" ]; then
    echo "Found kubeconfig at $HOME/.kube/config"
    ARGS+=("--kubeconfig=$HOME/.kube/config")
  elif [ -f "/var/run/secrets/kubernetes.io/serviceaccount/token" ]; then
    echo "Found in-cluster service account token"
    # Let loadbalancer handle in-cluster config automatically
  else
    echo "No kubeconfig found in standard locations"
  fi
fi

# Pass kubernetes flag if set
if [ "${KUBERNETES:-false}" = "true" ]; then
  ARGS+=("--kubernetes")
fi

# Pass debug flag if set
if [ "${DEBUG:-false}" = "true" ]; then
  ARGS+=("--debug")
fi

# Pass any user-provided arguments
ARGS+=("$@")

# Print final configuration
echo "Configuration:"
echo "- LINK_DEVICE: ${LINK_DEVICE:-auto-detect}"
echo "- KUBERNETES: ${KUBERNETES:-false}"
echo "- DEBUG: ${DEBUG:-false}"
# Print kubeconfig status
if [[ "${ARGS[@]}" =~ "--kubeconfig" ]]; then
  # Extract kubeconfig path from args
  for arg in "${ARGS[@]}"; do
    if [[ "$arg" == --kubeconfig=* ]]; then
      echo "- KUBECONFIG: ${arg#--kubeconfig=}"
      break
    fi
  done
else
  echo "- KUBECONFIG: auto-detect"
fi

# Detect architecture and select correct binary
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)
    BINARY_ARCH="amd64"
    ;;
  aarch64|arm64)
    BINARY_ARCH="arm64"
    ;;
  *)
    echo "Warning: Unknown architecture '$ARCH', defaulting to amd64"
    BINARY_ARCH="amd64"
    ;;
esac

echo "Detected architecture: $ARCH -> $BINARY_ARCH"

# Create symlinks for all utilities so they're in PATH
echo "Setting up architecture-specific binaries..."
for bin in /usr/local/bin/${BINARY_ARCH}/*; do
  if [ -f "$bin" ]; then
    BASENAME=$(basename "$bin")
    ln -sf "$bin" "/usr/local/bin/$BASENAME" 2>/dev/null || true
  fi
done

LOADBALANCER_BIN="/usr/local/bin/${BINARY_ARCH}/loadbalancer"

# Verify binary exists and is executable
if [ ! -x "$LOADBALANCER_BIN" ]; then
  echo "Error: Binary not found or not executable: $LOADBALANCER_BIN"
  echo "Available binaries:"
  ls -la /usr/local/bin/*/loadbalancer 2>/dev/null || echo "  (none found)"
  exit 1
fi

echo "Using loadbalancer binary: $LOADBALANCER_BIN"

# Print the final command
echo "Running: $LOADBALANCER_BIN ${ARGS[@]}"

# Execute loadbalancer with all arguments
# Use exec to ensure signals are properly passed to the loadbalancer process
exec "$LOADBALANCER_BIN" "${ARGS[@]}"