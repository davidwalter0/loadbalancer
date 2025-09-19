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

package utils

import (
	"context"
	"fmt"
	"net"
	"time"
)

// IsPortOpen checks if a port is open on the specified host
func IsPortOpen(host string, port int, timeout time.Duration) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// WaitForPortOpen waits for a port to be open on the specified host
func WaitForPortOpen(host string, port int, timeout, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for port %d to be open on %s", port, host)
		default:
			if IsPortOpen(host, port, interval) {
				return nil
			}
			time.Sleep(interval)
		}
	}
}

// WaitForPortClosed waits for a port to be closed on the specified host
func WaitForPortClosed(host string, port int, timeout, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for port %d to be closed on %s", port, host)
		default:
			if !IsPortOpen(host, port, interval) {
				return nil
			}
			time.Sleep(interval)
		}
	}
}

// GetAvailablePort finds an available port on the local system
func GetAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}