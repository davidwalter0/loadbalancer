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
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// HTTPClient is a utility for making HTTP requests with timeouts
type HTTPClient struct {
	Client  *http.Client
	Timeout time.Duration
}

// NewHTTPClient creates a new HTTP client with the specified timeout
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &HTTPClient{
		Client:  client,
		Timeout: timeout,
	}
}

// Get performs an HTTP GET request
func (c *HTTPClient) Get(url string) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return resp.StatusCode, body, nil
}

// WaitForHTTPStatus waits for the specified URL to return the expected status code
func (c *HTTPClient) WaitForHTTPStatus(url string, expectedStatus int, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, _, err := c.Get(url)
		if err == nil && status == expectedStatus {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for HTTP status %d from %s after %v", expectedStatus, url, timeout)
}

// WaitForHTTPContent waits for the specified URL to return content containing the expected string
func (c *HTTPClient) WaitForHTTPContent(url, expectedContent string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, body, err := c.Get(url)
		if err == nil && status == http.StatusOK && string(body) == expectedContent {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for HTTP content from %s after %v", url, timeout)
}