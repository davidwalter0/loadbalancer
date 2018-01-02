package mgr

import (
	"fmt"

	"k8s.io/api/core/v1"
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
				IP = "0.0.0.0"
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

// ServiceSink IP:NodePort from v1.Service
func ServiceSinks(Service *v1.Service) (Sink []string) {
	var IP string
	var NodePort int32
	for _, port := range Service.Spec.Ports {
		if port.NodePort > 0 {
			NodePort = port.NodePort
			// if len(Service.Spec.LoadBalancerIP) > 0 {
			// 	IP = Service.Spec.LoadBalancerIP
			// } else {
			// 	IP = "0.0.0.0"
			// }
			for _, IP = range []string{"172.17.4.201", "172.17.4.202"} {
				Sink = append(Sink, fmt.Sprintf("%s:%d", IP, NodePort))
			}
			break
		}
	}
	return
}
