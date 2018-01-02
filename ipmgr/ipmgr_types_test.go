package ipmgr

type TestCIDRType struct {
	IPCidrStr string
	Device    string
	CIDRType  *CIDR
	IPnet     string
	Mask      string
	Nil       bool
}

var testCIDRType = []TestCIDRType{

	{
		IPCidrStr: "127.0.0.224/24",
		Device:    "eth0",
		CIDRType:  &CIDR{IP: "127.0.0.224", Bits: "24"},
		IPnet:     "127.0.0.0",
		Mask:      "255.255.255.0",
		Nil:       false,
	},
	{
		IPCidrStr: "127.0.0.225/16",
		Device:    "eth0",
		CIDRType:  &CIDR{IP: "127.0.0.225", Bits: "16"},
		IPnet:     "127.0.0.0",
		Mask:      "255.255.0.0",
		Nil:       false,
	},
	{
		IPCidrStr: "127.0.0.225/22",
		Device:    "eth0",
		CIDRType:  &CIDR{IP: "127.0.0.225", Bits: "22"},
		IPnet:     "192.168.1.0",
		Mask:      "255.255.252.0",
		Nil:       false,
	},
	{
		IPCidrStr: "127.0.0.2/24",
		Device:    "eth0",
		CIDRType:  &CIDR{IP: "127.0.0.2", Bits: "24"},
		IPnet:     "127.0.0.0",
		Mask:      "255.255.255.0",
		Nil:       false,
	},
	{
		IPCidrStr: "127.0.0.224",
		Device:    "eth0",
		CIDRType:  nil,
		IPnet:     "",
		Mask:      "255.255.255.0",
		Nil:       true,
	},
}

var testLBIps = make(LoadBalancerIPs)
