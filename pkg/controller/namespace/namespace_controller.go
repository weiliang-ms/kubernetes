/*
Copyright 2015 The Kubernetes Authors.

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

package namespace

import (
	"fmt"
	"time"

	"golang.org/x/time/rate"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/component-base/metrics/prometheus/ratelimiter"
	"k8s.io/kubernetes/pkg/controller"
	"k8s.io/kubernetes/pkg/controller/namespace/deletion"

	"k8s.io/klog"
)

const (
	// namespaceDeletionGracePeriod is the time period to wait before processing a received namespace event.
	// This allows time for the following to occur:
	// * lifecycle admission plugins on HA apiservers to also observe a namespace
	//   deletion and prevent new objects from being created in the terminating namespace
	// * non-leader etcd servers to observe last-minute object creations in a namespace
	//   so this controller's cleanup can actually clean up all objects
	namespaceDeletionGracePeriod = 5 * time.Second
)

// NamespaceController is responsible for performing actions dependent upon a namespace phase
type NamespaceController struct {
	// lister that can list namespaces from a shared cache
	lister corelisters.NamespaceLister
	// returns true when the namespace cache is ready
	listerSynced cache.InformerSynced
	// namespaces that have been queued up for processing by workers
	queue workqueue.RateLimitingInterface
	// helper to delete all resources in the namespace when the namespace is deleted.
	namespacedResourcesDeleter deletion.NamespacedResourcesDeleterInterface
}

// NewNamespaceController creates a new NamespaceController
func NewNamespaceController(
	kubeClient clientset.Interface,
	metadataClient metadata.Interface,
	discoverResourcesFn func() ([]*metav1.APIResourceList, error),
	namespaceInformer coreinformers.NamespaceInformer,
	resyncPeriod time.Duration,
	finalizerToken v1.FinalizerName) *NamespaceController {

	// create the controller so we can inject the enqueue function
	namespaceController := &NamespaceController{
		queue:                      workqueue.NewNamedRateLimitingQueue(nsControllerRateLimiter(), "namespace"),
		namespacedResourcesDeleter: deletion.NewNamespacedResourcesDeleter(kubeClient.CoreV1().Namespaces(), metadataClient, kubeClient.CoreV1(), discoverResourcesFn, finalizerToken),
	}

	if kubeClient != nil && kubeClient.CoreV1().RESTClient().GetRateLimiter() != nil {
		ratelimiter.RegisterMetricAndTrackRateLimiterUsage("namespace_controller", kubeClient.CoreV1().RESTClient().GetRateLimiter())
	}

	// configure the namespace informer event handlers
	// 添加event回调函数
	namespaceInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				namespace := obj.(*v1.Namespace)
				namespaceController.enqueueNamespace(namespace)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				namespace := newObj.(*v1.Namespace)
				namespaceController.enqueueNamespace(namespace)
			},
		},
		resyncPeriod,
	)
	namespaceController.lister = namespaceInformer.Lister()
	namespaceController.listerSynced = namespaceInformer.Informer().HasSynced

	return namespaceController
}

// nsControllerRateLimiter is tuned for a faster than normal recycle time with default backoff speed and default overall
// requeing speed.  We do this so that namespace cleanup is reliably faster and we know that the number of namespaces being
// deleted is smaller than total number of other namespace scoped resources in a cluster.
func nsControllerRateLimiter() workqueue.RateLimiter {
	return workqueue.NewMaxOfRateLimiter(
		// this ensures that we retry namespace deletion at least every minute, never longer.
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 60*time.Second),
		// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)
}

// enqueueNamespace adds an object to the controller work queue
// obj could be an *v1.Namespace, or a DeletionFinalStateUnknown item.
func (nm *NamespaceController) enqueueNamespace(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %+v: %v", obj, err))
		return
	}

	namespace := obj.(*v1.Namespace)
	// don't queue if we aren't deleted
	// 如果ns没有被删除，直接返回，不添加至队列里待处理
	// 实际监听并处理的仅是删除事件，其他事件监听后续添加？
	if namespace.DeletionTimestamp == nil || namespace.DeletionTimestamp.IsZero() {
		return
	}

	// delay processing namespace events to allow HA api servers to observe namespace deletion,
	// and HA etcd servers to observe last minute object creations inside the namespace
	// 延时5s添加至队列
	/*
		namespaceDeletionGracePeriod是处理收到的命名空间事件之前要等待的时间段
		这样就有时间进行以下操作：
		1.高可用API-Server上的生命周期许可插件也可以观察名称空间
		2.删除并防止在终止命名空间中创建新对象
		3.用于观察命名空间中最后一分钟对象创建的非先导etcd服务器

		因此，该控制器的清理实际上可以清理所有对象

	*/
	nm.queue.AddAfter(key, namespaceDeletionGracePeriod)
}

// worker processes the queue of namespace objects.
// Each namespace can be in the queue at most once.
// The system ensures that no two workers can process
// the same namespace at the same time.
func (nm *NamespaceController) worker() {
	workFunc := func() bool {
		key, quit := nm.queue.Get()
		if quit {
			return true
		}
		defer nm.queue.Done(key)

		err := nm.syncNamespaceFromKey(key.(string))
		if err == nil {
			// no error, forget this entry and return
			nm.queue.Forget(key)
			return false
		}

		// 判断命名空间下是否仍有资源未被删除，如果还有未被删除资源，获取资源个数。随后将该命名空间延时加入队列
		// 如果没有未被删除资源，将该命名空间加入限速队列
		if estimate, ok := err.(*deletion.ResourcesRemainingError); ok {
			t := estimate.Estimate/2 + 1
			klog.V(4).Infof("Content remaining in namespace %s, waiting %d seconds", key, t)
			nm.queue.AddAfter(key, time.Duration(t)*time.Second)
		} else {
			// rather than wait for a full resync, re-add the namespace to the queue to be processed
			nm.queue.AddRateLimited(key)
			utilruntime.HandleError(fmt.Errorf("deletion of namespace %v failed: %v", key, err))
		}
		return false
	}

	for {
		quit := workFunc()

		if quit {
			return
		}
	}
}

// syncNamespaceFromKey looks for a namespace with the specified key in its store and synchronizes it
func (nm *NamespaceController) syncNamespaceFromKey(key string) (err error) {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing namespace %q (%v)", key, time.Since(startTime))
	}()

	// lister -> 带有List与Get方法的indexer对象
	namespace, err := nm.lister.Get(key)
	if errors.IsNotFound(err) {
		klog.Infof("Namespace has been deleted %v", key)
		return nil
	}
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Unable to retrieve namespace %v from store: %v", key, err))
		return err
	}
	// 删除ns
	return nm.namespacedResourcesDeleter.Delete(namespace.Name)
}

// Run starts observing the system with the specified number of workers.
func (nm *NamespaceController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer nm.queue.ShutDown()

	klog.Infof("Starting namespace controller")
	defer klog.Infof("Shutting down namespace controller")

	if !cache.WaitForNamedCacheSync("namespace", stopCh, nm.listerSynced) {
		return
	}

	klog.V(5).Info("Starting workers of namespace controller")
	for i := 0; i < workers; i++ {
		go wait.Until(nm.worker, time.Second, stopCh)
	}
	<-stopCh
}
