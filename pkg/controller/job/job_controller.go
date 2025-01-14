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

package job

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sort"
	"sync"
	"time"

	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/component-base/metrics/prometheus/ratelimiter"
	"k8s.io/kubernetes/pkg/controller"
	"k8s.io/utils/integer"

	"k8s.io/klog"
)

const statusUpdateRetries = 3

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = batch.SchemeGroupVersion.WithKind("Job")

var (
	// DefaultJobBackOff is the max backoff period, exported for the e2e test
	DefaultJobBackOff = 10 * time.Second
	// MaxJobBackOff is the max backoff period, exported for the e2e test
	MaxJobBackOff = 360 * time.Second
)

type JobController struct {
	kubeClient clientset.Interface
	podControl controller.PodControlInterface

	// To allow injection of updateJobStatus for testing.
	updateHandler func(job *batch.Job) error
	syncHandler   func(jobKey string) (bool, error)
	// podStoreSynced returns true if the pod store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	podStoreSynced cache.InformerSynced
	// jobStoreSynced returns true if the job store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	jobStoreSynced cache.InformerSynced

	// A TTLCache of pod creates/deletes each rc expects to see
	expectations controller.ControllerExpectationsInterface

	// A store of jobs
	jobLister batchv1listers.JobLister

	// A store of pods, populated by the podController
	podStore corelisters.PodLister

	// Jobs that need to be updated
	queue workqueue.RateLimitingInterface

	recorder record.EventRecorder
}

// 初始化job控制器函数
func NewJobController(podInformer coreinformers.PodInformer, jobInformer batchinformers.JobInformer, kubeClient clientset.Interface) *JobController {
	// 初始化广播对象
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})

	if kubeClient != nil && kubeClient.CoreV1().RESTClient().GetRateLimiter() != nil {
		ratelimiter.RegisterMetricAndTrackRateLimiterUsage("job_controller", kubeClient.CoreV1().RESTClient().GetRateLimiter())
	}

	// 初始化job控制器，包含：
	// - kube连接客户端
	// - 管理pod的RealPodControl
	// - 存储job控制器异常信息的*ControllerExpectations类型对象
	// - 队列：存放需要更新的job
	// - job-controller事件广播对象
	jm := &JobController{
		kubeClient: kubeClient,
		// pod控制器
		/*
			实现了以下接口
			type PodControlInterface interface {
				// 创建pod
				CreatePods(namespace string, template *v1.PodTemplateSpec, object runtime.Object) error
				// 指定Node节点创建pod
				CreatePodsOnNode(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error
				// 创建控制器托管类型pod（如pod -> Controlled By: Job/pi-job）
				CreatePodsWithControllerRef(namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error
				// 删除pod
				DeletePod(namespace string, podID string, object runtime.Object) error
				// 更新pod
				PatchPod(namespace, name string, data []byte) error
			}
		*/
		podControl: controller.RealPodControl{
			KubeClient: kubeClient,
			Recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "job-controller"}),
		},
		// 初始化存储job控制器异常信息的对象
		expectations: controller.NewControllerExpectations(),
		// 初始化队列
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(DefaultJobBackOff, MaxJobBackOff), "job"),
		// 初始化job-controller事件广播对象
		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "job-controller"}),
	}

	// 创建job资源Informer事件监控回调方法
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			jm.enqueueController(obj, true)
		},
		UpdateFunc: jm.updateJob,
		DeleteFunc: func(obj interface{}) {
			jm.enqueueController(obj, true)
		},
	})

	// 创建job controller监听对象
	jm.jobLister = jobInformer.Lister()
	jm.jobStoreSynced = jobInformer.Informer().HasSynced

	// 创建pod资源Informer事件监控回调方法
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    jm.addPod,
		UpdateFunc: jm.updatePod,
		DeleteFunc: jm.deletePod,
	})

	jm.podStore = podInformer.Lister()
	jm.podStoreSynced = podInformer.Informer().HasSynced

	/*
		更新job状态方法，状态包含:
		Conditions []JobCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
		StartTime *metav1.Time `json:"startTime,omitempty" protobuf:"bytes,2,opt,name=startTime"`
		CompletionTime *metav1.Time `json:"completionTime,omitempty" protobuf:"bytes,3,opt,name=completionTime"`
		Active int32 `json:"active,omitempty" protobuf:"varint,4,opt,name=active"`
		Succeeded int32 `json:"succeeded,omitempty" protobuf:"varint,5,opt,name=succeeded"`
		Failed int32 `json:"failed,omitempty" protobuf:"varint,6,opt,name=failed"`
	*/
	jm.updateHandler = jm.updateJobStatus
	// 同步job状态方法
	jm.syncHandler = jm.syncJob

	return jm
}

// Run the main goroutine responsible for watching and syncing jobs.
func (jm *JobController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer jm.queue.ShutDown()

	klog.Infof("Starting job controller")
	defer klog.Infof("Shutting down job controller")

	if !cache.WaitForNamedCacheSync("job", stopCh, jm.podStoreSynced, jm.jobStoreSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(jm.worker, time.Second, stopCh)
	}

	<-stopCh
}

// getPodJobs returns a list of Jobs that potentially match a Pod.
func (jm *JobController) getPodJobs(pod *v1.Pod) []*batch.Job {
	jobs, err := jm.jobLister.GetPodJobs(pod)
	if err != nil {
		return nil
	}
	if len(jobs) > 1 {
		// ControllerRef will ensure we don't do anything crazy, but more than one
		// item in this list nevertheless constitutes user error.
		utilruntime.HandleError(fmt.Errorf("user error! more than one job is selecting pods with labels: %+v", pod.Labels))
	}
	ret := make([]*batch.Job, 0, len(jobs))
	for i := range jobs {
		ret = append(ret, &jobs[i])
	}
	return ret
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (jm *JobController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) *batch.Job {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != controllerKind.Kind {
		return nil
	}
	job, err := jm.jobLister.Jobs(namespace).Get(controllerRef.Name)
	if err != nil {
		return nil
	}
	if job.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return job
}

// When a pod is created, enqueue the controller that manages it and update it's expectations.
func (jm *JobController) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	if pod.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		jm.deletePod(pod)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		job := jm.resolveControllerRef(pod.Namespace, controllerRef)
		if job == nil {
			return
		}
		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			return
		}
		jm.expectations.CreationObserved(jobKey)
		jm.enqueueController(job, true)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching controllers and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	for _, job := range jm.getPodJobs(pod) {
		jm.enqueueController(job, true)
	}
}

// When a pod is updated, figure out what job/s manage it and wake them up.
// If the labels of the pod have changed we need to awaken both the old
// and new job. old and cur must be *v1.Pod types.
func (jm *JobController) updatePod(old, cur interface{}) {
	curPod := cur.(*v1.Pod)
	oldPod := old.(*v1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}
	if curPod.DeletionTimestamp != nil {
		// when a pod is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
		// and after such time has passed, the kubelet actually deletes it from the store. We receive an update
		// for modification of the deletion timestamp and expect an job to create more pods asap, not wait
		// until the kubelet actually deletes the pod.
		jm.deletePod(curPod)
		return
	}

	// the only time we want the backoff to kick-in, is when the pod failed
	immediate := curPod.Status.Phase != v1.PodFailed

	curControllerRef := metav1.GetControllerOf(curPod)
	oldControllerRef := metav1.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if job := jm.resolveControllerRef(oldPod.Namespace, oldControllerRef); job != nil {
			jm.enqueueController(job, immediate)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		job := jm.resolveControllerRef(curPod.Namespace, curControllerRef)
		if job == nil {
			return
		}
		jm.enqueueController(job, immediate)
		return
	}

	// Otherwise, it's an orphan. If anything changed, sync matching controllers
	// to see if anyone wants to adopt it now.
	labelChanged := !reflect.DeepEqual(curPod.Labels, oldPod.Labels)
	if labelChanged || controllerRefChanged {
		for _, job := range jm.getPodJobs(curPod) {
			jm.enqueueController(job, immediate)
		}
	}
}

// When a pod is deleted, enqueue the job that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (jm *JobController) deletePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new job will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		pod, ok = tombstone.Obj.(*v1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a pod %+v", obj))
			return
		}
	}

	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	job := jm.resolveControllerRef(pod.Namespace, controllerRef)
	if job == nil {
		return
	}
	jobKey, err := controller.KeyFunc(job)
	if err != nil {
		return
	}
	jm.expectations.DeletionObserved(jobKey)
	jm.enqueueController(job, true)
}

func (jm *JobController) updateJob(old, cur interface{}) {
	oldJob := old.(*batch.Job)
	curJob := cur.(*batch.Job)

	// never return error
	key, err := controller.KeyFunc(curJob)
	if err != nil {
		return
	}
	jm.enqueueController(curJob, true)
	// check if need to add a new rsync for ActiveDeadlineSeconds
	if curJob.Status.StartTime != nil {
		curADS := curJob.Spec.ActiveDeadlineSeconds
		if curADS == nil {
			return
		}
		oldADS := oldJob.Spec.ActiveDeadlineSeconds
		if oldADS == nil || *oldADS != *curADS {
			now := metav1.Now()
			start := curJob.Status.StartTime.Time
			passed := now.Time.Sub(start)
			total := time.Duration(*curADS) * time.Second
			// AddAfter will handle total < passed
			jm.queue.AddAfter(key, total-passed)
			klog.V(4).Infof("job ActiveDeadlineSeconds updated, will rsync after %d seconds", total-passed)
		}
	}
}

// obj could be an *batch.Job, or a DeletionFinalStateUnknown marker item,
// immediate tells the controller to update the status right away, and should
// happen ONLY when there was a successful pod run.
func (jm *JobController) enqueueController(obj interface{}, immediate bool) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %+v: %v", obj, err))
		return
	}

	backoff := time.Duration(0)
	if !immediate {
		backoff = getBackoff(jm.queue, key)
	}

	// TODO: Handle overlapping controllers better. Either disallow them at admission time or
	// deterministically avoid syncing controllers that fight over pods. Currently, we only
	// ensure that the same controller is synced for a given pod. When we periodically relist
	// all controllers there will still be some replica instability. One way to handle this is
	// by querying the store for all controllers that this rc overlaps, as well as all
	// controllers that overlap this rc, and sorting them.
	jm.queue.AddAfter(key, backoff)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (jm *JobController) worker() {
	for jm.processNextWorkItem() {
	}
}

func (jm *JobController) processNextWorkItem() bool {
	key, quit := jm.queue.Get()
	if quit {
		return false
	}
	defer jm.queue.Done(key)

	// jm.syncHandler调用的实际为func (jm *JobController) syncJob(key string) (bool, error)
	// 该赋值在NewJobController()阶段中的：jm.syncHandler = jm.syncJob 完成
	forget, err := jm.syncHandler(key.(string))
	if err == nil {
		if forget {
			jm.queue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing job: %v", err))
	jm.queue.AddRateLimited(key)

	return true
}

// getPodsForJob returns the set of pods that this Job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned Pods are pointers into the cache.
func (jm *JobController) getPodsForJob(j *batch.Job) ([]*v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(j.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	pods, err := jm.podStore.Pods(j.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).

	// 检查job是否被删除（通过检测 meta.DeletionTimestamp属性）
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		// 通过job从apiserver查询job最新信息
		fresh, err := jm.kubeClient.BatchV1().Jobs(j.Namespace).Get(context.TODO(), j.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		// 判断本地缓存数据与apiserver数据是否一致
		if fresh.UID != j.UID {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", j.Namespace, j.Name, fresh.UID, j.UID)
		}
		return fresh, nil
	})

	// 返回job关联pod列表
	cm := controller.NewPodControllerRefManager(jm.podControl, j, selector, controllerKind, canAdoptFunc)
	return cm.ClaimPods(pods)
}

// syncJob will sync the job with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods created or deleted. This function is not meant to be invoked
// concurrently with the same key.
func (jm *JobController) syncJob(key string) (bool, error) {
	// 同步开始时间
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing job %q (%v)", key, time.Since(startTime))
	}()

	// key格式可能为：namespace/job-name或job-name
	// 通过key获取Job所在命名空间及job名称
	// 获取方式，通过截取key中的'/'
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	if len(ns) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid job key %q: either namespace or name is missing", key)
	}
	// 获取job实体，sharedJob类型为*v1.Job,
	sharedJob, err := jm.jobLister.Jobs(ns).Get(name)
	if err != nil {
		// 判断Job是否被删除
		if errors.IsNotFound(err) {
			klog.V(4).Infof("Job has been deleted: %v", key)
			jm.expectations.DeleteExpectations(key)
			return true, nil
		}
		return false, err
	}
	// 获取sharedJob指针的值
	job := *sharedJob

	// if job was finished previously, we don't want to redo the termination
	// 判断Job是否已经完成
	// 当 job 的 .status.conditions中有Complete或Failed的type且对应的status为true时表示该 job 已经执行完成
	/*
		status:
		  completionTime: "2019-12-18T14:16:47Z"
		  conditions:
		  - lastProbeTime: "2019-12-18T14:16:47Z"
		    lastTransitionTime: "2019-12-18T14:16:47Z"
		    status: "True"                // status 为 true
		    type: Complete                      // Complete
		  startTime: "2019-12-18T14:15:35Z"
		  succeeded: 2
	*/
	if IsJobFinished(&job) {
		return true, nil
	}

	// retrieve the previous number of retry
	// 获取job重试次数
	previousRetry := jm.queue.NumRequeues(key)

	// Check the expectations of the job before counting active pods, otherwise a new pod can sneak in
	// and update the expectations after we've retrieved active pods from the store. If a new pod enters
	// the store after we've checked the expectation, the job sync is just deferred till the next relist.
	// 检查job当前状态是否满足预期(是否需要同步)，判断条件如下：
	/*
		1、该 key job在 ControllerExpectations 中的 adds 和 dels 都 <= 0，即调用 apiserver 的创建和删除接口没有失败过；
		2、该 key job在 ControllerExpectations 中已经超过 5min 没有更新了，执行同步
		3、该 key job在 ControllerExpectations 中不存在，即该对象是新创建的，执行同步
		4、调用 GetExpectations 方法失败，内部错误，强制进行同步；
	*/
	jobNeedsSync := jm.expectations.SatisfiedExpectations(key)

	// 获取job关联的pod，若有孤儿pod的label与job的能匹配则进行关联；若已关联的pod label有变化则解除与job的关联关系；
	pods, err := jm.getPodsForJob(&job)
	if err != nil {
		return false, err
	}

	// 过滤active状态的pod列表,关于active状态pod的定义：
	// pod的状态不是`Failed`、`Succeeded`并且不含有DeletionTimestamp属性
	// 关于Pod状态解析：
	// Pending: pod已经被Kubernetes系统接受，但是还没有创建一个或多个容器映像。这包括通过网络下载镜像所花费的时间，这可能需要一段时间。
	// Running: pod已经调度到某一个节点，并且已经创建了所有的容器。至少有一个容器仍在运行，或正在启动或重新启动。
	// Succeeded: pod中所有容器成功终止，不会重新启动。
	// Failed: pod内的所有容器都已终止，且至少有一个容器因故障终止。（容器要么以非零状态退出，要么被系统终止。）
	// Unknown: 由于某些原因无法获得`pod`的状态，通常是由于与`pod`的主机通信时出现错误。
	activePods := controller.FilterActivePods(pods)
	// 获取活跃pod数量
	active := int32(len(activePods))
	// 获取Succeeded状态pod与Failed状态pod数量
	succeeded, failed := getStatus(pods)
	// 获取conditions信息（对应status.conditions）即pod当前状态的最新观测，样例如下：
	/*
		conditions:
		  - lastProbeTime: null
		    lastTransitionTime: "2021-07-12T04:28:35Z"
		    reason: PodCompleted
		    status: "True"
		    type: Initialized
		  - lastProbeTime: null
		    lastTransitionTime: "2021-07-12T04:28:43Z"
		    reason: PodCompleted
		    status: "False"
		    type: Ready
		  - lastProbeTime: null
		    lastTransitionTime: "2021-07-12T04:28:43Z"
		    reason: PodCompleted
		    status: "False"
		    type: ContainersReady
		  - lastProbeTime: null
		    lastTransitionTime: "2021-07-12T04:28:35Z"
		    status: "True"
		    type: PodScheduled
	*/
	conditions := len(job.Status.Conditions)
	// job first start
	// 获取job启动时间，如果为空，说明第一次启动
	// job.Status.StartTime设置为当前时间
	if job.Status.StartTime == nil {
		now := metav1.Now()
		job.Status.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		// 对应spec.activeDeadlineSeconds属性，默认一个Job将会一直运行除非Pod失败且遵从.spec.backoffLimit的限制，可以通过设置deadline来终止Job对象：
		// .spec.activeDeadlineSeconds设置为多少秒，一旦设置了这个参数，所有的Pod将会终止且Job的状态将会就变成 -> type: Failed reason: DeadlineExceeded，
		// .spec.activeDeadlineSeconds的优先级高于.spec.backoffLimit，
		// 因此Job重试过一个或多个失败的Pod达到限制的时间（activeDeadlineSeconds）后将不会部署额外的Pod， 即使此时backoffLimit还没达到。
		if job.Spec.ActiveDeadlineSeconds != nil {
			klog.V(4).Infof("Job %s has ActiveDeadlineSeconds will sync after %d seconds",
				key, *job.Spec.ActiveDeadlineSeconds)
			// .spec.activeDeadlineSeconds后将Job加入延迟同步队列里面
			jm.queue.AddAfter(key, time.Duration(*job.Spec.ActiveDeadlineSeconds)*time.Second)
		}
	}

	var manageJobErr error
	jobFailed := false
	var failureReason string
	var failureMessage string

	// 判断job是否有新增的Failed状态的pod
	jobHaveNewFailure := failed > job.Status.Failed
	// new failures happen when status does not reflect the failures and active
	// is different than parallelism, otherwise the previous controller loop
	// failed updating status so even if we pick up failure it is not a new one
	// 1.jobHaveNewFailure -> 判断job是否有新增的Failed状态的pod
	// 2.(active != *job.Spec.Parallelism) -> 判断pod存活数是否与job并发数一致
	// 3.(int32(previousRetry)+1 > *job.Spec.BackoffLimit) -> 判断 job 的重试次数是否超过了 job.Spec.BackoffLimit(默认是6次)
	exceedsBackoffLimit := jobHaveNewFailure && (active != *job.Spec.Parallelism) &&
		(int32(previousRetry)+1 > *job.Spec.BackoffLimit)

	// 1.job 处于 Failed状态
	// 2.pastBackoffLimitOnFailure检查容器restartCounts的总和是否超过BackoffLimit，此方法仅适用于restartPolicy == OnFailure的pod
	// 满足1、2任一条件，job被标记为：BackoffLimitExceeded状态（达到重试次数限制）
	if exceedsBackoffLimit || pastBackoffLimitOnFailure(&job, pods) {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncJob
		jobFailed = true
		failureReason = "BackoffLimitExceeded"
		failureMessage = "Job has reached the specified backoff limit"
		// 如果job设置了activedeadlinesseconds字段,
		// pastActiveDeadline检查作业是否设置ActiveDeadlineSeconds字段，
		// 若设置ActiveDeadlineSeconds字段且超过该字段时间，job被标记为Failed状态且原因为 DeadlineExceeded
	} else if pastActiveDeadline(&job) {
		jobFailed = true
		failureReason = "DeadlineExceeded"
		failureMessage = "Job was active longer than specified deadline"
	}

	// 如果Job状态为Failed
	if jobFailed {
		errCh := make(chan error, active)
		// 删除Job中的active状态Pod（pod的状态不是`Failed`、`Succeeded`并且不含有DeletionTimestamp属性）
		jm.deleteJobPods(&job, activePods, errCh)
		select {
		case manageJobErr = <-errCh:
			if manageJobErr != nil {
				break
			}
		default:
		}

		// update status values accordingly
		// 失败Pod总数追加之前活跃状态的Pod数量（活跃状态Pod上一步中中已被删除）
		failed += active
		// 活跃状态pod数量置零
		active = 0
		// 更新Job的status.Conditions的属性
		job.Status.Conditions = append(job.Status.Conditions, newCondition(batch.JobFailed, failureReason, failureMessage))
		// 更新Job事件(Events)
		jm.recorder.Event(&job, v1.EventTypeWarning, failureReason, failureMessage)
	} else {
		// 如果Job需要进行同步，且job的DeletionTimestamp属性为空
		if jobNeedsSync && job.DeletionTimestamp == nil {
			// 同步逻辑!!!
			active, manageJobErr = jm.manageJob(activePods, succeeded, &job)
		}
		completions := succeeded
		complete := false
		// 若job.Spec.Completions字段没有设置值则只要有一pod运行完成该job就为Completed状态
		if job.Spec.Completions == nil {
			// This type of job is complete when any pod exits with success.
			// Each pod is capable of
			// determining whether or not the entire Job is done.  Subsequent pods are
			// not expected to fail, but if they do, the failure is ignored.  Once any
			// pod succeeds, the controller waits for remaining pods to finish, and
			// then the job is complete.

			// 判断job是否完成
			if succeeded > 0 && active == 0 {
				complete = true
			}
		} else {
			// Job specifies a number of completions.  This type of job signals
			// success by having that number of successes.  Since we do not
			// start more pods than there are remaining completions, there should
			// not be any remaining active pods once this count is reached.
			// 若设置了 job.Spec.Completions 会通过判断已经运行完成状态的 pod 即 succeeded pod 数是否大于等于该值；
			if completions >= *job.Spec.Completions {
				complete = true
				if active > 0 {
					jm.recorder.Event(&job, v1.EventTypeWarning, "TooManyActivePods", "Too many active pods running after completion count reached")
				}
				if completions > *job.Spec.Completions {
					jm.recorder.Event(&job, v1.EventTypeWarning, "TooManySucceededPods", "Too many succeeded pods running after completion count reached")
				}
			}
		}
		// 通过以上判断若 job 运行完成了，则更新 job.Status.Conditions 和 job.Status.CompletionTime 字段
		if complete {
			job.Status.Conditions = append(job.Status.Conditions, newCondition(batch.JobComplete, "", ""))
			now := metav1.Now()
			job.Status.CompletionTime = &now
			jm.recorder.Event(&job, v1.EventTypeNormal, "Completed", "Job completed")
		}
	}

	forget := false
	// Check if the number of jobs succeeded increased since the last check. If yes "forget" should be true
	// This logic is linked to the issue: https://github.com/kubernetes/kubernetes/issues/56853 that aims to
	// improve the Job backoff policy when parallelism > 1 and few Jobs failed but others succeed.
	// In this case, we should clear the backoff delay.

	if job.Status.Succeeded < succeeded {
		forget = true
	}

	// no need to update the job if the status hasn't changed since last time
	// 如果 job 的 status 有变化，将 job 的 status 更新到 apiserver
	if job.Status.Active != active || job.Status.Succeeded != succeeded || job.Status.Failed != failed || len(job.Status.Conditions) != conditions {
		job.Status.Active = active
		job.Status.Succeeded = succeeded
		job.Status.Failed = failed

		if err := jm.updateHandler(&job); err != nil {
			return forget, err
		}

		if jobHaveNewFailure && !IsJobFinished(&job) {
			// returning an error will re-enqueue Job after the backoff period
			return forget, fmt.Errorf("failed pod(s) detected for job key %q", key)
		}

		forget = true
	}

	return forget, manageJobErr
}

func (jm *JobController) deleteJobPods(job *batch.Job, pods []*v1.Pod, errCh chan<- error) {
	// TODO: below code should be replaced with pod termination resulting in
	// pod failures, rather than killing pods. Unfortunately none such solution
	// exists ATM. There's an open discussion in the topic in
	// https://github.com/kubernetes/kubernetes/issues/14602 which might give
	// some sort of solution to above problem.
	// kill remaining active pods
	wait := sync.WaitGroup{}
	nbPods := len(pods)
	wait.Add(nbPods)
	for i := int32(0); i < int32(nbPods); i++ {
		go func(ix int32) {
			defer wait.Done()
			if err := jm.podControl.DeletePod(job.Namespace, pods[ix].Name, job); err != nil && !apierrors.IsNotFound(err) {
				defer utilruntime.HandleError(err)
				klog.V(2).Infof("Failed to delete %v, job %q/%q deadline exceeded", pods[ix].Name, job.Namespace, job.Name)
				errCh <- err
			}
		}(i)
	}
	wait.Wait()
}

// pastBackoffLimitOnFailure checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods with restartPolicy == OnFailure
func pastBackoffLimitOnFailure(job *batch.Job, pods []*v1.Pod) bool {
	if job.Spec.Template.Spec.RestartPolicy != v1.RestartPolicyOnFailure {
		return false
	}
	result := int32(0)
	for i := range pods {
		po := pods[i]
		if po.Status.Phase == v1.PodRunning || po.Status.Phase == v1.PodPending {
			for j := range po.Status.InitContainerStatuses {
				stat := po.Status.InitContainerStatuses[j]
				result += stat.RestartCount
			}
			for j := range po.Status.ContainerStatuses {
				stat := po.Status.ContainerStatuses[j]
				result += stat.RestartCount
			}
		}
	}
	if *job.Spec.BackoffLimit == 0 {
		return result > 0
	}
	return result >= *job.Spec.BackoffLimit
}

// pastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func pastActiveDeadline(job *batch.Job) bool {
	if job.Spec.ActiveDeadlineSeconds == nil || job.Status.StartTime == nil {
		return false
	}
	now := metav1.Now()
	start := job.Status.StartTime.Time
	duration := now.Time.Sub(start)
	allowedDuration := time.Duration(*job.Spec.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

func newCondition(conditionType batch.JobConditionType, reason, message string) batch.JobCondition {
	return batch.JobCondition{
		Type:               conditionType,
		Status:             v1.ConditionTrue,
		LastProbeTime:      metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getStatus returns no of succeeded and failed pods running a job
func getStatus(pods []*v1.Pod) (succeeded, failed int32) {
	succeeded = int32(filterPods(pods, v1.PodSucceeded))
	failed = int32(filterPods(pods, v1.PodFailed))
	return
}

// manageJob is the core method responsible for managing the number of running
// pods according to what is specified in the job.Spec.
// Does NOT modify <activePods>.
// jm.manageJob它主要做的事情就是根据 job 配置的并发数来确认当前处于 active 的 pods 数量是否合理，如果不合理的话则进行调整
func (jm *JobController) manageJob(activePods []*v1.Pod, succeeded int32, job *batch.Job) (int32, error) {
	// 初始化同步锁，
	var activeLock sync.Mutex
	// 获取job活跃的pod数量
	active := int32(len(activePods))
	// 获取job可运行的pod数
	parallelism := *job.Spec.Parallelism
	jobKey, err := controller.KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for job %#v: %v", job, err))
		return 0, nil
	}

	var errCh chan error
	// 判断如果处于 active 状态的 pods 数大于 job 设置的并发数 job.Spec.Parallelism，
	// 则删除多余的 active pods
	if active > parallelism {
		diff := active - parallelism
		errCh = make(chan error, diff)
		jm.expectations.ExpectDeletions(jobKey, int(diff))
		klog.V(4).Infof("Too many pods running job %q, need %d, deleting %d", jobKey, parallelism, diff)
		// Sort the pods in the order such that not-ready < ready, unscheduled
		// < scheduled, and pending < running. This ensures that we delete pods
		// in the earlier stages whenever possible.
		// 对现有pod做排序，优先删除优先级低的
		//将activePods转换为controller.ActivePods类型，使其具备排序能力
		// 排序规则：
		// 1、判断是否绑定了 node：Unassigned < assigned；
		// 2、判断 pod phase：PodPending < PodUnknown < PodRunning；
		// 3、判断 pod 状态：Not ready < ready；
		// 4、若 pod 都为 ready，则按运行时间排序，运行时间最短会被删除：empty time < less time < more time；
		// 5、根据 pod 重启次数排序：higher restart counts < lower restart counts；
		// 6、按 pod 创建时间进行排序：Empty creation time pods < newer pods < older pods；
		sort.Sort(controller.ActivePods(activePods))

		active -= diff
		wait := sync.WaitGroup{}
		wait.Add(int(diff))
		// 并发删除冗余的active Pod
		for i := int32(0); i < diff; i++ {
			go func(ix int32) {
				defer wait.Done()
				if err := jm.podControl.DeletePod(job.Namespace, activePods[ix].Name, job); err != nil {
					// Decrement the expected number of deletes because the informer won't observe this deletion
					jm.expectations.DeletionObserved(jobKey)
					if !apierrors.IsNotFound(err) {
						klog.V(2).Infof("Failed to delete %v, decremented expectations for job %q/%q", activePods[ix].Name, job.Namespace, job.Name)
						activeLock.Lock()
						active++
						activeLock.Unlock()
						errCh <- err
						utilruntime.HandleError(err)
					}

				}
			}(i)
		}
		wait.Wait()
		// 5、若处于 active 状态的 pods 数小于 job 设置的并发数，则需要创建出新的 pod
	} else if active < parallelism {
		// 6、首先计算出 diff 数
		wantActive := int32(0)
		if job.Spec.Completions == nil {
			// Job does not specify a number of completions.  Therefore, number active
			// should be equal to parallelism, unless the job has seen at least
			// once success, in which leave whatever is running, running.
			if succeeded > 0 {
				wantActive = active
			} else {
				wantActive = parallelism
			}
		} else {
			// Job specifies a specific number of completions.  Therefore, number
			// active should not ever exceed number of remaining completions.
			wantActive = *job.Spec.Completions - succeeded
			if wantActive > parallelism {
				wantActive = parallelism
			}
		}
		diff := wantActive - active
		if diff < 0 {
			utilruntime.HandleError(fmt.Errorf("More active than wanted: job %q, want %d, have %d", jobKey, wantActive, active))
			diff = 0
		}
		if diff == 0 {
			return active, nil
		}

		jm.expectations.ExpectCreations(jobKey, int(diff))
		errCh = make(chan error, diff)
		klog.V(4).Infof("Too few pods running job %q, need %d, creating %d", jobKey, wantActive, diff)

		active += diff
		wait := sync.WaitGroup{}

		/*
		  当前尝试启动大量的pod，出现失败与错误时，有可能原因一致。
		  例如，一个配额很低的项目，如果试图创建大量的pod，那么在其中一个pod失败后，就不会向API服务发送pod创建请求。
		*/
		// 7、批量创建 pod，呈指数级增长
		for batchSize := int32(integer.IntMin(int(diff), controller.SlowStartInitialBatchSize)); diff > 0; batchSize = integer.Int32Min(2*batchSize, diff) {
			errorCount := len(errCh)
			wait.Add(int(batchSize))
			for i := int32(0); i < batchSize; i++ {
				go func() {
					defer wait.Done()
					err := jm.podControl.CreatePodsWithControllerRef(job.Namespace, &job.Spec.Template, job, metav1.NewControllerRef(job, controllerKind))
					// 失败原因为'命名空间处于Terminating终止状态：' -> 返回
					if err != nil {
						if errors.HasStatusCause(err, v1.NamespaceTerminatingCause) {
							// If the namespace is being torn down, we can safely ignore
							// this error since all subsequent creations will fail.
							return
						}
					}

					if err != nil {
						defer utilruntime.HandleError(err)
						// Decrement the expected number of creates because the informer won't observe this pod
						klog.V(2).Infof("Failed creation, decrementing expectations for job %q/%q", job.Namespace, job.Name)
						jm.expectations.CreationObserved(jobKey)
						activeLock.Lock()
						active--
						activeLock.Unlock()
						errCh <- err
					}
				}()
			}
			wait.Wait()
			// any skipped pods that we never attempted to start shouldn't be expected.
			// 9、若有创建失败的操作记录在 expectations 中
			skippedPods := diff - batchSize
			if errorCount < len(errCh) && skippedPods > 0 {
				klog.V(2).Infof("Slow-start failure. Skipping creation of %d pods, decrementing expectations for job %q/%q", skippedPods, job.Namespace, job.Name)
				active -= skippedPods
				for i := int32(0); i < skippedPods; i++ {
					// Decrement the expected number of creates because the informer won't observe this pod
					// 给定控制器的期望计数 -1
					jm.expectations.CreationObserved(jobKey)
				}
				// The skipped pods will be retried later. The next controller resync will
				// retry the slow start process.
				break
			}
			diff -= batchSize
		}
	}

	select {
	case err := <-errCh:
		// all errors have been reported before, we only need to inform the controller that there was an error and it should re-try this job once more next time.
		if err != nil {
			return active, err
		}
	default:
	}

	return active, nil
}

func (jm *JobController) updateJobStatus(job *batch.Job) error {
	jobClient := jm.kubeClient.BatchV1().Jobs(job.Namespace)
	var err error
	for i := 0; i <= statusUpdateRetries; i = i + 1 {
		var newJob *batch.Job
		newJob, err = jobClient.Get(context.TODO(), job.Name, metav1.GetOptions{})
		if err != nil {
			break
		}
		newJob.Status = job.Status
		if _, err = jobClient.UpdateStatus(context.TODO(), newJob, metav1.UpdateOptions{}); err == nil {
			break
		}
	}

	return err
}

func getBackoff(queue workqueue.RateLimitingInterface, key interface{}) time.Duration {
	exp := queue.NumRequeues(key)

	if exp <= 0 {
		return time.Duration(0)
	}

	// The backoff is capped such that 'calculated' value never overflows.
	backoff := float64(DefaultJobBackOff.Nanoseconds()) * math.Pow(2, float64(exp-1))
	if backoff > math.MaxInt64 {
		return MaxJobBackOff
	}

	calculated := time.Duration(backoff)
	if calculated > MaxJobBackOff {
		return MaxJobBackOff
	}
	return calculated
}

// filterPods returns pods based on their phase.
func filterPods(pods []*v1.Pod, phase v1.PodPhase) int {
	result := 0
	for i := range pods {
		if phase == pods[i].Status.Phase {
			result++
		}
	}
	return result
}
