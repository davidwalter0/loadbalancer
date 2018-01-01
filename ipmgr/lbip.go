package ipmgr

import (
	"fmt"

	"github.com/davidwalter0/go-mutex"
	"github.com/vishvananda/netlink"
)

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
		netlink.AddrDel(link, addr)
		delete(*mips, IPNet)
	}
}

// AddAddr adds an address to a network device
func (mips *LoadBalancerIPs) AddAddr(IPNet, device string) {
	defer monitor()()
	if addr, ok := (*mips)[IPNet]; !ok {
		link, _ := netlink.LinkByName(device)
		addr, _ = netlink.ParseAddr(IPNet)
		netlink.AddrAdd(link, addr)
		(*mips)[IPNet] = addr
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

// LinkList from netlink
func LinkList() ([]netlink.Link, error) {
	return netlink.LinkList()
}

// LinkNames list all network device names
func LinkNames() (names []string) {
	if links, err := LinkList(); err == nil {
		for _, link := range links {
			names = append(names, link.Attrs().Name)
		}
	}
	return
}

// LinkAddrList of all devices for all families
func LinkAddrList() (Addrs []netlink.Addr) {
	if links, err := LinkList(); err == nil {
		for _, linkName := range links {
			link, _ := netlink.LinkByName(linkName.Attrs().Name)
			if addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL); err == nil {
				for _, addr := range addrs {
					Addrs = append(Addrs, addr)
				}
			}
		}
	}
	return
}

// LinkAddrListByName addresses for one device for all address families
func LinkAddrListByName(linkName string) (Addrs []netlink.Addr) {
	if link, err := netlink.LinkByName(linkName); err == nil {
		if addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL); err == nil {
			for _, addr := range addrs {
				Addrs = append(Addrs, addr)
			}
		}
	} else {
		fmt.Println(err)
	}
	return
}

// LinkIPv4AddrListByName addresses for one device for ipv4
func LinkIPv4AddrListByName(linkName string) (Addrs []netlink.Addr) {
	if link, err := netlink.LinkByName(linkName); err == nil {
		if addrs, err := netlink.AddrList(link, netlink.FAMILY_V4); err == nil {
			for _, addr := range addrs {
				Addrs = append(Addrs, addr)
			}
		}
	} else {
		fmt.Println(err)
	}
	return
}

// LinkIPv6AddrListByName addresses for one device for ipv6
func LinkIPv6AddrListByName(linkName string) (Addrs []netlink.Addr) {
	if link, err := netlink.LinkByName(linkName); err == nil {
		if addrs, err := netlink.AddrList(link, netlink.FAMILY_V6); err == nil {
			for _, addr := range addrs {
				Addrs = append(Addrs, addr)
			}
		}
	} else {
		fmt.Println(err)
	}
	return
}

func DevCIDR(dev string) (c *CIDR) {
	for i, addr := range LinkIPv4AddrListByName(dev) {
		if i == 0 {
			c = Addr2CIDR(addr)
			break
		}
	}
	return
}
