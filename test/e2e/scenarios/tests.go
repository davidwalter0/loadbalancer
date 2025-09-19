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
	"testing"

	"github.com/davidwalter0/loadbalancer/test/e2e/framework"
)

// TestFunc is a function that runs a test scenario
type TestFunc func(*testing.T, *framework.Framework)

// GetTests returns all test scenarios
func GetTests() map[string]TestFunc {
	return map[string]TestFunc{
		"BasicConnectivity":            TestBasicConnectivity,
		"MultipleServices":             TestMultipleServices,
		"ServiceUpdates":               TestServiceUpdates,
		"EndpointChanges":              TestEndpointChanges,
		"HealthCheck":                  TestHealthCheck,
		"LoadBalancing":                TestLoadBalancing,
		"ShutdownAndRestart":           TestShutdownAndRestart,
		"HighAvailabilityConfiguration": TestHighAvailabilityConfiguration,
	}
}