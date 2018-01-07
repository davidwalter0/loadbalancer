// Package mgr manages listeners for each forwarding pipe definition
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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/llb/helper"
	"github.com/davidwalter0/llb/ipmgr"
	"github.com/davidwalter0/llb/listener"
	"github.com/davidwalter0/llb/nodemgr"
	"github.com/davidwalter0/llb/share"
	"github.com/davidwalter0/llb/tracer"
	"github.com/davidwalter0/llb/watch"
)

var managedLBIPs ipmgr.LoadBalancerIPs = make(ipmgr.LoadBalancerIPs)
var shutdown = make(chan struct{})

// retries number of attempts
var retries = 5

// logStatusTimeout in seconds
var logStatusTimeout = time.Duration(600)

// Mgr management info for listeners
type Mgr struct {
	Listeners      map[string]*listener.ManagedListener
	Mutex          *mutex.Mutex
	EnvCfg         *share.ServerCfg
	NodeWatcher    *watch.QueueMgr
	ServiceWatcher *watch.QueueMgr
	InCluster      bool
	*kubernetes.Clientset
}

// NewMgr create a new Mgr
func NewMgr(EnvCfg *share.ServerCfg, Clientset *kubernetes.Clientset) *Mgr {
	return &Mgr{
		Listeners: make(map[string]*listener.ManagedListener),
		Mutex:     &mutex.Mutex{},
		EnvCfg:    EnvCfg,
		InCluster: false,
		Clientset: Clientset,
	}
}

// Shutdown this manager
func (mgr *Mgr) Shutdown() {
	close(shutdown)
	for key, ml := range mgr.Listeners {
		log.Println("Shutting down listener for", key)
		mgr.Close(key)
		ml.RemoveExternalIP()
	}
}

// Monitor lifts mutex deferable lock to Mgr object
func (mgr *Mgr) Monitor() func() {
	defer trace.Tracer.ScopedTrace()()
	return mgr.Mutex.MonitorTrace()
}

// Run primary processing loop
func (mgr *Mgr) Run() {
	log.Println("LinkDefaultCIDR", ipmgr.DefaultCIDR)

	listOpts := &metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker", IncludeUninitialized: false}

	mgr.NodeWatcher = watch.NewQueueMgrListOpt(watch.NodeAPIName, mgr.Clientset, listOpts)
	mgr.ServiceWatcher = watch.NewQueueMgr(watch.ServiceAPIName, mgr.Clientset)

	go mgr.NodeWatch()
	go mgr.ServiceWatch()

	select {
	case <-shutdown:
	}
}

// Close removes a pipe definition
func (mgr *Mgr) Close(Key string) {
	defer trace.Tracer.ScopedTrace()()
	defer mgr.Mutex.MonitorTrace("Remove")()

	if listener, ok := mgr.Listeners[Key]; ok {
		mgr.Listeners[Key].Close()
		managedLBIPs.RemoveAddr(listener.CIDR.String(), listener.CIDR.LinkDevice)
		delete(mgr.Listeners, Key)
	}
}

// Open adds/update a pipe definition, creates Managed
// Listeners, IPs for load balancers with specified external ports
func (mgr *Mgr) Open(Service *v1.Service) {
	defer mgr.Mutex.MonitorTrace("Update")()
	defer trace.Tracer.ScopedTrace()()
	var Key = helper.ServiceKey(Service)
	if current, ok := mgr.Listeners[Key]; !ok {
		managedListener := NewManagedListenerFromV1Service(Service, mgr.EnvCfg, mgr.Clientset)
		managedLBIPs.AddAddr(managedListener.CIDR.String(), managedListener.CIDR.LinkDevice)
		managedListener.Listener = Listen(helper.ServiceSource(Service))
		mgr.Listeners[Key] = managedListener
		mgr.Listeners[Key].Open()
	} else {
		managedListener := NewManagedListenerFromV1Service(Service, mgr.EnvCfg, mgr.Clientset)
		if !managedListener.Equal(current) {
			mgr.Listeners[Key].Close()
			managedLBIPs.RemoveAddr(current.CIDR.String(), current.CIDR.LinkDevice)
			mgr.Listeners[Key] = managedListener
			managedLBIPs.AddAddr(managedListener.CIDR.String(), managedListener.CIDR.LinkDevice)
			managedListener.Listener = Listen(helper.ServiceSource(Service))
			mgr.Listeners[Key].Open()
		}
	}
}

// Listen open listener on address
func Listen(address string) (listener net.Listener) {
	defer trace.Tracer.ScopedTrace()()
	var err error
	if false {
		defer trace.Tracer.ScopedTrace(fmt.Sprintf("listener:%v err: %v", listener, err))()
	}

	for i := 0; i < retries; i++ {
		if listener, err = net.Listen("tcp", address); err != nil {
			log.Printf("net.Listen(\"tcp\", %s ) failed: %v\n", address, err)
		} else {
			return listener
		}
		time.Sleep(time.Second * time.Duration(i))
	}
	panic("listen failed: " + address)
}

// NodeWatch manage node workers list dynamically
func (mgr *Mgr) NodeWatch() {
	nodeList := nodemgr.NodeListPtr()
	go mgr.NodeWatcher.Run(1, 1)
	for {
		select {
		case <-shutdown:
			log.Println("NodeWatch shutting down...")
			return
		case item, ok := <-mgr.NodeWatcher.QueueItems:
			if ok {
				Node := item.Interface.(*v1.Node)
				// if mgr.EnvCfg.Debug {
				// 	log.Printf("Event %s for node %s with type %s\n", item.Key, Node.Name, item.EventType)
				// }
				switch item.EventType {
				case watch.ADD:
					log.Printf("Event %s for node %s with type %s\n", item.Key, Node.Name, item.EventType)
					nodeList.AddNode(Node.Spec.ExternalID)
				// case watch.UPDATE:
				// 	// Expect that nodes can't change their ip address w/o
				// 	// destroy / create, ignore UPDATE for now
				case watch.DELETE:
					log.Printf("Event %s for node %s with type %s\n", item.Key, Node.Name, item.EventType)
					nodeList.RemoveNode(Node.Spec.ExternalID)
				}
			} else {
				log.Fatal("Error in Nodes Channel")
			}
		}
	}
}

// ServiceWatch watch.QueueMgr for LoadBalancers
func (mgr *Mgr) ServiceWatch() {
	go mgr.ServiceWatcher.Run(1, 1)
	for {
		select {
		case <-shutdown:
			log.Println("ServiceWatch shutting down...")
			return
		case item, ok := <-mgr.ServiceWatcher.QueueItems:
			if ok {
				switch item.EventType {
				case watch.ADD:
					fallthrough
				case watch.UPDATE:
					Service := item.Interface.(*v1.Service)
					if mgr.EnvCfg.Debug {
						log.Printf("Event %s for service %s with type %s\n", item.Key, Service.Spec.Type, item.EventType)
					}
					switch Service.Spec.Type {
					case "LoadBalancer":
						mgr.Open(Service)
					default:
						mgr.Close(item.Key)
					}
				case watch.DELETE:
					log.Printf("Event %s for type %s\n", item.Key, item.EventType)
					mgr.Close(item.Key)
				}
			} else {
				log.Fatal("error in Services Channel")
			}
		}
	}
}
