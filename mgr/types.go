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

package mgr

import (
	"fmt"
	"log"
	"net"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/pipe"
)

// ManagedListeners manages multiple managed listeners for multiple
// ports and endpoints
type ManagedListeners struct {
	Listeners ManagedListenerMap
	Mutex     *mutex.Mutex `json:"-"`
	Debug     bool         `json:"-"`
	Clientset *kubernetes.Clientset
	Active    uint64
	Service   *v1.Service
	*ipmgr.CIDR
	IPs
	Ports
}

// ManagedListener and it's dependent objects
type ManagedListener struct {
	pipe.Definition
	Listener  net.Listener          `json:"-"`
	Pipes     map[*pipe.Pipe]bool   `json:"-"`
	Mutex     *mutex.Mutex          `json:"-"`
	Debug     bool                  `json:"-"`
	MapAdd    chan *pipe.Pipe       `json:"-"`
	MapRm     chan *pipe.Pipe       `json:"-"`
	Clientset *kubernetes.Clientset `json:"-"`
	StopWatch chan bool             `json:"-"`
	n         uint64
	Active    uint64
	Key       string
	Service   *v1.Service
	Endpoints *v1.Endpoints
	Changed   bool
	Create    time.Time
	Port
	IPs
	Ports
	*ipmgr.CIDR
}

type internalClient struct {
	*kubernetes.Clientset
	*v1.Service
}

func (i *internalClient) get() (ep *v1.Endpoints) {
	var err error
	var ns = i.Service.ObjectMeta.Namespace
	var name = i.Service.ObjectMeta.Name
	client := i.Clientset.CoreV1().Endpoints(ns)
	ep, err = client.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
	}
	return
}

// Endpoints returns *v1.Endpoints
func Endpoints(
	Clientset *kubernetes.Clientset,
	Service *v1.Service) (ep *v1.Endpoints) {

	var c = 0
	var i = internalClient{Clientset, Service}
	for ep = i.get(); ep == nil && c < 3; ep = i.get() {
		time.Sleep(time.Second)
		c++
	}
	return
}

// IP IP address
type IP string

// IPs sortable slice of IP addresses
type IPs []IP

// ManagedListenerMap of managed listeners by port
type ManagedListenerMap map[Port]*ManagedListener

// Port number
type Port int32

// PortSet set of ports
type PortSet map[Port]bool

// IPSet set of address
type IPSet map[IP]bool

// PortByIP ports for an address
type PortByIP map[IP]PortSet

// IPByPort ports for an address
type IPByPort map[Port]IPSet

// Ports sortable Port slice
type Ports []Port

func (ports Ports) Len() int           { return len(ports) }
func (ports Ports) Less(i, j int) bool { return ports[i] < ports[j] }
func (ports Ports) Swap(i, j int)      { ports[i], ports[j] = ports[j], ports[i] }

func (ips IPs) Len() int           { return len(ips) }
func (ips IPs) Less(i, j int) bool { return ips[i] < ips[j] }
func (ips IPs) Swap(i, j int)      { ips[i], ips[j] = ips[j], ips[i] }

// ToPorts from a set
func (portSet PortSet) ToPorts() (ports Ports) {
	ports = Ports{}
	for port := range portSet {
		ports = append(ports, port)
	}
	return
}

// ToIPs from a set
func (ipSet IPSet) ToIPs() (ips IPs) {
	ips = IPs{}
	for ip := range ipSet {
		ips = append(ips, ip)
	}
	return
}

// PortSet from port slice
func (ports Ports) PortSet() (ps PortSet) {
	if len(ports) > 0 {
		ps = make(PortSet)
		for _, port := range ports {
			ps[port] = true
		}
	}
	return
}

// Equal compare 2 port arrays
func (ports Ports) Equal(rhs Ports) bool {
	lhs := ports
	if lhs == nil && rhs == nil {
		return true
	}

	if lhs == nil || rhs == nil || len(lhs) != len(rhs) {
		return false
	}

	for i, n := range lhs {
		if n != rhs[i] {
			return false
		}
	}
	return true
}

// Equal compare 2 IP arrays
func (ips IPs) Equal(rhs IPs) bool {
	lhs := ips
	if lhs == nil && rhs == nil {
		return true
	}

	if lhs == nil || rhs == nil || len(lhs) != len(rhs) {
		return false
	}

	for i, n := range lhs {
		if n != rhs[i] {
			return false
		}
	}
	return true
}

// Address ip + ':' + port
func Address(ip IP, port Port) string {
	return fmt.Sprintf("%s:%d", ip, port)
}
