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
	"os"
	"strings"
)

// InterfaceInfo contains information about a network interface
type InterfaceInfo struct {
	Name        string
	IsUp        bool
	IsWireless  bool
	IsPhysical  bool
	IsVirtual   bool
	Description string
	IPv4Addrs   []string
}

// DetectWirelessInterfaces checks sysfs to detect wireless interfaces (no external commands)
func DetectWirelessInterfaces() (map[string]bool, error) {
	wirelessIfaces := make(map[string]bool)

	// Get all network interfaces
	netIfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Check each interface for wireless directory in sysfs
	for _, iface := range netIfaces {
		// Check if /sys/class/net/<iface>/wireless exists
		wirelessPath := fmt.Sprintf("/sys/class/net/%s/wireless", iface.Name)
		if _, err := os.Stat(wirelessPath); err == nil {
			wirelessIfaces[iface.Name] = true
			continue
		}

		// Alternative: Check if /sys/class/net/<iface>/phy80211 exists
		phy80211Path := fmt.Sprintf("/sys/class/net/%s/phy80211", iface.Name)
		if _, err := os.Stat(phy80211Path); err == nil {
			wirelessIfaces[iface.Name] = true
		}
	}

	return wirelessIfaces, nil
}

// GetAllInterfaces returns information about all network interfaces
func GetAllInterfaces() ([]InterfaceInfo, error) {
	// Get all network interfaces
	netIfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %v", err)
	}
	
	// Detect wireless interfaces using sysfs (no external commands)
	wirelessIfaces, err := DetectWirelessInterfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to detect wireless interfaces: %v", err)
	}
	
	var interfaces []InterfaceInfo
	
	for _, iface := range netIfaces {
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		
		// Get IP addresses
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		
		var ipv4Addrs []string
		for _, addr := range addrs {
			// Convert to CIDR notation
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			
			// Only include IPv4 addresses
			if ipv4 := ipNet.IP.To4(); ipv4 != nil {
				ipv4Addrs = append(ipv4Addrs, ipNet.String())
			}
		}
		
		// Check interface type
		description, isPhysical, isVirtual, _ := CheckInterfaceType(iface.Name)
		
		interfaces = append(interfaces, InterfaceInfo{
			Name:        iface.Name,
			IsUp:        iface.Flags&net.FlagUp != 0,
			IsWireless:  wirelessIfaces[iface.Name],
			IsPhysical:  isPhysical,
			IsVirtual:   isVirtual,
			Description: description,
			IPv4Addrs:   ipv4Addrs,
		})
	}
	
	return interfaces, nil
}

// FindFirstWiredInterface returns the first physical wired interface that is up
func FindFirstWiredInterface(interfaces []InterfaceInfo) *InterfaceInfo {
	// First look for physical interfaces (preferred)
	for _, iface := range interfaces {
		if iface.IsUp && !iface.IsWireless && iface.IsPhysical && len(iface.IPv4Addrs) > 0 {
			return &iface
		}
	}
	
	// If no physical interfaces found, try any non-virtual interface as a fallback
	for _, iface := range interfaces {
		if iface.IsUp && !iface.IsWireless && !iface.IsVirtual && len(iface.IPv4Addrs) > 0 {
			return &iface
		}
	}
	
	// Last resort: any wired interface (including virtual)
	for _, iface := range interfaces {
		if iface.IsUp && !iface.IsWireless && len(iface.IPv4Addrs) > 0 {
			return &iface
		}
	}
	
	return nil
}

// FindFirstWirelessInterface returns the first wireless interface that is up
func FindFirstWirelessInterface(interfaces []InterfaceInfo) *InterfaceInfo {
	for _, iface := range interfaces {
		if iface.IsUp && iface.IsWireless && len(iface.IPv4Addrs) > 0 {
			return &iface
		}
	}
	return nil
}

// CheckInterfaceType tries to determine if an interface is a virtual or physical interface
func CheckInterfaceType(ifaceName string) (string, bool, bool, error) {
	description := "Unknown"
	isPhysical := false
	isVirtual := false
	
	// Check common virtual interface naming patterns first (this will work without any tools)
	if strings.HasPrefix(ifaceName, "br-") || 
	   strings.HasPrefix(ifaceName, "docker") ||
	   strings.HasPrefix(ifaceName, "veth") ||
	   strings.HasPrefix(ifaceName, "virbr") ||
	   strings.HasPrefix(ifaceName, "vnet") ||
	   strings.HasPrefix(ifaceName, "tun") ||
	   strings.HasPrefix(ifaceName, "tap") ||
	   strings.HasPrefix(ifaceName, "cni") ||
	   strings.HasPrefix(ifaceName, "flannel") ||
	   strings.HasPrefix(ifaceName, "cilium") ||
	   strings.HasPrefix(ifaceName, "calico") ||
	   ifaceName == "lxcbr0" ||
	   ifaceName == "docker0" {
		isVirtual = true
		
		if strings.HasPrefix(ifaceName, "br-") || ifaceName == "lxcbr0" || ifaceName == "docker0" {
			description = "Bridge"
		} else if strings.HasPrefix(ifaceName, "veth") {
			description = "Virtual (veth)"
		} else if strings.HasPrefix(ifaceName, "docker") {
			description = "Docker"
		} else {
			description = "Virtual"
		}
		
		return description, isPhysical, isVirtual, nil
	}
	
	// Common physical interface naming patterns (will work without tools)
	if strings.HasPrefix(ifaceName, "eth") ||
	   strings.HasPrefix(ifaceName, "en") ||
	   strings.HasPrefix(ifaceName, "wl") ||
	   strings.HasPrefix(ifaceName, "ww") {
		isPhysical = true
		description = "Physical (by naming convention)"
	}
	
	// Try to get driver information from sysfs (no external commands)
	driverPath := fmt.Sprintf("/sys/class/net/%s/device/driver", ifaceName)
	if driverLink, err := os.Readlink(driverPath); err == nil {
		isPhysical = true
		// Extract driver name from path like: ../../../bus/pci/drivers/r8152
		parts := strings.Split(driverLink, "/")
		if len(parts) > 0 {
			driverName := parts[len(parts)-1]
			description = fmt.Sprintf("Physical (%s)", driverName)
			return description, isPhysical, isVirtual, nil
		}
		description = "Physical"
		return description, isPhysical, isVirtual, nil
	}

	// Check device subsystem to determine if USB or PCI (pure Go sysfs)
	subsystemPath := fmt.Sprintf("/sys/class/net/%s/device/subsystem", ifaceName)
	if subsystemLink, err := os.Readlink(subsystemPath); err == nil {
		if strings.Contains(subsystemLink, "usb") {
			isPhysical = true
			description = "Physical (USB)"
			return description, isPhysical, isVirtual, nil
		}
		if strings.Contains(subsystemLink, "pci") {
			isPhysical = true
			if description == "Physical (by naming convention)" {
				description = "Physical (PCI)"
			}
			return description, isPhysical, isVirtual, nil
		}
	}
	
	// If we already determined it's physical by naming convention, return that
	if isPhysical {
		return description, isPhysical, isVirtual, nil
	}
	
	// If we still don't know, check driver directory exists (works on most Linux systems)
	if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s/device/driver", ifaceName)); err == nil {
		isPhysical = true
		description = "Physical (has driver)"
		return description, isPhysical, isVirtual, nil
	}
	
	return description, isPhysical, isVirtual, nil
}

// GetPreferredInterface returns the best interface to use
// Preference order:
// 1. Physical wired interface
// 2. Any non-virtual wired interface
// 3. Any wired interface
// 4. Wireless interface
func GetPreferredInterface() (*InterfaceInfo, error) {
	interfaces, err := GetAllInterfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %v", err)
	}
	
	// Try to find a wired interface first
	if wiredIface := FindFirstWiredInterface(interfaces); wiredIface != nil {
		return wiredIface, nil
	}
	
	// If no wired interface, try wireless
	if wirelessIface := FindFirstWirelessInterface(interfaces); wirelessIface != nil {
		return wirelessIface, nil
	}
	
	return nil, fmt.Errorf("no suitable network interface found")
}