package mgr

import (
	"fmt"
	"log"
	"net"
	"time"

	"k8s.io/api/core/v1"

	"github.com/davidwalter0/loadbalancer/helper"

	"github.com/cenkalti/backoff"
)

// Listen creates a listener and retries until the listener works with
// exponential backoff
func (mgr *Mgr) Listen(Service *v1.Service) {

	var serviceKey = helper.ServiceKey(Service)
	var managedListener *ManagedListener
	var ServicePrefix = "Service: " + serviceKey
	var created bool

	mgr.GetCreate(serviceKey, Service, &managedListener, &created)
	if !created && time.Now().Sub(managedListener.Create) < 5*time.Second {
		log.Printf("%s skip listener create", ServicePrefix)
		return
	}

	var create = created || Service.Spec.LoadBalancerIP != managedListener.Service.Spec.LoadBalancerIP

	if create {
		log.Printf("%s listener create start", ServicePrefix)
		managedListener.Close()
		managedLBIPs.RemoveAddr(managedListener.CIDR.String(), managedListener.CIDR.LinkDevice)
		managedLBIPs.AddAddr(managedListener.CIDR.String(), managedListener.CIDR.LinkDevice)
		managedListener.Listener = Listen(serviceKey, helper.ServiceSource(Service))
		go managedListener.Open()
		log.Printf("%s listener created %v", ServicePrefix, managedListener.Listener.Addr())
	}

}

// Listen open listener on address
func Listen(serviceKey, address string) (listener net.Listener) {

	var ServicePrefix = "Service: " + serviceKey
	var err error

	Try := func() (err error) {
		if listener, err = net.Listen("tcp", address); err != nil {
			err = fmt.Errorf("%s net.Listen(tcp, %s) %v %v",
				ServicePrefix,
				address,
				err,
				listener,
			)
		}
		return
	}

	ExpBackoff := ConfigureBackoff(10*time.Second, 1*time.Minute, 3*time.Minute)

	Notify := func(err error, t time.Duration) {
		log.Printf("%v started %s elapsed %s break after %s",
			err,
			ExpBackoff.StartTime().Format("15.04.999"),
			DurationString(ExpBackoff.GetElapsedTime()),
			DurationString(ExpBackoff.MaxElapsedTime))
	}

	for {
		if err = backoff.RetryNotify(Try, ExpBackoff, Notify); err != nil {
			log.Printf("%s Listen retry timeout: %v, %v", ServicePrefix, err, listener)
		} else {
			log.Printf("%s listener created %v %v", ServicePrefix, err, listener)
			break
		}
	}
	return
}
