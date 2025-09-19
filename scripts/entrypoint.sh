#!/bin/bash
set -e

# Function to detect available network interfaces (informational only)
detect_interfaces() {
  echo "Available network interfaces:"
  echo "----------------------------"

  # Get list of interfaces excluding loopback
  interfaces=$(ip -o link show | grep -v 'lo:' | awk -F': ' '{print $2}')

  i=1
  for iface in $interfaces; do
    ip_addr=$(ip -o -4 addr show dev $iface | awk '{print $4}' | cut -d'/' -f1)
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

  # Try to detect if we can connect to Kubernetes with kubectl
  if kubectl get nodes > /dev/null 2>&1; then
    echo "Kubernetes connection detected"
    # Set Kubernetes flag if not explicitly overridden to false
    if [ "${KUBERNETES}" != "false" ]; then
      export KUBERNETES=true
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

# Print the final command
echo "Running: loadbalancer ${ARGS[@]}"

# Execute loadbalancer with all arguments
# Use exec to ensure signals are properly passed to the loadbalancer process
exec loadbalancer "${ARGS[@]}"