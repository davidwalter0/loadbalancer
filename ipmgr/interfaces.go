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

package ipmgr

import (
	"fmt"
	"net"
)

// Interface represents a network interface with its associated addresses
type Interface struct {
	Name      string
	Addresses []string
	IsUp      bool
}

// ListInterfaces returns a list of all network interfaces on the system
func ListInterfaces() ([]Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %v", err)
	}

	var result []Interface
	for _, iface := range interfaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var addrStrings []string
		for _, addr := range addrs {
			addrStrings = append(addrStrings, addr.String())
		}

		result = append(result, Interface{
			Name:      iface.Name,
			Addresses: addrStrings,
			IsUp:      iface.Flags&net.FlagUp != 0,
		})
	}

	return result, nil
}

// GetValidInterfaces returns a list of valid interfaces for use as link devices
// Valid interfaces are those that:
// - Are up
// - Have at least one IPv4 address
// - Are not loopback
func GetValidInterfaces() ([]Interface, error) {
	interfaces, err := ListInterfaces()
	if err != nil {
		return nil, err
	}

	var validInterfaces []Interface
	for _, iface := range interfaces {
		if !iface.IsUp {
			continue
		}

		// Check if interface has at least one IPv4 address
		hasIPv4 := false
		for _, addr := range iface.Addresses {
			ip, _, err := net.ParseCIDR(addr)
			if err != nil {
				continue
			}
			if ip.To4() != nil {
				hasIPv4 = true
				break
			}
		}

		if hasIPv4 {
			validInterfaces = append(validInterfaces, iface)
		}
	}

	return validInterfaces, nil
}