package main

import (
	"fmt"
	"os"

	"github.com/vishvananda/netlink"
)

// const (
// 	FAMILY_ALL  = nl.FAMILY_ALL
// 	FAMILY_V4   = nl.FAMILY_V4
// 	FAMILY_V6   = nl.FAMILY_V6
// 	FAMILY_MPLS = nl.FAMILY_MPLS
// )
func main() {
	if links, err := netlink.LinkList(); err != nil {
		fmt.Println(err)
	} else {
		for i, link := range links {
			fmt.Println(i, link)
		}
	}
	ether, _ := netlink.LinkByName("wlan0")
	if addrs, err := netlink.AddrList(ether, netlink.FAMILY_ALL); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	} else {
		for i, addr := range addrs {
			fmt.Printf("%d addr %v\n", i, addr)
		}
	}

	if addrs, err := netlink.AddrList(ether, netlink.FAMILY_V4); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	} else {
		for i, addr := range addrs {
			fmt.Printf("IPv4 %d addr %v\n", i, addr)
		}
	}
}
