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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/davidwalter0/go-cfg/v3/pkg/config"
	"github.com/davidwalter0/loadbalancer/kubeconfig"
	"github.com/davidwalter0/loadbalancer/watch"
)

// ServerCfg runtime options config struct
type ServerCfg struct {
	Debug      bool   `json:"debug"       doc:"increase verbosity"                               default:"false"`
	Kubeconfig string `json:"kubeconfig"  doc:"kubernetes auth secrets / configuration file"     default:"cluster/auth/kubeconfig"`
	Kubernetes bool   `json:"kubernetes"  doc:"use kubernetes dynamic endpoint from service/ns" default:"true"`
}

// Read from env variables or command line flags
func (envCfg *ServerCfg) Read() {
	var err error
	manager := config.NewManager()
	if err = manager.Load(envCfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

var envCfg = &ServerCfg{}

func init() {
	// Load the configuration
	envCfg.Read()
}

// Cfg exposes common configuration item
func Cfg() *ServerCfg {
	return envCfg
}

func main() {
	clientset := kubeconfig.NewClientset(envCfg.Kubeconfig)
	if clientset == nil {
		log.Fatal("Kubernetes connection failed")
	}

	serviceWatcher := watch.NewQueueMgr(watch.ServiceAPIName, clientset)
	go serviceWatcher.Run(1, 1)

	time.Sleep(10 * time.Second)
	for i := 0; i < 100; i++ {
		select {
		case item := <-serviceWatcher.QueueItems:
			service := item.Interface.(*v1.Service)
			fmt.Println(service.Name)

			if service != nil && service.Spec.Type == v1.ServiceTypeLoadBalancer {
				// for key, value := range service.ObjectMeta.Labels {
				// 	fmt.Printf("  %-32s %s\n", key, value)
				// }

				var err error
				if jsonText, err := json.MarshalIndent(service.Spec, "", "  "); err == nil {
					fmt.Printf("-------------------------\nservice\n-------------------------\n%s\n", string(jsonText))
				} else {
					fmt.Println(err)
				}

				var endpoint *v1.Endpoints
				namespace := service.ObjectMeta.Namespace
				name := service.ObjectMeta.Name
				endpoint, err = clientset.CoreV1().Endpoints(namespace).Get(context.Background(), name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					fmt.Printf("Endpoints %s in namespace %s not found\n", name, namespace)
				} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
					fmt.Printf("Error getting endpoint %s in namespace %s: %v\n", name, namespace, statusError.ErrStatus.Message)
				} else if err != nil {
					panic(err.Error())
				} else {
					fmt.Printf("Found endpoint %s in namespace %s\n", name, namespace)
					if endpoint != nil {
						for _, subset := range endpoint.Subsets {
							if jsonText, err := json.MarshalIndent(subset.Addresses, "", "  "); err == nil {
								fmt.Println(string(jsonText))
							} else {
								fmt.Println(err)
							}

							if jsonText, err := json.MarshalIndent(subset.Ports, "", "  "); err == nil {
								fmt.Println(string(jsonText))
							} else {
								fmt.Println(err)
							}
						}
					}
				}
			}

		default:
			// non blocking channel, choose default and break
			break
		}
	}
}
