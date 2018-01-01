package main

import (
	"fmt"
	"net"

	"github.com/davidwalter0/llb/ipmgr"

	"github.com/vishvananda/netlink"
)

func main() {
	var m = make(ipmgr.LoadBalancerIPs)
	var list = []string{"192.168.0.129/24", "192.168.0.130/24", "192.168.0.131/24"}
	for _, IPNet := range list {
		m[IPNet], _ = netlink.ParseAddr(IPNet)
	}
	fmt.Println(m.String())
	fmt.Println(m)
	// link, _ := netlink.LinkByName("wlan0")
	// addr, _ := netlink.ParseAddr("192.168.0.129/24")
	// netlink.AddrAdd(link, addr)

	linkNames := ipmgr.LinkNames()
	fmt.Println(linkNames)

	for _, name := range linkNames {
		var devCIDR *ipmgr.CIDR = ipmgr.DevCIDR(name)
		fmt.Printf("%s\n", name)
		for i, addr := range ipmgr.LinkIPv4AddrListByName(name) {
			fmt.Printf("  %3d match %v %v %v\n", i, devCIDR, addr, devCIDR.MatchAddr(&addr))
			fmt.Printf("  %3d %v\n", i, addr)
			ip, ipnet, err := net.ParseCIDR(addr.IPNet.String())
			fmt.Printf("  %3d %v %v %v %v\n", i, addr.IPNet.String(), ip, ipnet, err)
		}
		for i, addr := range ipmgr.LinkIPv6AddrListByName(name) {
			fmt.Printf("  %3d %v %s\n", i, addr, name)
		}
	}
}
