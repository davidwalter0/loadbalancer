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

package helper

import (
	"fmt"
	"log"
	"time"

	"k8s.io/api/core/v1"

	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/nodemgr"
)

// ServiceKey from v1.Service info for map lookup in listeners
func ServiceKey(Service *v1.Service) string {
	return Service.ObjectMeta.Namespace + "/" + Service.ObjectMeta.Name
}

// ServiceSource IP:NodePort from v1.Service
func ServiceSource(Service *v1.Service) (Source string) {
	var IP string
	var Port int32
	for _, port := range Service.Spec.Ports {
		if port.Port > 0 {
			Port = port.Port
			if len(Service.Spec.LoadBalancerIP) > 0 {
				IP = Service.Spec.LoadBalancerIP
			} else {
				// IP = "0.0.0.0"
				IP = ipmgr.DefaultCIDR.IP
			}
			Source = fmt.Sprintf("%s:%d", IP, Port)
			break
		}
	}
	return
}

// ServiceSourceIP IP from v1.Service
func ServiceSourceIP(Service *v1.Service) (IP string) {
	if len(Service.Spec.LoadBalancerIP) > 0 {
		IP = Service.Spec.LoadBalancerIP
	}
	return
}

// ServiceSinks returns connection endpoints for a service
// When NodePorts are available, it returns node:nodeport pairs
// When NodePorts aren't available, it falls back to using ClusterIP:port
func ServiceSinks(Service *v1.Service) (Sink []string) {
	// Try NodePort approach first
	nodeList := nodemgr.NodeListPtr()
	var NodePort int32
	var hasNodePort bool

	// Check if we have any NodePort
	for _, port := range Service.Spec.Ports {
		if port.NodePort > 0 {
			NodePort = port.NodePort
			hasNodePort = true
			break
		}
	}

	// If we have a NodePort, build node:nodeport sinks
	if hasNodePort {
		// Get nodes with retry
		var nodes []string
		for nodes = nodeList.GetNodes(); len(nodes) == 0; nodes = nodeList.GetNodes() {
			log.Println("Node List is empty, sleep a bit")
			time.Sleep(time.Second)
		}

		// Build sinks list with NodePorts
		if len(nodes) > 0 {
			for _, node := range nodes {
				Sink = append(Sink, fmt.Sprintf("%s:%d", node, NodePort))
			}
			log.Printf("Using NodePorts for service %s/%s", Service.Namespace, Service.Name)
			return Sink
		}
	}

	// No NodePort available or no nodes found, try ClusterIP approach
	if hasNodePort {
		log.Println("Service has NodePort but no nodes found, falling back to ClusterIP")
	} else {
		log.Println("Service has no NodePort assigned, falling back to ClusterIP")
	}

	// Get the service port
	var servicePort int32
	for _, port := range Service.Spec.Ports {
		if port.Port > 0 {
			servicePort = port.Port
			break
		}
	}

	// If the service has a valid ClusterIP and port, use that
	if servicePort > 0 && Service.Spec.ClusterIP != "" && Service.Spec.ClusterIP != "None" {
		// Use the ClusterIP:Port as the sink
		endpoint := fmt.Sprintf("%s:%d", Service.Spec.ClusterIP, servicePort)
		Sink = append(Sink, endpoint)
		log.Printf("Using ClusterIP:Port %s for service %s/%s", endpoint, Service.Namespace, Service.Name)
		return Sink
	}

	// No valid endpoints found
	log.Printf("No valid endpoints found for service %s/%s", Service.Namespace, Service.Name)
	return nil
}

// NodeSinks IP:NodePort from v1.Service
func NodeSinks(Service *v1.Service) (Sink []string) {
	nodeList := nodemgr.NodeListPtr()
	// var IP string
	var NodePort int32
	var hasNodePort bool

	// First check if we have any NodePort
	for _, port := range Service.Spec.Ports {
		if port.NodePort > 0 {
			NodePort = port.NodePort
			hasNodePort = true
			break
		}
	}

	// If we don't have a NodePort (allocateLoadBalancerNodePorts=false)
	// then we can't build node sinks
	if !hasNodePort {
		log.Println("Service has no NodePort assigned, skipping node sinks")
		return Sink
	}

	// Get nodes with retry
	var nodes []string
	for nodes = nodeList.GetNodes(); len(nodes) == 0; nodes = nodeList.GetNodes() {
		log.Println("Node List is empty, sleep a bit")
		time.Sleep(time.Second)
	}

	// Build sinks list
	for _, node := range nodes {
		Sink = append(Sink, fmt.Sprintf("%s:%d", node, NodePort))
	}

	return
}
