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
