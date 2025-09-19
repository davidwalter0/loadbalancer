# Deploying Loadbalancer with k3d

This guide explains how to deploy the loadbalancer to a k3d cluster, exposing it with host networking to allow it to manage IPs on the host network interfaces.

## Using as an Alternative to MetalLB

This loadbalancer provides a lightweight alternative to MetalLB for managing LoadBalancer services in k3d clusters. Unlike MetalLB, this solution:

1. Runs as a single container with host networking
2. Automatically detects network interfaces
3. Allocates IPs from the host's subnet by default
4. Does not require BGP protocol configuration

## Prerequisites

- Docker installed
- k3d installed (https://k3d.io/)
- kubectl installed

## Creating a k3d Cluster

1. Create a k3d cluster with port mappings for services:

```bash
k3d cluster create loadbalancer-cluster \
  --agents 2 \
  --k3s-arg "--disable=traefik@server:0" \
  --port "80:80@loadbalancer" \
  --port "443:443@loadbalancer"
```

2. Verify the cluster is running:

```bash
kubectl get nodes
```

## Running the Loadbalancer with Host Networking

There are two ways to run the loadbalancer with k3d:

### Option 1: Using the provided scripts (Recommended)

The repository includes scripts to run the loadbalancer with proper host networking:

```bash
# Run with auto-detected network interface
./scripts/run-with-k3d.sh

# Or specify a network interface
./scripts/run-with-k3d.sh --interface eth0

# Enable debug mode
./scripts/run-with-k3d.sh --debug

# Use a specific Docker registry
./scripts/run-with-k3d.sh --registry kdc1:5000
```

This script will:
1. Create a k3d cluster if it doesn't exist
2. Configure kubectl to use the cluster
3. Apply RBAC resources
4. Create a ConfigMap with the selected interface
5. Build the Docker image
6. Run the loadbalancer with host networking

### Option 2: Manual deployment

#### Building the Loadbalancer Image

1. Build the loadbalancer Docker image:

```bash
cd /path/to/loadbalancer
docker build -t loadbalancer:latest .
```

2. Import the image into k3d:

```bash
k3d image import loadbalancer:latest -c loadbalancer-cluster
```

#### Deploying the Loadbalancer (Kubernetes DaemonSet)

1. Create the necessary RBAC permissions:

```bash
kubectl apply -f k3d/rbac.yaml
```

2. Deploy the loadbalancer as a privileged DaemonSet:

```bash
kubectl apply -f k3d/daemonset.yaml
```

#### Running as a Docker Container with Host Networking

Alternatively, you can run the loadbalancer directly as a Docker container with host networking:

```bash
# Run with auto-detected network interface
./scripts/run-docker.sh

# Or specify a network interface
./scripts/run-docker.sh --interface eth0

# Use a specific Docker registry
./scripts/run-docker.sh --registry kdc1:5000
```

## Verifying the Deployment

1. Check if the pods are running:

```bash
kubectl get pods -n kube-system -l app=loadbalancer
```

2. Check the logs:

```bash
kubectl logs -n kube-system -l app=loadbalancer
```

## Testing with a Sample Service

1. Deploy a sample service:

```bash
kubectl apply -f k3d/sample-service.yaml
```

2. The service should now be accessible through the loadbalancer's IP.

## Troubleshooting

- If the loadbalancer can't manage network interfaces, ensure the container has the necessary privileges (NET_ADMIN capability).
- If the pod can't be scheduled, check node taints and pod tolerations.
- If services aren't accessible, verify that the loadbalancer is correctly watching for service changes.

## Configuring LoadBalancer Services

When deploying a service with `type: LoadBalancer`, the loadbalancer will automatically allocate an IP address from the subnet of the configured network interface.

Example service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
  # Optional: specify a fixed IP
  # loadBalancerIP: 192.168.1.100
```

## Avoiding Conflicts with MetalLB

If you have MetalLB installed in the same cluster:

1. **Uninstall MetalLB first** - Running both can cause conflicts as they both try to assign IPs to the same services
2. If you need both:
   - Configure this loadbalancer with a specific CIDR using the ConfigMap
   - Configure MetalLB to use a non-overlapping IP range
   - Use service annotations to control which controller handles which service

## Auto-detection of Network Interfaces

The loadbalancer will auto-detect a suitable network interface when:

1. The `linkdevice` field in the ConfigMap is empty (default)
2. The `LINK_DEVICE` environment variable is not set

When auto-detecting, it selects the first network interface that:
- Is up
- Is not a loopback
- Has an IPv4 address

To use a specific interface, set the `linkdevice` field in the ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: loadbalancer-config
  namespace: kube-system
data:
  linkdevice: "eth0"  # Specify your desired interface
```