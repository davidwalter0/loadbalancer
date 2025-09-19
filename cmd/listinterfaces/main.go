/*

Copyright 2018 David Walter.

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
	"log"

	"github.com/davidwalter0/loadbalancer/ipmgr"
)

func main() {
	// List all valid interfaces
	interfaces, err := ipmgr.GetValidInterfaces()
	if err != nil {
		log.Fatalf("Failed to list interfaces: %v", err)
	}

	fmt.Println("Available network interfaces:")
	fmt.Println("-----------------------------")
	
	for i, iface := range interfaces {
		fmt.Printf("%d. %s\n", i+1, iface.Name)
		fmt.Printf("   Addresses:\n")
		for _, addr := range iface.Addresses {
			fmt.Printf("     - %s\n", addr)
		}
		fmt.Println()
	}

	if len(interfaces) == 0 {
		fmt.Println("No valid network interfaces found")
	} else {
		fmt.Printf("\nTo use one of these interfaces with loadbalancer, use:\n")
		fmt.Printf("--linkdevice <interface-name>\n\n")
		fmt.Printf("Example: --linkdevice %s\n", interfaces[0].Name)
	}
}