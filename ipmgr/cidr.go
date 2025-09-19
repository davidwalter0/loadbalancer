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
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
)

// CIDR ip/bits CIDR struct
type CIDR struct {
	IP         string
	Bits       string
	LinkDevice string
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
// Returns nil if the CIDR format is invalid
func StringToCIDR(cidr string) (c *CIDR) {
	// Ensure valid CIDR format
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Try to add the default /32 suffix if it's just an IP
		if ip := net.ParseIP(cidr); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				cidr = cidr + "/32"
				_, ipNet, err = net.ParseCIDR(cidr)
				if err != nil {
					fmt.Printf("Error parsing CIDR %s: %v\n", cidr, err)
					return nil
				}
			} else {
				// IPv6 address
				cidr = cidr + "/128"
				_, ipNet, err = net.ParseCIDR(cidr)
				if err != nil {
					fmt.Printf("Error parsing CIDR %s: %v\n", cidr, err)
					return nil
				}
			}
		} else {
			fmt.Printf("Error parsing CIDR %s: %v\n", cidr, err)
			return nil
		}
	}

	// Use normalized IP from net.ParseCIDR
	ip := ipNet.IP.String()
	ones, _ := ipNet.Mask.Size()
	mask := fmt.Sprintf("%d", ones)

	c = &CIDR{IP: ip, Bits: mask}
	fmt.Printf("Parsed CIDR %s as IP: %s, Bits: %s\n", cidr, ip, mask)
	return
}

// Addr2CIDR convert from netlink.Addr
func Addr2CIDR(addr netlink.Addr) (c *CIDR) {
	cidr := addr.IPNet.String()
	split := strings.Split(cidr, "/")
	c = &CIDR{IP: split[0], Bits: split[1], LinkDevice: addr.Label}
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
	if len(c.LinkDevice) > 0 {
		return fmt.Sprintf("%s/%s %s", c.IP, c.Bits, c.LinkDevice)
	} else {
		return fmt.Sprintf("%s/%s", c.IP, c.Bits)
	}
}
