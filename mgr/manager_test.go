/*

Copyright 2025 David Walter.

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
	"github.com/davidwalter0/loadbalancer/health"
	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/share"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

// Helper function to create a test service
func createTestService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				"loadbalancer.example.com/health-check-enabled": "true",
				"loadbalancer.example.com/health-check-interval": "1s",
			},
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeLoadBalancer,
			Ports: []v1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   v1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": "test",
			},
		},
	}
}

// Helper function to create test endpoints
func createTestEndpoints() *v1.Endpoints {
	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "192.168.1.1",
					},
					{
						IP: "192.168.1.2",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Port: 8080,
					},
				},
			},
		},
	}
}

func TestNewMgr(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()
	
	// Create config
	config := &share.ServerCfg{
		ForwarderCfg: share.ForwarderCfg{
			Debug:      true,
			Kubeconfig: "test-config",
			Kubernetes: true,
			LinkDevice: "eth0",
		},
	}
	
	// Create manager
	mgr := NewMgr(config, clientset)
	
	// Verify manager was created
	assert.NotNil(t, mgr, "Expected manager to be created")
	assert.Equal(t, config, mgr.EnvCfg, "Expected config to be set")
	assert.Equal(t, clientset, mgr.Clientset, "Expected clientset to be set")
	assert.NotNil(t, mgr.Listeners, "Expected listeners map to be initialized")
	assert.NotNil(t, mgr.Mutex, "Expected mutex to be initialized")
}

func TestGetCreate(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()
	
	// Create config
	config := &share.ServerCfg{
		ForwarderCfg: share.ForwarderCfg{
			Debug:      true,
			Kubeconfig: "test-config",
			Kubernetes: true,
			LinkDevice: "eth0",
		},
	}
	
	// Set test CIDR
	ipmgr.DefaultCIDR = &ipmgr.CIDR{IP: "192.168.1.1", Bits: "24", LinkDevice: "eth0"}
	ipmgr.IP = ipmgr.DefaultCIDR.IP
	ipmgr.Bits = ipmgr.DefaultCIDR.Bits
	ipmgr.LinkDevice = "eth0"
	ipmgr.Debug = true
	
	// Create manager
	mgr := NewMgr(config, clientset)
	
	// Create test service
	service := createTestService()
	
	// Test GetCreate
	var created bool
	ml := mgr.GetCreate("default/test-service", service, &created)
	
	// Verify listener was created
	assert.True(t, created, "Expected listener to be created")
	assert.NotNil(t, ml, "Expected listener to be returned")
	assert.Equal(t, "default/test-service", ml.Key, "Expected key to match")
	assert.Equal(t, service, ml.Service, "Expected service to match")
	
	// Verify health checker was created
	assert.NotNil(t, ml.HealthChecker, "Expected health checker to be created")
	assert.True(t, ml.HealthChecker.IsEnabled(), "Expected health checker to be enabled")
	
	// Test GetCreate again with same key
	created = false
	ml2 := mgr.GetCreate("default/test-service", service, &created)
	
	// Verify listener was not created again
	assert.False(t, created, "Expected listener to not be created again")
	assert.Equal(t, ml, ml2, "Expected same listener to be returned")
	
	// Verify listener was added to manager
	assert.Equal(t, 1, len(mgr.Listeners), "Expected one listener in manager")
	assert.Equal(t, ml, mgr.Listeners["default/test-service"], "Expected listener to be in manager")
}

func TestGetSet(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()
	
	// Create config
	config := &share.ServerCfg{
		ForwarderCfg: share.ForwarderCfg{
			Debug:      true,
			Kubeconfig: "test-config",
			Kubernetes: true,
			LinkDevice: "eth0",
		},
	}
	
	// Create manager
	mgr := NewMgr(config, clientset)
	
	// Create test service
	service := createTestService()
	
	// Create managed listener
	ml := NewManagedListener(service, config, clientset)
	
	// Test Set
	mgr.Set("default/test-service", ml)
	
	// Test Get
	ml2, ok := mgr.Get("default/test-service")
	
	// Verify listener was retrieved
	assert.True(t, ok, "Expected listener to be found")
	assert.Equal(t, ml, ml2, "Expected retrieved listener to match")
	
	// Test Get with non-existent key
	ml3, ok := mgr.Get("non-existent")
	
	// Verify listener was not found
	assert.False(t, ok, "Expected listener to not be found")
	assert.Nil(t, ml3, "Expected nil listener to be returned")
}

func TestClose(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()
	
	// Create config
	config := &share.ServerCfg{
		ForwarderCfg: share.ForwarderCfg{
			Debug:      true,
			Kubeconfig: "test-config",
			Kubernetes: true,
			LinkDevice: "eth0",
		},
	}
	
	// Create manager
	mgr := NewMgr(config, clientset)
	
	// Create test service
	service := createTestService()
	
	// Create managed listener
	ml := NewManagedListener(service, config, clientset)
	
	// Enable health checks for the listener for test coverage
	ml.HealthChecker = health.NewChecker(health.DefaultSettings())
	ml.HealthChecker.Debug = true
	ml.HealthChecker.StartChecking()
	
	// Add listener to manager
	mgr.Set("default/test-service", ml)
	
	// Verify listener is in manager
	assert.Equal(t, 1, len(mgr.Listeners), "Expected one listener in manager")
	
	// Close listener
	mgr.Close("default/test-service")
	
	// Verify listener was removed
	assert.Equal(t, 0, len(mgr.Listeners), "Expected no listeners in manager")
}

func TestSetEndpoint(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()
	
	// Create config
	config := &share.ServerCfg{
		ForwarderCfg: share.ForwarderCfg{
			Debug:      true,
			Kubeconfig: "test-config",
			Kubernetes: true,
			LinkDevice: "eth0",
		},
	}
	
	// Create manager
	mgr := NewMgr(config, clientset)
	
	// Create test service and endpoints
	service := createTestService()
	endpoints := createTestEndpoints()
	
	// Create managed listener
	ml := NewManagedListener(service, config, clientset)
	
	// Add listener to manager
	mgr.Set("default/test-service", ml)
	
	// Set endpoints
	mgr.SetEndpoint("default/test-service", endpoints)
	
	// Verify endpoints were set
	assert.Equal(t, endpoints, ml.Endpoints, "Expected endpoints to be set")
	assert.True(t, ml.EndpointsChanged, "Expected EndpointsChanged to be true")
}

func TestManagedListenerUpdateEndpoints(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()
	
	// Create config
	config := &share.ServerCfg{
		ForwarderCfg: share.ForwarderCfg{
			Debug:      true,
			Kubeconfig: "test-config",
			Kubernetes: true,
			LinkDevice: "eth0",
		},
	}
	
	// Create test service and endpoints
	service := createTestService()
	endpoints := createTestEndpoints()
	
	// Create managed listener
	ml := NewManagedListener(service, config, clientset)
	
	// Set health checker
	ml.HealthChecker = health.NewChecker(health.DefaultSettings())
	
	// Set endpoints
	ml.Endpoints = endpoints
	ml.EndpointsChanged = true
	
	// Update endpoints
	ml.UpdateEndpoints()
	
	// Verify IPs and ports were extracted
	assert.Equal(t, 2, len(ml.IPs), "Expected 2 IPs")
	assert.Equal(t, IP("192.168.1.1"), ml.IPs[0], "Expected first IP to match")
	assert.Equal(t, IP("192.168.1.2"), ml.IPs[1], "Expected second IP to match")
	
	assert.Equal(t, 1, len(ml.Ports), "Expected 1 port")
	assert.Equal(t, Port(8080), ml.Ports[0], "Expected port to match")
	assert.Equal(t, Port(8080), ml.Port, "Expected primary port to match")
	
	assert.False(t, ml.EndpointsChanged, "Expected EndpointsChanged to be false")
}

func TestShutdown(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()
	
	// Create config
	config := &share.ServerCfg{
		ForwarderCfg: share.ForwarderCfg{
			Debug:      true,
			Kubeconfig: "test-config",
			Kubernetes: true,
			LinkDevice: "eth0",
		},
	}
	
	// Create manager
	mgr := NewMgr(config, clientset)
	
	// Create test service
	service := createTestService()
	
	// Create managed listener
	ml := NewManagedListener(service, config, clientset)
	
	// Add listener to manager
	mgr.Set("default/test-service", ml)
	
	// Shutdown manager
	mgr.Shutdown()
	
	// Verify all listeners were removed
	assert.Equal(t, 0, len(mgr.Listeners), "Expected no listeners in manager after shutdown")
	
	// Verify shutdown channel was closed
	select {
	case <-shutdown:
		// Channel is closed, which is expected
	default:
		t.Fatal("Expected shutdown channel to be closed")
	}
	
	// Reset shutdown channel for other tests
	shutdown = make(chan struct{})
}