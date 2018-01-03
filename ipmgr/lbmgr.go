package ipmgr

import ()

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
				c.LinkDevice = link
				return link
			}
		}
	}
	return ""
}
