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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/davidwalter0/go-cfg"
	"github.com/davidwalter0/loadbalancer/kubeconfig"
	"github.com/davidwalter0/loadbalancer/watch"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var debug = false

// ServerCfg runtime options config struct
type ServerCfg struct {
	Debug      bool   `json:"debug"       doc:"increase verbosity"                               default:"false"`
	Context    string `json:"context"     doc:"context set in configuration"                     default:"northlake"`
	Namespace  string `json:"namespace"   doc:"namespace"                                        default:"typhoon"`
	Kubeconfig string `json:"kubeconfig"  doc:"kubernetes auth secrets / configuration file"     default:"cluster/auth/kubeconfig"`
	Kubernetes bool   `json:"kubernetes"  doc:"use kubernetes dynamic endpoint from service/ns" default:"true"`
}

// Read from env variables or command line flags
func (envCfg *ServerCfg) Read() {
	var err error
	if err = cfg.AddStruct(envCfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
	cfg.Finalize()
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

type Service struct {
	v1.Service
}

func (service *Service) ServiceSelector(gateway string) string {
	parts := strings.Split(gateway, "-")
	if len(parts) == 2 {
		lane := parts[0]
		fmt.Printf("Lane %s selector name: %s service: %s\n", lane, service.Spec.Selector["name"], gateway)
		return lane
	} else {
		fmt.Printf("Lane Selector not found %+v\n", service.Spec.Selector)
		return ""
	}
}

type Lanes struct {
	Active   string
	Inactive string
}

func (service *Service) Active() string {
	lanes := service.Lanes()
	if lanes == nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("Lanes not returned"))
		return ""
	}
	return lanes.Active
}

func (service *Service) Inactive() string {
	lanes := service.Lanes()
	if lanes == nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("Lanes not returned"))
		return ""
	}
	return lanes.Inactive
}

func (service *Service) Lanes() *Lanes {
	selector := service.Spec.Selector
	gateway := selector["name"]
	parts := strings.Split(gateway, "-")
	if len(parts) == 2 {
		lane := parts[0]
		// fmt.Printf("Lane %s selector name: %s service: %s\n", lane, service.Spec.Selector["name"], gateway)
		var lanes *Lanes
		switch lane {
		case "green":
			lanes = &Lanes{lane, "blue"}
		case "blue":
			lanes = &Lanes{lane, "green"}
		}

		return lanes
	}
	// fmt.Printf("Lane Selector not found %+v\n", service.Spec.Selector)
	return nil
}

func main() {
	var namespace = "typhoon"

	for _, context := range []string{"hillsboro", "northlake", envCfg.Context} {
		Another(context, namespace)
	}
	os.Exit(0)
	clientset := kubeconfig.NewClientset(envCfg.Kubeconfig)
	if clientset == nil {
		log.Fatal("Kubernetes connection failed")
	}

	services, err := clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
	for i, service := range services.Items {
		name := service.ObjectMeta.Name
		if service.ObjectMeta.Name == "gateway" {
			if debug {
				fmt.Printf("Active lane = %-32s %+v\n", name, service.Spec)
				fmt.Printf("Lane Selector Name = %+v\n", service.Spec.Selector["name"])
			}
			svc := Service{service}
			svc.ServiceSelector(service.Spec.Selector["name"])
			fmt.Printf("Lanes %+v\n", svc.Lanes())
			fmt.Println("Active", svc.Active())
			fmt.Println("Inactive", svc.Inactive())
			if debug {
				if jsonText, err := json.MarshalIndent(service, "", "  "); err == nil {
					fmt.Printf("-------------------------\nservice\n-------------------------\n%s\n", string(jsonText))
				} else {
					fmt.Println(err)
				}
			}
			break
		} else {
			continue
		}
		fmt.Println(i, err)
		if jsonText, err := json.MarshalIndent(service, "", "  "); err == nil {
			fmt.Printf("-------------------------\nservice\n-------------------------\n%s\n", string(jsonText))
		} else {
			fmt.Println(err)
		}
		// if service.Object.Name == "gateway" {
		// 	fmt.Println("Active lane =", service.Object)
		// }
		// fmt.Println("Active lane =", service.Object)
		// }
	}
	// if false {
	if true {
		serviceWatcher := watch.NewQueueMgr(watch.ServiceAPIName, clientset)
		go serviceWatcher.Run(1, 1)

		time.Sleep(10 * time.Second)
		for { // i := 0; i < 100; i++ {
			select {
			case item := <-serviceWatcher.QueueItems:
				service := item.Interface.(*v1.Service)

				if service != nil && strings.Index(service.Name, "gateway") >= 0 && service.ObjectMeta.Name == "gateway" {
					svc := Service{*service}
					svc.ServiceSelector(service.Spec.Selector["name"])
					fmt.Printf("Lanes %+v\n", svc.Lanes())
					fmt.Println("Active", svc.Active())
					fmt.Println("Inactive", svc.Inactive())
					// fmt.Println(service.Name)
					// name := service.ObjectMeta.Name
					// fmt.Println("Active lane =", service.ObjectMeta, name)
					return
				}
			default:
				// non blocking channel, choose default and break
				break
			}
		}
	}
}

func Another(context, namespace string) {
	// namespace := "typhoon"

	config, clientset, err := GetKubeClient(context)
	if err != nil {
		fmt.Println(err)
	}
	services, err := clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Config: %+v\n", config)
	for i, service := range services.Items {
		name := service.ObjectMeta.Name
		if service.ObjectMeta.Name == "gateway" {
			if debug {
				fmt.Printf("Active lane = %-32s %+v\n", name, service.Spec)
				fmt.Printf("Lane Selector Name = %+v\n", service.Spec.Selector["name"])
			}
			svc := Service{service}
			svc.ServiceSelector(service.Spec.Selector["name"])
			fmt.Printf("Lanes %+v\n", svc.Lanes())
			fmt.Println("Active", svc.Active())
			fmt.Println("Inactive", svc.Inactive())
			if debug {
				if jsonText, err := json.MarshalIndent(service, "", "  "); err == nil {
					fmt.Printf("-------------------------\nservice\n-------------------------\n%s\n", string(jsonText))
				} else {
					fmt.Println(err)
				}
			}
			break
		} else {
			continue
		}
		fmt.Println(i, err)
		if jsonText, err := json.MarshalIndent(service, "", "  "); err == nil {
			fmt.Printf("-------------------------\nservice\n-------------------------\n%s\n", string(jsonText))
		} else {
			fmt.Println(err)
		}
		// if service.Object.Name == "gateway" {
		// 	fmt.Println("Active lane =", service.Object)
		// }
		// fmt.Println("Active lane =", service.Object)
		// }
	}
}

// GetKubeClient creates a Kubernetes config and client for a given kubeconfig context.
// func GetKubeClient(context string) (*rest.Config, kubernetes.Interface, error) {
func GetKubeClient(context string) (*rest.Config, *kubernetes.Clientset, error) {
	config, err := configForContext(context)
	if err != nil {
		return nil, nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return config, client, nil
}

// configForContext creates a Kubernetes REST client configuration for a given kubeconfig context.
func configForContext(context string) (*rest.Config, error) {
	config, err := getConfig(context).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config for context %q: %s", context, err)
	}
	return config, nil
}

// getConfig returns a Kubernetes client config for a given context.
func getConfig(context string) clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

	if context != "" {
		overrides.CurrentContext = context
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
}

// func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
// 	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
// 		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
// 		&clientcmd.ConfigOverrides{
// 			CurrentContext: context,
// 		}).ClientConfig()
// }
