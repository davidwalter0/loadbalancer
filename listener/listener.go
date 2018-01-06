package listener

import (
	"encoding/json"
	"log"
	"net"
	// "sync"
	"sync/atomic"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/davidwalter0/go-mutex"
	"github.com/davidwalter0/llb/helper"
	"github.com/davidwalter0/llb/ipmgr"
	"github.com/davidwalter0/llb/kubeconfig"
	"github.com/davidwalter0/llb/pipe"
	"github.com/davidwalter0/llb/share"
	"github.com/davidwalter0/llb/tracer"
)

var retries = 3

// ManagedListener and it's dependent objects
type ManagedListener struct {
	pipe.Definition
	Listener net.Listener        `json:"-"`
	Pipes    map[*pipe.Pipe]bool `json:"-"`
	Mutex    *mutex.Mutex        `json:"-"`
	// Wg         sync.WaitGroup      `json:"-"`
	Kubernetes bool `json:"-"`
	Debug      bool `json:"-"`
	n          uint64
	MapAdd     chan *pipe.Pipe
	MapRm      chan *pipe.Pipe
	StopWatch  chan bool
	Clientset  *kubernetes.Clientset
	Active     uint64
	V1Service  *v1.Service
	InCluster  bool
	*ipmgr.CIDR
}

// Monitor for this ManagedListener
func (ml *ManagedListener) Monitor(args ...interface{}) func() {
	if ml != nil {
		defer trace.Tracer.ScopedTrace(args...)()
		return ml.Mutex.MonitorTrace(args...)
	}
	return func() {}
}

// Insert pipe to map of pipes in managed listener
func (ml *ManagedListener) Insert(pipe *pipe.Pipe) {
	defer trace.Tracer.ScopedTrace("MapAdd", *pipe)()
	pipe.State = share.Open
	defer pipe.Monitor()()
	ml.Pipes[pipe] = true
	ml.Active = uint64(len(ml.Pipes))
}

// Delete pipe from map of pipes in managed listener
func (ml *ManagedListener) Delete(pipe *pipe.Pipe) {
	defer trace.Tracer.ScopedTrace("MapRm", *pipe)()
	pipe.State = share.Closed
	defer pipe.Monitor()()
	delete(ml.Pipes, pipe)
	ml.Active = uint64(len(ml.Pipes))
}

// PipeMapHandler adds, removes, closes and single threads access to map list
func (ml *ManagedListener) PipeMapHandler() {
	if ml != nil {
		for {
			select {
			case pipe := <-ml.MapAdd:
				if pipe != nil {
					ml.Insert(pipe)
				}
			case pipe := <-ml.MapRm:
				if pipe != nil {
					ml.Delete(pipe)
				}
			}
		}
	}
}

// Open listener for this endPtDef
func (ml *ManagedListener) Open() {
	if ml != nil {
		defer trace.Tracer.ScopedTrace()()
		go ml.Listening()
		go ml.PipeMapHandler()
		ml.SetExternalIP()
	}
}

// LoadEndpoints queries the service name for endpoints
func (ml *ManagedListener) LoadEndpoints() {
	if ml != nil && ml.InCluster {
		defer ml.Monitor()()
		var ep = pipe.EP{}
		if ep = kubeconfig.Endpoints(ml.Clientset, ml.Name, ml.Namespace); !ep.Equal(&ml.Endpoints) {
			ml.Endpoints = ep
		}
	}
}

// NextEndPoint returns the next host:port pair if more than one
// available round robin selection
func (ml *ManagedListener) NextEndPoint() (sink string) {
	if ml != nil {
		defer trace.Tracer.ScopedTrace()()
		defer ml.Monitor()()
		var n uint64
		// Don't use k8s endpoint lookup if not in a k8s cluster
		sinks := helper.ServiceSinks(ml.V1Service)
		n = atomic.AddUint64(&ml.n, 1) % uint64(len(sinks))
		sink = sinks[n]
	}
	return
}

// Accept expose ManagedListener's listener
func (ml *ManagedListener) Accept() (net.Conn, error) {
	defer trace.Tracer.ScopedTrace()()
	return ml.Listener.Accept()
}

// StopWatchNotify checking for endpoints
func (ml *ManagedListener) StopWatchNotify() {
	if ml != nil {
		ml.StopWatch <- true
	}
}

// EpWatcher check for endpoints
func (ml *ManagedListener) EpWatcher() {
	if ml != nil && ml.InCluster {
		ticker := time.NewTicker(share.TickDelay)
		defer ticker.Stop()
		for {
			select {
			case <-ml.StopWatch:
				return
			case <-ticker.C:
				ml.LoadEndpoints()
				if ml.Debug {
					log.Println(ml.Key, ml.Source, ml.Sink, ml.Name, ml.Namespace, ml.Debug, ml.Endpoints, "active", ml.Active)
				}
			}
		}
	}
}

// Listening on managed listener
func (ml *ManagedListener) Listening() {
	defer trace.Tracer.ScopedTrace()()
	defer ml.StopWatchNotify()
	go ml.EpWatcher()
	for {
		var err error
		var SourceConn, SinkConn net.Conn
		if SourceConn, err = ml.Accept(); err != nil {
			log.Printf("Source connection failed: %v\n", err)
			break
		}
		sink := ml.NextEndPoint()
		log.Println(sink)
		SinkConn, err = net.Dial("tcp", sink)
		if err != nil {
			log.Printf("Sink connection failed: %v\n", err)
			break
		}
		var pipe = pipe.NewPipe(ml.Key, ml.MapAdd, ml.MapRm, ml.Mutex, SourceConn, SinkConn, &ml.Definition)
		go pipe.Connect()
	}
}

// Close a listener and it's children
func (ml *ManagedListener) Close() {
	if ml != nil {
		defer trace.Tracer.ScopedTrace()()
		if ml.Listener != nil {
			if err := ml.Listener.Close(); err != nil {
				log.Println("Error closing listener", ml.Listener)
			}
			defer ml.Monitor()()
			var pipes = []*pipe.Pipe{}
			for pipe := range ml.Pipes {
				pipe.Close()
				pipes = append(pipes, pipe)
			}
			for _, pipe := range pipes {
				delete(ml.Pipes, pipe)
			}
		}
		ml.RemoveExternalIP()
	}
}

// SetExternalIP for service spec
func (ml *ManagedListener) SetExternalIP() {
	if ml.Clientset == nil {
		panic("Clientset nil, can't UpdateExternalIP")
	}
	client := ml.Clientset.CoreV1().Services(ml.V1Service.ObjectMeta.Namespace)
	ml.refreshV1ServiceSpec(client)
	var ip = ipmgr.IP
	if ml.CIDR != nil && len(ml.CIDR.IP) > 0 {
		ip = ml.CIDR.IP
	}
	ml.V1Service.Spec.ExternalIPs = []string{ip}
	log.Println("Setting ExternalIP", ml.V1Service.Spec.ExternalIPs)
	client.Update(ml.V1Service)
	ml.refreshV1ServiceSpec(client)
}

func (ml *ManagedListener) refreshV1ServiceSpec(client v1core.ServiceInterface) {
	service, err := client.Get(ml.V1Service.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		ml.V1Service = service
	}
}

// RemoveExternalIP from service spec
func (ml *ManagedListener) RemoveExternalIP() {
	if ml.Clientset == nil {
		panic("Clientset nil, can't UpdateExternalIP")
	}
	client := ml.Clientset.CoreV1().Services(ml.V1Service.ObjectMeta.Namespace)
	log.Println("Removing ExternalIP", ml.V1Service.Spec.ExternalIPs)
	ml.refreshV1ServiceSpec(client)
	ml.V1Service.Spec.ExternalIPs = []string{}
	// ml.V1Service.ObjectMeta.Annotations = make(map[string]string)
	_, err := client.Update(ml.V1Service)
	if err != nil {
		log.Println("Error Removing ExternalIPs", err)
		if ml.Debug {
			dumpJSON(ml.V1Service)
		}
	}
	for i := 0; i < 3 && err != nil; i++ {
		log.Printf("Removing ExternalIPs Retry %d %v", i, err)
		time.Sleep(time.Second)
		ml.refreshV1ServiceSpec(client)
		ml.V1Service.Spec.ExternalIPs = []string{}
		_, err = client.Update(ml.V1Service)

	}
}

func dumpJSON(obj interface{}) {
	if jsonText, err := json.MarshalIndent(obj, "", "  "); err == nil {
		log.Println(string(jsonText))
	} else {
		log.Println(err)
	}
}
