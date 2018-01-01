
---
**Example golang run to use the netlink library to add an ip address**

Similar to the commands `ip addr add ...` following this source

```
package main

import (
	"github.com/vishvananda/netlink"
)

func main() {
	eth, _ := netlink.LinkByName("wlan0")
	addr, _ := netlink.ParseAddr("192.168.0.129/24")
	netlink.AddrAdd(eth, addr)
}

```

---
**multiple ip addresses for one device using ip commands**

`https://askubuntu.com/questions/547289/how-can-i-from-cli-assign-multiple-ip-addresses-to-one-interface`

```
If you need an additional IP address just for the moment you can add it to any interface on your machine with

 sudo ip address add <ip-address>/<netmask> dev <interface>
for instance

 sudo ip address add 172.16.100.17/24 dev eth0
would add 172.16.100.17 using a 24bit netmask to the list of addresses configured for your eth0.

You can check the result with

ip address show eth0
and you can delete this address again with

sudo ip address del 172.16.100.17/24 dev eth0
Of course these changes are lost when you reboot your machine.

To make the additional addresses permanent you can edit the file /etc/network/interfaces by adding as many stanzas of the form

iface eth0 static
    address 172.16.100.17/24
so that it looks like

iface eth0 inet dhcp

iface eth0 inet static
    address 172.16.100.17/24

iface eth0 inet static
    address 172.16.24.11/24
You can even keep the dhcp for the primary address.

To activate these settings without a reboot use ifdown/ifup like

sudo ifdown eth0 && sudo ifup eth0
It is essential to put those two commands into one line if you are remoting into the server because the first one will drop your connection! Given in this way the ssh-session will survive.
```
