package ipmgr

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringToCIDRType(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
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

func TestCIDRString2Mask(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
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

func TestCIDRStringify(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
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

func TestAddr2CIDR(t *testing.T) {
	var c *CIDR
	for _, next := range testCIDRType {
		c = StringToCIDR(next.IPCidrStr)
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
