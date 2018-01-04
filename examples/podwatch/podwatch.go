package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"k8s.io/api/core/v1"

	"github.com/davidwalter0/go-cfg"
	"github.com/davidwalter0/llb/kubeconfig"
	"github.com/davidwalter0/llb/watch"
)

// ServerCfg runtime options config struct
type ServerCfg struct {
	Debug      bool   `json:"debug"       doc:"increase verbosity"                               default:"false"`
	Kubeconfig string `json:"kubeconfig"  doc:"kubernetes auth secrets / configuration file"     default:"cluster/auth/kubeconfig"`
	Kubernetes bool   `json:"kubernetes"  doc:"use kubernetes dynamic endpoints from service/ns" default:"true"`
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

func main() {
	clientset := kubeconfig.NewClientset(envCfg.Kubeconfig)
	if clientset == nil {
		log.Fatal("Kubernetes connection failed")
	}

	podWatcher := watch.NewQueueMgr(watch.PodAPIName, clientset)
	go podWatcher.Run()

	time.Sleep(10 * time.Second)
	for i := 0; i < 100; i++ {
		select {
		case item := <-podWatcher.QueueItems:
			pod := item.Interface.(*v1.Pod)
			fmt.Println(pod.Name)
			for key, value := range pod.ObjectMeta.Labels {
				fmt.Printf("  %-32s %s\n", key, value)
			}
			if false {
				if jsonText, err := json.MarshalIndent(pod, "", "  "); err == nil {
					fmt.Println(string(jsonText))
				} else {
					fmt.Println(err)
				}
			}
		default:
			// non blocking channel, choose default and break
			break
		}
	}
}
