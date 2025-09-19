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
	"os"

	"github.com/davidwalter0/loadbalancer/pkg/interfaces"
)

func main() {
	// Get all interfaces
	allInterfaces, err := interfaces.GetAllInterfaces()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	// Print all interfaces
	fmt.Println("All Network Interfaces:")
	fmt.Println("=======================")
	for _, iface := range allInterfaces {
		fmt.Printf("Name: %s\n", iface.Name)
		fmt.Printf("  Type: %s\n", iface.Description)
		fmt.Printf("  Is Up: %t\n", iface.IsUp)
		fmt.Printf("  Is Wireless: %t\n", iface.IsWireless)
		fmt.Printf("  Is Physical: %t\n", iface.IsPhysical)
		fmt.Printf("  Is Virtual: %t\n", iface.IsVirtual)
		fmt.Printf("  IPv4 Addresses: %v\n", iface.IPv4Addrs)
		fmt.Println()
	}
	
	// Find best interface with priority order
	fmt.Println("\nBest Interface Detection with Priority:")
	fmt.Println("=======================================")
	
	// Define priority order
	preferredTypes := []string{"wired-physical", "wired-nonvirtual", "wired", "wireless", "any"}
	fmt.Println("Priority order:")
	fmt.Println("1. Physical wired interface")
	fmt.Println("2. Non-virtual wired interface")
	fmt.Println("3. Any wired interface")
	fmt.Println("4. Wireless interface")
	fmt.Println("5. Any interface with IP")
	fmt.Println()
	
	bestIface, err := interfaces.GetBestInterface(preferredTypes)
	if err != nil {
		fmt.Printf("Error finding best interface: %v\n", err)
		os.Exit(1)
	}
	
	if bestIface != nil {
		fmt.Println("Best Interface Found:")
		fmt.Println("====================")
		fmt.Printf("Name: %s\n", bestIface.Name)
		fmt.Printf("Type: %s\n", bestIface.Description)
		fmt.Printf("Is Up: %t\n", bestIface.IsUp)
		fmt.Printf("Is Wireless: %t\n", bestIface.IsWireless)
		fmt.Printf("Is Physical: %t\n", bestIface.IsPhysical)
		fmt.Printf("Is Virtual: %t\n", bestIface.IsVirtual)
		fmt.Printf("IPv4 Addresses: %v\n", bestIface.IPv4Addrs)
	} else {
		fmt.Println("No suitable interface found.")
	}
}