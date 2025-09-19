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

package pipe

import (
	"bytes"
	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/share"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// MockConn implements net.Conn for testing
type MockConn struct {
	Reader *bytes.Buffer
	Writer *bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func NewMockConn() *MockConn {
	return &MockConn{
		Reader: bytes.NewBuffer(nil),
		Writer: bytes.NewBuffer(nil),
	}
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.EOF
	}
	return m.Reader.Read(b)
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.Writer.Write(b)
}

func (m *MockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *MockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *MockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestNewPipe(t *testing.T) {
	// Create mock connections
	sourceConn := NewMockConn()
	sinkConn := NewMockConn()
	
	// Create channels for test
	mapAdd := make(chan *Pipe, 3)
	mapRm := make(chan *Pipe, 3)
	
	// Create definition
	definition := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
	}
	
	// Create pipe
	go func() {
		pipe := NewPipe("test/key", mapAdd, mapRm, sourceConn, sinkConn, definition)
		assert.NotNil(t, pipe, "Expected pipe to be created")
	}()
	
	// Verify pipe was added to mapAdd channel
	select {
	case pipe := <-mapAdd:
		assert.NotNil(t, pipe, "Expected pipe to be added to mapAdd channel")
		assert.Equal(t, "test/key", pipe.Key, "Expected pipe key to match")
		assert.Equal(t, sourceConn, pipe.SourceConn, "Expected source connection to match")
		assert.Equal(t, sinkConn, pipe.SinkConn, "Expected sink connection to match")
		assert.Equal(t, share.Open, pipe.State, "Expected pipe state to be Open")
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for pipe to be added to mapAdd channel")
	}
}

func TestPipeConnect(t *testing.T) {
	// Create mock connections
	sourceConn := NewMockConn()
	sinkConn := NewMockConn()
	
	// Write test data to source connection
	testData := []byte("Hello, World!")
	sourceConn.Reader.Write(testData)
	
	// Create channels for test
	mapAdd := make(chan *Pipe, 3)
	mapRm := make(chan *Pipe, 3)
	
	// Create definition
	definition := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
	}
	
	// Create and connect pipe
	pipe := NewPipe("test/key", mapAdd, mapRm, sourceConn, sinkConn, definition)
	
	// Drain mapAdd channel
	<-mapAdd
	
	// Connect the pipe (this would normally be done in a goroutine)
	done := make(chan bool)
	go func() {
		pipe.Connect()
		done <- true
	}()
	
	// Wait for data to be copied
	time.Sleep(50 * time.Millisecond)
	
	// Verify data was copied from source to sink
	assert.Equal(t, testData, sinkConn.Writer.Bytes(), "Expected data to be copied from source to sink")
	
	// Write response data to sink connection
	responseData := []byte("Response!")
	sinkConn.Reader.Write(responseData)
	
	// Wait for data to be copied
	time.Sleep(50 * time.Millisecond)
	
	// Verify data was copied from sink to source
	assert.Equal(t, responseData, sourceConn.Writer.Bytes(), "Expected data to be copied from sink to source")
	
	// Close connections to complete the test
	sourceConn.Close()
	sinkConn.Close()
	
	// Wait for connect to complete
	select {
	case <-done:
		// Test completed successfully
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for pipe to complete")
	}
}

func TestPipeClose(t *testing.T) {
	// Create mock connections
	sourceConn := NewMockConn()
	sinkConn := NewMockConn()
	
	// Create channels for test
	mapAdd := make(chan *Pipe, 3)
	mapRm := make(chan *Pipe, 3)
	
	// Create definition
	definition := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
	}
	
	// Create pipe
	pipe := NewPipe("test/key", mapAdd, mapRm, sourceConn, sinkConn, definition)
	
	// Drain mapAdd channel
	<-mapAdd
	
	// Verify initial state
	assert.Equal(t, share.Open, pipe.State, "Expected initial state to be Open")
	assert.False(t, sourceConn.IsClosed(), "Expected source connection to be open")
	assert.False(t, sinkConn.IsClosed(), "Expected sink connection to be open")
	
	// Close the pipe
	go pipe.Close()
	
	// Verify pipe was added to mapRm channel
	select {
	case removed := <-mapRm:
		assert.Equal(t, pipe, removed, "Expected pipe to be added to mapRm channel")
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for pipe to be added to mapRm channel")
	}
	
	// Verify connections were closed
	assert.True(t, sourceConn.IsClosed(), "Expected source connection to be closed")
	assert.True(t, sinkConn.IsClosed(), "Expected sink connection to be closed")
	
	// Verify state was updated
	assert.Equal(t, share.Closed, pipe.State, "Expected state to be Closed")
	
	// Verify calling Close again has no effect
	mapRm2 := make(chan *Pipe, 3) // New channel to ensure we don't get the previous removal
	pipe.MapRm = mapRm2
	pipe.Close()
	
	// Verify pipe was not added to mapRm channel again
	select {
	case <-mapRm2:
		t.Fatal("Pipe should not be added to mapRm channel again")
	case <-time.After(50 * time.Millisecond):
		// This is expected
	}
}

func TestDefinitionEqual(t *testing.T) {
	// Create base definition
	def1 := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
		Mode:      "tcp",
	}
	
	// Create identical definition
	def2 := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
		Mode:      "tcp",
	}
	
	// Create different definition
	def3 := &Definition{
		Key:       "different/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
		Mode:      "tcp",
	}
	
	// Test equality
	assert.True(t, def1.Equal(def2), "Expected identical definitions to be equal")
	assert.False(t, def1.Equal(def3), "Expected different definitions to not be equal")
	assert.False(t, def2.Equal(def3), "Expected different definitions to not be equal")
	
	// Test with more variations
	def4 := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "different:9090", // Different sink
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
		Mode:      "tcp",
	}
	
	def5 := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     false, // Different debug
		Mode:      "tcp",
	}
	
	assert.False(t, def1.Equal(def4), "Expected definitions with different sinks to not be equal")
	assert.False(t, def1.Equal(def5), "Expected definitions with different debug settings to not be equal")
}

func TestDefinitionCopy(t *testing.T) {
	// Create source definition
	src := &Definition{
		Key:       "test/key",
		Source:    "127.0.0.1:8080",
		Sink:      "127.0.0.1:9090",
		Name:      "test-service",
		Namespace: "default",
		Debug:     true,
		Mode:      "tcp",
	}
	
	// Create destination definition with different values
	dst := &Definition{
		Key:       "old/key",
		Source:    "old:8080",
		Sink:      "old:9090",
		Name:      "old-service",
		Namespace: "old",
		Debug:     false,
		Mode:      "old",
	}
	
	// Copy values
	result := dst.Copy(src)
	
	// Verify copy operation
	assert.Equal(t, dst, result, "Expected Copy to return the destination")
	assert.Equal(t, src.Key, dst.Key, "Expected Key to be copied")
	assert.Equal(t, src.Source, dst.Source, "Expected Source to be copied")
	assert.Equal(t, src.Sink, dst.Sink, "Expected Sink to be copied")
	assert.Equal(t, src.Name, dst.Name, "Expected Name to be copied")
	assert.Equal(t, src.Namespace, dst.Namespace, "Expected Namespace to be copied")
	assert.Equal(t, src.Debug, dst.Debug, "Expected Debug to be copied")
	assert.Equal(t, src.Mode, dst.Mode, "Expected Mode to be copied")
}

func TestEPEqual(t *testing.T) {
	// Create EP slices
	ep1 := EP{"endpoint1", "endpoint2", "endpoint3"}
	ep2 := EP{"endpoint3", "endpoint1", "endpoint2"} // Same elements, different order
	ep3 := EP{"endpoint1", "endpoint2", "endpoint4"} // Different elements
	ep4 := EP{"endpoint1", "endpoint2"} // Fewer elements
	
	// Test equality
	assert.True(t, ep1.Equal(&ep2), "Expected EPs with same elements to be equal regardless of order")
	assert.False(t, ep1.Equal(&ep3), "Expected EPs with different elements to not be equal")
	assert.False(t, ep1.Equal(&ep4), "Expected EPs with different length to not be equal")
	
	// Test with nil
	var nilEP *EP
	assert.False(t, ep1.Equal(nilEP), "Expected comparison with nil to return false")
	assert.False(t, (*EP)(nil).Equal(&ep1), "Expected comparison from nil to return false")
	assert.True(t, (*EP)(nil).Equal((*EP)(nil)), "Expected nil to equal nil")
}