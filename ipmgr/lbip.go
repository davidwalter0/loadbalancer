package ipmgr

import (
	"fmt"

	"github.com/davidwalter0/go-mutex"
	"github.com/vishvananda/netlink"
)

var _InTest_ bool

// const (
// 	FAMILY_ALL  = nl.FAMILY_ALL
// 	FAMILY_V4   = nl.FAMILY_V4
// 	FAMILY_V6   = nl.FAMILY_V6
// 	FAMILY_MPLS = nl.FAMILY_MPLS
// )
// type IPNet net.IPNet

var monitor = mutex.NewMonitor()

// LoadBalancerIPs load balancer IPs
type LoadBalancerIPs map[string]*netlink.Addr

// RemoveAddr from networks
func (mips *LoadBalancerIPs) RemoveAddr(IPNet string) {
	defer monitor()()
	if addr, ok := (*mips)[IPNet]; ok {
		link, _ := netlink.LinkByName(addr.Label)
		if link != nil && !_InTest_ {

			netlink.AddrDel(link, addr)
		}
		delete(*mips, IPNet)
	}
}

// AddAddr adds an address to a network device
func (mips *LoadBalancerIPs) AddAddr(IPNet, device string) {
	defer monitor()()
	if addr, ok := (*mips)[IPNet]; !ok {
		link, _ := netlink.LinkByName(device)
		if link != nil {
			addr, _ = netlink.ParseAddr(IPNet)
			if addr != nil && !_InTest_ {
				if err := netlink.AddrAdd(link, addr); err == nil {
					(*mips)[IPNet] = addr
				}
			}
		}
	}
}

// keys of managed ip map | not thread safe
func (mips *LoadBalancerIPs) keys() (IPNets []string) {
	for key := range *mips {
		IPNets = append(IPNets, key)
	}
	return
}

// Keys of managed ip map | thread safe
func (mips *LoadBalancerIPs) Keys() (IPNets []string) {
	defer monitor()()
	return mips.keys()
}

// String from managed ip map | thread safe
func (mips *LoadBalancerIPs) String() string {
	defer monitor()()
	return fmt.Sprintf("%v", mips.keys())
}
