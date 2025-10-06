#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LB_CONTAINER="loadbalancer"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}E2E Test: IP Conflict & Port Allocation${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to wait for service to be processed
wait_for_service() {
    local service_name=$1
    local timeout=10
    echo -e "${YELLOW}Waiting for service $service_name to be processed...${NC}"
    for i in $(seq 1 $timeout); do
        if kubectl get svc $service_name &>/dev/null; then
            sleep 2
            return 0
        fi
        sleep 1
    done
    return 1
}

# Function to show loadbalancer logs for a service
show_lb_logs() {
    local service_name=$1
    echo -e "${BLUE}--- Load Balancer Logs for $service_name ---${NC}"
    docker logs $LB_CONTAINER 2>&1 | grep -E "$service_name.*(LoadBalancerIP|allocation|Allocated|Reusing|Successfully|FAILED)" || echo "No logs found"
    echo ""
}

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up test resources...${NC}"
    kubectl delete -f $TEST_DIR/service1-port80.yaml --ignore-not-found=true 2>/dev/null || true
    kubectl delete -f $TEST_DIR/service2-port80-conflict.yaml --ignore-not-found=true 2>/dev/null || true
    kubectl delete -f $TEST_DIR/service3-port80-dynamic.yaml --ignore-not-found=true 2>/dev/null || true
    sleep 2
}

# Cleanup before starting
cleanup

echo -e "${GREEN}=== Step 1: Deploy Service 1 (port 80, dynamic IP) ===${NC}"
kubectl apply -f $TEST_DIR/service1-port80.yaml
wait_for_service web-service-1
show_lb_logs web-service-1

SERVICE1_IP=$(kubectl get svc web-service-1 -o jsonpath='{.spec.externalIPs[0]}' 2>/dev/null || echo "none")
echo -e "${GREEN}Service 1 External IP: ${SERVICE1_IP}${NC}"
echo ""

echo -e "${GREEN}=== Step 2: Deploy Service 2 (port 80, explicit IP=${SERVICE1_IP} - should CONFLICT) ===${NC}"
# Update the manifest with the actual IP from service1
sed "s/loadBalancerIP: .*/loadBalancerIP: ${SERVICE1_IP}/" $TEST_DIR/service2-port80-conflict.yaml | kubectl apply -f -
wait_for_service web-service-2
show_lb_logs web-service-2

SERVICE2_IP=$(kubectl get svc web-service-2 -o jsonpath='{.spec.externalIPs[0]}' 2>/dev/null || echo "none")
echo -e "${RED}Service 2 External IP: ${SERVICE2_IP} (should be empty/none due to conflict)${NC}"
echo ""

echo -e "${GREEN}=== Step 3: Deploy Service 3 (port 80, dynamic IP - should get next IP) ===${NC}"
kubectl apply -f $TEST_DIR/service3-port80-dynamic.yaml
wait_for_service web-service-3
show_lb_logs web-service-3

SERVICE3_IP=$(kubectl get svc web-service-3 -o jsonpath='{.spec.externalIPs[0]}' 2>/dev/null || echo "none")
echo -e "${GREEN}Service 3 External IP: ${SERVICE3_IP}${NC}"
echo ""

echo -e "${BLUE}=== Summary ===${NC}"
kubectl get svc web-service-1 web-service-2 web-service-3
echo ""

echo -e "${BLUE}=== IP Addresses on Interface ===${NC}"
ip addr show | grep -E "inet 192.168.0.19" || echo "No IPs found"
echo ""

echo -e "${BLUE}=== Testing Services ===${NC}"
if [ "$SERVICE1_IP" != "none" ] && [ ! -z "$SERVICE1_IP" ]; then
    echo -e "${GREEN}Testing Service 1 at http://${SERVICE1_IP}:${NC}"
    curl -s --connect-timeout 2 http://${SERVICE1_IP} 2>&1 | head -1 || echo "Failed to connect"
    echo ""
fi

if [ "$SERVICE2_IP" != "none" ] && [ ! -z "$SERVICE2_IP" ]; then
    echo -e "${RED}Testing Service 2 at http://${SERVICE2_IP} (unexpected - should have failed):${NC}"
    curl -s --connect-timeout 2 http://${SERVICE2_IP} 2>&1 | head -1 || echo "Failed to connect (expected)"
    echo ""
else
    echo -e "${GREEN}Service 2 correctly has no IP (conflict detected) ✓${NC}"
    echo ""
fi

if [ "$SERVICE3_IP" != "none" ] && [ ! -z "$SERVICE3_IP" ]; then
    echo -e "${GREEN}Testing Service 3 at http://${SERVICE3_IP}:${NC}"
    curl -s --connect-timeout 2 http://${SERVICE3_IP} 2>&1 | head -1 || echo "Failed to connect"
    echo ""
fi

echo -e "${BLUE}=== Full Load Balancer Logs ===${NC}"
docker logs $LB_CONTAINER 2>&1 | grep -E "web-service"
echo ""

echo -e "${YELLOW}Test complete. Cleanup? (y/n)${NC}"
read -t 10 -n 1 answer || answer="y"
echo ""
if [ "$answer" = "y" ]; then
    cleanup
    echo -e "${GREEN}Cleanup complete.${NC}"
else
    echo -e "${YELLOW}Skipping cleanup. Run 'kubectl delete -f $TEST_DIR/*.yaml' to clean up manually.${NC}"
fi
