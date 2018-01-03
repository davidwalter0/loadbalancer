package ipmgr

type TestCIDRType struct {
	IPCidrStr  string
	LinkDevice string
	CIDRType   *CIDR
	IPnet      string
	Mask       string
	Nil        bool
}

var testCIDRType = []TestCIDRType{

	{
		IPCidrStr:  "192.168.254.224/24",
		LinkDevice: "eth0",
		CIDRType:   &CIDR{IP: "192.168.254.224", Bits: "24"},
		IPnet:      "192.168.254.0",
		Mask:       "255.255.255.0",
		Nil:        false,
	},
	{
		IPCidrStr:  "192.168.254.225/16",
		LinkDevice: "eth0",
		CIDRType:   &CIDR{IP: "192.168.254.225", Bits: "16"},
		IPnet:      "192.168.254.0",
		Mask:       "255.255.0.0",
		Nil:        false,
	},
	{
		IPCidrStr:  "192.168.254.225/22",
		LinkDevice: "eth0",
		CIDRType:   &CIDR{IP: "192.168.254.225", Bits: "22"},
		IPnet:      "192.168.1.0",
		Mask:       "255.255.252.0",
		Nil:        false,
	},
	{
		IPCidrStr:  "192.168.254.2/24",
		LinkDevice: "eth0",
		CIDRType:   &CIDR{IP: "192.168.254.2", Bits: "24"},
		IPnet:      "192.168.254.0",
		Mask:       "255.255.255.0",
		Nil:        false,
	},
	{
		IPCidrStr:  "192.168.254.224",
		LinkDevice: "eth0",
		CIDRType:   nil,
		IPnet:      "",
		Mask:       "255.255.255.0",
		Nil:        true,
	},
}

var testLBIps = make(LoadBalancerIPs)
