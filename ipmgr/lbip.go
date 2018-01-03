package ipmgr

import (
	"fmt"
	"log"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/llb/global"
	"github.com/davidwalter0/llb/tracer"

	"github.com/vishvananda/netlink"
)

var _InTest_ bool

var monitor = mutex.NewMonitor()

// LinkAddr manage links track in use Count
type LinkAddr struct {
	*netlink.Addr
	Count int
}

// LoadBalancerIPs load balancer IPs
type LoadBalancerIPs map[string]*LinkAddr

// AddAddr adds an address to a network LinkDevice
func (mips *LoadBalancerIPs) AddAddr(IPNet, LinkDevice string) {
	log.Printf("AddAddr %v %v\n", IPNet, LinkDevice)
	defer monitor()()
	defer trace.Tracer.ScopedTrace()()
	for key := range *mips {
		fmt.Println(key)
	}

	if linkAddr, ok := (*mips)[IPNet]; !ok {
		if link, err := netlink.LinkByName(LinkDevice); err == nil {
			if global.Cfg().Debug {
				fmt.Printf("AddAddr %v %v link: %v\n", IPNet, LinkDevice, link)
			}
			if addr, err := netlink.ParseAddr(IPNet); err == nil {
				linkAddr = &LinkAddr{Addr: addr, Count: 1}
				if global.Cfg().Debug {
					fmt.Printf("AddAddr %v %v LinkAddr: %v\n", IPNet, LinkDevice, *linkAddr)
				}
				if !_InTest_ {
					if err := netlink.AddrAdd(link, addr); err == nil {
						(*mips)[IPNet] = linkAddr
					} else {
						if global.Cfg().Debug {
							fmt.Println("Warning: managing existing ip", IPNet, LinkDevice)
							fmt.Println(err)
						}
						(*mips)[IPNet] = linkAddr
					}
				}
				if global.Cfg().Debug {
					fmt.Printf("AddAddr %v %v LinkAddr: %v Count: %d\n", IPNet, LinkDevice, *linkAddr, linkAddr.Count)
				}
			} else {
				fmt.Println(err)
			}
		} else {
			fmt.Println(err)
		}
	} else {
		linkAddr.Count++
	}
}

// RemoveAddr from networks
func (mips *LoadBalancerIPs) RemoveAddr(IPNet, LinkDevice string) {
	log.Printf("RemoveAddr %v %v\n", IPNet, LinkDevice)
	if DefaultCIDR.String() == IPNet {
		log.Printf("RemoveAddr Skips IPNet/CIDR rule %v on device %v\n", IPNet, LinkDevice)
		return
	}
	defer monitor()()
	defer trace.Tracer.ScopedTrace()()
	for key := range *mips {
		fmt.Println(key)
	}
	if linkAddr, ok := (*mips)[IPNet]; ok {
		if global.Cfg().Debug {
			fmt.Printf("RemoveAddr %v %v LinkAddr: %v Count: %d\n", IPNet, LinkDevice, *linkAddr, linkAddr.Count)
		}
		addr := linkAddr.Addr
		linkAddr.Count--
		if linkAddr.Count <= 0 {
			if global.Cfg().Debug {
				fmt.Println("RemoveAddr", addr, ok)
			}
			if link, err := netlink.LinkByName(LinkDevice); err == nil {
				if global.Cfg().Debug {
					fmt.Printf("RemoveAddr %v %v link: %v\n", IPNet, LinkDevice, link)
					fmt.Println(addr, link)
				}
				if link != nil && !_InTest_ {
					if err := netlink.AddrDel(link, addr); err != nil {
						fmt.Println(err)
					}
				}
			} else {
				fmt.Println(err)
			}
			delete(*mips, IPNet)
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
