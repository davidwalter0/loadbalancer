package ipmgr

import ()

// var IPs LoadBalancerIPs = make(LoadBalancerIPs)

// GetEtherIfaceFromIP returns the link device name for which an
// address would be assigned
func (c *CIDR) GetEtherIfaceFromIP() string {
	for _, link := range LinkNames() {
		if addr := LinkDefaultAddr(link); addr != nil {
			if c.MatchAddr(addr) {
				return link
			}
		}
	}
	return ""
}

// SetEtherIfaceFromIP returns the link device name for which an
// address would be assigned
func (c *CIDR) SetEtherIfaceFromIP() string {
	for _, link := range LinkNames() {
		if addr := LinkDefaultAddr(link); addr != nil {
			if c.MatchAddr(addr) {
				c.Dev = link
				return link
			}
		}
	}
	return ""
}

/*
1. get load balancer ip if specified and service port
2. if --default ether is set for the load balancer use that device to
   assign new addresses
3. add loadbalancer ip to default routing interface
*/
