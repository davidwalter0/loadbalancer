/*

Copyright 2017 The Kubernetes Authors.

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

/*

- based on k8s.io/client-go/examples/workqueue
- refactored to enable multiple watchers via new QueueMgr

*/

package watch

import (
	"fmt"
	"log"
	"os"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/watch"
)

// Interface abstraction for returned type
type Interface interface{}

// QueueItem publishable object from watch
type QueueItem struct {
	Key string
	K8sAPIName
	EventType
	Interface
}

// QueueItems channel for downstream consumption
type QueueItems chan QueueItem

// QueueController allows the creation of multiple independent queue
// controllers
type QueueController struct {
	K8sAPIName
	cache.Indexer
	workqueue.RateLimitingInterface
	cache.Controller
	QueueItems
}

// NewQueueController creates a watch callback interface
func NewQueueController(typeName K8sAPIName, RateLimitingInterface workqueue.RateLimitingInterface, Indexer cache.Indexer, Controller cache.Controller) *QueueController {
	return &QueueController{
		K8sAPIName:            typeName,
		Indexer:               Indexer,
		RateLimitingInterface: RateLimitingInterface,
		Controller:            Controller,
		QueueItems:            make(chan QueueItem, 100),
	}
}

// Next pushes the event item to it's queue
func (c *QueueController) Next() bool {
	// Wait until there is a new item in the working queue
	event, quit := c.RateLimitingInterface.Get()
	if quit {
		return false
	}

	// Tell the queue that we are done with processing this event. This
	// unblocks the event for other workers This allows safe parallel
	// processing because two services with the same event are never
	// processed in parallel.
	defer c.RateLimitingInterface.Done(event)

	// Invoke the method containing the business logic
	err := c.Publish(event.(QueueItem))
	// Handle the error if something went wrong during the execution of
	// the business logic
	c.handleErr(err, event.(QueueItem))
	return true
}

// Publish is the business logic of the controller publishing to the
// mgr's service channel.
func (c *QueueController) Publish(event QueueItem) error {
	key := event.Key
	// obj, exists, err := c.Indexer.GetByKey(key)
	obj, _, err := c.Indexer.GetByKey(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Fetching object with key %s from store failed with %v", log.Prefix(), key, err)
		return err
	}

	fmt.Printf("QueueItem %s for %s %s\n", event.EventType, event.K8sAPIName, key)
	event.Interface = obj
	c.QueueItems <- event

	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *QueueController) handleErr(err error, event QueueItem) { //interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the event on every
		// successful synchronization.  This ensures that future
		// processing of updates for this event is not delayed because of
		// an outdated error history.
		c.RateLimitingInterface.Forget(event)
		return
	}

	// This controller retries 5 times if something goes wrong. After
	// that, it stops trying.
	if c.RateLimitingInterface.NumRequeues(event) < 5 {
		log.Printf("Error syncing %s %v: %v", event.K8sAPIName, event, err)

		// Re-enqueue the event rate limited. Based on the rate limiter on
		// the queue and the re-enqueue history, the event will be
		// processed later again.
		c.RateLimitingInterface.AddRateLimited(event)
		return
	}

	c.RateLimitingInterface.Forget(event)
	// Report to an external entity that, even after several retries, we
	// could not successfully process this event
	runtime.HandleError(err)
	log.Printf("Dropping %s %q out of the queue: %v", event.K8sAPIName, event, err)
}

// Run a QueueController to manage a throttled queue of data
func (c *QueueController) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.RateLimitingInterface.ShutDown()
	log.Printf("Starting %s controller", c.K8sAPIName)

	go c.Controller.Run(stopCh)

	// Wait for all involved caches to be synced, before processing
	// items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.Controller.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Printf("Stopping %s controller", c.K8sAPIName)
}

func (c *QueueController) runWorker() {
	for c.Next() {
	}
}

// QueueMgr abstract the parts of a watcher
type QueueMgr struct {
	*kubernetes.Clientset
	*cache.ListWatch
	*QueueController
}

// NewQueueMgr takes a type string (services, pods, nodes, ...)  and a
// *kubernetes.Clientset, v1TypePointer is the type of the reference
// object &v1.Service{}, &v1.Node{}...
func NewQueueMgr(watchTypeName K8sAPIName, clientset *kubernetes.Clientset) *QueueMgr {
	var queueMgr = &QueueMgr{}
	queueMgr.ListWatch = cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), string(watchTypeName), v1.NamespaceAll, fields.Everything())
	// create the workqueue
	ratelimiter := workqueue.DefaultControllerRateLimiter()
	queue := workqueue.NewRateLimitingQueue(ratelimiter)
	var v1TypePointer pkgruntime.Object
	switch watchTypeName {
	case "services":
		v1TypePointer = &v1.Service{}
	case "nodes":
		v1TypePointer = &v1.Node{}
	case "pods":
		v1TypePointer = &v1.Pod{}
	case "endpoints":
		v1TypePointer = &v1.Endpoints{}
	}

	indexer, controller := cache.NewIndexerInformer(queueMgr.ListWatch, v1TypePointer, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(QueueItem{Key: key, K8sAPIName: watchTypeName, EventType: ADD})
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(QueueItem{Key: key, K8sAPIName: watchTypeName, EventType: UPDATE})
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// event function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(QueueItem{Key: key, K8sAPIName: watchTypeName, EventType: DELETE})
			}
		},
	}, cache.Indexers{})

	queueMgr.QueueController = NewQueueController(watchTypeName, queue, indexer, controller)

	return queueMgr
}

// V1TypePointer return a valid type pointer derived from the
// K8sAPIName
func V1TypePointer(watchTypeName K8sAPIName) (v1TypePointer pkgruntime.Object) {
	switch watchTypeName {
	case "services":
		v1TypePointer = &v1.Service{}
	case "nodes":
		v1TypePointer = &v1.Node{}
	case "pods":
		v1TypePointer = &v1.Pod{}
	case "endpoints":
		v1TypePointer = &v1.Endpoints{}
	default:
		panic("V1TypePointer")
	}
	return
}

// NewQueueMgrListOpt takes a type string (services, pods, nodes, ...)  and a
// *kubernetes.Clientset, v1TypePointer is the type of the reference
// object &v1.Service{}, &v1.Node{}...
func NewQueueMgrListOpt(watchTypeName K8sAPIName, clientset *kubernetes.Clientset, options *metav1.ListOptions) *QueueMgr {
	var queueMgr = &QueueMgr{}
	queueMgr.ListWatch = cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), string(watchTypeName), v1.NamespaceAll, fields.Everything())
	// create the workqueue
	ratelimiter := workqueue.DefaultControllerRateLimiter()
	queue := workqueue.NewRateLimitingQueue(ratelimiter)
	var v1TypePointer = V1TypePointer(watchTypeName)

	watcher := queueMgr.ListWatch.Watch
	lister := queueMgr.ListWatch.List
	newWatch := func(ignored metav1.ListOptions) (watch.Interface, error) {
		return watcher(*options)
	}

	newList := func(ignored metav1.ListOptions) (pkgruntime.Object, error) {
		return lister(*options)
	}

	queueMgr.ListWatch = &cache.ListWatch{ListFunc: newList, WatchFunc: newWatch}

	indexer, controller := cache.NewIndexerInformer(queueMgr.ListWatch, v1TypePointer, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(QueueItem{Key: key, K8sAPIName: watchTypeName, EventType: ADD})
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(QueueItem{Key: key, K8sAPIName: watchTypeName, EventType: UPDATE})
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// event function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(QueueItem{Key: key, K8sAPIName: watchTypeName, EventType: DELETE})
			}
		},
	}, cache.Indexers{})

	queueMgr.QueueController = NewQueueController(watchTypeName, queue, indexer, controller)

	return queueMgr
}

// Run creates a QueueController and executes watchers
func (queueMgr *QueueMgr) Run() {

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go queueMgr.QueueController.Run(1, stop)

	// Wait forever
	select {}
}
