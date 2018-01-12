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
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"

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
				ml.Changed = true
				ml.Endpoints = ep
			}
		}
	}
}

// SetService from nodes or others
func (ml *ManagedListener) SetService(Service *v1.Service) {
	defer ml.Monitor()()
	ml.Service = Service
	ml.Changed = true
}

// UpdateEndpoints when in a cluster and processing asynchronous
// updates manage changes
func (ml *ManagedListener) UpdateEndpoints() {
	if InCluster() {
		if ml.Endpoints == nil {
			ml.Endpoints = Endpoints(ml.Clientset, ml.Service)
			ml.Changed = true
		}
		if ml.Changed || !ml.canListen() {
			ml.IPs = EndpointIPs(ml.Endpoints)
			ml.Ports = EndpointSubsetPorts(ml.Endpoints)
			if len(ml.Ports) > 0 {
				ml.Port = ml.Ports[0]
			} else {
				log.Println(fmt.Errorf("%s", "No port endpoints available"))
			}
		}
		ml.Changed = false
	}
}

// Insert pipe to map of pipes in managed listener
func (ml *ManagedListener) Insert(pipe *pipe.Pipe) {
	defer trace.Tracer.ScopedTrace("MapAdd", *pipe)()
	pipe.State = share.Open
	defer pipe.Monitor()()
	ml.Pipes[pipe] = true
	ml.Active = uint64(len(ml.Pipes))
}

// Delete pipe from map of pipes in managed listener
func (ml *ManagedListener) Delete(pipe *pipe.Pipe) {
	defer trace.Tracer.ScopedTrace("MapRm", *pipe)()
	pipe.State = share.Closed
	defer pipe.Monitor()()
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
			n = atomic.AddUint64(&ml.n, 1) % uint64(len(sinks))
			sink = sinks[n]
		} else {
			n = atomic.AddUint64(&ml.n, 1) % uint64(len(ml.IPs))
			if len(ml.Ports) > 0 {
				sink = Address(ml.IPs[n], ml.Port)
			}
		}
		log.Println("sinks", sinks)
		log.Println("sink", sink, ml.Port)
	}
	return
}

// Accept expose ManagedListener's listener
func (ml *ManagedListener) Accept() (net.Conn, error) {
	defer trace.Tracer.ScopedTrace()()
	return ml.Listener.Accept()
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

	for {
		if InCluster() {
			ml.UpdateEndpoints()
			for !ml.canListen() {
				log.Println(fmt.Errorf("!canListen options not set\nport: %v\nips: %v\nports: %v\nservice: %v\nep: %v",
					ml.Port,
					ml.IPs,
					ml.Ports,
					ml.Service,
					ml.Endpoints,
				))
				time.Sleep(time.Second)
				ml.UpdateEndpoints()
			}
		}
		var err error
		var SourceConn, SinkConn net.Conn
		if SourceConn, err = ml.Accept(); err != nil {
			log.Printf("Source connection failed: %v\n", err)
			break
		}
		sink := ml.Next()
		log.Println(sink)
		SinkConn, err = net.Dial("tcp", sink)
		if err != nil {
			log.Printf("Sink connection failed: %v\n", err)
			break
		}
		var pipe = pipe.NewPipe(ml.Key, ml.MapAdd, ml.MapRm, ml.Mutex, SourceConn, SinkConn, &ml.Definition)
		go pipe.Connect()
	}
}

// Close a listener and it's children
func (ml *ManagedListener) Close() {
	if ml != nil {
		defer trace.Tracer.ScopedTrace()()
		if ml.Listener != nil {
			if err := ml.Listener.Close(); err != nil {
				log.Println("Error closing listener", ml.Listener)
			}
			defer ml.Monitor()()
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
	client.Update(ml.Service)
	ml.refreshServiceSpec(client)
}

func (ml *ManagedListener) refreshServiceSpec(client v1core.ServiceInterface) {
	service, err := client.Get(ml.Service.ObjectMeta.Name, metav1.GetOptions{})
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
	_, err := client.Update(ml.Service)
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
		_, err = client.Update(ml.Service)

	}
}

func dumpJSON(obj interface{}) {
	if jsonText, err := json.MarshalIndent(obj, "", "  "); err == nil {
		log.Println(string(jsonText))
	} else {
		log.Println(err)
	}
}
