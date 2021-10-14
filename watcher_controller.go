package main

import (
	"flag"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller demonstrates how to implement a controller with client-go.
type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

// NewController creates a new Controller
func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		informer: informer,
		queue:    queue,
		indexer:  indexer,
	}
}

func (c *Controller) ProcessNextItem() bool {
	// wait till there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	// tell queue that key has been processed. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// invoke method containing business logic
	err := c.syncToStdout(key.(string))
	c.handleError(err, key)
	return true
}

// syncToStdout is the business logic of the controller
func (c *Controller) syncToStdout(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Pod, so that we will see a delete for one pod
		fmt.Printf("Deployment %s does not exist anymore\n", key)
	} else {
		// Note that you also have to check the uid if you have a local controlled resource, which
		// is dependent on the actual instance, to detect that a Pod was recreated with the same name
		fmt.Printf("Sync/Add/Update for deployment %s\n", obj.(*appsv1.Deployment).GetName())
	}

	return nil
}

// handleError checks if an error happened and makes sure we retry later
func (c *Controller) handleError(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// the controller tries 5 times if something goes wrong
	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing deployment %v:%v", key, err)

		// re-enqueue the key rate limited
		// based on the rate limiter on the queue and re-enqueue history
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// report to an external entity that we couldn't process this key even after 5 retries
	runtime.HandleError(err)
	klog.Infof("Dropping deployment %q out of queue: %v", key, err)
}

// Run begins watching and syncing
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let workers stop when we are done
	defer c.queue.ShutDown()
	klog.Info("Starting deployment controller")

	go c.informer.Run(stopCh)

	// wait for all caches to sync before processing items from the cache
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping deployment controller")
}

func (c *Controller) runWorker() {
	for c.ProcessNextItem() {
	}
}

func main() {
	var kubeconfig string
	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.Parse()

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	// create pod watcher
	// podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything())
	deploymentListWatcher := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "deployments", v1.NamespaceDefault, fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// bind the workqueue to a cache with the help of an informer
	// thus whenever the cache is update the key is added to the workqueue
	indexer, informer := cache.NewIndexerInformer(deploymentListWatcher, &appsv1.Deployment{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue
			// thus for deletes we have to use this key function
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	controller := NewController(queue, indexer, informer)

	// we can now warm up the cache for initial sync
	// we will add a "dummy" deployment to the cache
	// the controller will be notified of its absence after the cache has synced
	indexer.Add(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: v1.NamespaceDefault,
		},
	})
	// indexer.Add(&v1.Pod{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      "dummy",
	// 		Namespace: v1.NamespaceDefault,
	// 	},
	// })

	// start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	// wait forever
	select {}
}
