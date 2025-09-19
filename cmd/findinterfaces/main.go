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

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
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

// DetectWirelessInterfaces uses the 'iw' command to detect wireless interfaces
func DetectWirelessInterfaces() (map[string]bool, error) {
	// Run 'iw dev' to list wireless interfaces
	cmd := exec.Command("iw", "dev")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run 'iw dev': %v", err)
	}
	
	// Parse output to find interface names
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	
	wirelessIfaces := make(map[string]bool)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Interface ") {
			ifaceName := strings.TrimPrefix(line, "Interface ")
			wirelessIfaces[ifaceName] = true
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
	
	// Detect wireless interfaces
	wirelessIfaces, err := DetectWirelessInterfaces()
	if err != nil {
		// Continue without wireless detection if 'iw' fails
		fmt.Printf("Warning: failed to detect wireless interfaces: %v\n", err)
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
	
	// Try to use ethtool if available for more detailed info
	ethtoolAvailable := isCommandAvailable("ethtool")
	if ethtoolAvailable {
		cmd := exec.Command("ethtool", "-i", ifaceName)
		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), "bus-info:") {
			isPhysical = true
			description = "Physical"
			
			// Extract driver information if available
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "driver:") {
					driver := strings.TrimSpace(strings.TrimPrefix(line, "driver:"))
					description = fmt.Sprintf("Physical (%s)", driver)
					break
				}
			}
			
			return description, isPhysical, isVirtual, nil
		}
	}
	
	// If ethtool is not available or didn't provide results, check sysfs
	if isCommandAvailable("readlink") {
		// Check if it's a USB device by looking at sysfs
		cmd := exec.Command("sh", "-c", fmt.Sprintf("readlink -f /sys/class/net/%s/device 2>/dev/null | grep -q usb", ifaceName))
		if err := cmd.Run(); err == nil {
			isPhysical = true
			description = "Physical (USB)"
			return description, isPhysical, isVirtual, nil
		}
		
		// Check for PCI devices through sysfs as a fallback to ethtool
		cmd = exec.Command("sh", "-c", fmt.Sprintf("readlink -f /sys/class/net/%s/device 2>/dev/null | grep -q pci", ifaceName))
		if err := cmd.Run(); err == nil {
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

// isCommandAvailable checks if a command is available on the system
func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func main() {
	interfaces, err := GetAllInterfaces()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Print all interfaces
	fmt.Println("All Network Interfaces:")
	fmt.Println("=======================")
	for _, iface := range interfaces {
		fmt.Printf("Name: %s\n", iface.Name)
		fmt.Printf("  Type: %s\n", iface.Description)
		fmt.Printf("  Is Up: %t\n", iface.IsUp)
		fmt.Printf("  Is Wireless: %t\n", iface.IsWireless)
		fmt.Printf("  Is Physical: %t\n", iface.IsPhysical)
		fmt.Printf("  Is Virtual: %t\n", iface.IsVirtual)
		fmt.Printf("  IPv4 Addresses: %v\n", iface.IPv4Addrs)
		fmt.Println()
	}
	
	// Find first wired interface
	wiredIface := FindFirstWiredInterface(interfaces)
	if wiredIface != nil {
		fmt.Println("\nFirst Physical Wired Interface (Up):")
		fmt.Println("===================================")
		fmt.Printf("Name: %s\n", wiredIface.Name)
		fmt.Printf("Type: %s\n", wiredIface.Description)
		fmt.Printf("IPv4 Addresses: %v\n", wiredIface.IPv4Addrs)
	} else {
		fmt.Println("\nNo physical wired interfaces found.")
	}
	
	// Find first wireless interface
	wirelessIface := FindFirstWirelessInterface(interfaces)
	if wirelessIface != nil {
		fmt.Println("\nFirst Wireless Interface (Up):")
		fmt.Println("=============================")
		fmt.Printf("Name: %s\n", wirelessIface.Name)
		fmt.Printf("Type: %s\n", wirelessIface.Description)
		fmt.Printf("IPv4 Addresses: %v\n", wirelessIface.IPv4Addrs)
	} else {
		fmt.Println("\nNo wireless interfaces found.")
	}
}