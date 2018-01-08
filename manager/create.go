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
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/loadbalancer/helper"
	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/listener"
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
		EnableEp:  true,
		InCluster: false,
		Name:      Service.ObjectMeta.Name,
		Namespace: Service.ObjectMeta.Namespace,
		Debug:     envCfg.Debug,
		Endpoints: helper.ServiceSinks(Service),
	}
}

// NewManagedListenerFromV1Service from a kubernetes v1.Service
func NewManagedListenerFromV1Service(Service *v1.Service,
	envCfg *share.ServerCfg,
	Clientset *kubernetes.Clientset) (ml *listener.ManagedListener) {
	defer trace.Tracer.ScopedTrace()()
	var ip = helper.ServiceSourceIP(Service)
	var c *ipmgr.CIDR = &ipmgr.CIDR{IP: ip, Bits: ipmgr.Bits, LinkDevice: ipmgr.LinkDevice}

	ml = &listener.ManagedListener{
		Definition: *NewPipeDefinition(Service, envCfg),
		Listener:   nil,
		Pipes:      make(map[*pipe.Pipe]bool),
		Mutex:      &mutex.Mutex{},
		Kubernetes: envCfg.Kubernetes,
		MapAdd:     make(chan *pipe.Pipe, 3),
		MapRm:      make(chan *pipe.Pipe, 3),
		StopWatch:  make(chan bool, 3),
		Debug:      envCfg.Debug,
		Clientset:  Clientset,
		Active:     0,
		V1Service:  Service,
		InCluster:  false,
		CIDR:       c,
	}
	return
}
