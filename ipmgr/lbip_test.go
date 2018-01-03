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
