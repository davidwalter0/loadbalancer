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

package framework

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ServiceConfig holds configuration for creating a test service
type ServiceConfig struct {
	Name        string
	Namespace   string
	Ports       []ServicePort
	Selector    map[string]string
	Annotations map[string]string
}

// ServicePort represents a port configuration for a service
type ServicePort struct {
	Name       string
	Port       int32
	TargetPort int32
	Protocol   corev1.Protocol
}

// CreateLoadBalancerService creates a LoadBalancer type service
func (f *Framework) CreateLoadBalancerService(config ServiceConfig) (*corev1.Service, error) {
	if config.Namespace == "" {
		config.Namespace = f.Namespace
	}

	servicePorts := []corev1.ServicePort{}
	for _, p := range config.Ports {
		protocol := p.Protocol
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		
		servicePort := corev1.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: intstr.FromInt(int(p.TargetPort)),
			Protocol:   protocol,
		}
		servicePorts = append(servicePorts, servicePort)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        config.Name,
			Namespace:   config.Namespace,
			Annotations: config.Annotations,
			Labels: map[string]string{
				"e2e-test": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: config.Selector,
			Ports:    servicePorts,
		},
	}

	log.Printf("Creating LoadBalancer service %s/%s", config.Namespace, config.Name)
	
	createdService, err := f.KubeClient.CoreV1().Services(config.Namespace).Create(
		context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Register cleanup function
	f.AddCleanupFunc(func() error {
		return f.DeleteService(config.Name, config.Namespace)
	})

	return createdService, nil
}

// DeleteService deletes a service
func (f *Framework) DeleteService(name, namespace string) error {
	if namespace == "" {
		namespace = f.Namespace
	}

	log.Printf("Deleting service %s/%s", namespace, name)
	
	err := f.KubeClient.CoreV1().Services(namespace).Delete(
		context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// WaitForServiceExternalIP waits until the service has an external IP assigned
func (f *Framework) WaitForServiceExternalIP(name, namespace string, timeout time.Duration) (string, error) {
	if namespace == "" {
		namespace = f.Namespace
	}

	log.Printf("Waiting for service %s/%s to get an external IP", namespace, name)
	
	var externalIP string
	
	err := wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		service, err := f.KubeClient.CoreV1().Services(namespace).Get(
			context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			externalIP = service.Status.LoadBalancer.Ingress[0].IP
			return true, nil
		}
		
		if len(service.Spec.ExternalIPs) > 0 {
			externalIP = service.Spec.ExternalIPs[0]
			return true, nil
		}
		
		return false, nil
	})
	
	if err != nil {
		return "", fmt.Errorf("timed out waiting for service external IP: %w", err)
	}
	
	log.Printf("Service %s/%s has external IP: %s", namespace, name, externalIP)
	return externalIP, nil
}

// CreateTestPod creates a pod that serves HTTP traffic for testing
func (f *Framework) CreateTestPod(name string, labels map[string]string, port int32) (*corev1.Pod, error) {
	if labels == nil {
		labels = map[string]string{
			"app": name,
		}
	}
	
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: port,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/",
								Port: intstr.FromInt(int(port)),
							},
						},
						InitialDelaySeconds: 5,
						TimeoutSeconds:      1,
						PeriodSeconds:       5,
					},
				},
			},
		},
	}
	
	log.Printf("Creating test pod %s/%s", f.Namespace, name)
	
	createdPod, err := f.KubeClient.CoreV1().Pods(f.Namespace).Create(
		context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create test pod: %w", err)
	}
	
	// Register cleanup function
	f.AddCleanupFunc(func() error {
		return f.DeletePod(name)
	})
	
	// Wait for pod to be ready
	err = f.WaitForPodReady(name, 2*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("test pod failed to become ready: %w", err)
	}
	
	return createdPod, nil
}

// DeletePod deletes a pod
func (f *Framework) DeletePod(name string) error {
	log.Printf("Deleting pod %s/%s", f.Namespace, name)
	
	err := f.KubeClient.CoreV1().Pods(f.Namespace).Delete(
		context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete pod: %w", err)
	}
	
	return nil
}

// WaitForPodReady waits for a pod to be in ready state
func (f *Framework) WaitForPodReady(name string, timeout time.Duration) error {
	log.Printf("Waiting for pod %s/%s to be ready", f.Namespace, name)
	
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		pod, err := f.KubeClient.CoreV1().Pods(f.Namespace).Get(
			context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return false, fmt.Errorf("pod %s/%s has completed with phase: %s", f.Namespace, name, pod.Status.Phase)
		}
		
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		
		return false, nil
	})
}