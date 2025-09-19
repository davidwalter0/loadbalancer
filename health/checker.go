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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

// AddEndpoint adds an endpoint to be health checked
func (c *Checker) AddEndpoint(address string) {
	if !c.IsEnabled() {
		return
	}

	defer c.Monitor("AddEndpoint", address)()
	
	if _, ok := c.Endpoints[address]; !ok {
		c.Endpoints[address] = &EndpointStatus{
			Address:      address,
			Status:       Unknown,
			LastCheck:    time.Time{},
			FailureCount: 0,
			SuccessCount: 0,
		}
		
		if c.Debug {
			log.Printf("Health check: Added endpoint %s", address)
		}
	}
}

// RemoveEndpoint removes an endpoint from health checking
func (c *Checker) RemoveEndpoint(address string) {
	if !c.IsEnabled() {
		return
	}

	defer c.Monitor("RemoveEndpoint", address)()
	
	if _, ok := c.Endpoints[address]; ok {
		delete(c.Endpoints, address)
		
		if c.Debug {
			log.Printf("Health check: Removed endpoint %s", address)
		}
	}
}

// GetStatus returns the health status of an endpoint
func (c *Checker) GetStatus(address string) Status {
	if !c.IsEnabled() {
		return Healthy // If checks are disabled, consider all endpoints healthy
	}

	defer c.Monitor("GetStatus", address)()
	
	if endpoint, ok := c.Endpoints[address]; ok {
		return endpoint.Status
	}
	
	return Unknown
}

// IsHealthy returns whether an endpoint is healthy
func (c *Checker) IsHealthy(address string) bool {
	if !c.IsEnabled() {
		return true // If checks are disabled, consider all endpoints healthy
	}

	status := c.GetStatus(address)
	return status == Healthy
}

// GetHealthyEndpoints returns all healthy endpoints
func (c *Checker) GetHealthyEndpoints() []string {
	if !c.IsEnabled() {
		// If health checks are disabled, return all endpoints
		addresses := make([]string, 0, len(c.Endpoints))
		for addr := range c.Endpoints {
			addresses = append(addresses, addr)
		}
		return addresses
	}

	defer c.Monitor("GetHealthyEndpoints")()
	
	healthy := make([]string, 0, len(c.Endpoints))
	for addr, endpoint := range c.Endpoints {
		if endpoint.Status == Healthy {
			healthy = append(healthy, addr)
		}
	}
	
	return healthy
}

// StartChecking begins the health check loop
func (c *Checker) StartChecking() {
	if !c.IsEnabled() {
		return
	}

	go func() {
		ticker := time.NewTicker(c.Settings.Interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-c.Stop:
				log.Println("Health checker shutting down...")
				return
			case <-ticker.C:
				c.checkAll()
			}
		}
	}()
	
	log.Println("Health checker started")
}

// StopChecking stops the health check loop
func (c *Checker) StopChecking() {
	if !c.IsEnabled() {
		return
	}

	// Safe channel close with mutex to prevent race conditions
	defer c.Monitor()()

	// Add a recover to handle already closed channel
	defer func() {
		if r := recover(); r != nil {
			// Channel was already closed, just log and continue
			log.Printf("Warning: Health checker channel was already closed: %v", r)
		}
	}()

	// Set enabled to false first to prevent double closing
	c.Settings.Enabled = false

	// Close the channel
	close(c.Stop)
}

// checkAll performs health checks on all endpoints
func (c *Checker) checkAll() {
	defer c.Monitor("checkAll")()
	
	for addr, endpoint := range c.Endpoints {
		go c.checkEndpoint(addr, endpoint)
	}
}

// checkEndpoint performs a health check on a single endpoint
func (c *Checker) checkEndpoint(address string, endpoint *EndpointStatus) {
	start := time.Now()
	success := false
	
	switch c.Settings.Type {
	case TCPCheck:
		success = c.tcpCheck(address)
	case HTTPCheck:
		success = c.httpCheck(address)
	}
	
	checkDuration := time.Since(start)
	
	defer c.Monitor("checkEndpoint", address)()
	
	endpoint.LastCheck = time.Now()
	endpoint.LastCheckTime = checkDuration
	
	if success {
		endpoint.SuccessCount++
		endpoint.FailureCount = 0
		
		if endpoint.Status != Healthy && endpoint.SuccessCount >= c.Settings.HealthyThreshold {
			if endpoint.Status != Unknown {
				log.Printf("Health check: Endpoint %s is now healthy", address)
			}
			endpoint.Status = Healthy
		}
	} else {
		endpoint.FailureCount++
		endpoint.SuccessCount = 0
		
		if endpoint.Status != Unhealthy && endpoint.FailureCount >= c.Settings.UnhealthyThreshold {
			if endpoint.Status != Unknown {
				log.Printf("Health check: Endpoint %s is now unhealthy", address)
			}
			endpoint.Status = Unhealthy
		}
	}
	
	if c.Debug {
		log.Printf("Health check: %s - %v (took %v)", address, success, checkDuration)
	}
}

// tcpCheck performs a TCP health check
func (c *Checker) tcpCheck(address string) bool {
	conn, err := net.DialTimeout("tcp", address, c.Settings.Timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// httpCheck performs an HTTP health check
func (c *Checker) httpCheck(address string) bool {
	client := &http.Client{
		Timeout: c.Settings.Timeout,
	}
	
	url := fmt.Sprintf("http://%s%s", address, c.Settings.HTTPPath)
	
	resp, err := client.Get(url)
	if err != nil {
		if c.Debug {
			log.Printf("HTTP health check failed for %s: %v", address, err)
		}
		return false
	}
	
	defer resp.Body.Close()
	
	// Drain the response body to reuse the connection
	_, _ = ioutil.ReadAll(resp.Body)
	
	if c.Settings.HTTPStatusCodeMatch > 0 && resp.StatusCode != c.Settings.HTTPStatusCodeMatch {
		if c.Debug {
			log.Printf("HTTP health check failed for %s: got status %d, expected %d", 
				address, resp.StatusCode, c.Settings.HTTPStatusCodeMatch)
		}
		return false
	}
	
	return true
}