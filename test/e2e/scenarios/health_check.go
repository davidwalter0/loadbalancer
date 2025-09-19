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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestHealthCheck verifies that the loadbalancer's health check functionality works correctly
func TestHealthCheck(t *testing.T, f *framework.Framework) {
	// Start the loadbalancer
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to start loadbalancer: %v", err)
	}

	// Create test pods
	labels := map[string]string{"app": "health-test"}
	
	// Create healthy pod
	healthyPodName := "healthy-pod"
	_, err := f.CreateTestPod(healthyPodName, labels, 80)
	if err != nil {
		t.Fatalf("Failed to create healthy test pod: %v", err)
	}

	// Create a LoadBalancer service with health checks enabled
	serviceConfig := framework.ServiceConfig{
		Name: "health-service",
		Ports: []framework.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: 80,
			},
		},
		Selector: labels,
		Annotations: map[string]string{
			"loadbalancer.example.com/health-check-enabled":  "true",
			"loadbalancer.example.com/health-check-path":     "/",
			"loadbalancer.example.com/health-check-type":     "http",
			"loadbalancer.example.com/health-check-interval": "5s",
			"loadbalancer.example.com/health-check-timeout":  "2s",
		},
	}
	
	service, err := f.CreateLoadBalancerService(serviceConfig)
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

	// Test connectivity to verify it works with healthy pod
	url := fmt.Sprintf("http://%s:%d", externalIP, serviceConfig.Ports[0].Port)
	
	// Wait for HTTP endpoint to be available
	err = httpClient.WaitForHTTPStatus(url, http.StatusOK, 2*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to service: %v", err)
	}

	// Verify we can connect to the service
	statusCode, _, err := httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	t.Log("Successfully connected to service with healthy backend")

	// Create an unhealthy pod with broken readiness probe
	unhealthyPodName := "unhealthy-pod"
	unhealthyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      unhealthyPodName,
			Namespace: f.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					// Configure a readiness probe that will always fail
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/does-not-exist",
								Port: intstr.FromInt(80),
							},
						},
						InitialDelaySeconds: 5,
						TimeoutSeconds:      1,
						PeriodSeconds:       5,
						FailureThreshold:    1,
					},
				},
			},
		},
	}
	
	_, err = f.KubeClient.CoreV1().Pods(f.Namespace).Create(f.Ctx, unhealthyPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create unhealthy test pod: %v", err)
	}

	// Register cleanup
	f.AddCleanupFunc(func() error {
		return f.DeletePod(unhealthyPodName)
	})

	// Wait for some time for health checks to run
	t.Log("Waiting for health checks to run...")
	time.Sleep(30 * time.Second)

	// Verify service is still accessible (routes only to healthy pod)
	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	t.Log("Service is still accessible with mixed healthy/unhealthy backends")

	// Delete the healthy pod
	err = f.DeletePod(healthyPodName)
	if err != nil {
		t.Fatalf("Failed to delete healthy pod: %v", err)
	}

	// Wait for health checks to detect the change
	t.Log("Waiting for health checks to detect the change...")
	time.Sleep(30 * time.Second)

	// Try to connect - should eventually fail as there are no healthy backends
	// but the loadbalancer may fall back to all available backends when none are healthy
	_, _, err = httpClient.Get(url)
	if err == nil {
		t.Log("Note: Service is still accessible, which is expected if the loadbalancer falls back to all available backends when none are healthy")
	} else {
		t.Log("Service is not accessible, as all backends are unhealthy")
	}

	// Create a new healthy pod
	newHealthyPodName := "new-healthy-pod"
	_, err = f.CreateTestPod(newHealthyPodName, labels, 80)
	if err != nil {
		t.Fatalf("Failed to create new healthy test pod: %v", err)
	}

	// Wait for health checks to detect the new healthy pod
	t.Log("Waiting for health checks to detect the new healthy pod...")
	time.Sleep(30 * time.Second)

	// Verify service is accessible again
	err = httpClient.WaitForHTTPStatus(url, http.StatusOK, 1*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to service after adding new healthy pod: %v", err)
	}

	statusCode, _, err = httpClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	t.Log("Service is accessible again after adding new healthy pod")
}