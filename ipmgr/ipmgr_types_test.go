/*

Copyright 2018 David Walter.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

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
