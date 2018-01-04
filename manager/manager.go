// Package mgr manages listeners for each forwarding pipe definition
package mgr

import (
	"fmt"
	"log"
	"net"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/llb/ipmgr"
	"github.com/davidwalter0/llb/listener"
	"github.com/davidwalter0/llb/share"
	"github.com/davidwalter0/llb/tracer"
	"github.com/davidwalter0/llb/watch"
)

var managedLBIPs ipmgr.LoadBalancerIPs = make(ipmgr.LoadBalancerIPs)

// retries number of attempts
var retries = 5

// logStatusTimeout in seconds
var logStatusTimeout = time.Duration(600)

// Mgr management info for listeners
type Mgr struct {
	Listeners map[string]*listener.ManagedListener
	Mutex     *mutex.Mutex
	EnvCfg    *share.ServerCfg
	*kubernetes.Clientset
	InCluster bool
}

// NewMgr create a new Mgr
func NewMgr(EnvCfg *share.ServerCfg, Clientset *kubernetes.Clientset) *Mgr {
	return &Mgr{
		Listeners: make(map[string]*listener.ManagedListener),
		Mutex:     &mutex.Mutex{},
		EnvCfg:    EnvCfg,
		Clientset: Clientset,
		InCluster: false,
	}
}

// Monitor lifts mutex deferable lock to Mgr object
func (mgr *Mgr) Monitor() func() {
	defer trace.Tracer.ScopedTrace()()
	return mgr.Mutex.MonitorTrace()
}

// Run primary processing loop
func (mgr *Mgr) Run() {
	serviceWatcher := watch.NewQueueMgr(watch.ServiceAPIName, mgr.Clientset)
	go serviceWatcher.Run()

	for {
		select {
		case item, ok := <-serviceWatcher.QueueItems:
			if !ok {
				log.Fatal("error in Services Channel")
			} else {
				switch item.EventType {
				case watch.ADD:
					fallthrough
				case watch.UPDATE:
					Service := item.Interface.(*v1.Service)
					fmt.Printf("Event %s for service %s with type %s\n", item.Key, Service.Spec.Type, item.EventType)
					switch Service.Spec.Type {
					case "LoadBalancer":
						mgr.UpdatePipe(Service)
					default:
						mgr.RemovePipe(item.Key)
					}
				case watch.DELETE:
					mgr.RemovePipe(item.Key)
				}
			}
		}
	}
}

// RemovePipe adds/updates or removes a pipe definition
func (mgr *Mgr) RemovePipe(Key string) {
	defer trace.Tracer.ScopedTrace()()
	defer mgr.Mutex.MonitorTrace("Remove")()

	if listener, ok := mgr.Listeners[Key]; ok {
		mgr.Listeners[Key].Close()
		managedLBIPs.RemoveAddr(listener.CIDR.String(), listener.CIDR.LinkDevice)
		delete(mgr.Listeners, Key)
	}
}

// UpdatePipe adds/update a pipe definition, creates Managed
// Listeners, IPs for load balancers with specified external ports
func (mgr *Mgr) UpdatePipe(Service *v1.Service) {
	defer mgr.Mutex.MonitorTrace("Update")()
	defer trace.Tracer.ScopedTrace()()
	var Key = ServiceKey(Service)
	if current, ok := mgr.Listeners[Key]; !ok {
		managedListener := NewManagedListenerFromV1Service(Service, mgr.EnvCfg, mgr.Clientset)
		managedLBIPs.AddAddr(managedListener.CIDR.String(), managedListener.CIDR.LinkDevice)
		managedListener.Listener = Listen(ServiceSource(Service))
		mgr.Listeners[Key] = managedListener
		mgr.Listeners[Key].Open()
	} else {
		managedListener := NewManagedListenerFromV1Service(Service, mgr.EnvCfg, mgr.Clientset)
		if !managedListener.Equal(current) {
			mgr.Listeners[Key].Close()
			managedLBIPs.RemoveAddr(current.CIDR.String(), current.CIDR.LinkDevice)
			mgr.Listeners[Key] = managedListener
			managedLBIPs.AddAddr(managedListener.CIDR.String(), managedListener.CIDR.LinkDevice)
			managedListener.Listener = Listen(ServiceSource(Service))
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
