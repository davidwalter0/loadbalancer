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

package ipmgr

import (
	"fmt"
	"log"

	"github.com/davidwalter0/loadbalancer/pkg/interfaces"
)

// AutoDetectInterface returns the best interface to use based on priority:
// 1. Physical wired interface with IPv4 address
// 2. Any non-virtual wired interface with IPv4 address
// 3. Any wired interface with IPv4 address
// 4. Wireless interface with IPv4 address
// Returns the interface name if successful, error otherwise
func AutoDetectInterface() (string, error) {
	log.Println("Auto-detecting best network interface...")
	
	// Get the preferred interface using the new interfaces package
	preferredTypes := []string{"wired-physical", "wired-nonvirtual", "wired", "wireless", "any"}
	iface, err := interfaces.GetBestInterface(preferredTypes)
	if err != nil {
		return "", fmt.Errorf("failed to auto-detect interface: %v", err)
	}
	
	if iface == nil {
		return "", fmt.Errorf("no suitable interface found")
	}
	
	log.Printf("Auto-detected interface: %s (%s) with IP addresses: %v", 
		iface.Name, iface.Description, iface.IPv4Addrs)
	
	return iface.Name, nil
}

// GetInterfaceDetails returns detailed information about a specific interface
func GetInterfaceDetails(ifaceName string) (*interfaces.InterfaceInfo, error) {
	return interfaces.GetInterfaceByName(ifaceName)
}