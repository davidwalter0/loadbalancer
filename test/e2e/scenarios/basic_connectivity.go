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

// TestBasicConnectivity verifies that the loadbalancer can forward traffic to a backend service
func TestBasicConnectivity(t *testing.T, f *framework.Framework) {
	// Start the loadbalancer
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to start loadbalancer: %v", err)
	}

	// Create a test pod
	podName := "nginx-test"
	podPort := int32(80)
	labels := map[string]string{"app": "nginx-test"}
	
	_, err := f.CreateTestPod(podName, labels, podPort)
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	// Create a LoadBalancer service
	serviceConfig := framework.ServiceConfig{
		Name: "nginx-service",
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

	// Test connectivity to the service via external IP
	url := fmt.Sprintf("http://%s:%d", externalIP, serviceConfig.Ports[0].Port)
	
	// Wait for HTTP endpoint to be available
	err = httpClient.WaitForHTTPStatus(url, http.StatusOK, 2*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to service: %v", err)
	}

	// Make a request to verify content
	statusCode, body, err := httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	t.Logf("Successfully connected to service at %s", url)
	t.Logf("Response status: %d, body length: %d", statusCode, len(body))
}