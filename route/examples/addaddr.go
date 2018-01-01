package main

import (
	"github.com/vishvananda/netlink"
)

func main() {
	ether, _ := netlink.LinkByName("wlan0")
	cidrAddr, _ := netlink.ParseAddr("192.168.0.129/24")
	netlink.AddrAdd(ether, cidrAddr)
}
