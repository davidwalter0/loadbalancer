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
	"log"
	"github.com/davidwalter0/loadbalancer/global"
	"github.com/davidwalter0/loadbalancer/ipmgr"
)

// Initialize configures the IP manager with the provided link device
// If no link device is specified, it will attempt to auto-detect one using the following priority:
// 1. Physical wired interface with IPv4 address
// 2. Any non-virtual wired interface with IPv4 address
// 3. Any wired interface with IPv4 address
// 4. Wireless interface with IPv4 address
// If a link device is specified but doesn't exist, it will fallback to auto-detection
func Initialize() {
	linkDevice := global.Cfg().LinkDevice

	// Auto-detect link device if not specified
	if linkDevice == "" {
		log.Println("No link device specified, attempting auto-detection with priority...")
		var err error
		linkDevice, err = ipmgr.AutoDetectInterface()
		if err != nil {
			log.Fatalf("Error auto-detecting network interface: %v", err)
		}

		if linkDevice == "" {
			log.Fatal("No suitable network interface found for auto-detection")
		}

		// Update the global config
		global.Cfg().LinkDevice = linkDevice
	}

	log.Printf("Using link device: %s", linkDevice)

	// Check if restricted CIDR is specified
	restrictedCIDR := global.Cfg().RestrictedCIDR
	if restrictedCIDR != "" {
		log.Printf("Using restricted CIDR range: %s", restrictedCIDR)
		ipmgr.DefaultCIDR = ipmgr.StringToCIDR(restrictedCIDR)
		ipmgr.DefaultCIDR.LinkDevice = linkDevice
	} else {
		// If no restricted CIDR, check if the specified link device exists
		ipmgr.DefaultCIDR = ipmgr.LinkDefaultCIDR(linkDevice)
	}

	if ipmgr.DefaultCIDR == nil {
		log.Printf("Link device '%s' not found or has no IPv4 addresses, falling back to auto-detection...", linkDevice)

		var err error
		linkDevice, err = ipmgr.AutoDetectInterface()
		if err != nil {
			log.Fatalf("Error auto-detecting network interface: %v", err)
		}

		if linkDevice == "" {
			log.Fatal("No suitable network interface found for auto-detection")
		}

		log.Printf("Auto-detected fallback interface: %s", linkDevice)

		// Update the global config and retry
		global.Cfg().LinkDevice = linkDevice

		// If restricted CIDR is specified, use it
		restrictedCIDR := global.Cfg().RestrictedCIDR
		if restrictedCIDR != "" {
			log.Printf("Using restricted CIDR range: %s", restrictedCIDR)
			ipmgr.DefaultCIDR = ipmgr.StringToCIDR(restrictedCIDR)
			ipmgr.DefaultCIDR.LinkDevice = linkDevice
		} else {
			ipmgr.DefaultCIDR = ipmgr.LinkDefaultCIDR(linkDevice)
		}

		if ipmgr.DefaultCIDR == nil {
			log.Fatalf("Auto-detected interface '%s' also has no IPv4 addresses", linkDevice)
		}
	}

	ipmgr.IP = ipmgr.DefaultCIDR.IP
	ipmgr.Bits = ipmgr.DefaultCIDR.Bits
	ipmgr.LinkDevice = linkDevice
	ipmgr.Debug = global.Cfg().Debug

	// Initialize IP pool from the CIDR range
	var err error
	ipmgr.IPPoolInstance, err = ipmgr.NewIPPool(ipmgr.DefaultCIDR.String())
	if err != nil {
		log.Fatalf("Failed to create IP pool from CIDR %s: %v", ipmgr.DefaultCIDR.String(), err)
	}
	log.Printf("Initialized IP pool for CIDR: %s", ipmgr.DefaultCIDR.String())
}
