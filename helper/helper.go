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

// ServiceSinks IP:NodePort from v1.Service
func ServiceSinks(Service *v1.Service) (Sink []string) {
	nodeList := nodemgr.NodeListPtr()
	// var IP string
	var NodePort int32
	for _, port := range Service.Spec.Ports {
		if port.NodePort > 0 {
			NodePort = port.NodePort
			var nodes []string
			for nodes = nodeList.GetNodes(); len(nodes) == 0; nodes = nodeList.GetNodes() {
				log.Println("Node List is empty, sleep a bit")
				time.Sleep(time.Second)
			}
			for _, node := range nodes {
				Sink = append(Sink, fmt.Sprintf("%s:%d", node, NodePort))
			}
			break
		}
	}
	return
}
