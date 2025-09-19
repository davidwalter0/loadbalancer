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

package ipmgr

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestAddAddr(t *testing.T) {
	// disabled: future test validations in a safe environment use of
	// existing networks may break connectivity
	if false {
		assert := assert.New(t)
		assert.Equal(len(testLBIps), 0)
		_InTest_ = true
		var c *CIDR
		for _, next := range testCIDRType {
			c = StringToCIDR(next.IPCidrStr)
			switch next.Nil {
			case true:
				assert.Nil(c)
			case false:
				if assert.NotNil(c) {
					testLBIps.AddAddr(next.IPCidrStr, next.LinkDevice)
				}
			}
		}
		assert.NotEqual(0, len(testLBIps))
		assert.Equal(4, len(testLBIps))
	}
}

func TestRemoveAddr(t *testing.T) {
	// disabled: future test validations in a safe environment use of
	// existing networks may break connectivity
	if false {
		assert := assert.New(t)
		assert.Equal(4, len(testLBIps))
		_InTest_ = true
		var c *CIDR
		for _, next := range testCIDRType {
			c = StringToCIDR(next.IPCidrStr)
			switch next.Nil {
			case true:
				assert.Nil(c)
			case false:
				if assert.NotNil(c) {
					testLBIps.RemoveAddr(next.IPCidrStr, next.LinkDevice)
				}
			}
		}
		assert.Equal(0, len(testLBIps))
	}
}

func TestLoadBalancerIPsAddRemove(t *testing.T) {
	// Set up test environment
	_InTest_ = true
	Debug = true
	
	// Create a new LoadBalancerIPs
	lbIPs := make(LoadBalancerIPs)
	
	// Test adding an IP
	cidr := "192.168.1.100/24"
	device := "eth0"
	
	lbIPs.AddAddr(cidr, device)
	
	// Verify IP was added
	assert.Equal(t, 1, len(lbIPs), "Expected 1 IP in the map")
	_, exists := lbIPs[cidr]
	assert.True(t, exists, "Expected IP to be in the map")
	
	if exists {
		assert.Equal(t, 1, lbIPs[cidr].Count, "Expected reference count to be 1")
	}
	
	// Test adding the same IP again
	lbIPs.AddAddr(cidr, device)
	
	// Verify reference count was incremented
	assert.Equal(t, 1, len(lbIPs), "Expected 1 IP in the map after adding same IP")
	if _, exists := lbIPs[cidr]; exists {
		assert.Equal(t, 2, lbIPs[cidr].Count, "Expected reference count to be 2")
	}
	
	// Test removing the IP once
	lbIPs.RemoveAddr(cidr, device)
	
	// Verify reference count was decremented but IP not removed
	assert.Equal(t, 1, len(lbIPs), "Expected IP to still be in map after first removal")
	if _, exists := lbIPs[cidr]; exists {
		assert.Equal(t, 1, lbIPs[cidr].Count, "Expected reference count to be 1 after first removal")
	}
	
	// Test removing the IP again
	lbIPs.RemoveAddr(cidr, device)
	
	// Verify IP was removed
	assert.Equal(t, 0, len(lbIPs), "Expected IP to be removed after second removal")
	_, exists = lbIPs[cidr]
	assert.False(t, exists, "Expected IP to not be in the map")
}

func TestLoadBalancerIPsKeys(t *testing.T) {
	// Set up test environment
	_InTest_ = true
	Debug = true
	
	// Create a new LoadBalancerIPs
	lbIPs := make(LoadBalancerIPs)
	
	// Add multiple IPs
	lbIPs.AddAddr("192.168.1.100/24", "eth0")
	lbIPs.AddAddr("192.168.1.101/24", "eth0")
	lbIPs.AddAddr("192.168.1.102/24", "eth0")
	
	// Test Keys method
	keys := lbIPs.Keys()
	
	// Sort keys for predictable comparison
	sort.Strings(keys)
	
	// Verify keys
	assert.Equal(t, 3, len(keys), "Expected 3 keys")
	assert.Equal(t, "192.168.1.100/24", keys[0], "Expected first key to be 192.168.1.100/24")
	assert.Equal(t, "192.168.1.101/24", keys[1], "Expected second key to be 192.168.1.101/24")
	assert.Equal(t, "192.168.1.102/24", keys[2], "Expected third key to be 192.168.1.102/24")
}

func TestLoadBalancerIPsString(t *testing.T) {
	// Set up test environment
	_InTest_ = true
	Debug = true
	
	// Create a new LoadBalancerIPs
	lbIPs := make(LoadBalancerIPs)
	
	// Test empty map
	assert.Equal(t, "[]", lbIPs.String(), "Expected empty array string for empty map")
	
	// Add an IP
	lbIPs.AddAddr("192.168.1.100/24", "eth0")
	
	// Test with one IP
	assert.Contains(t, lbIPs.String(), "192.168.1.100/24", "Expected string to contain the IP")
}

func TestLoadBalancerIPsInvalidIP(t *testing.T) {
	// Set up test environment
	_InTest_ = true
	Debug = true
	
	// Create a new LoadBalancerIPs
	lbIPs := make(LoadBalancerIPs)
	
	// Test adding an invalid IP
	invalidCIDR := "invalid/24"
	device := "eth0"
	
	lbIPs.AddAddr(invalidCIDR, device)
	
	// Verify IP was not added
	assert.Equal(t, 0, len(lbIPs), "Expected invalid IP to not be added")
	
	// Test removing an invalid IP
	lbIPs.RemoveAddr(invalidCIDR, device)
	
	// Verify no errors occurred
	assert.Equal(t, 0, len(lbIPs), "Expected no change after removing invalid IP")
}

func TestLoadBalancerIPsReferenceCount(t *testing.T) {
	// Set up test environment
	_InTest_ = true
	Debug = true
	
	// Create a new LoadBalancerIPs
	lbIPs := make(LoadBalancerIPs)
	
	// Add the same IP multiple times
	cidr := "192.168.1.100/24"
	device := "eth0"
	
	for i := 0; i < 5; i++ {
		lbIPs.AddAddr(cidr, device)
	}
	
	// Verify reference count
	assert.Equal(t, 1, len(lbIPs), "Expected 1 IP in the map")
	if _, exists := lbIPs[cidr]; exists {
		assert.Equal(t, 5, lbIPs[cidr].Count, "Expected reference count to be 5")
	}
	
	// Remove the IP multiple times
	for i := 0; i < 4; i++ {
		lbIPs.RemoveAddr(cidr, device)
	}
	
	// Verify IP still exists with reference count 1
	assert.Equal(t, 1, len(lbIPs), "Expected IP to still be in map")
	if _, exists := lbIPs[cidr]; exists {
		assert.Equal(t, 1, lbIPs[cidr].Count, "Expected reference count to be 1")
	}
	
	// Remove the IP one more time
	lbIPs.RemoveAddr(cidr, device)
	
	// Verify IP was removed
	assert.Equal(t, 0, len(lbIPs), "Expected IP to be removed")
}
