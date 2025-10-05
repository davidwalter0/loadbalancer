package mgr

import (
	"fmt"
	"log"
	"net"
	"strings"
	"syscall"
	"time"

	"k8s.io/api/core/v1"

	"github.com/davidwalter0/backoff"
	"github.com/davidwalter0/loadbalancer/helper"
	"github.com/davidwalter0/loadbalancer/ipmgr"
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

		// Create the listener with IP allocation retry
		managedListener.Listener = ListenWithIPAllocation(serviceKey, Service,
			managedListener.CIDR.LinkDevice, managedListener.Canceled)

		if managedListener.Listener != nil {
			// Update Source field to reflect actual listener address
			managedListener.Source = managedListener.Listener.Addr().String()
			managedListener.Definition.Source = managedListener.Source

			managedListener.Open()

			log.Printf("%s listener created %v",
				ServicePrefix, managedListener.Listener.Addr())
		} else {
			log.Printf("%s listener create failed", ServicePrefix)
		}
	}

}

// ListenWithIPAllocation attempts to bind to an available IP from the pool
// If binding fails due to "address already in use", it will try the next IP
// If Service.Spec.LoadBalancerIP is set, it will attempt to use that specific IP
func ListenWithIPAllocation(serviceKey string, Service *v1.Service, linkDevice string, cancel chan struct{}) (listener net.Listener) {
	var ServicePrefix = fmt.Sprintf("Service %-32.32s", serviceKey)
	var allocatedIP string
	var port string

	// Get port from service
	for _, p := range Service.Spec.Ports {
		if p.Port > 0 {
			port = fmt.Sprintf("%d", p.Port)
			break
		}
	}

	if port == "" {
		log.Printf("%s No valid port found in service", ServicePrefix)
		return nil
	}

	// Check if a specific IP is requested
	requestedIP := Service.Spec.LoadBalancerIP
	if requestedIP != "" {
		log.Printf("%s Specific LoadBalancerIP requested: %s:%s", ServicePrefix, requestedIP, port)

		// Try to allocate the specific IP:port combination
		err := ipmgr.IPPoolInstance.AllocateSpecific(requestedIP, port)
		if err != nil {
			log.Printf("%s FAILED to allocate requested LoadBalancerIP %s:%s - Reason: %v", ServicePrefix, requestedIP, port, err)
			log.Printf("%s Service will not be created due to explicit IP request failure", ServicePrefix)
			return nil
		}

		allocatedIP = requestedIP
		cidrStr := fmt.Sprintf("%s/%s", allocatedIP, ipmgr.Bits)
		address := fmt.Sprintf("%s:%s", allocatedIP, port)

		log.Printf("%s Attempting to bind to requested %s", ServicePrefix, address)

		// Add the IP address to the interface (only if not already added)
		managedLBIPs.AddAddr(cidrStr, linkDevice)

		// Try to listen on this address
		listener, err := net.Listen("tcp", address)
		if err == nil {
			log.Printf("%s Successfully bound to requested %s", ServicePrefix, address)
			return listener
		}

		// Failed to bind to requested IP
		log.Printf("%s FAILED to bind to requested %s - Reason: %v", ServicePrefix, address, err)
		log.Printf("%s Service will not be created due to bind failure on explicit IP", ServicePrefix)
		ipmgr.IPPoolInstance.Release(allocatedIP, port)
		// Note: We don't remove the address from the interface here because
		// another service might still be using this IP on a different port
		return nil
	}

	// No specific IP requested, allocate dynamically with port-aware allocation
	log.Printf("%s No LoadBalancerIP specified, using dynamic allocation for port %s", ServicePrefix, port)

	// Try to allocate with port (will reuse existing IPs if port is available)
	var reused bool
	var err error
	allocatedIP, reused, err = ipmgr.IPPoolInstance.AllocateWithPort(port)
	if err != nil {
		log.Printf("%s FAILED to allocate IP for port %s - Reason: %v", ServicePrefix, port, err)
		return nil
	}

	if reused {
		log.Printf("%s Reusing existing IP %s for new port %s (IP consolidation)", ServicePrefix, allocatedIP, port)
	} else {
		log.Printf("%s Allocated new IP %s for port %s", ServicePrefix, allocatedIP, port)
	}

	cidrStr := fmt.Sprintf("%s/%s", allocatedIP, ipmgr.Bits)
	address := fmt.Sprintf("%s:%s", allocatedIP, port)

	// Add the IP address to the interface (idempotent if already added)
	managedLBIPs.AddAddr(cidrStr, linkDevice)

	// Try to listen on this address
	listener, err = net.Listen("tcp", address)
	if err == nil {
		// Success! Update the service's LoadBalancerIP field
		Service.Spec.LoadBalancerIP = allocatedIP
		log.Printf("%s Successfully bound to %s", ServicePrefix, address)
		return listener
	}

	// Failed to bind - this shouldn't happen since we allocated the port
	log.Printf("%s FAILED to bind to allocated %s - Reason: %v (unexpected, port was marked as available)", ServicePrefix, address, err)
	ipmgr.IPPoolInstance.Release(allocatedIP, port)
	// Only remove address if no other ports are using this IP
	managedLBIPs.RemoveAddr(cidrStr, linkDevice)
	return nil
}

// isAddressInUse checks if the error is due to "address already in use"
func isAddressInUse(err error) bool {
	if err == nil {
		return false
	}

	// Check for syscall.EADDRINUSE
	if opErr, ok := err.(*net.OpError); ok {
		if osErr, ok := opErr.Err.(*syscall.Errno); ok {
			return *osErr == syscall.EADDRINUSE
		}
		// Also check the string for "address already in use"
		return strings.Contains(opErr.Error(), "address already in use")
	}

	return strings.Contains(err.Error(), "address already in use")
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
