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
