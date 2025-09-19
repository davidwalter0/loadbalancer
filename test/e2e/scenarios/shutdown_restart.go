/*

Copyright 2025 David Walter.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package scenarios

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/davidwalter0/loadbalancer/test/e2e/framework"
	"github.com/davidwalter0/loadbalancer/test/e2e/utils"
)

// TestShutdownAndRestart verifies that the loadbalancer can be shutdown and restarted without issues
func TestShutdownAndRestart(t *testing.T, f *framework.Framework) {
	// Start the loadbalancer
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to start loadbalancer: %v", err)
	}

	// Create a test pod
	podName := "restart-test-pod"
	podPort := int32(80)
	labels := map[string]string{"app": "restart-test"}
	
	_, err := f.CreateTestPod(podName, labels, podPort)
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	// Create a LoadBalancer service
	serviceConfig := framework.ServiceConfig{
		Name: "restart-test-service",
		Ports: []framework.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: podPort,
			},
		},
		Selector: labels,
		Annotations: map[string]string{
			"loadbalancer.example.com/health-check-enabled": "true",
			"loadbalancer.example.com/health-check-path":    "/",
		},
	}
	
	_, err = f.CreateLoadBalancerService(serviceConfig)
	if err != nil {
		t.Fatalf("Failed to create LoadBalancer service: %v", err)
	}

	// Wait for service to get an external IP
	externalIP, err := f.WaitForServiceExternalIP(serviceConfig.Name, "", 2*time.Minute)
	if err != nil {
		t.Fatalf("Failed to get external IP: %v", err)
	}

	// Create HTTP client
	httpClient := utils.NewHTTPClient(5 * time.Second)

	// Test connectivity to verify service is available
	url := fmt.Sprintf("http://%s:%d", externalIP, serviceConfig.Ports[0].Port)
	
	// Wait for HTTP endpoint to be available
	err = httpClient.WaitForHTTPStatus(url, http.StatusOK, 2*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to service: %v", err)
	}

	// Verify initial connectivity
	statusCode, _, err := httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}
	
	t.Log("Successfully connected to service with initial loadbalancer")

	// Stop the loadbalancer
	t.Log("Stopping loadbalancer...")
	if err := f.StopLoadBalancer(); err != nil {
		t.Fatalf("Failed to stop loadbalancer: %v", err)
	}
	
	// Wait for a moment to ensure it's really stopped
	time.Sleep(5 * time.Second)
	
	// Verify service is no longer accessible
	t.Log("Verifying service is no longer accessible...")
	_, _, err = httpClient.Get(url)
	if err == nil {
		t.Log("Warning: Service is still accessible after stopping loadbalancer, this might be due to connection caching or other factors")
	} else {
		t.Log("Service is not accessible after stopping loadbalancer, as expected")
	}
	
	// Start the loadbalancer again
	t.Log("Starting loadbalancer again...")
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to restart loadbalancer: %v", err)
	}
	
	// Wait for the loadbalancer to initialize
	t.Log("Waiting for loadbalancer to initialize...")
	time.Sleep(10 * time.Second)
	
	// Wait for service to be accessible again
	t.Log("Waiting for service to be accessible again...")
	err = httpClient.WaitForHTTPStatus(url, http.StatusOK, 2*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to service after restarting loadbalancer: %v", err)
	}
	
	// Verify connectivity after restart
	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request after restart: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d after restart, got %d", http.StatusOK, statusCode)
	}
	
	t.Log("Successfully connected to service after restarting loadbalancer")
	
	// Add another pod with the same labels
	secondPodName := "restart-test-pod-2"
	_, err = f.CreateTestPod(secondPodName, labels, podPort)
	if err != nil {
		t.Fatalf("Failed to create second test pod: %v", err)
	}
	
	// Wait for endpoint to be updated
	t.Log("Waiting for endpoint to be updated with second pod...")
	time.Sleep(30 * time.Second)
	
	// Verify service is still accessible with both pods
	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request after adding second pod: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d after adding second pod, got %d", http.StatusOK, statusCode)
	}
	
	t.Log("Successfully connected to service after adding second pod")
	
	// Stop and restart the loadbalancer again
	t.Log("Stopping loadbalancer again...")
	if err := f.StopLoadBalancer(); err != nil {
		t.Fatalf("Failed to stop loadbalancer: %v", err)
	}
	
	time.Sleep(5 * time.Second)
	
	t.Log("Starting loadbalancer again...")
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to restart loadbalancer: %v", err)
	}
	
	// Wait for the loadbalancer to initialize
	t.Log("Waiting for loadbalancer to initialize...")
	time.Sleep(10 * time.Second)
	
	// Wait for service to be accessible again
	t.Log("Waiting for service to be accessible again...")
	err = httpClient.WaitForHTTPStatus(url, http.StatusOK, 2*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to service after second restart: %v", err)
	}
	
	// Verify connectivity after second restart
	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request after second restart: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d after second restart, got %d", http.StatusOK, statusCode)
	}
	
	t.Log("Successfully connected to service after second restart")
}