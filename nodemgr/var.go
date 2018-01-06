package nodemgr

var nodelist = NewNodeList()

// NodeListPtr returns an instance of a managed list of nodes
func NodeListPtr() *NodeList {
	return nodelist
}
