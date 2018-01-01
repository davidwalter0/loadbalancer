package main

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

type ManagedIP map[string]*netlink.Addr

func (mips *ManagedIP) RemoveAddr(IPNet string) {
	defer monitor()()
	if addr, ok := (*mips)[IPNet]; ok {
		link, _ := netlink.LinkByName(addr.Label)
		netlink.AddrDel(link, addr)
		delete(*mips, IPNet)
	}
}

func (mips *ManagedIP) AddAddr(IPNet, device string) {
	defer monitor()()
	if addr, ok := (*mips)[IPNet]; !ok {
		link, _ := netlink.LinkByName(device)
		addr, _ = netlink.ParseAddr(IPNet)
		netlink.AddrAdd(link, addr)
		(*mips)[IPNet] = addr
	}
}

func (mips *ManagedIP) keys() (IPNets []string) {
	for key := range *mips {
		IPNets = append(IPNets, key)
	}
	return
}

func (mips *ManagedIP) Keys() (IPNets []string) {
	defer monitor()()
	return mips.keys()
}

func (mips *ManagedIP) String() string {
	defer monitor()()
	return fmt.Sprintf("%v", mips.keys())
}

func LinkList() ([]netlink.Link, error) {
	return netlink.LinkList()
}

func LinkNames() (names []string) {
	if links, err := LinkList(); err == nil {
		for _, link := range links {
			names = append(names, link.Attrs().Name)
		}
	}
	return
}

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
