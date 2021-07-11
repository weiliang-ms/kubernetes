# job-controller

## 核心概念

### 声明

> 声明一个`job`

    cat <<EOF | kubectl apply -f -
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: pi-job
    spec:
      template:
        spec:
          containers:
          - name: pi
            image: perl
            command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
          restartPolicy: Never
    EOF
    
> 查看状态

    [root@node1 kubectl get pod -w
    NAME           READY   STATUS    RESTARTS   AGE
    pi-job-hh6rg   1/1     Running   0          20s
    pi-job-hh6rg   0/1     Completed   0          21s
    
> 清理Job

    kubectl delete job pi-job
    
### 自动清理job

每次`job`执行完成后手动回收非常麻烦，`k8s`在`v1.12`版本中加入了`TTLAfterFinished feature-gates`，
启用该特性后会启动一个`TTL`控制器，在创建`job`时指定后可在`job`运行完成后自动回收相关联的`pod`，

由于`k8s1.21`版本前，该特性还是`alpha`版本，需要给`kube-controller-manager`和 `kube-apiserver`开启`TTLAfterFinished `才能生效。

    `--feature-gates=`
    
添加
    
    `TTLAfterFinished=true`

> 运行完成10秒内自动删除

`ttlSecondsAfterFinished`为`k8s1.12`的`alpha`特性，`k8s1.21`升级为`beta`
该参数设置后`job`在运行完成后在指定时间内就会自动删除掉

    cat <<EOF | kubectl apply -f -
        apiVersion: batch/v1
        kind: Job
        metadata:
          name: pi-job
        spec:
          ttlSecondsAfterFinished: 5
          template:
            spec:
              containers:
              - name: pi
                image: perl
                imagePullPolicy: IfNotPresent
                command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
              restartPolicy: Never
    EOF
    
观测`pod`状态

    [root@node1 ~]# kubectl get pod -w
    NAME           READY   STATUS    RESTARTS   AGE
    pi-job-8rthr   1/1     Running   0          7s
    pi-job-8rthr   0/1     Completed   0          11s
    pi-job-8rthr   0/1     Terminating   0          16s

`job`完成`5s`被回收
    
## 源码分析

### startJobController

首先还是直接看`jobController`的启动方法`startJobController`，
该方法中调用`NewJobController`初始化`jobController`然后调用`Run`方法启动`jobController`。
从初始化流程中可以看到`JobController`监听`pod`和`job`两种资源，其中`ConcurrentJobSyncs`默认值为`5`。

[startJobController](../../cmd/kube-controller-manager/app/batch.go)

    // job控制器启动方式
    func startJobController(ctx ControllerContext) (http.Handler, bool, error) {
        // 判断job控制器是否已存在
    	if !ctx.AvailableResources[schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}] {
    		return nil, false, nil
    	}
    	// 
    	// 开启协程，创建job控制器并运行
    	go job.NewJobController(
    	    // 监听pod job资源
    		ctx.InformerFactory.Core().V1().Pods(),
    		ctx.InformerFactory.Batch().V1().Jobs(),
    		ctx.ClientBuilder.ClientOrDie("job-controller"),
    	).Run(int(ctx.ComponentConfig.JobController.ConcurrentJobSyncs), ctx.Stop)
    	return nil, true, nil
    }

### NewJobController()方法解析

> 源码实体

    // 初始化job控制器函数
    func NewJobController(podInformer coreinformers.PodInformer, jobInformer batchinformers.JobInformer, kubeClient clientset.Interface) *JobController {
    	// 创建事件通知器
    	eventBroadcaster := record.NewBroadcaster()
    	eventBroadcaster.StartLogging(klog.Infof)
    	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
    
    
        // 
    	if kubeClient != nil && kubeClient.CoreV1().RESTClient().GetRateLimiter() != nil {
    		ratelimiter.RegisterMetricAndTrackRateLimiterUsage("job_controller", kubeClient.CoreV1().RESTClient().GetRateLimiter())
    	}
    
        // 初始化job控制器对象
    	jm := &JobController{
    	    // kube客户端对象
    		kubeClient: kubeClient,
    		// pod控制器（包含创建、删除、更新Pod）
    		podControl: controller.RealPodControl{
    			KubeClient: kubeClient,
    			Recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "job-controller"}),
    		},
    		expectations: controller.NewControllerExpectations(),
    		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(DefaultJobBackOff, MaxJobBackOff), "job"),
    		recorder:     eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "job-controller"}),
    	}
        // 配置监听
    	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    	    // 
    		AddFunc: func(obj interface{}) {
    			jm.enqueueController(obj, true)
    		},
    		UpdateFunc: jm.updateJob,
    		DeleteFunc: func(obj interface{}) {
    			jm.enqueueController(obj, true)
    		},
    	})
    	jm.jobLister = jobInformer.Lister()
    	jm.jobStoreSynced = jobInformer.Informer().HasSynced
    
    	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    		AddFunc:    jm.addPod,
    		UpdateFunc: jm.updatePod,
    		DeleteFunc: jm.deletePod,
    	})
    	jm.podStore = podInformer.Lister()
    	jm.podStoreSynced = podInformer.Informer().HasSynced
    
    	jm.updateHandler = jm.updateJobStatus
    	jm.syncHandler = jm.syncJob
    
    	return jm
    }

### Job控制器Run方法

[job_controller.go 140行](../../pkg/controller/job/job_controller.go)

以下是`jobController`的`Run`方法，其中核心逻辑是调用`jm.worker`执行`syncLoop`操作，
`worker`方法是`syncJob`方法的别名，最终调用的是`syncJob`

    // Run the main goroutine responsible for watching and syncing jobs.
    // 监听同步job状态
    func (jm *JobController) Run(workers int, stopCh <-chan struct{}) {
        // 捕获协程内的异常panic，进行处理
        // 在goroutine里使用defer+recover进行异常处理，可以保证goroutine发生panic，不会对主进程产生影响
    	defer utilruntime.HandleCrash()
    	// 方法结束前销毁Job队列？
    	defer jm.queue.ShutDown()
    
    	klog.Infof("Starting job controller")
    	defer klog.Infof("Shutting down job controller")
    
        // 同步job控制器管理的job pod至缓存对象jm中
    	if !cache.WaitForNamedCacheSync("job", stopCh, jm.podStoreSynced, jm.jobStoreSynced) {
    		return
    	}
    
    	for i := 0; i < workers; i++ {
    		go wait.Until(jm.worker, time.Second, stopCh)
    	}
    
        // 等待信道被关闭
    	<-stopCh
    } 
    
> 关于协程异常处理

在`goroutine`里使用`defer+recover`进行异常处理，可以保证`goroutine`发生`panic`，不会对主进程产生影响
    
不做`defer+recover`进行异常处理时

    func main()  {
    
    	go func() {
    		panic("goroutine panic")
    	}()
    	
    	time.Sleep(1*time.Second)
    
    	fmt.Println("ok")
    }
    
    // fmt.Println("ok")无法执行，主进程直接退出
    
`defer+recover`进行异常处理时

    func main(){
    	go func() {
    		defer func() {
    			if err := recover(); err != nil {
    				// 打印出err信息
    				fmt.Println(reflect.ValueOf(err).String())
    				// 也可以继续panic
    				//panic(err.Error)
    			}
    		}()
    		panic("goroutine error")
    	}()
    
    	// goroutine发生panic，只会使协程中断，但是不影响主进程，主进程还可以正常运行。
    	for{
    		time.Sleep(1*time.Second)
    		fmt.Println("ok")
    	}
    }
    // 输出如下
    goroutine error
    ok
    ok
    ...
    
> 关于`stopCh`

在`Go`语言中，有一种特殊的`struct{}`类型的`channel`，它不能被写入任何数据，
只有通过`close()`函数进行关闭操作，才能进行输出操作。
`struct`类型的channel不占用任何内存！！！

应用场景-等待某任务的结束：

    done := make(chan struct{})
	begin := time.Now()
	go func() {
		fmt.Println("[goroutine] begin goroutine process...")
		time.Sleep(time.Second * 10)
		close(done)
		defer fmt.Println("[goroutine] return main process...")
	}()
	// do some other bits
	// wait for that long running thing to finish
	fmt.Println("[main] before goroutine done...")
	fmt.Println(time.Now().Sub(begin).Seconds())
	// 阻塞到协程内的方法执行完毕
	<-done
	fmt.Println("[main] goroutine done...")
	fmt.Println(time.Now().Sub(begin).Seconds())
    
    // 输出如下：
   
    [main] before goroutine done...
    [goroutine] begin goroutine process...
    0.0006752
    [goroutine] return main progrecess...
    [main] goroutine done...
    10.0011074

> 解析定时同步机制原理-1
    
`workers`为每次同步数量由上层调用传入
`time.Second`表示同步频率为每秒一次
`stopCh`传入`ch`对象,当`ch`被关闭时(close(stopCh)),协程结束
如不执行`<-stopCh`，则创建协程后直接退出
    
    for i := 0; i < workers; i++ {
        go wait.Until(jm.worker, time.Second, stopCh)
    }
    <-stopCh
    
`for`循环的作用: 初始化`works`个协程,通过下面样例验证:

    package main
    import (
    	"fmt"
    	"k8s.io/apimachinery/pkg/util/wait"
    	"time"
    )
    
    func main(){
    	var stopCh <- chan struct{}
    	for i := 0; i < 5; i++ {
    		go wait.Until(currentTime, time.Second, stopCh)
    	}
    
    	<-stopCh
    }
    
    func currentTime()  {
    	fmt.Println(time.Now().Unix())
    }
    
    // 输出
    1625912664
    1625912664
    1625912664
    1625912664
    1625912664
    1625912665
    1625912665
    1625912665
    1625912665
    ...

> 解析定时同步机制原理-2

`wait.Until`调用`Until`函数,`Until`函数调用`JitterUntil`函数

定时器在`f()`函数执行完成后开始运行

    func Until(f func(), period time.Duration, stopCh <-chan struct{}) {
    	JitterUntil(f, period, 0.0, true, stopCh)
    }

> 解析定时同步机制原理-3
    
`JitterUntil`函数实体如下：

- `JitterUntil`周期性（默认为1秒）运行`f`函数。
- 入参`jitterFactor`如果是正的（默认0.0），定时器在`f()`函数的每一次运行之前被抖动
- `sliding`: 默认入参为`true`，即定时任务间隔时间（默认1秒）不包含执行`f()`函数所需的时间

    func JitterUntil(f func(), period time.Duration, jitterFactor float64, sliding bool, stopCh <-chan struct{}) {
    	BackoffUntil(f, NewJitteredBackoffManager(period, jitterFactor, &clock.RealClock{}), sliding, stopCh)
    }
    
> 解析定时同步机制原理-4

根据`backoff`的定时器来循环触发`f`函数，直到`stopCh`关闭

`BackoffUntil`函数实体如下：

    // BackoffUntil一直循环，周期性的（每秒）运行f()函数，直到stodCh通道关闭
    func BackoffUntil(f func(), backoff BackoffManager, sliding bool, stopCh <-chan struct{}) {
    	// 定义计时器
    	var t clock.Timer
    	// 开启循环流程
    	for {
    	    // step 1
    	    // 在golang中select没有优先级选择，为了避免额外执行f(),在每次循环开始后会先判断stopCh是否关闭
    	    // 如果stopCh通道关闭，退出循环（定时任务）
    		select {
    		case <-stopCh:
    			return
    		default:
    		}
    
            // 默认sliding被设置为true，该逻辑不会被执行
    		if !sliding {
    			t = backoff.Backoff()
    		}
    
            // step 2
            // 执行f()函数，并捕捉异常
    		func() {
    			defer runtime.HandleCrash()
    			f()
    		}()
    
            // step 3
            // sliding被设置为true，执行定时器赋值（默认定时器间隔1s）
    		if sliding {
    		    // 本质为定时器（带抖动属性）
    			t = backoff.Backoff()
    		}
    
            // // step 3
            // 在golang中select没有优先级选择，为了避免额外执行f(),判断stopCh是否关闭
    		select {
            // 如果stopCh通道关闭，提前退出循环（无需等待至定时结束进入下一轮for{}再退出）
    		case <-stopCh:
    			return
            // 阻塞至定时结束
            // time.Timer需要对通道进行释放才能达到定时的效果
    		case <-t.C():
    		}
    	}
    }
    
原理同下：

    package main
  
    import (
    	"fmt"
    	"k8s.io/apimachinery/pkg/util/runtime"
    	"time"
    )
    
    func main(){
    	var t time.Timer
    	var stopCh <- chan struct{}
    	for {
    		select {
    		case <-stopCh:
    			return
    		default:
    		}
    
    		func() {
    			defer runtime.HandleCrash()
    			fmt.Println("执行f()函数逻辑...")
    		}()
    
    		t=*time.NewTimer(time.Second)
    
    		select {
    		case <-stopCh:
    			return
    		case <-t.C:
    		}
    	}
    	<-stopCh
    }
 
至此，`Job-controller`开启了定时执行`f()`的流程
 
### 关于worker函数

也就是`Run`函数中引用的`jm.worker`

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

> `worker()`实体如下

实际调用`jm.processNextWorkItem()`

    func (jm *JobController) worker() {
        for jm.processNextWorkItem() {
        }
    }

> 分析`processNextWorkItem()`-1

[pkg/controller/job/job_controller.go 第384行](../../pkg/controller/job/job_controller.go)

    func (jm *JobController) processNextWorkItem() bool {
        // 获取job控制器队列
    	key, quit := jm.queue.Get()
    	// 判断队列是否被关闭
    	if quit {
    		return false
    	}
    	// 函数结束前，标记key更新状态完毕
    	defer jm.queue.Done(key)
    
        // 同步
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
    
> 分析`processNextWorkItem()`-2

关于`jm.queue.Get()`分析

    key, quit := jm.queue.Get()
    
- `jm.queue`队列存储的是需要更新的`job`
- `Get()`是[workqueue](../../staging/src/k8s.io/client-go/util/workqueue/queue.go) 包中`Interface`
接口的一个方法，其他方法如下：
    - `Add(item interface{})`
    - `Len() int`
    - `Get() (item interface{}, shutdown bool)`
    - `Done(item interface{})`
    - `ShutDown()`
    - `ShuttingDown() bool`

`Get()`对应实现为

    func (q *Type) Get() (item interface{}, shutdown bool) {
        // 创建同步锁
    	q.cond.L.Lock()
    	// 函数执行结束前释放同步锁
    	defer q.cond.L.Unlock()
    	// 判读队列状态
    	for len(q.queue) == 0 && !q.shuttingDown {
    	    // 如果队列开启，且队列无要处理对象执行Wait()方法
    		q.cond.Wait()
    	}
    	
    	if len(q.queue) == 0 {
    		// We must be shutting down.
    		return nil, true
    	}
    
        // 获取队列第一个元素
    	item, q.queue = q.queue[0], q.queue[1:]
    
        // 获取元素的metrics信息
    	q.metrics.get(item)
    
    	q.processing.insert(item)
    	q.dirty.delete(item)
    
    	return item, false
    }
