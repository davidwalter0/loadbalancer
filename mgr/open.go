package mgr

import (
	"fmt"
	"log"
	"net"
	"time"

	"k8s.io/api/core/v1"

	"github.com/davidwalter0/backoff"
	"github.com/davidwalter0/loadbalancer/helper"
	"github.com/davidwalter0/loadbalancer/pipe"
)

// Listen creates a listener and retries until the listener works with
// exponential backoff
func (mgr *Mgr) Listen(Service *v1.Service) {

	var serviceKey = helper.ServiceKey(Service)
	var managedListener *ManagedListener
	var ServicePrefix = fmt.Sprintf("Service %-32.32s", serviceKey)
	var created bool

	managedListener = mgr.GetCreate(serviceKey, Service, &created)
	if !created && time.Now().Sub(managedListener.Create) < 5*time.Second {
		log.Printf("%s skip listener create", ServicePrefix)
		return
	}
	var lhs *pipe.Definition = NewPipeDefinition(Service, mgr.EnvCfg)
	var rhs *pipe.Definition = &managedListener.Definition

	if !created && lhs.Equal(rhs) {
		equal := lhs.Equal(rhs)
		log.Println(ServicePrefix, "Skip creating", equal)
		log.Println(ServicePrefix, "equal", equal)
		log.Println(ServicePrefix, "lhs", lhs)
		log.Println(ServicePrefix, "rhs", rhs)
		return
	}

	var create = created ||
		Service.Spec.LoadBalancerIP != managedListener.Service.Spec.LoadBalancerIP

	if create {
		log.Printf("%s listener create start", ServicePrefix)
		managedListener.Close()

		managedLBIPs.RemoveAddr(managedListener.CIDR.String(),
			managedListener.CIDR.LinkDevice)

		managedLBIPs.AddAddr(managedListener.CIDR.String(),
			managedListener.CIDR.LinkDevice)

		managedListener.Listener = Listen(serviceKey,
			helper.ServiceSource(Service), managedListener.Canceled)

		if managedListener.Listener != nil {
			managedListener.Open()

			log.Printf("%s listener created %v",
				ServicePrefix, managedListener.Listener.Addr())
		} else {
			log.Printf("%s listener create failed", ServicePrefix)
		}
	}

}

// Listen open listener on address
func Listen(
	serviceKey, address string, cancel chan struct{}) (
	listener net.Listener) {

	var ServicePrefix = fmt.Sprintf("Service %-32.32s", serviceKey)
	var err error

	Try := func() (err error) {
		listener, err = net.Listen("tcp", address)
		return
	}

	ExpBackoff := ConfigureBackoff(
		10*time.Second, 1*time.Minute, 3*time.Minute, cancel)

	Notify := func(err error, t time.Duration) {
		log.Printf("%v started %s elapsed %s break after %s",
			err,
			ExpBackoff.StartTime().Format("15.04.999"),
			DurationString(ExpBackoff.GetElapsedTime()),
			DurationString(ExpBackoff.MaxElapsedTime))
	}
	for {
		select {
		case <-cancel:
			log.Printf("%s Listen retry canceled by cancel", ServicePrefix)
		default:
			if err = backoff.RetryNotify(Try, ExpBackoff, Notify); err != nil {
				log.Printf("%s Listen retry timeout: %v, %v", ServicePrefix, err, listener)
			} else {
				log.Printf("%s listener created %v %v", ServicePrefix, err, listener)
				return
			}
		}
	}
}
