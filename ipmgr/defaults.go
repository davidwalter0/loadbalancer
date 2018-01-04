package ipmgr

import (
	"github.com/davidwalter0/llb/global"
)

// IP default link IP
var IP string

// Bits default link CIDR bits abbrev
var Bits string

// LinkDevice default link device to use for external ip addresses
var LinkDevice string

// DefaultCIDR default link device to use for external ip addresses
var DefaultCIDR *CIDR

func init() {
	DefaultCIDR = LinkDefaultCIDR(global.Cfg().LinkDevice)
	IP = DefaultCIDR.IP
	Bits = DefaultCIDR.Bits
	LinkDevice = global.Cfg().LinkDevice
}
