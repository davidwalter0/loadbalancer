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

		// Skip if CIDR is empty or invalid
		if managedListener.CIDR == nil {
			log.Printf("%s CIDR is nil, cannot create listener", ServicePrefix)
			return
		}

		cidrStr := managedListener.CIDR.String()
		if cidrStr == "/" || cidrStr == "/32" || cidrStr == "/128" {
			log.Printf("%s invalid CIDR: %s, cannot create listener", ServicePrefix, cidrStr)
			return
		}

		// Log the CIDR being used
		log.Printf("%s using CIDR: %s on device: %s", ServicePrefix, cidrStr, managedListener.CIDR.LinkDevice)

		// Remove old IP address if it exists
		managedLBIPs.RemoveAddr(cidrStr, managedListener.CIDR.LinkDevice)

		// Add the IP address
		managedLBIPs.AddAddr(cidrStr, managedListener.CIDR.LinkDevice)

		// Create the listener
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
		log.Printf("%v elapsed %s break after %s",
			err,
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
