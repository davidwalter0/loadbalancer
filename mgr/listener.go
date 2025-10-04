/*

Copyright 2018-2025 David Walter.

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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/davidwalter0/backoff"
	"github.com/davidwalter0/loadbalancer/helper"
	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/pipe"
	"github.com/davidwalter0/loadbalancer/share"
	"github.com/davidwalter0/loadbalancer/tracer"
)

// Monitor for this ManagedListener
func (ml *ManagedListener) Monitor(args ...interface{}) func() {
	if ml != nil {
		defer trace.Tracer.ScopedTrace(args...)()
		return ml.Mutex.MonitorTrace(args...)
	}
	return func() {}
}

// SetEndpoint from nodes or others
func (ml *ManagedListener) SetEndpoint(ep *v1.Endpoints) {
	defer ml.Monitor()()
	if InCluster() {
		if ml.Endpoints != nil && ep != nil {
			lhsPorts := EndpointIPs(ml.Endpoints)
			lhsIPs := EndpointSubsetPorts(ml.Endpoints)
			rhsPorts := EndpointIPs(ep)
			rhsIPs := EndpointSubsetPorts(ep)
			if !lhsPorts.Equal(rhsPorts) || !lhsIPs.Equal(rhsIPs) {
				ml.EndpointsChanged = true
				ml.Endpoints = ep
			}
		}
	}
}

// SetService from nodes or others
func (ml *ManagedListener) SetService(Service *v1.Service) {
	defer ml.Monitor()()
	ml.Service = Service
	ml.EndpointsChanged = true
}

// UpdateEndpointsWithBackoff retry until endoints are found
func (ml *ManagedListener) UpdateEndpointsWithBackoff() {
	var ServicePrefix = fmt.Sprintf("Service %-32.32s", ml.Key)

	Try := func() (err error) {
		defer ml.Monitor()()
		ml.UpdateEndpoints()
		if !ml.canListen() {
			err = fmt.Errorf("%s !canListen %v:%v",
				ServicePrefix,
				ml.IPs,
				ml.Port,
			)
		}
		return
	}

	ExpBackoff := ConfigureBackoff(5*time.Second, 1*time.Minute, 3*time.Minute, ml.Canceled)

	Notify := func(err error, t time.Duration) {
		log.Printf("%v elapsed %s break after %s",
			err,
			DurationString(ExpBackoff.GetElapsedTime()),
			DurationString(ExpBackoff.MaxElapsedTime))
	}

	var err error
	for {
		if err = backoff.RetryNotify(Try, ExpBackoff, Notify); err != nil {
			log.Printf("%s accept retry timeout: %v\n", ServicePrefix, err)
			continue
		}
		break
	}
}

// UpdateEndpoints when in a cluster and processing asynchronous
// updates manage changes
func (ml *ManagedListener) UpdateEndpoints() {
	if InCluster() {
		if ml.Endpoints == nil {
			ml.Endpoints = Endpoints(ml.Clientset, ml.Service)
			ml.EndpointsChanged = true
		}
		if ml.EndpointsChanged || !ml.canListen() {
			// Save old IPs for health checker updates
			oldIPs := ml.IPs
			
			ml.IPs = EndpointIPs(ml.Endpoints)
			ml.Ports = EndpointSubsetPorts(ml.Endpoints)
			if len(ml.Ports) > 0 {
				ml.Port = ml.Ports[0]
			} else {
				log.Println(fmt.Errorf("%s", "No port endpoints available"))
			}
			
			// Update health checker if enabled
			if ml.HealthChecker != nil && ml.HealthChecker.IsEnabled() && len(ml.Ports) > 0 {
				// Remove old endpoints
				for _, ip := range oldIPs {
					address := Address(ip, ml.Port)
					ml.HealthChecker.RemoveEndpoint(address)
				}
				
				// Add new endpoints
				for _, ip := range ml.IPs {
					address := Address(ip, ml.Port)
					ml.HealthChecker.AddEndpoint(address)
				}
				
				if ml.Debug {
					log.Printf("Updated health check endpoints for service %s: %v", ml.Key, ml.IPs)
				}
			}
			
			ml.EndpointsChanged = false
		}
	}
}

// Insert pipe to map of pipes in managed listener
func (ml *ManagedListener) Insert(pipe *pipe.Pipe) {
	defer trace.Tracer.ScopedTrace("MapAdd", *pipe)()
	pipe.State = share.Open
	defer ml.Monitor()()
	ml.Pipes[pipe] = true
	ml.Active = uint64(len(ml.Pipes))
}

// Delete pipe from map of pipes in managed listener
func (ml *ManagedListener) Delete(pipe *pipe.Pipe) {
	defer trace.Tracer.ScopedTrace("MapRm", *pipe)()
	pipe.State = share.Closed
	defer ml.Monitor()()
	delete(ml.Pipes, pipe)
	ml.Active = uint64(len(ml.Pipes))
}

// PipeMapHandler adds, removes, closes and single threads access to map list
func (ml *ManagedListener) PipeMapHandler() {
	if ml != nil {
		for {
			select {
			case pipe := <-ml.MapAdd:
				if pipe != nil {
					ml.Insert(pipe)
				}
			case pipe := <-ml.MapRm:
				if pipe != nil {
					ml.Delete(pipe)
				}
			}
		}
	}
}

// Open / start Listening and run PipeMapHandler
func (ml *ManagedListener) Open() {
	if ml != nil {
		defer trace.Tracer.ScopedTrace()()
		go ml.Listening()
		go ml.PipeMapHandler()
		ml.SetExternalIP()
		
		// Start health checks if enabled
		if ml.HealthChecker != nil && ml.HealthChecker.IsEnabled() {
			// Add all endpoints to health checker
			for _, ip := range ml.IPs {
				if len(ml.Ports) > 0 {
					address := Address(ip, ml.Port)
					ml.HealthChecker.AddEndpoint(address)
				}
			}
			
			// Start the health checker
			ml.HealthChecker.Debug = ml.Debug
			ml.HealthChecker.StartChecking()
			log.Printf("Health checking started for service %s with %d endpoints", ml.Key, len(ml.IPs))
		}
	}
}

// Next returns the next host:port pair if more than one
// available round robin selection
func (ml *ManagedListener) Next() (sink string) {
	if ml != nil {
		defer trace.Tracer.ScopedTrace()()
		defer ml.Monitor()()
		var n uint64
		// Don't use k8s endpoint lookup if not in a k8s cluster
		var sinks []string
		if !InCluster() {
			sinks = helper.ServiceSinks(ml.Service)
			if len(sinks) > 0 {
				n = atomic.AddUint64(&ml.n, 1) % uint64(len(sinks))
				sink = sinks[n]
			} else {
				log.Printf("Warning: No available service sinks for %s", ml.Key)
				return ""
			}
		} else {
			// Use health checker if enabled to select only healthy endpoints
			if ml.HealthChecker != nil && ml.HealthChecker.IsEnabled() && len(ml.Ports) > 0 {
				// Get only healthy endpoints
				healthyAddresses := ml.HealthChecker.GetHealthyEndpoints()

				// If no healthy endpoints, fall back to all endpoints
				if len(healthyAddresses) == 0 {
					if ml.Debug {
						log.Printf("Warning: No healthy endpoints for service %s, using all available endpoints", ml.Key)
					}

					if len(ml.IPs) == 0 {
						log.Printf("Error: No IP addresses available for service %s", ml.Key)
						return ""
					}

					n = atomic.AddUint64(&ml.n, 1) % uint64(len(ml.IPs))
					if len(ml.Ports) > 0 {
						sink = Address(ml.IPs[n], ml.Port)
					}
				} else {
					// Select from healthy endpoints
					n = atomic.AddUint64(&ml.n, 1) % uint64(len(healthyAddresses))
					sink = healthyAddresses[n]

					if ml.Debug {
						log.Printf("Selected healthy endpoint %s from %d available for service %s",
							sink, len(healthyAddresses), ml.Key)
					}
				}
			} else {
				// Standard round-robin selection without health checks
				if len(ml.IPs) == 0 {
					log.Printf("Error: No IP addresses available for service %s", ml.Key)
					return ""
				}

				n = atomic.AddUint64(&ml.n, 1) % uint64(len(ml.IPs))
				if len(ml.Ports) > 0 {
					sink = Address(ml.IPs[n], ml.Port)
				}
			}
		}
		log.Printf("Service %-32.32s %s ~ %s:%d\n", ml.Key, ml.Source, sink, ml.Port)
	}
	return
}

// Accept expose ManagedListener's listener
func (ml *ManagedListener) Accept() (net.Conn, error) {
	defer trace.Tracer.ScopedTrace()()
	if ml != nil && ml.Listener != nil {
		return ml.Listener.Accept()
	}
	return nil, fmt.Errorf("Accept called on nil Managed Listener")
}

// StopWatchNotify checking for endpoints
func (ml *ManagedListener) StopWatchNotify() {
	if ml != nil {
		ml.StopWatch <- true
	}
}

func (ml *ManagedListener) canListen() (ok bool) {
	return ml.Port > 0 && len(ml.IPs) > 0 && len(ml.Ports) > 0 && ml.Endpoints != nil && ml.Service != nil
}

// Listening on managed listener
func (ml *ManagedListener) Listening() {
	defer trace.Tracer.ScopedTrace()()
	defer ml.StopWatchNotify()
	var ServicePrefix = fmt.Sprintf("Service %-32.32s", ml.Key)
	if ml.Canceled == nil {
		log.Println("Listener Canceled chan is nil returning...")
		return
	}
	canceled := ml.Canceled

	for {
		if InCluster() {
			ml.UpdateEndpointsWithBackoff()
		}

		var err error
		var SourceConn, SinkConn net.Conn
		if SourceConn, err = ml.Accept(); err != nil {
			log.Printf("%s Downstream connection failed: %v\n", ServicePrefix, err)
			select {
			case <-canceled:
				log.Println("Canceled Listener shutting down...")
				return
			case <-shutdown:
				log.Println("Listener shutting down...")
				return
			default:
			}
			continue
		}
		sink := ml.Next()
		SinkConn, err = net.Dial("tcp", sink)
		if err != nil {
			log.Printf("%s Upstream connection failed: %v\n", ServicePrefix, err)
			continue
		}
		select {
		case <-canceled:
			log.Println("Canceled Listener shutting down...")
			return
		case <-shutdown:
			log.Println("Listener shutting down...")
			return
		default:
			var pipe = pipe.NewPipe(ml.Key, ml.MapAdd, ml.MapRm, SourceConn, SinkConn, &ml.Definition)
			go pipe.Connect()
		}
	}
}

// Close a listener and it's children
func (ml *ManagedListener) Close() {
	if ml != nil {
		defer trace.Tracer.ScopedTrace()()
		defer ml.Monitor()()
		
		// Stop health checker if enabled
		if ml.HealthChecker != nil && ml.HealthChecker.IsEnabled() {
			ml.HealthChecker.StopChecking()
			log.Printf("Health checking stopped for service %s", ml.Key)
		}
		
		if ml.Canceled != nil {
			// Add a recover to handle already closed channel
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Channel was already closed, just log and continue
						log.Printf("Warning: Canceled channel was already closed for service %s: %v", ml.Key, r)
					}
				}()

				close(ml.Canceled)
			}()

			ml.Canceled = make(chan struct{})
		}
		if ml.Listener != nil {
			// Release the allocated IP back to the pool before closing
			if ml.Service != nil && ml.Service.Spec.LoadBalancerIP != "" {
				ip := ml.Service.Spec.LoadBalancerIP

				// Only release if it's not the default CIDR base IP
				if ipmgr.DefaultCIDR != nil && ip != ipmgr.DefaultCIDR.IP {
					if ipmgr.IPPoolInstance != nil {
						port := fmt.Sprintf("%d", ml.Port)
						ipmgr.IPPoolInstance.Release(ip, port)
						log.Printf("Released IP %s:%s back to pool for service %s", ip, port, ml.Key)
					}
				}
			}

			if err := ml.Listener.Close(); err != nil {
				log.Println("Error closing listener", ml.Listener)
			}
			var pipes = []*pipe.Pipe{}
			for pipe := range ml.Pipes {
				go pipe.Close()
				pipes = append(pipes, pipe)
			}
			for _, pipe := range pipes {
				delete(ml.Pipes, pipe)
			}
		}
		ml.RemoveExternalIP()
	}
}

// SetExternalIP for service spec
func (ml *ManagedListener) SetExternalIP() {
	if ml.Clientset == nil {
		panic("Clientset nil, can't SetExternalIP")
	}
	client := ml.Clientset.CoreV1().Services(ml.Service.ObjectMeta.Namespace)
	ml.refreshServiceSpec(client)
	var ip = ipmgr.IP
	if ml.CIDR != nil && len(ml.CIDR.IP) > 0 {
		ip = ml.CIDR.IP
	}
	ml.Service.Spec.ExternalIPs = []string{ip}
	log.Println("SetExternalIP", ml.Service.Spec.ExternalIPs)
	if _, err := client.Update(context.Background(), ml.Service, metav1.UpdateOptions{}); err != nil {
		log.Println("Problem with client update:", err)
	}
	ml.refreshServiceSpec(client)
}

func (ml *ManagedListener) refreshServiceSpec(client v1core.ServiceInterface) {
	service, err := client.Get(context.Background(), ml.Service.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		ml.Service = service
	}
}

// RemoveExternalIP from service spec
func (ml *ManagedListener) RemoveExternalIP() {
	if ml.Clientset == nil {
		panic("Clientset nil, can't UpdateExternalIP")
	}
	client := ml.Clientset.CoreV1().Services(ml.Service.ObjectMeta.Namespace)
	log.Println("Removing ExternalIP", ml.Service.Spec.ExternalIPs)
	ml.refreshServiceSpec(client)
	ml.Service.Spec.ExternalIPs = []string{}
	// ml.Service.ObjectMeta.Annotations = make(map[string]string)
	_, err := client.Update(context.Background(), ml.Service, metav1.UpdateOptions{})
	if err != nil {
		log.Println("Error Removing ExternalIPs", err)
		if ml.Debug {
			dumpJSON(ml.Service)
		}
	}
	for i := 0; i < 3 && err != nil; i++ {
		log.Printf("Removing ExternalIPs Retry %d %v", i, err)
		time.Sleep(time.Second)
		ml.refreshServiceSpec(client)
		ml.Service.Spec.ExternalIPs = []string{}
		_, err = client.Update(context.Background(), ml.Service, metav1.UpdateOptions{})

	}
}

func dumpJSON(obj interface{}) {
	if jsonText, err := json.MarshalIndent(obj, "", "  "); err == nil {
		log.Println(string(jsonText))
	} else {
		log.Println(err)
	}
}
