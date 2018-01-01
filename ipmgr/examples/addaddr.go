package main

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

func main() {
	if ether, err := netlink.LinkByName("wlan0"); err == nil {
		if cidrAddr, err := netlink.ParseAddr("192.168.0.129/24"); err == nil {
			netlink.AddrAdd(ether, cidrAddr)
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Println(err)
	}
	if ether, err := netlink.LinkByName("wlan0"); err == nil {
		if cidrAddr, err := netlink.ParseAddr("192.168.0.131"); err == nil {
			netlink.AddrAdd(ether, cidrAddr)
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Println(err)
	}
}
