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
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidwalter0/loadbalancer/test/e2e/framework"
	"github.com/davidwalter0/loadbalancer/test/e2e/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestLoadBalancing verifies that the loadbalancer distributes traffic across multiple backends
func TestLoadBalancing(t *testing.T, f *framework.Framework) {
	// Start the loadbalancer
	if err := f.StartLoadBalancer(); err != nil {
		t.Fatalf("Failed to start loadbalancer: %v", err)
	}

	// Create multiple backends with unique identifiers
	numBackends := 3
	labels := map[string]string{"app": "lb-test"}
	
	// Create pods with custom index pages
	for i := 1; i <= numBackends; i++ {
		podName := fmt.Sprintf("lb-test-pod-%d", i)
		
		// Create a ConfigMap with custom index.html
		indexHTML := fmt.Sprintf("<html><body><h1>Backend %d</h1></body></html>", i)
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("index-html-%d", i),
				Namespace: f.Namespace,
			},
			Data: map[string]string{
				"index.html": indexHTML,
			},
		}
		
		_, err := f.KubeClient.CoreV1().ConfigMaps(f.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create ConfigMap: %v", err)
		}
		
		// Register cleanup
		cmName := configMap.Name
		f.AddCleanupFunc(func() error {
			return f.KubeClient.CoreV1().ConfigMaps(f.Namespace).Delete(context.TODO(), cmName, metav1.DeleteOptions{})
		})
		
		// Create pod with the custom index.html mounted
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
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
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "index-html",
								MountPath: "/usr/share/nginx/html/",
							},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: metav1.FromInt(80),
								},
							},
							InitialDelaySeconds: 5,
							TimeoutSeconds:      1,
							PeriodSeconds:       5,
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "index-html",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: configMap.Name,
								},
							},
						},
					},
				},
			},
		}
		
		_, err = f.KubeClient.CoreV1().Pods(f.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}
		
		// Register cleanup
		podNameCopy := podName
		f.AddCleanupFunc(func() error {
			return f.DeletePod(podNameCopy)
		})
		
		// Wait for pod to be ready
		err = f.WaitForPodReady(podName, 2*time.Minute)
		if err != nil {
			t.Fatalf("Pod failed to become ready: %v", err)
		}
	}

	// Create a LoadBalancer service
	serviceConfig := framework.ServiceConfig{
		Name: "lb-test-service",
		Ports: []framework.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: 80,
			},
		},
		Selector: labels,
	}
	
	_, err := f.CreateLoadBalancerService(serviceConfig)
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

	// Make multiple requests and count responses from each backend
	numRequests := 100
	backendHits := make(map[int]int)
	var mu sync.Mutex

	// Make requests concurrently to simulate load
	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			statusCode, body, err := httpClient.Get(url)
			if err != nil {
				t.Logf("Failed to make HTTP request: %v", err)
				return
			}
			
			if statusCode != http.StatusOK {
				t.Logf("Expected status code %d, got %d", http.StatusOK, statusCode)
				return
			}
			
			// Extract backend number from response
			bodyStr := string(body)
			if strings.Contains(bodyStr, "Backend") {
				// Parse backend number
				startIndex := strings.Index(bodyStr, "Backend") + 8
				endIndex := strings.Index(bodyStr[startIndex:], "<")
				if endIndex == -1 {
					endIndex = len(bodyStr) - startIndex
				}
				
				backendStr := strings.TrimSpace(bodyStr[startIndex : startIndex+endIndex])
				backendNum, err := strconv.Atoi(backendStr)
				if err == nil {
					mu.Lock()
					backendHits[backendNum]++
					mu.Unlock()
				}
			}
		}()
	}

	// Wait for all requests to complete
	wg.Wait()

	// Log distribution of backend hits
	t.Logf("Distribution of backend hits across %d requests:", numRequests)
	for i := 1; i <= numBackends; i++ {
		hits := backendHits[i]
		percentage := float64(hits) / float64(numRequests) * 100
		t.Logf("Backend %d: %d hits (%.2f%%)", i, hits, percentage)
		
		// Verify each backend received some traffic
		if hits == 0 {
			t.Errorf("Backend %d received no traffic", i)
		}
	}

	// Calculate standard deviation to check for distribution quality
	mean := float64(numRequests) / float64(numBackends)
	var sumSquaredDiff float64
	for i := 1; i <= numBackends; i++ {
		diff := float64(backendHits[i]) - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := sumSquaredDiff / float64(numBackends)
	
	// Log distribution quality metrics
	t.Logf("Mean hits per backend: %.2f", mean)
	t.Logf("Variance: %.2f", stdDev)
	
	// Check if the distribution is reasonably balanced
	// This is a simple check - in production you might want more sophisticated analysis
	maxHits := 0
	minHits := numRequests
	for i := 1; i <= numBackends; i++ {
		if backendHits[i] > maxHits {
			maxHits = backendHits[i]
		}
		if backendHits[i] < minHits {
			minHits = backendHits[i]
		}
	}
	
	// Check that no backend gets more than 50% more than the expected average
	maxAllowed := int(mean * 1.5)
	if maxHits > maxAllowed {
		t.Errorf("Load distribution is unbalanced. Max hits: %d, expected max: %d", maxHits, maxAllowed)
	}
}