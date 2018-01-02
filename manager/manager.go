// Package mgr manages listeners for each forwarding pipe definition
package mgr

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/llb/global"
	"github.com/davidwalter0/llb/ipmgr"
	"github.com/davidwalter0/llb/listener"
	"github.com/davidwalter0/llb/pipe"
	"github.com/davidwalter0/llb/share"
	"github.com/davidwalter0/llb/tracer"
)

// Services chan of *v1.Service to create and monitor
var Services chan *v1.Service

// RemoveServices channel of services to close
var RemovedServices chan string

// IP default link IP
var IP string

// Bits default link CIDR bits abbrev
var Bits string

// LinkDev default link device to use for external ip services
var LinkDev string

// LinkAddrs list of addresses on the default link device
var LinkAddrs = make(map[string]int)

func init() {
	c := ipmgr.LinkDefaultCIDR(global.Cfg().LinkDevice)
	IP = c.IP
	Bits = c.Bits
	LinkDev = global.Cfg().LinkDevice
	LinkAddrs[IP] = 1
	Services = make(chan *v1.Service, 100)
	RemovedServices = make(chan string, 100)
	fmt.Println("Defaults", IP, Bits, LinkDev, c)
}

// retries number of attempts
var retries = 3

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

var mgIPs ipmgr.LoadBalancerIPs = make(ipmgr.LoadBalancerIPs)

// NewPipeDefinition from a kubernetes v1.Service
func NewPipeDefinition(Service *v1.Service, envCfg *share.ServerCfg) *pipe.Definition {
	defer trace.Tracer.ScopedTrace()()
	return &pipe.Definition{
		Key:       ServiceKey(Service),
		Source:    ServiceSource(Service),
		EnableEp:  true,
		InCluster: false,
		Name:      Service.ObjectMeta.Name,
		Namespace: Service.ObjectMeta.Namespace,
		Debug:     envCfg.Debug,
		Endpoints: ServiceSinks(Service),
	}
}

// NewManagedListenerFromV1Service from a kubernetes v1.Service
func NewManagedListenerFromV1Service(Service *v1.Service,
	envCfg *share.ServerCfg,
	Clientset *kubernetes.Clientset) (ml *listener.ManagedListener) {

	ip := ServiceSourceIP(Service)
	var c *ipmgr.CIDR = &ipmgr.CIDR{IP: ip, Bits: Bits, Dev: LinkDev}
	if len(ip) > 0 {
		if count, found := LinkAddrs[ip]; !found || count == 0 {
			mgIPs.AddAddr(c.String(), c.Dev)
			LinkAddrs[ip]++
		}
	}

	ml = &listener.ManagedListener{
		Definition: *NewPipeDefinition(Service, envCfg),
		Listener:   Listen(ServiceSource(Service)),
		Pipes:      make(map[*pipe.Pipe]bool),
		Mutex:      &mutex.Mutex{},
		Kubernetes: envCfg.Kubernetes,
		MapAdd:     make(chan *pipe.Pipe, 3),
		MapRm:      make(chan *pipe.Pipe, 3),
		StopWatch:  make(chan bool, 3),
		Debug:      envCfg.Debug,
		Clientset:  Clientset,
		Active:     0,
		Interface:  Service,
		InCluster:  false,
		CIDR:       c,
	}
	return
}

// Run primary processing loop
func (mgr *Mgr) Run() {
	var EnvCfg = mgr.EnvCfg

	for {
		if EnvCfg.Debug {
			log.Println("Loop in Run()")
		}
		select {
		case Service, ok := <-Services:
			if !ok {
				log.Fatal("error in Listener Channel")
			}
			var err error
			var jsonbytes []byte
			if jsonbytes, err = json.Marshal(Service); err == nil {
				fmt.Printf("JSON for Service\nKey: %s\n%s\n",
					ServiceKey(Service),
					string(jsonbytes))

			} else {
				fmt.Println(err)
			}

			mgr.UpdatePipe(Service)
		case Service, ok := <-RemovedServices:
			if !ok {
				log.Fatal("error in Listener Channel")
			}
			mgr.RemovePipe(Service)
		case delay := <-time.After(time.Second * logStatusTimeout):
			if EnvCfg.Debug {
				log.Printf("Timed out after %d seconds %v\n", logStatusTimeout, delay)
			}
		}
	}
}

// RemovePipe adds/updates or removes a pipe definition
func (mgr *Mgr) RemovePipe(Key string) {
	defer trace.Tracer.ScopedTrace()()

	if listener, ok := mgr.Listeners[Key]; ok {
		mgr.Listeners[Key].Close()
		defer mgr.Mutex.MonitorTrace("Merge")()
		ip := listener.CIDR.IP
		LinkAddrs[ip]--
		if ip != IP && LinkAddrs[ip] == 0 {
			mgIPs.RemoveAddr(listener.CIDR.String())
			delete(mgIPs, ip)
		}
		delete(mgr.Listeners, Key)
	}
}

// UpdatePipe adds/update a pipe definition
func (mgr *Mgr) UpdatePipe(Service *v1.Service) {
	// defer mgr.Mutex.MonitorTrace("Merge")()
	// defer trace.Tracer.ScopedTrace()()
	var Key = ServiceKey(Service)
	fmt.Println("before checking for existing Key", Key)
	if rhs, ok := mgr.Listeners[Key]; !ok {
		ml := NewManagedListenerFromV1Service(Service, mgr.EnvCfg, mgr.Clientset)
		fmt.Println("Not found adding Key", Key)
		mgr.Listeners[Key] = ml
		fmt.Println("Opening Key", Key)
		mgr.Listeners[Key].Open()
	} else {
		ml := NewManagedListenerFromV1Service(Service, mgr.EnvCfg, mgr.Clientset)
		if !ml.Equal(rhs) {
			mgr.Listeners[Key].Close()
			mgr.Listeners[Key] = ml
			mgr.Listeners[Key].Open()
		}
	}
	for key, value := range mgr.Listeners {
		fmt.Printf("key %s value %v\n", key, value)
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
		listener, err = net.Listen("tcp", address)
		if err != nil {
			log.Printf("net.Listen(\"tcp\", %s ) failed: %v\n", address, err)
		} else {
			return listener
		}
	}
	return
}
