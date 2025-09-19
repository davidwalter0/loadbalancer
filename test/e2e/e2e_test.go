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

package e2e

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/davidwalter0/loadbalancer/test/e2e/framework"
	"github.com/davidwalter0/loadbalancer/test/e2e/scenarios"
)

var (
	kubeconfig      string
	namespace       string
	networkIface    string
	lbBinary        string
	testFilter      string
	cleanupOnFail   bool
	runClusterTests bool
)

func init() {
	// Get kubeconfig from environment or default location
	if home, err := os.UserHomeDir(); err == nil {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "Path to kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	}

	// Set up other flags
	flag.StringVar(&namespace, "namespace", "", "Namespace to use for tests (defaults to auto-generated)")
	flag.StringVar(&networkIface, "iface", "eth0", "Network interface to use for tests")
	flag.StringVar(&lbBinary, "lb-binary", "", "Path to loadbalancer binary")
	flag.StringVar(&testFilter, "test-filter", "", "Filter for test names to run (comma-separated)")
	flag.BoolVar(&cleanupOnFail, "cleanup-on-fail", true, "Clean up resources even when tests fail")
	flag.BoolVar(&runClusterTests, "cluster-tests", true, "Run tests that require a Kubernetes cluster")

	// Override with environment variables if set
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	if os.Getenv("E2E_NAMESPACE") != "" {
		namespace = os.Getenv("E2E_NAMESPACE")
	}
	if os.Getenv("E2E_NETWORK_IFACE") != "" {
		networkIface = os.Getenv("E2E_NETWORK_IFACE")
	}
	if os.Getenv("LOADBALANCER_BINARY") != "" {
		lbBinary = os.Getenv("LOADBALANCER_BINARY")
	}
	if os.Getenv("E2E_TEST_FILTER") != "" {
		testFilter = os.Getenv("E2E_TEST_FILTER")
	}
	if os.Getenv("E2E_CLEANUP_ON_FAIL") != "" {
		cleanupOnFail = os.Getenv("E2E_CLEANUP_ON_FAIL") == "true"
	}
	if os.Getenv("E2E_CLUSTER_TESTS") != "" {
		runClusterTests = os.Getenv("E2E_CLUSTER_TESTS") == "true"
	}

	// Ensure binary is set
	if lbBinary != "" {
		os.Setenv("LOADBALANCER_BINARY", lbBinary)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	// Set up signal handling
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		fmt.Println("\nReceived interrupt, cleaning up...")
		os.Exit(1)
	}()

	// Run tests
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	if !runClusterTests {
		t.Skip("Skipping cluster tests")
	}

	// Create framework
	f, err := framework.New(namespace, kubeconfig, networkIface)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}

	// Set up test environment
	if err := f.Setup(); err != nil {
		t.Fatalf("Failed to set up test environment: %v", err)
	}

	// Register cleanup
	defer func() {
		if !cleanupOnFail && t.Failed() {
			log.Println("Test failed and cleanup-on-fail=false, skipping cleanup")
			log.Printf("Test namespace %s has been preserved for debugging", f.Namespace)
			return
		}
		if err := f.Teardown(); err != nil {
			t.Logf("Warning: failed to clean up test environment: %v", err)
		}
	}()

	// Parse test filter
	var testFilters []string
	if testFilter != "" {
		testFilters = strings.Split(testFilter, ",")
	}

	// Run test scenarios
	testFuncs := scenarios.GetTests()
	for name, testFunc := range testFuncs {
		// Skip if filter is specified and test name doesn't match
		if len(testFilters) > 0 {
			matched := false
			for _, filter := range testFilters {
				if strings.Contains(name, filter) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		t.Run(name, func(t *testing.T) {
			testFunc(t, f)
		})
	}
}