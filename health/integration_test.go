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
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
	"time"
)

// Integration test for the health checker with real network services
func TestHealthCheckerIntegration(t *testing.T) {
	// Skip test in CI environments
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Set up test TCP server
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create TCP test server: %v", err)
	}
	defer tcpListener.Close()
	
	tcpAddr := tcpListener.Addr().String()
	
	// Accept TCP connections in background
	go func() {
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	
	// Set up test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	httpServer := &http.Server{
		Handler: mux,
	}
	
	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create HTTP test server: %v", err)
	}
	defer httpListener.Close()
	
	httpAddr := httpListener.Addr().String()
	
	go httpServer.Serve(httpListener)
	defer httpServer.Close()
	
	// Create health checker
	settings := &Settings{
		Enabled:             true,
		Type:                TCPCheck,
		Interval:            time.Millisecond * 100,
		Timeout:             time.Millisecond * 500,
		HealthyThreshold:    1,
		UnhealthyThreshold:  1,
		HTTPPath:            "/health",
		HTTPStatusCodeMatch: 200,
	}
	
	checker := NewChecker(settings)
	checker.Debug = true
	
	// Test TCP health check
	t.Run("TCP Health Check", func(t *testing.T) {
		// Add TCP endpoint
		checker.AddEndpoint(tcpAddr)
		
		// Start health checks
		checker.StartChecking()
		
		// Wait for checks to run
		time.Sleep(time.Millisecond * 200)
		
		// Check TCP endpoint status
		status := checker.GetStatus(tcpAddr)
		assert.Equal(t, Healthy, status, "Expected TCP endpoint to be Healthy")
		
		// Verify IsHealthy helper
		assert.True(t, checker.IsHealthy(tcpAddr), "Expected IsHealthy to return true")
		
		// Get healthy endpoints
		healthy := checker.GetHealthyEndpoints()
		assert.Contains(t, healthy, tcpAddr, "Expected healthy endpoints to include TCP endpoint")
		
		// Close TCP server to simulate failure
		tcpListener.Close()
		
		// Wait for health check to detect failure
		time.Sleep(time.Millisecond * 600)
		
		// Check TCP endpoint status again
		status = checker.GetStatus(tcpAddr)
		assert.Equal(t, Unhealthy, status, "Expected TCP endpoint to be Unhealthy after server closed")
		
		// Verify IsHealthy helper
		assert.False(t, checker.IsHealthy(tcpAddr), "Expected IsHealthy to return false")
		
		// Remove TCP endpoint
		checker.RemoveEndpoint(tcpAddr)
		
		// Stop health checks
		checker.StopChecking()
	})
	
	// Reset checker for HTTP test
	checker = NewChecker(&Settings{
		Enabled:             true,
		Type:                HTTPCheck,
		Interval:            time.Millisecond * 100,
		Timeout:             time.Millisecond * 500,
		HealthyThreshold:    1,
		UnhealthyThreshold:  1,
		HTTPPath:            "/health",
		HTTPStatusCodeMatch: 200,
	})
	checker.Debug = true
	
	// Test HTTP health check
	t.Run("HTTP Health Check", func(t *testing.T) {
		// Add HTTP endpoint
		checker.AddEndpoint(httpAddr)
		
		// Start health checks
		checker.StartChecking()
		
		// Wait for checks to run
		time.Sleep(time.Millisecond * 200)
		
		// Check HTTP endpoint status
		status := checker.GetStatus(httpAddr)
		assert.Equal(t, Healthy, status, "Expected HTTP endpoint to be Healthy")
		
		// Close HTTP server to simulate failure
		httpListener.Close()
		
		// Wait for health check to detect failure
		time.Sleep(time.Millisecond * 600)
		
		// Check HTTP endpoint status again
		status = checker.GetStatus(httpAddr)
		assert.Equal(t, Unhealthy, status, "Expected HTTP endpoint to be Unhealthy after server closed")
		
		// Stop health checks
		checker.StopChecking()
	})
}

// Test for health status transitions
func TestHealthStatusTransitions(t *testing.T) {
	// Create mock TCP checker that controls success/failure
	type mockChecker struct {
		successFlag bool
	}
	
	origTCPCheck := tcpCheck
	defer func() { tcpCheck = origTCPCheck }()
	
	mock := &mockChecker{successFlag: true}
	tcpCheck = func(c *Checker, address string) bool {
		return mock.successFlag
	}
	
	// Create health checker with specific thresholds
	settings := &Settings{
		Enabled:            true,
		Type:               TCPCheck,
		Interval:           time.Millisecond * 10,
		Timeout:            time.Millisecond * 100,
		HealthyThreshold:   3,  // Need 3 successes to become healthy
		UnhealthyThreshold: 2,  // Need 2 failures to become unhealthy
	}
	
	checker := NewChecker(settings)
	checker.Debug = true
	
	// Add endpoint
	endpoint := "test-endpoint:80"
	checker.AddEndpoint(endpoint)
	
	// Verify initial status is Unknown
	assert.Equal(t, Unknown, checker.GetStatus(endpoint), "Expected initial status to be Unknown")
	
	// Run manual health checks with successes
	for i := 0; i < 2; i++ {
		checker.checkEndpoint(endpoint, checker.Endpoints[endpoint])
		assert.Equal(t, Unknown, checker.GetStatus(endpoint), "Expected status to still be Unknown after 2 successes")
		assert.Equal(t, i+1, checker.Endpoints[endpoint].SuccessCount, "Expected success count to increase")
	}
	
	// Third success should transition to Healthy
	checker.checkEndpoint(endpoint, checker.Endpoints[endpoint])
	assert.Equal(t, Healthy, checker.GetStatus(endpoint), "Expected status to be Healthy after 3 successes")
	assert.Equal(t, 3, checker.Endpoints[endpoint].SuccessCount, "Expected success count to be 3")
	
	// Simulate a failure
	mock.successFlag = false
	checker.checkEndpoint(endpoint, checker.Endpoints[endpoint])
	assert.Equal(t, Healthy, checker.GetStatus(endpoint), "Expected status to still be Healthy after 1 failure")
	assert.Equal(t, 1, checker.Endpoints[endpoint].FailureCount, "Expected failure count to be 1")
	assert.Equal(t, 0, checker.Endpoints[endpoint].SuccessCount, "Expected success count to be reset to 0")
	
	// Second failure should transition to Unhealthy
	checker.checkEndpoint(endpoint, checker.Endpoints[endpoint])
	assert.Equal(t, Unhealthy, checker.GetStatus(endpoint), "Expected status to be Unhealthy after 2 failures")
	assert.Equal(t, 2, checker.Endpoints[endpoint].FailureCount, "Expected failure count to be 2")
	
	// Simulate recovery
	mock.successFlag = true
	for i := 0; i < 2; i++ {
		checker.checkEndpoint(endpoint, checker.Endpoints[endpoint])
		assert.Equal(t, Unhealthy, checker.GetStatus(endpoint), "Expected status to still be Unhealthy after 2 successes")
		assert.Equal(t, i+1, checker.Endpoints[endpoint].SuccessCount, "Expected success count to increase")
	}
	
	// Third success should transition back to Healthy
	checker.checkEndpoint(endpoint, checker.Endpoints[endpoint])
	assert.Equal(t, Healthy, checker.GetStatus(endpoint), "Expected status to be Healthy after 3 successes")
	assert.Equal(t, 3, checker.Endpoints[endpoint].SuccessCount, "Expected success count to be 3")
	assert.Equal(t, 0, checker.Endpoints[endpoint].FailureCount, "Expected failure count to be reset to 0")
}

// Refactored function for testability
var tcpCheck = func(c *Checker, address string) bool {
	conn, err := net.DialTimeout("tcp", address, c.Settings.Timeout)
	if err != nil {
		if c.Debug {
			fmt.Printf("TCP health check failed for %s: %v\n", address, err)
		}
		return false
	}
	
	defer conn.Close()
	return true
}