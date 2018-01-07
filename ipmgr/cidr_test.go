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
	"testing"
)

func initializer(c *CIDR) {
	IP = c.IP
	Bits = c.Bits
	LinkDevice = c.LinkDevice
}

func TestStringToCIDRType(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
		if c != nil {
			initializer(c)
			switch next.Nil {
			case true:
				assert.Nil(t, c)
			case false:
				if assert.NotNil(t, c) {
					assert.Equal(t, *next.CIDRType, *c)
				}
			}
		}
	}
}

func TestCIDRString2Mask(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
		if c != nil {
			initializer(c)
			m, err := IPv4CIDRString2Mask(next.IPCidrStr)
			switch next.Nil {
			case true:
				assert.Nil(t, c)
				assert.NotNil(t, err)
			case false:
				assert.Nil(t, err)
				if assert.NotNil(t, c) {
					assert.Equal(t, next.Mask, m)
				}
			}
		}
	}
}

func TestCIDRStringify(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
		if c != nil {
			initializer(c)
			switch next.Nil {
			case true:
				assert.Nil(t, c)
			case false:
				if assert.NotNil(t, c) {
					assert.Equal(t, next.IPCidrStr, c.String())
				}
			}
		}
	}
}

func TestAddr2CIDR(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
		if c != nil {
			initializer(c)
			switch next.Nil {
			case true:
				assert.Nil(t, c)
			case false:
				if assert.NotNil(t, c) {
					assert.Equal(t, next.IPCidrStr, c.String())
					addr := c.CIDR2Addr()
					if assert.NotNil(t, addr) {
						assert.Equal(t, next.IPCidrStr, addr.String())
					}
				}
			}
		}
	}
}
