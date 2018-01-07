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

package nodemgr

import (
	"sort"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/llb/tracer"
)

// NodeList has
type NodeList struct {
	*mutex.Mutex
	Nodes  map[string]bool
	List   []string
	Change bool
}

// NewNodeList returns a *NodeList
func NewNodeList() *NodeList {
	defer trace.Tracer.ScopedTrace()()
	return &NodeList{
		Mutex: mutex.NewMutex(),
		Nodes: map[string]bool{},
	}
}

// AddNode to node list
func (nl *NodeList) AddNode(node string) {
	defer trace.Tracer.ScopedTrace()()
	defer nl.Mutex.Monitor()()
	nl.Nodes[node] = true
	nl.updateList()
}

// RemoveNode to node list
func (nl *NodeList) RemoveNode(node string) {
	defer trace.Tracer.ScopedTrace()()
	defer nl.Mutex.Monitor()()
	delete(nl.Nodes, node)
	nl.updateList()

}

// updateList changes the list
func (nl *NodeList) updateList() {
	nl.List = []string{}
	for node := range nl.Nodes {
		nl.List = append(nl.List, node)
	}
	sort.Strings(nl.List)
}

// GetNodes to node list
func (nl *NodeList) GetNodes() []string {
	defer trace.Tracer.ScopedTrace()()
	defer nl.Mutex.Monitor()()
	return nl.List
}

// HasNode to node list
func (nl *NodeList) HasNode(node string) bool {
	defer trace.Tracer.ScopedTrace()()
	defer nl.Mutex.Monitor()()
	_, ok := nl.Nodes[node]
	return ok
}
