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

// SubNet returns the ip, subnetwork and error (if any) of the CIDR
func SubNet(addr *netlink.Addr) (ip net.IP, subnet *net.IPNet, err error) {
	ip, subnet, err = net.ParseCIDR(addr.IPNet.String())
	return
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

// StringToCIDR cidr ip/bits string to type CIDR
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

// IPv4CIDRString2Mask ip/bit string to subnet mask
func IPv4CIDRString2Mask(ip string) (string, error) {
	_, ipv4Net, err := net.ParseCIDR(ip)
	if err != nil {
		return "", err
	}
	return ipv4MaskString(ipv4Net.Mask), err
}

func ipv4MaskString(m []byte) string {
	if len(m) != 4 {
		panic("ipv4Mask: len must be 4 bytes")
	}

	return fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
}

// CIDR2Addr convert from CIDR to a netlink.Addr
func CIDR2Addr(c *CIDR) (addr *netlink.Addr) {
	cidr := c.IP + "/" + c.Bits
	ipv4, ipv4Net, err := net.ParseCIDR(cidr)
	if err == nil {
		ipnet := &net.IPNet{IP: ipv4, Mask: ipv4Net.Mask}
		addr = &netlink.Addr{IPNet: ipnet}
	}
	return addr
}

// CIDR2Addr convert from CIDR to a netlink.Addr
func (c *CIDR) CIDR2Addr() (addr *netlink.Addr) {
	return CIDR2Addr(c)
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
