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
	"sort"

	"k8s.io/api/core/v1"
)

// ServiceTargetPorts get the ports for a Service port array
func ServiceTargetPorts(svc *v1.Service) (ports Ports) {
	ports = Ports{}
	var portSet = make(PortSet)
	if svc.Spec.Ports != nil {
		for _, port := range svc.Spec.Ports {
			value := port.TargetPort.IntValue()
			if value != 0 {
				portSet[Port(value)] = true
			}
		}
	}
	ports = portSet.ToPorts()
	sort.Sort(ports)
	return
}

// ServicePort Port from v1.Service
func ServicePort(Service *v1.Service) (port Port) {
	for _, portObj := range Service.Spec.Ports {
		if portObj.Port > 0 {
			port = Port(portObj.Port)
			break
		}
	}
	return
}

// EndpointIPs get the ips from EndpointSubsets
func EndpointIPs(ep *v1.Endpoints) (ips IPs) {
	var ipSet = make(IPSet)
	ips = IPs{}
	if ep != nil && ep.Subsets != nil {
		for _, subset := range ep.Subsets {
			for _, address := range subset.Addresses {
				ipSet[IP(address.IP)] = true
			}
		}
	}
	ips = ipSet.ToIPs()
	sort.Sort(ips)
	return
}

// EndpointSubsetPorts get the ports from EndpointSubsets
func EndpointSubsetPorts(ep *v1.Endpoints) (ports Ports) {
	var portSet = make(PortSet)
	ports = Ports{}
	if ep != nil && ep.Subsets != nil {
		for _, subset := range ep.Subsets {
			if subset.Ports != nil {
				for _, port := range subset.Ports {
					portSet[Port(port.Port)] = true
				}
			}
		}
	}
	ports = portSet.ToPorts()
	sort.Sort(ports)
	return
}

// SinkPort Port from v1.Service or v1.Endpoints
func SinkPort(Service *v1.Service, ep *v1.Endpoints) (port Port) {
	if InCluster() {
		var ports Ports
		if ep != nil {
			ports = EndpointSubsetPorts(ep)
			if len(ports) > 0 {
				port = ports[0]
			}
		}
	} else {
		// Look for NodePort in the service
		hasNodePort := false
		for _, portObj := range Service.Spec.Ports {
			if portObj.NodePort > 0 {
				port = Port(portObj.NodePort)
				hasNodePort = true
				break
			}
		}

		// If no NodePort is found, use the service port instead
		if !hasNodePort {
			for _, portObj := range Service.Spec.Ports {
				if portObj.Port > 0 {
					port = Port(portObj.Port)
					break
				}
			}
		}
	}
	return
}
