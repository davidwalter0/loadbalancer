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

package interfaces

import (
	"fmt"
	"net"
	"strings"
)

// GetInterfaceByName returns a specific interface by name
func GetInterfaceByName(name string) (*InterfaceInfo, error) {
	interfaces, err := GetAllInterfaces()
	if err != nil {
		return nil, err
	}
	
	for _, iface := range interfaces {
		if iface.Name == name {
			return &iface, nil
		}
	}
	
	return nil, fmt.Errorf("interface %s not found", name)
}

// GetInterfaceAddress returns the first IPv4 address of an interface
func GetInterfaceAddress(ifaceName string) (string, error) {
	iface, err := GetInterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}
	
	if len(iface.IPv4Addrs) == 0 {
		return "", fmt.Errorf("no IPv4 address found for interface %s", ifaceName)
	}
	
	// Return the first address without subnet mask
	cidr := iface.IPv4Addrs[0]
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	
	return ip.String(), nil
}

// FilterInterfaces returns interfaces that match the specified criteria
func FilterInterfaces(interfaces []InterfaceInfo, isUp, isPhysical, isVirtual, isWireless *bool) []InterfaceInfo {
	var filtered []InterfaceInfo
	
	for _, iface := range interfaces {
		// Skip interfaces with no IPv4 addresses
		if len(iface.IPv4Addrs) == 0 {
			continue
		}
		
		// Apply filters if specified
		if isUp != nil && iface.IsUp != *isUp {
			continue
		}
		
		if isPhysical != nil && iface.IsPhysical != *isPhysical {
			continue
		}
		
		if isVirtual != nil && iface.IsVirtual != *isVirtual {
			continue
		}
		
		if isWireless != nil && iface.IsWireless != *isWireless {
			continue
		}
		
		filtered = append(filtered, iface)
	}
	
	return filtered
}

// PrintInterfaceInfo outputs information about an interface
func PrintInterfaceInfo(iface *InterfaceInfo) string {
	if iface == nil {
		return "No interface found"
	}
	
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("Name: %s\n", iface.Name))
	builder.WriteString(fmt.Sprintf("Type: %s\n", iface.Description))
	builder.WriteString(fmt.Sprintf("Is Up: %t\n", iface.IsUp))
	builder.WriteString(fmt.Sprintf("Is Wireless: %t\n", iface.IsWireless))
	builder.WriteString(fmt.Sprintf("Is Physical: %t\n", iface.IsPhysical))
	builder.WriteString(fmt.Sprintf("Is Virtual: %t\n", iface.IsVirtual))
	builder.WriteString(fmt.Sprintf("IPv4 Addresses: %v\n", iface.IPv4Addrs))
	
	return builder.String()
}

// GetBestInterface returns the best interface based on the provided preferences
// Preference order is determined by the order of the types in the preferredTypes slice
// Example: []string{"wired-physical", "wired", "wireless"}
func GetBestInterface(preferredTypes []string) (*InterfaceInfo, error) {
	interfaces, err := GetAllInterfaces()
	if err != nil {
		return nil, err
	}
	
	for _, prefType := range preferredTypes {
		switch prefType {
		case "wired-physical":
			// Physical wired interface
			isUp := true
			isWireless := false
			isPhysical := true
			filtered := FilterInterfaces(interfaces, &isUp, &isPhysical, nil, &isWireless)
			if len(filtered) > 0 {
				return &filtered[0], nil
			}
		case "wired-nonvirtual":
			// Any non-virtual wired interface
			isUp := true
			isWireless := false
			isVirtual := false
			filtered := FilterInterfaces(interfaces, &isUp, nil, &isVirtual, &isWireless)
			if len(filtered) > 0 {
				return &filtered[0], nil
			}
		case "wired":
			// Any wired interface
			isUp := true
			isWireless := false
			filtered := FilterInterfaces(interfaces, &isUp, nil, nil, &isWireless)
			if len(filtered) > 0 {
				return &filtered[0], nil
			}
		case "wireless":
			// Wireless interface
			isUp := true
			isWireless := true
			filtered := FilterInterfaces(interfaces, &isUp, nil, nil, &isWireless)
			if len(filtered) > 0 {
				return &filtered[0], nil
			}
		case "any":
			// Any interface with an IP
			isUp := true
			filtered := FilterInterfaces(interfaces, &isUp, nil, nil, nil)
			if len(filtered) > 0 {
				return &filtered[0], nil
			}
		}
	}
	
	return nil, fmt.Errorf("no interface found matching the preferred types")
}