package main

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

func main() {
	var m ManagedIP = make(ManagedIP)
	var list = []string{"192.168.0.129/24", "192.168.0.130/24", "192.168.0.131/24"}
	for _, IPNet := range list {
		m[IPNet], _ = netlink.ParseAddr(IPNet)
	}
	fmt.Println(m.String())
	fmt.Println(m)
	// link, _ := netlink.LinkByName("wlan0")
	// addr, _ := netlink.ParseAddr("192.168.0.129/24")
	// netlink.AddrAdd(link, addr)

	linkNames := LinkNames()
	fmt.Println(linkNames)
	for _, name := range linkNames {
		fmt.Printf("%s\n", name)
		for i, addr := range LinkIPv4AddrListByName(name) {
			fmt.Printf("  %3d %v\n", i, addr)
		}
		for i, addr := range LinkIPv6AddrListByName(name) {
			fmt.Printf("  %3d %v\n", i, addr)
		}
	}
}
