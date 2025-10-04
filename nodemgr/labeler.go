/*

Copyright 2018-2025 David Walter.

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
	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// WorkerRoleLabel is the standard Kubernetes worker node label
	WorkerRoleLabel = "node-role.kubernetes.io/worker"
)

// LabelWorkerNodes applies the worker role label to all nodes that don't have it
func LabelWorkerNodes(clientset *kubernetes.Clientset) error {
	if clientset == nil {
		log.Println("Cannot label nodes: clientset is nil")
		return nil
	}

	// Get all nodes in the cluster
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing nodes: %v", err)
		return err
	}

	log.Printf("Found %d nodes in cluster", len(nodes.Items))

	labeledCount := 0
	for _, node := range nodes.Items {
		// Check if node already has the worker label
		if _, hasLabel := node.Labels[WorkerRoleLabel]; hasLabel {
			log.Printf("Node %s already has worker label, skipping", node.Name)
			continue
		}

		// Add the worker label
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		node.Labels[WorkerRoleLabel] = ""

		// Update the node
		_, err := clientset.CoreV1().Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Error labeling node %s: %v", node.Name, err)
			continue
		}

		log.Printf("Successfully labeled node %s with %s", node.Name, WorkerRoleLabel)
		labeledCount++
	}

	if labeledCount > 0 {
		log.Printf("Labeled %d nodes with worker role", labeledCount)
	} else {
		log.Println("No nodes needed labeling")
	}

	return nil
}
