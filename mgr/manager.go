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

// Package mgr manages listeners for each LoadBalancer definition
// configured in the kubernetes services
package mgr

import (
	"log"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/nodemgr"
	"github.com/davidwalter0/loadbalancer/share"
	"github.com/davidwalter0/loadbalancer/tracer"
	"github.com/davidwalter0/loadbalancer/watch"
)

var managedLBIPs ipmgr.LoadBalancerIPs = make(ipmgr.LoadBalancerIPs)
var shutdown = make(chan struct{})

// retries number of attempts
var retries = 1

// logStatusTimeout in seconds
var logStatusTimeout = time.Duration(600)

// Mgr management info for listeners
type Mgr struct {
	Listeners       map[string]*ManagedListener
	Mutex           *mutex.Mutex
	EnvCfg          *share.ServerCfg
	NodeWatcher     *watch.QueueMgr
	ServiceWatcher  *watch.QueueMgr
	EndpointWatcher *watch.QueueMgr
	*kubernetes.Clientset
}

// NewMgr create a new Mgr
func NewMgr(EnvCfg *share.ServerCfg, Clientset *kubernetes.Clientset) *Mgr {
	return &Mgr{
		Listeners: make(map[string]*ManagedListener),
		Mutex:     &mutex.Mutex{},
		EnvCfg:    EnvCfg,
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

// Get a listener by key
func (mgr *Mgr) Get(Key string) (ml *ManagedListener, ok bool) {
	defer trace.Tracer.ScopedTrace()()
	defer mgr.Mutex.MonitorTrace("Get")()
	ml, ok = mgr.Listeners[Key]
	return
}

// GetCreate returns  a listener by key
func (mgr *Mgr) GetCreate(Key string, Service *v1.Service, mlp **ManagedListener, created *bool) (ml *ManagedListener) {
	defer trace.Tracer.ScopedTrace()()
	defer mgr.Mutex.MonitorTrace("Get")()
	var ok bool
	if ml, ok = mgr.Listeners[Key]; !ok {
		ml = NewManagedListener(Service, mgr.EnvCfg, mgr.Clientset)
		mgr.Listeners[Key] = ml
		*created = true
	}
	*mlp = ml
	return
}

// Get a listener by key
func (mgr *Mgr) Set(Key string, ml *ManagedListener) {
	defer trace.Tracer.ScopedTrace()()
	defer mgr.Mutex.MonitorTrace("Set")()
	mgr.Listeners[Key] = ml
}

// Run primary processing loop
func (mgr *Mgr) Run() {
	log.Println("LinkDefaultCIDR", ipmgr.DefaultCIDR)

	listOpts := &metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker", IncludeUninitialized: false}

	mgr.NodeWatcher = watch.NewQueueMgrListOpt(watch.NodeAPIName, mgr.Clientset, listOpts)
	mgr.ServiceWatcher = watch.NewQueueMgr(watch.ServiceAPIName, mgr.Clientset)
	mgr.EndpointWatcher = watch.NewQueueMgr(watch.EndpointAPIName, mgr.Clientset)

	go mgr.NodeWatch()
	go mgr.ServiceWatch()
	if InCluster() {
		go mgr.EndpointWatch()
	}

	select {
	case <-shutdown:
	}
}

// Close removes a pipe definition
func (mgr *Mgr) Close(Key string) {
	defer trace.Tracer.ScopedTrace()()
	defer mgr.Mutex.MonitorTrace("Close")()

	if ml, ok := mgr.Listeners[Key]; ok {
		mgr.Listeners[Key].Close()
		managedLBIPs.RemoveAddr(ml.CIDR.String(), ml.CIDR.LinkDevice)
		delete(mgr.Listeners, Key)
	}
}

// NodeWatch manage node workers list dynamically
func (mgr *Mgr) NodeWatch() {
	nodeList := nodemgr.NodeListPtr()
	go mgr.NodeWatcher.Run(1, 10)
	for {
		select {
		case <-shutdown:
			log.Println("NodeWatch shutting down...")
			return
		case item, ok := <-mgr.NodeWatcher.QueueItems:
			if ok {
				Node := item.Interface.(*v1.Node)
				switch item.EventType {
				case watch.ADD:
					log.Printf("NodeWatcher Event %s for node %s with type %s\n", item.Key, Node.Name, item.EventType)
					nodeList.AddNode(Node.Spec.ExternalID)
				// case watch.UPDATE:
				// 	// Expect that nodes can't change their ip address w/o
				// 	// destroy / create, ignore UPDATE for now
				case watch.DELETE:
					log.Printf("NodeWatcher Event %s for node %s with type %s\n", item.Key, Node.Name, item.EventType)
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
	go mgr.ServiceWatcher.Run(1, 5)
	for {
		select {
		case <-shutdown:
			log.Println("ServiceWatch shutting down...")
			return
		case item, ok := <-mgr.ServiceWatcher.QueueItems:
			if ok {
				switch item.EventType {
				case watch.ADD:
					log.Printf("ServiceWatcher Event %s for type %s\n", item.Key, item.EventType)
					fallthrough
				case watch.UPDATE:
					Service := item.Interface.(*v1.Service)
					if mgr.EnvCfg.Debug {
						log.Printf("ServiceWatcher Event %s for service %s with type %s\n", item.Key, Service.Spec.Type, item.EventType)
					}
					switch Service.Spec.Type {
					case v1.ServiceTypeLoadBalancer:
						go mgr.Listen(Service)
					default:
						mgr.Close(item.Key)
					}
				case watch.DELETE:
					log.Printf("ServiceWatcher Event %s for type %s\n", item.Key, item.EventType)
					mgr.Close(item.Key)
				}
			} else {
				log.Fatal("error in Services Channel")
			}
		}
	}
}

// SetService sets the endpoint addresses for a managed listener with
// lock
func (mgr *Mgr) SetService(Key string, Service *v1.Service) {
	if ml, ok := mgr.Get(Key); ok {
		ml.SetService(Service)
	}
}

// SetEndpoint sets the endpoint addresses for a managed listener with
// lock
func (mgr *Mgr) SetEndpoint(Key string, ep *v1.Endpoints) {
	if ml, ok := mgr.Get(Key); ok {
		ml.SetEndpoint(ep)
	}
}

// EndpointWatch watch.QueueMgr for LoadBalancers
func (mgr *Mgr) EndpointWatch() {
	go mgr.EndpointWatcher.Run(1, 1)
	for {
		select {
		case <-shutdown:
			log.Println("EndpointWatch shutting down...")
			return
		case item, ok := <-mgr.EndpointWatcher.QueueItems:
			if ok {
				if _, ok := mgr.Get(item.Key); ok {
					if mgr.EnvCfg.Debug {
						log.Printf("EndpointWatcher Event %s for type %s\n", item.Key, item.EventType)
					}
					ep := item.Interface.(*v1.Endpoints)
					switch item.EventType {
					case watch.ADD:
						log.Printf("EndpointWatcher Event %s for type %s\n", item.Key, item.EventType)
						fallthrough
					case watch.UPDATE:
						log.Printf("EndpointWatcher Event %s for type %s\n", item.Key, item.EventType)
						mgr.SetEndpoint(item.Key, ep)
					case watch.DELETE:
						log.Printf("EndpointWatcher Event %s for type %s\n", item.Key, item.EventType)
						mgr.SetEndpoint(item.Key, ep)
					}
				}
			} else {
				log.Fatal("error in Endpoints Channel")
			}
		}
	}
}
