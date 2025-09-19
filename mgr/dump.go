/*

Copyright 2018-2025 David Walter.

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

package mgr

import (
	"fmt"
	"strings"
)

// GetPorts returns a formatted list of ports for the managed listener
func (ml *ManagedListener) GetPorts() string {
	if ml == nil {
		return "N/A"
	}

	defer ml.Monitor()()

	if len(ml.Ports) == 0 {
		return fmt.Sprintf("%d (primary)", ml.Port)
	}

	// Format ports as a comma-separated list
	ports := make([]string, len(ml.Ports))
	for i, port := range ml.Ports {
		if port == ml.Port {
			ports[i] = fmt.Sprintf("%d (primary)", port)
		} else {
			ports[i] = fmt.Sprintf("%d", port)
		}
	}

	return strings.Join(ports, ", ")
}

// GetEndpoints returns a formatted list of the managed listener's endpoints
func (ml *ManagedListener) GetEndpoints() string {
	if ml == nil {
		return "N/A"
	}

	defer ml.Monitor()()

	if len(ml.IPs) == 0 {
		return "No endpoints"
	}

	// Format IPs as a comma-separated list with health status if health checker is enabled
	endpoints := make([]string, len(ml.IPs))

	for i, ip := range ml.IPs {
		// Add health status if health checker is enabled
		if ml.HealthChecker != nil && ml.HealthChecker.IsEnabled() && len(ml.Ports) > 0 {
			address := Address(ip, ml.Port)
			if ml.HealthChecker.IsHealthy(address) {
				endpoints[i] = fmt.Sprintf("%s (healthy)", string(ip))
			} else {
				endpoints[i] = fmt.Sprintf("%s (unhealthy)", string(ip))
			}
		} else {
			endpoints[i] = string(ip)
		}
	}

	return strings.Join(endpoints, ", ")
}

// DumpServicesInfo returns a multi-line string with information about all services
func (mgr *Mgr) DumpServicesInfo() string {
	if mgr == nil {
		return "Manager is nil"
	}

	defer mgr.Mutex.MonitorTrace("DumpServicesInfo")()

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Load Balancer Services Summary:\n"))
	builder.WriteString(fmt.Sprintf("Link Device: %s, CIDR: %s\n", mgr.EnvCfg.LinkDevice, mgr.GetCIDR()))
	builder.WriteString(fmt.Sprintf("Total Active Services: %d\n\n", len(mgr.Listeners)))

	if len(mgr.Listeners) == 0 {
		builder.WriteString("No active services found\n")
		return builder.String()
	}

	// Create a sorted list of service keys
	keys := make([]string, 0, len(mgr.Listeners))
	for k := range mgr.Listeners {
		keys = append(keys, k)
	}

	// Sort the keys alphabetically
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	// Build the output for each service
	for _, key := range keys {
		listener := mgr.Listeners[key]

		builder.WriteString(fmt.Sprintf("Service: %s\n", key))
		builder.WriteString(fmt.Sprintf("  Type: LoadBalancer\n"))
		builder.WriteString(fmt.Sprintf("  External IP: %s\n", listener.CIDR.IP))
		builder.WriteString(fmt.Sprintf("  Ports: %s\n", listener.GetPorts()))
		builder.WriteString(fmt.Sprintf("  Endpoints: %s\n", listener.GetEndpoints()))
		builder.WriteString(fmt.Sprintf("  Active Connections: %d\n", listener.Active))

		// Add health checker info if enabled
		if listener.HealthChecker != nil && listener.HealthChecker.IsEnabled() {
			builder.WriteString(fmt.Sprintf("  Health Checking: Enabled\n"))
			healthyCount := len(listener.HealthChecker.GetHealthyEndpoints())
			totalCount := len(listener.IPs)
			builder.WriteString(fmt.Sprintf("  Healthy Endpoints: %d/%d\n", healthyCount, totalCount))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}