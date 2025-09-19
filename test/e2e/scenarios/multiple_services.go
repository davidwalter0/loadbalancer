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
	"sync"
	"testing"
	"time"

	"github.com/davidwalter0/loadbalancer/test/e2e/framework"
	"github.com/davidwalter0/loadbalancer/test/e2e/utils"
)

// TestMultipleServices verifies that the loadbalancer can handle multiple services simultaneously
func TestMultipleServices(t *testing.T, f *framework.Framework) {
	// Start the loadbalancer
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to start loadbalancer: %v", err)
	}

	// Number of services to create
	numServices := 3
	
	// Create multiple services and backends
	services := make([]string, numServices)
	externalIPs := make([]string, numServices)
	
	for i := 0; i < numServices; i++ {
		// Create a test pod
		podName := fmt.Sprintf("multi-service-pod-%d", i)
		podPort := int32(8080 + i)
		labels := map[string]string{fmt.Sprintf("app"): fmt.Sprintf("multi-service-%d", i)}
		
		_, err := f.CreateTestPod(podName, labels, podPort)
		if err != nil {
			t.Fatalf("Failed to create test pod %s: %v", podName, err)
		}

		// Create a LoadBalancer service
		serviceName := fmt.Sprintf("multi-service-%d", i)
		servicePort := int32(80 + i)
		
		serviceConfig := framework.ServiceConfig{
			Name: serviceName,
			Ports: []framework.ServicePort{
				{
					Name:       "http",
					Port:       servicePort,
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
			t.Fatalf("Failed to create LoadBalancer service %s: %v", serviceName, err)
		}
		
		services[i] = serviceName
	}
	
	// Wait for all services to get external IPs
	t.Log("Waiting for all services to get external IPs...")
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error
	
	for i, serviceName := range services {
		wg.Add(1)
		go func(index int, name string) {
			defer wg.Done()
			
			externalIP, err := f.WaitForServiceExternalIP(name, "", 2*time.Minute)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to get external IP for service %s: %w", name, err))
				mu.Unlock()
				return
			}
			
			mu.Lock()
			externalIPs[index] = externalIP
			mu.Unlock()
		}(i, serviceName)
	}
	
	wg.Wait()
	
	// Check if there were any errors
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("%v", err)
		}
		t.Fatalf("Failed to get external IPs for all services")
	}
	
	// Create HTTP client
	httpClient := utils.NewHTTPClient(5 * time.Second)
	
	// Test connectivity to all services simultaneously
	t.Log("Testing connectivity to all services...")
	
	var connectErrors []error
	for i, serviceName := range services {
		wg.Add(1)
		go func(index int, name string) {
			defer wg.Done()
			
			externalIP := externalIPs[index]
			servicePort := int32(80 + index)
			url := fmt.Sprintf("http://%s:%d", externalIP, servicePort)
			
			// Wait for HTTP endpoint to be available
			err := httpClient.WaitForHTTPStatus(url, http.StatusOK, 2*time.Minute, 5*time.Second)
			if err != nil {
				mu.Lock()
				connectErrors = append(connectErrors, fmt.Errorf("failed to connect to service %s: %w", name, err))
				mu.Unlock()
				return
			}
			
			// Make a request to verify content
			statusCode, _, err := httpClient.Get(url)
			if err != nil {
				mu.Lock()
				connectErrors = append(connectErrors, fmt.Errorf("failed to make HTTP request to service %s: %w", name, err))
				mu.Unlock()
				return
			}
			
			if statusCode != http.StatusOK {
				mu.Lock()
				connectErrors = append(connectErrors, fmt.Errorf("expected status code %d for service %s, got %d", http.StatusOK, name, statusCode))
				mu.Unlock()
				return
			}
			
			t.Logf("Successfully connected to service %s at %s", name, url)
		}(i, serviceName)
	}
	
	wg.Wait()
	
	// Check if there were any connection errors
	if len(connectErrors) > 0 {
		for _, err := range connectErrors {
			t.Errorf("%v", err)
		}
		t.Fatalf("Failed to connect to all services")
	}
	
	// Perform load test on all services simultaneously
	t.Log("Performing load test on all services...")
	
	numRequests := 20
	var loadTestErrors []error
	
	for i := 0; i < numRequests; i++ {
		wg.Add(numServices)
		
		for j := 0; j < numServices; j++ {
			go func(serviceIndex int) {
				defer wg.Done()
				
				externalIP := externalIPs[serviceIndex]
				servicePort := int32(80 + serviceIndex)
				url := fmt.Sprintf("http://%s:%d", externalIP, servicePort)
				
				statusCode, _, err := httpClient.Get(url)
				if err != nil {
					mu.Lock()
					loadTestErrors = append(loadTestErrors, fmt.Errorf("load test request failed for service %s: %w", services[serviceIndex], err))
					mu.Unlock()
					return
				}
				
				if statusCode != http.StatusOK {
					mu.Lock()
					loadTestErrors = append(loadTestErrors, fmt.Errorf("load test expected status %d for service %s, got %d", http.StatusOK, services[serviceIndex], statusCode))
					mu.Unlock()
				}
			}(j)
		}
	}
	
	wg.Wait()
	
	// Check if there were any load test errors
	if len(loadTestErrors) > 0 {
		for _, err := range loadTestErrors {
			t.Errorf("%v", err)
		}
		t.Fatalf("Load test failed for some services")
	}
	
	t.Log("Successfully tested all services simultaneously")
}