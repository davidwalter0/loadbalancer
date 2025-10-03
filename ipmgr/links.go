/*

Copyright 2018-2025 David Walter.

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
	"log"

	"github.com/vishvananda/netlink"
)

// LinkList array of ether devices from netlink lib
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
		log.Println(err)
	}
	return
}

// LinkIPv4AddrListByName addresses for one device for ipv4
func LinkIPv4AddrListByName(linkName string) (Addrs []netlink.Addr) {
	if Debug {
		log.Println(linkName)
	}
	if link, err := netlink.LinkByName(linkName); err == nil {
		if Debug {
			log.Println(link, err)
		}
		if addrs, err := netlink.AddrList(link, netlink.FAMILY_V4); err == nil {
			if Debug {
				log.Println(addrs, err)
			}
			for _, addr := range addrs {
				if Debug {
					log.Println(addr)
				}
				Addrs = append(Addrs, addr)
			}
		}
	} else {
		log.Println(err)
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
		log.Println(err)
	}
	return
}

// LinkDefaultAddr assume that the first address is canonical for a device
// and shouldn't be modified, removed and describes the subnet
// bits/routes
func LinkDefaultAddr(link string) *netlink.Addr {
	for _, addr := range LinkIPv4AddrListByName(link) {
		return &addr
	}
	return nil
}

// LinkDefaultCIDR assume that the first address is canonical for a
// device and shouldn't be modified, removed and describes the subnet
// bits/routes
func LinkDefaultCIDR(link string) *CIDR {
	for _, addr := range LinkIPv4AddrListByName(link) {
		return Addr2CIDR(addr)
	}
	return nil
}
