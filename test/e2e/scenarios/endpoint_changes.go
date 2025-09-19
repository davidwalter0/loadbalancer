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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestEndpointChanges verifies that the loadbalancer handles endpoint changes correctly
func TestEndpointChanges(t *testing.T, f *framework.Framework) {
	// Start the loadbalancer
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to start loadbalancer: %v", err)
	}

	// Create a test pod
	podName := "endpoint-test-pod"
	podPort := int32(80)
	labels := map[string]string{"app": "endpoint-test"}
	
	_, err := f.CreateTestPod(podName, labels, podPort)
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	// Create a LoadBalancer service
	serviceConfig := framework.ServiceConfig{
		Name: "endpoint-test-service",
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
	
	t.Log("Successfully connected to service with initial pod")

	// Add another pod with the same labels
	secondPodName := "endpoint-test-pod-2"
	_, err = f.CreateTestPod(secondPodName, labels, podPort)
	if err != nil {
		t.Fatalf("Failed to create second test pod: %v", err)
	}
	
	// Wait for endpoint to be updated
	t.Log("Waiting for endpoint to be updated with second pod...")
	time.Sleep(30 * time.Second)
	
	// Verify service is still accessible
	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request after adding second pod: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d after adding second pod, got %d", http.StatusOK, statusCode)
	}
	
	t.Log("Successfully connected to service after adding second pod")
	
	// Remove the first pod
	err = f.DeletePod(podName)
	if err != nil {
		t.Fatalf("Failed to delete first pod: %v", err)
	}
	
	// Wait for endpoint to be updated
	t.Log("Waiting for endpoint to be updated after removing first pod...")
	time.Sleep(30 * time.Second)
	
	// Verify service is still accessible
	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request after removing first pod: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d after removing first pod, got %d", http.StatusOK, statusCode)
	}
	
	t.Log("Successfully connected to service after removing first pod")
	
	// Change service selector to point to non-existent pods
	t.Log("Changing service selector to point to non-existent pods...")
	
	service, err := f.KubeClient.CoreV1().Services(f.Namespace).Get(f.Ctx, serviceConfig.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}
	
	service.Spec.Selector = map[string]string{"app": "non-existent"}
	
	_, err = f.KubeClient.CoreV1().Services(f.Namespace).Update(f.Ctx, service, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update service selector: %v", err)
	}
	
	// Wait for endpoint to be updated
	t.Log("Waiting for endpoint to be updated after changing selector...")
	time.Sleep(30 * time.Second)
	
	// The service might become inaccessible or return an error now
	_, _, err = httpClient.Get(url)
	if err != nil {
		t.Log("Service became inaccessible after removing all endpoints, as expected")
	} else {
		t.Log("Service is still accessible, which might be expected if the loadbalancer maintains stale endpoints")
	}
	
	// Change service selector back to include the second pod
	t.Log("Changing service selector back to include existing pods...")
	
	service, err = f.KubeClient.CoreV1().Services(f.Namespace).Get(f.Ctx, serviceConfig.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}
	
	service.Spec.Selector = labels
	
	_, err = f.KubeClient.CoreV1().Services(f.Namespace).Update(f.Ctx, service, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update service selector: %v", err)
	}
	
	// Wait for endpoint to be updated
	t.Log("Waiting for endpoint to be updated after restoring selector...")
	time.Sleep(30 * time.Second)
	
	// Verify service is accessible again
	err = httpClient.WaitForHTTPStatus(url, http.StatusOK, 1*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to service after restoring endpoints: %v", err)
	}
	
	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request after restoring endpoints: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d after restoring endpoints, got %d", http.StatusOK, statusCode)
	}
	
	t.Log("Successfully connected to service after restoring endpoints")
}