package ipmgr

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
)

// CIDR ip/bits CIDR struct
type CIDR struct {
	IP   string
	Bits string
	Dev  string
}

// MatchAddr return true when the CIDR blocks/subnets match
func (c *CIDR) MatchAddr(addr *netlink.Addr) bool {

	var err error
	var lhsIPNet, rhsIPNet *net.IPNet
	_, lhsIPNet, err = net.ParseCIDR(addr.IPNet.String())
	if err != nil {
		return false
	}
	_, rhsIPNet, err = net.ParseCIDR(c.String())
	if err != nil {
		return false
	}
	return lhsIPNet.String() == rhsIPNet.String()
}

// StringToCIDR cidr string ip/bits to CIDR type
func StringToCIDR(cidr string) (c *CIDR) {
	split := strings.Split(cidr, "/")
	if len(split) == 2 {
		c = &CIDR{IP: split[0], Bits: split[1]}
	}
	return
}

// Addr2CIDR convert from netlink.Addr
func Addr2CIDR(addr netlink.Addr) (c *CIDR) {
	cidr := addr.IPNet.String()
	split := strings.Split(cidr, "/")
	c = &CIDR{IP: split[0], Bits: split[1], Dev: addr.Label}
	return
}

// FromString ip/bits CIDR notation to CIDR
func (c *CIDR) FromString(cidr string) {
	split := strings.Split(cidr, "/")
	if len(split) == 2 {
		c.IP, c.Bits = split[0], split[1]
	}
}

// String from CIDR
func (c *CIDR) String() string {
	return fmt.Sprintf("%s/%s", c.IP, c.Bits)
}

// CIDRDevString from CIDR with device
func (c *CIDR) CIDRDevString() string {
	if len(c.Dev) > 0 {
		return fmt.Sprintf("%s/%s %s", c.IP, c.Bits, c.Dev)
	} else {
		return fmt.Sprintf("%s/%s", c.IP, c.Bits)
	}
}
