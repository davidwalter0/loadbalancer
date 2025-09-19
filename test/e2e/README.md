# End-to-End Testing

This directory contains end-to-end tests for the loadbalancer. The tests verify that the loadbalancer works correctly in a real Kubernetes environment.

## Prerequisites

- A running Kubernetes cluster (local or remote)
- `kubectl` configured to access the cluster
- Permissions to create and manage resources in the cluster
- Built loadbalancer binary

## Running Tests

### Using Make

The simplest way to run the tests is using the provided Make targets:

```bash
# Run all tests
make e2e-test

# Run only basic connectivity tests
make e2e-test-basic

# Run only health check tests
make e2e-test-health

# Run tests without cleaning up on failure (for debugging)
make e2e-test-debug

# Run tests in CI environment
make e2e-test-ci
```

### Direct Execution

You can also run the tests directly:

```bash
# Build the loadbalancer binary
go build -o bin/loadbalancer

# Run all tests
LOADBALANCER_BINARY=$(pwd)/bin/loadbalancer go test ./test/e2e -v

# Run specific tests
LOADBALANCER_BINARY=$(pwd)/bin/loadbalancer E2E_TEST_FILTER=BasicConnectivity,HealthCheck go test ./test/e2e -v
```

## Configuration

The tests can be configured using environment variables or command-line flags:

| Environment Variable   | Command-line Flag       | Description                           | Default           |
|------------------------+-------------------------+---------------------------------------+-------------------|
| `KUBECONFIG`           | `--kubeconfig`          | Path to kubeconfig file               | `~/.kube/config`  |
| `E2E_NAMESPACE`        | `--namespace`           | Namespace to use for tests            | auto-generated    |
| `E2E_NETWORK_IFACE`    | `--iface`               | Network interface for loadbalancer    | `eth0`            |
| `LOADBALANCER_BINARY`  | `--lb-binary`           | Path to loadbalancer binary           | auto-detected     |
| `E2E_TEST_FILTER`      | `--test-filter`         | Tests to run (comma-separated)        | all tests         |
| `E2E_CLEANUP_ON_FAIL`  | `--cleanup-on-fail`     | Clean up resources on test failure    | `true`            |
| `E2E_CLUSTER_TESTS`    | `--cluster-tests`       | Run tests requiring a cluster         | `true`            |

## Available Tests

The following test scenarios are available:

- `BasicConnectivity`: Verifies basic connectivity to a service
- `MultipleServices`: Tests multiple services simultaneously
- `ServiceUpdates`: Tests handling of service updates
- `EndpointChanges`: Tests handling of endpoint changes
- `HealthCheck`: Verifies health check functionality
- `LoadBalancing`: Tests load distribution across backends
- `ShutdownAndRestart`: Tests stopping and restarting the loadbalancer
- `HighAvailabilityConfiguration`: Tests HA features

## Test Structure

- `framework/`: Core test framework utilities
- `scenarios/`: Test scenario implementations
- `utils/`: Utility functions for HTTP and network testing
- `e2e_test.go`: Main test runner

## Documentation

For more detailed information, see the [E2E Testing Documentation](../doc/e2e-testing.org).