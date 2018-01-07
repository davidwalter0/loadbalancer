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
