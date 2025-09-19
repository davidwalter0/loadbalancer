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
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/loadbalancer/health"
	"github.com/davidwalter0/loadbalancer/helper"
	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/pipe"
	"github.com/davidwalter0/loadbalancer/share"
	"github.com/davidwalter0/loadbalancer/tracer"
)

// NewPipeDefinition from a kubernetes v1.Service
func NewPipeDefinition(Service *v1.Service, envCfg *share.ServerCfg) *pipe.Definition {
	defer trace.Tracer.ScopedTrace()()
	return &pipe.Definition{
		Key:       helper.ServiceKey(Service),
		Source:    helper.ServiceSource(Service),
		Name:      Service.ObjectMeta.Name,
		Namespace: Service.ObjectMeta.Namespace,
		Debug:     envCfg.Debug,
	}
}

// NewManagedListener from a kubernetes v1.Service
func NewManagedListener(Service *v1.Service,
	envCfg *share.ServerCfg,
	Clientset *kubernetes.Clientset) (ml *ManagedListener) {
	defer trace.Tracer.ScopedTrace()()
	var ip = helper.ServiceSourceIP(Service)
	var c *ipmgr.CIDR = &ipmgr.CIDR{IP: ip, Bits: ipmgr.Bits, LinkDevice: ipmgr.LinkDevice}
	var ep *v1.Endpoints
	if InCluster() {
		ep = Endpoints(Clientset, Service)
	}

	// Create health checker with default settings
	healthSettings := health.DefaultSettings()
	
	// Check for health check annotations on the service
	if Service.Annotations != nil {
		if val, ok := Service.Annotations["loadbalancer.example.com/health-check-enabled"]; ok {
			healthSettings.Enabled = (val == "true")
		}
		if val, ok := Service.Annotations["loadbalancer.example.com/health-check-type"]; ok {
			if val == "http" {
				healthSettings.Type = health.HTTPCheck
			}
		}
		if val, ok := Service.Annotations["loadbalancer.example.com/health-check-path"]; ok && val != "" {
			healthSettings.HTTPPath = val
		}
		if val, ok := Service.Annotations["loadbalancer.example.com/health-check-interval"]; ok {
			if interval, err := time.ParseDuration(val); err == nil {
				healthSettings.Interval = interval
			}
		}
		if val, ok := Service.Annotations["loadbalancer.example.com/health-check-timeout"]; ok {
			if timeout, err := time.ParseDuration(val); err == nil {
				healthSettings.Timeout = timeout
			}
		}
	}

	ml = &ManagedListener{
		Definition: *NewPipeDefinition(Service, envCfg),
		Listener:   nil,
		Pipes:      make(map[*pipe.Pipe]bool),
		Mutex:      &mutex.Mutex{},
		MapAdd:     make(chan *pipe.Pipe, 3),
		MapRm:      make(chan *pipe.Pipe, 3),
		StopWatch:  make(chan bool, 3),
		Debug:      envCfg.Debug,
		Key:        helper.ServiceKey(Service),
		Clientset:  Clientset,
		Canceled:   make(chan struct{}),
		Active:     0,
		Service:    Service,
		Create:     time.Now(),
		Endpoints:  ep,
		CIDR:       c,
		HealthChecker: health.NewChecker(healthSettings),
		// Port:       port,
		// IPs:        ips,
		// Ports:      ports,
	}
	return
}

/*
// NewManagedListeners from a kubernetes v1.Service
func NewManagedListeners(Service *v1.Service,
	envCfg *share.ServerCfg,
	Clientset *kubernetes.Clientset) (ml *ManagedListeners) {
	defer trace.Tracer.ScopedTrace()()
	var ip = helper.ServiceSourceIP(Service)
	var c *ipmgr.CIDR = &ipmgr.CIDR{IP: ip, Bits: ipmgr.Bits, LinkDevice: ipmgr.LinkDevice}

	ml = &ManagedListeners{
		Listeners:  make(ManagedListenerMap),
		Mutex:      &mutex.Mutex{},
		Debug:      envCfg.Debug,
		n:          0,
		Clientset:  Clientset,
		Active:     0,
		Service:    Service,
		CIDR:       c,
		Addresses:  helper.UpstreamAddresses(),
		Ports:      helper.ServicePorts(Service),
	}
	return
}
*/
