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

// Package framework provides the core functionality for E2E testing of the loadbalancer
package framework

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Framework provides the core functionality and context for E2E tests
type Framework struct {
	KubeClient         *kubernetes.Clientset
	Namespace          string
	KubeConfigPath     string
	LoadBalancerBinary string
	LoadBalancerArgs   []string
	LoadBalancerCmd    *exec.Cmd
	LoadBalancerPID    int
	NetworkInterface   string
	TestDataDir        string
	Ctx                context.Context
	Cancel             context.CancelFunc
	CleanupFuncs       []func() error
}

// New creates a new E2E test framework
func New(namespace, kubeconfig, networkInterface string) (*Framework, error) {
	if namespace == "" {
		namespace = "e2e-test-" + fmt.Sprintf("%d", time.Now().Unix())
	}

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	if networkInterface == "" {
		networkInterface = "eth0" // Default for testing
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	f := &Framework{
		KubeClient:       client,
		Namespace:        namespace,
		KubeConfigPath:   kubeconfig,
		NetworkInterface: networkInterface,
		TestDataDir:      filepath.Join("test", "e2e", "testdata"),
		Ctx:              ctx,
		Cancel:           cancel,
		CleanupFuncs:     []func() error{},
	}

	// Find loadbalancer binary
	binaryPath := os.Getenv("LOADBALANCER_BINARY")
	if binaryPath == "" {
		// Try to find in common locations
		candidates := []string{
			"./loadbalancer",
			"./bin/loadbalancer",
			"../loadbalancer",
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				binaryPath = path
				break
			}
		}
		if binaryPath == "" {
			return nil, fmt.Errorf("loadbalancer binary not found, set LOADBALANCER_BINARY environment variable")
		}
	}
	f.LoadBalancerBinary = binaryPath

	return f, nil
}

// Setup prepares the E2E test environment
func (f *Framework) Setup() error {
	log.Printf("Setting up E2E test environment in namespace %s", f.Namespace)
	
	// Create test namespace
	if err := f.CreateNamespace(); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	
	// Register cleanup function to delete namespace
	f.AddCleanupFunc(func() error {
		return f.DeleteNamespace()
	})
	
	return nil
}

// Teardown cleans up the E2E test environment
func (f *Framework) Teardown() error {
	log.Println("Tearing down E2E test environment")
	
	// Stop loadbalancer process if running
	if f.LoadBalancerCmd != nil && f.LoadBalancerCmd.Process != nil {
		log.Println("Stopping loadbalancer process")
		if err := f.StopLoadBalancer(); err != nil {
			log.Printf("Warning: failed to stop loadbalancer: %v", err)
		}
	}
	
	// Run cleanup functions in reverse order
	for i := len(f.CleanupFuncs) - 1; i >= 0; i-- {
		if err := f.CleanupFuncs[i](); err != nil {
			log.Printf("Warning: cleanup function failed: %v", err)
		}
	}
	
	// Cancel context
	f.Cancel()
	
	return nil
}

// AddCleanupFunc adds a function to be called during teardown
func (f *Framework) AddCleanupFunc(fn func() error) {
	f.CleanupFuncs = append(f.CleanupFuncs, fn)
}

// StartLoadBalancer starts the loadbalancer process
func (f *Framework) StartLoadBalancer() error {
	log.Println("Starting loadbalancer process")
	
	args := append([]string{
		"--kubeconfig=" + f.KubeConfigPath,
		"--linkdevice=" + f.NetworkInterface,
		"--debug=true",
	}, f.LoadBalancerArgs...)
	
	cmd := exec.CommandContext(f.Ctx, f.LoadBalancerBinary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start loadbalancer: %w", err)
	}
	
	f.LoadBalancerCmd = cmd
	f.LoadBalancerPID = cmd.Process.Pid
	
	log.Printf("Loadbalancer started with PID %d", f.LoadBalancerPID)
	
	// Allow time for loadbalancer to initialize
	time.Sleep(2 * time.Second)
	
	return nil
}

// StopLoadBalancer stops the loadbalancer process
func (f *Framework) StopLoadBalancer() error {
	if f.LoadBalancerCmd == nil || f.LoadBalancerCmd.Process == nil {
		return nil
	}
	
	log.Println("Stopping loadbalancer process")
	
	// Try graceful shutdown first
	if err := f.LoadBalancerCmd.Process.Signal(os.Interrupt); err != nil {
		log.Printf("Warning: failed to send interrupt signal: %v", err)
		// Force kill if interrupt fails
		if err := f.LoadBalancerCmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill loadbalancer process: %w", err)
		}
	}
	
	// Wait for process to exit
	_, err := f.LoadBalancerCmd.Process.Wait()
	if err != nil {
		return fmt.Errorf("error waiting for loadbalancer process to exit: %w", err)
	}
	
	f.LoadBalancerCmd = nil
	f.LoadBalancerPID = 0
	
	return nil
}

// RunWithTimeout runs a function with a timeout
func (f *Framework) RunWithTimeout(timeout time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(f.Ctx, timeout)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("timed out after %v", timeout)
	}
}

// WaitForCondition waits for a condition to be true with the specified timeout and interval
func (f *Framework) WaitForCondition(timeout, interval time.Duration, condition func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		satisfied, err := condition()
		if err != nil {
			return err
		}
		if satisfied {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for condition after %v", timeout)
}