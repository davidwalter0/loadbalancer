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

// Package health provides health checking capabilities for LoadBalancer endpoints
package health

import (
	"fmt"
	"time"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/loadbalancer/tracer"
)

// Status represents the health status of an endpoint
type Status int

const (
	// Unknown endpoint has not been checked yet
	Unknown Status = iota
	// Healthy endpoint is responding to health checks
	Healthy
	// Unhealthy endpoint is not responding to health checks
	Unhealthy
)

// String returns a string representation of the health status
func (s Status) String() string {
	switch s {
	case Unknown:
		return "Unknown"
	case Healthy:
		return "Healthy"
	case Unhealthy:
		return "Unhealthy"
	default:
		return fmt.Sprintf("Invalid(%d)", s)
	}
}

// CheckType represents the type of health check to perform
type CheckType int

const (
	// TCPCheck simply verifies TCP connection establishment
	TCPCheck CheckType = iota
	// HTTPCheck sends a HTTP request and verifies the response
	HTTPCheck
)

// String returns a string representation of the check type
func (c CheckType) String() string {
	switch c {
	case TCPCheck:
		return "TCP"
	case HTTPCheck:
		return "HTTP"
	default:
		return fmt.Sprintf("Invalid(%d)", c)
	}
}

// EndpointStatus contains the health status information for an endpoint
type EndpointStatus struct {
	Address       string    // Host:port of the endpoint
	Status        Status    // Current health status
	LastCheck     time.Time // Time of last health check
	FailureCount  int       // Number of consecutive failures
	SuccessCount  int       // Number of consecutive successes
	LastCheckTime time.Duration // Duration of last check
}

// Settings contains configuration for health checks
type Settings struct {
	Enabled             bool          // Whether health checking is enabled
	Type                CheckType     // Type of health check to perform
	Interval            time.Duration // Time between health checks
	Timeout             time.Duration // Timeout for health checks
	HealthyThreshold    int           // Number of successes to mark as healthy
	UnhealthyThreshold  int           // Number of failures to mark as unhealthy
	HTTPPath            string        // Path for HTTP health checks
	HTTPStatusCodeMatch int           // Expected HTTP status code
}

// DefaultSettings returns the default health check settings
func DefaultSettings() *Settings {
	return &Settings{
		Enabled:            true,
		Type:               TCPCheck,
		Interval:           time.Second * 10,
		Timeout:            time.Second * 5,
		HealthyThreshold:   2,
		UnhealthyThreshold: 3,
		HTTPPath:           "/health",
		HTTPStatusCodeMatch: 200,
	}
}

// Checker performs health checks on endpoints
type Checker struct {
	Settings     *Settings
	Endpoints    map[string]*EndpointStatus
	Mutex        *mutex.Mutex
	Stop         chan struct{}
	Debug        bool
}

// NewChecker creates a new health checker with the given settings
func NewChecker(settings *Settings) *Checker {
	if settings == nil {
		settings = DefaultSettings()
	}
	
	return &Checker{
		Settings:  settings,
		Endpoints: make(map[string]*EndpointStatus),
		Mutex:     &mutex.Mutex{},
		Stop:      make(chan struct{}),
	}
}

// Monitor returns a mutex monitor function
func (c *Checker) Monitor(args ...interface{}) func() {
	defer trace.Tracer.ScopedTrace(args...)()
	return c.Mutex.MonitorTrace(args...)
}

// IsEnabled returns whether health checking is enabled
func (c *Checker) IsEnabled() bool {
	if c == nil || c.Settings == nil {
		return false
	}
	return c.Settings.Enabled
}