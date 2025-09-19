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

package health

import (
	"net"
	"net/http"
	"testing"
	"time"
)

func TestHealthCheckerTCP(t *testing.T) {
	// Create a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	
	serverAddr := listener.Addr().String()
	
	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	
	defer listener.Close()
	
	// Create health checker with custom settings for faster test
	settings := &Settings{
		Enabled:            true,
		Type:               TCPCheck,
		Interval:           time.Millisecond * 100,
		Timeout:            time.Millisecond * 500,
		HealthyThreshold:   1,
		UnhealthyThreshold: 1,
	}
	
	checker := NewChecker(settings)
	
	// Add the test server endpoint
	checker.AddEndpoint(serverAddr)
	
	// Start health checks
	checker.StartChecking()
	
	// Wait for checks to run
	time.Sleep(time.Millisecond * 200)
	
	// Check status
	status := checker.GetStatus(serverAddr)
	if status != Healthy {
		t.Errorf("Expected endpoint to be Healthy, got %s", status)
	}
	
	// Close the listener to simulate failure
	listener.Close()
	
	// Wait for checks to run
	time.Sleep(time.Millisecond * 600)
	
	// Check status
	status = checker.GetStatus(serverAddr)
	if status != Unhealthy {
		t.Errorf("Expected endpoint to be Unhealthy after server closed, got %s", status)
	}
	
	// Stop the checker
	checker.StopChecking()
}

func TestHealthCheckerHTTP(t *testing.T) {
	// Create a test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	server := &http.Server{
		Handler: mux,
	}
	
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	
	serverAddr := listener.Addr().String()
	
	go server.Serve(listener)
	defer server.Close()
	
	// Create health checker with custom settings for faster test
	settings := &Settings{
		Enabled:             true,
		Type:                HTTPCheck,
		Interval:            time.Millisecond * 100,
		Timeout:             time.Millisecond * 500,
		HealthyThreshold:    1,
		UnhealthyThreshold:  1,
		HTTPPath:            "/health",
		HTTPStatusCodeMatch: 200,
	}
	
	checker := NewChecker(settings)
	
	// Add the test server endpoint
	checker.AddEndpoint(serverAddr)
	
	// Start health checks
	checker.StartChecking()
	
	// Wait for checks to run
	time.Sleep(time.Millisecond * 200)
	
	// Check status
	status := checker.GetStatus(serverAddr)
	if status != Healthy {
		t.Errorf("Expected endpoint to be Healthy, got %s", status)
	}
	
	// Stop the checker
	checker.StopChecking()
}

func TestGetHealthyEndpoints(t *testing.T) {
	// Create health checker
	settings := &Settings{
		Enabled:            true,
		Type:               TCPCheck,
		Interval:           time.Second,
		Timeout:            time.Millisecond * 500,
		HealthyThreshold:   1,
		UnhealthyThreshold: 1,
	}
	
	checker := NewChecker(settings)
	
	// Add endpoints with various statuses
	checker.AddEndpoint("endpoint1:80")
	checker.AddEndpoint("endpoint2:80")
	checker.AddEndpoint("endpoint3:80")
	
	// Manually set statuses for testing
	checker.Endpoints["endpoint1:80"].Status = Healthy
	checker.Endpoints["endpoint2:80"].Status = Unhealthy
	checker.Endpoints["endpoint3:80"].Status = Healthy
	
	// Get healthy endpoints
	healthy := checker.GetHealthyEndpoints()
	
	// Check results
	if len(healthy) != 2 {
		t.Errorf("Expected 2 healthy endpoints, got %d", len(healthy))
	}
	
	// Check specific endpoints
	foundEndpoint1 := false
	foundEndpoint3 := false
	
	for _, endpoint := range healthy {
		if endpoint == "endpoint1:80" {
			foundEndpoint1 = true
		} else if endpoint == "endpoint3:80" {
			foundEndpoint3 = true
		}
	}
	
	if !foundEndpoint1 {
		t.Error("Expected endpoint1:80 to be in healthy endpoints")
	}
	
	if !foundEndpoint3 {
		t.Error("Expected endpoint3:80 to be in healthy endpoints")
	}
}

func TestHealthCheckerDisabled(t *testing.T) {
	// Create disabled health checker
	settings := &Settings{
		Enabled: false,
	}
	
	checker := NewChecker(settings)
	
	// Add an endpoint
	checker.AddEndpoint("endpoint1:80")
	
	// Check that it's considered healthy when checks are disabled
	if !checker.IsHealthy("endpoint1:80") {
		t.Error("Expected endpoint to be considered healthy when checks are disabled")
	}
	
	// Get healthy endpoints
	healthy := checker.GetHealthyEndpoints()
	
	// All endpoints should be included
	if len(healthy) != 1 {
		t.Errorf("Expected all endpoints to be included when disabled, got %d", len(healthy))
	}
}