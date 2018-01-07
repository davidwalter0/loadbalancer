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

package pipe

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/llb/ipmgr"
	"github.com/davidwalter0/llb/share"
	"github.com/davidwalter0/llb/tracer"
)

// NewPipe creates a Pipe and returns a pointer to the same
func NewPipe(Key string, mapAdd, mapRm chan *Pipe, mutex *mutex.Mutex, source, sink net.Conn, definition *Definition) (pipe *Pipe) {
	defer trace.Tracer.ScopedTrace()()
	defer mutex.Monitor()()
	pipe = &Pipe{
		Key:        Key,
		SourceConn: source,
		SinkConn:   sink,
		MapRm:      mapRm,
		Mutex:      mutex,
		Definition: *definition,
	}

	pipe.Endpoints = definition.Endpoints
	mapAdd <- pipe
	return
}

// Pipe a connection initiated by the return from listen and the
// up/down stream host:port pairs
type Pipe struct {
	Key        string
	SourceConn net.Conn
	SinkConn   net.Conn
	MapRm      chan *Pipe
	State      uint64
	Mutex      *mutex.Mutex
	Mode       string
	Definition
	*ipmgr.CIDR
}

// Monitor lock link into
func (pipe *Pipe) Monitor(args ...interface{}) func() {
	if pipe != nil {
		defer trace.Tracer.ScopedTrace(args...)()
		return pipe.Mutex.MonitorTrace(args...)
	}
	return func() {}
}

// Connect opens a link between source and sink
func (pipe *Pipe) Connect() {
	if pipe != nil {
		done := make(chan bool, 2)
		defer trace.Tracer.ScopedTrace()()
		go func() {
			defer trace.Tracer.ScopedTrace()()
			defer pipe.Close()
			if _, err := io.Copy(pipe.SinkConn, pipe.SourceConn); err != nil {
				log.Println(err)
			}
			done <- true
		}()
		go func() {
			defer trace.Tracer.ScopedTrace()()
			defer pipe.Close()
			if _, err := io.Copy(pipe.SourceConn, pipe.SinkConn); err != nil {
				log.Println(err)
			}
			done <- true
		}()
		go func() {
			log.Println("share.Queue.Push(&pipe.Definition)", pipe.Definition)
			// share.Queue.Push(&pipe.Definition)
			for {
				ticker := time.NewTicker(share.TickDelay)
				defer ticker.Stop()
				select {
				case <-done:
					return
				case <-ticker.C:
					log.Println("share.Queue.Push(&pipe.Definition)", pipe.Definition)
					// share.Queue.Push(&pipe.Definition)
				}
			}
		}()
	}
}

// Close a link between source and sink
func (pipe *Pipe) Close() {
	if pipe != nil {
		defer trace.Tracer.ScopedTrace()()
		defer pipe.Monitor()()
		if pipe.State == share.Open {
			pipe.MapRm <- pipe
			pipe.State = share.Closed
			if err := pipe.SinkConn.Close(); err != nil {
				log.Println(err)
			}
			if err := pipe.SourceConn.Close(); err != nil {
				log.Println(err)
			}
		}
	}
}
