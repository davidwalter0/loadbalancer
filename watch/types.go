package watch

// EventType for watcher
type EventType string

const (
	// ADD EventType
	ADD EventType = "ADD"
	// DELETE EventType
	DELETE EventType = "DELETE"
	// UPDATE EventType
	UPDATE EventType = "UPDATE"
)

// K8sAPIName for v1 service types
type K8sAPIName string

const (
	// NodeAPIName k8s api name
	NodeAPIName K8sAPIName = "nodes"
	// ServiceAPIName k8s api name
	ServiceAPIName K8sAPIName = "services"
	// EndpointAPIName k8s api name
	EndpointAPIName K8sAPIName = "endpoints"
	// PodAPIName k8s api name
	PodAPIName K8sAPIName = "pods"
)
