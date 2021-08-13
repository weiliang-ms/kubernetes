# GarbageCollectorController源码解析

在`k8s`系统中，当删除一个对象时，其对应的`controller`并不会执行删除对象的操作，
在`kubernetes`中对象的回收操作是由`GarbageCollectorController`负责的，
其作用就是当删除一个对象时，会根据指定的删除策略回收该对象及其依赖对象，本文会深入分析垃圾收集背后的实现。

## kubernetes中的删除策略

`kubernetes`中有三种删除策略：`Orphan`、`Foreground`和`Background`，三种删除策略的意义分别为:
- `Orphan`策略: 非级联删除，删除对象时，不会自动删除它的依赖或者是子对象，这些依赖被称作是原对象的孤儿对象。
例如当执行以下命令时会使用`Orphan`策略进行删除，此时`ds`的依赖对象`ontrollerrevision`不会被删除
```shell
kubectl delete ds/nginx-ds --cascade=false
```
- `Foreground`策略: 在该模式下，对象首先进入`删除中`状态，
即会设置对象的`deletionTimestamp`字段并且对象的`metadata.finalizers`字段包含了值`foregroundDeletion`，
此时该对象依然存在，然后垃圾收集器会删除该对象的所有依赖对象，
垃圾收集器在删除了所有`Blocking`状态的依赖对象（指其子对象中`ownerReference.blockOwnerDeletion=true`的对象）之后，
然后才会删除对象本身。

- `Background`策略: 在该模式下，`kubernetes`会立即删除该对象，然后垃圾收集器会在后台删除这些该对象的依赖对象

在`v1.9`以前的版本中，大部分`controller`默认的删除策略为`Orphan`，从`v1.9`开始，
对于`apps/v1`下的资源默认使用`Background`模式。
以上三种删除策略都可以在删除对象时通过设置`deleteOptions.propagationPolicy`字段进行指定，如下所示

```shell
curl -k -v -XDELETE  -H "Accept: application/json" -H "Content-Type: application/json" -d '{"propagationPolicy":"Foreground"}' 'https://192.168.99.108:8443/apis/apps/v1/namespaces/default/daemonsets/nginx-ds'
```

## finalizer机制

`finalizer`是在删除对象时设置的一个`hook`，其目的是为了让对象在删除前确认其子对象已经被完全删除，
`k8s`中默认有两种`finalizer`：`OrphanFinalizer`和`ForegroundFinalizer`，`finalizer`存在于对象的`ObjectMeta`中，
当一个对象的依赖对象被删除后其对应的`finalizers`字段也会被移除，只有`finalizers`字段为空时，`apiserver`才会删除该对象。

`Finalizers`允许`Operator`控制器实现异步的`pre-delete hook`。
比如你给`API`类型中的每个对象都创建了对应的外部资源，你希望在`k8s`删除对应资源时同时删除关联的外部资源， 
那么可以通过`Finalizers`来实现。

`Finalizers`是由字符串组成的列表，当`Finalizers`字段存在时，相关资源不允许被强制删除。
存在`Finalizers`字段的的资源对象接收的第一个删除请求设置`metadata.deletionTimestamp`字段的值，
但不删除具体资源，在该字段设置后，`finalizer`列表中的对象只能被删除，不能做其他操作。

当`metadata.deletionTimestamp`字段非空时，`controller watch`对象并执行对应`finalizers`的动作，
当所有动作执行完后，需要清空`finalizers` ，之后`k8s`会删除真正想要删除的资源

## 源码解析

`GarbageCollectorController`负责回收`kubernetes`中的资源，
要回收`kubernetes`中所有资源首先得监控所有资源，`GarbageCollectorControlle`会监听集群中所有可删除资源产生的所有事件，
这些事件会被放入到一个队列中，然后`controller`会启动多个`goroutine`处理队列中的事件，
若为删除事件会根据对象的删除策略删除关联的对象，对于非删除事件会更新对象之间的依赖关系。

首先还是看`GarbageCollectorController`的启动方法`startGarbageCollectorController`

### startGarbageCollectorController()

其主要逻辑为:
1. 初始化`discoveryClient`，`discoveryClient`主要用来获取集群中的所有资源
2. 调用`garbagecollector.GetDeletableResources`获取集群内所有可删除的资源对象，
支持`delete`,`list`,`watch`三种操作的`resource`称为`deletableResource`
3. 忽略`event`资源删除对象
4. 初始化`garbageCollector`对象
5. 调用`garbageCollector.Run`启动`garbageCollector`
6. 调用`garbageCollector.Sync`监听集群中的`DeletableResources`，
当出现新的`DeletableResources`时同步到`monitors`中，确保监控集群中的所有资源
7. 调用`garbagecollector.NewDebugHandler`注册`debug`接口，用来提供集群内所有对象的关联关系

```go
func startGarbageCollectorController(ctx ControllerContext) (http.Handler, bool, error) {
	if !ctx.ComponentConfig.GarbageCollectorController.EnableGarbageCollector {
		return nil, false, nil
	}
	// 1、初始化 discoveryClient
	gcClientset := ctx.ClientBuilder.ClientOrDie("generic-garbage-collector")
	discoveryClient := cacheddiscovery.NewMemCacheClient(gcClientset.Discovery())

	config := ctx.ClientBuilder.ConfigOrDie("generic-garbage-collector")
	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return nil, true, err
	}
	// 2、获取 deletableResource
	// Get an initial set of deletable resources to prime the garbage collector.
	deletableResources := garbagecollector.GetDeletableResources(discoveryClient)
	ignoredResources := make(map[schema.GroupResource]struct{})
	// 3、忽略删除event资源
	for _, r := range ctx.ComponentConfig.GarbageCollectorController.GCIgnoredResources {
		ignoredResources[schema.GroupResource{Group: r.Group, Resource: r.Resource}] = struct{}{}
	}
	// 4、初始化garbageCollector对象
	garbageCollector, err := garbagecollector.NewGarbageCollector(
		metadataClient,
		ctx.RESTMapper,
		deletableResources,
		ignoredResources,
		ctx.ObjectOrMetadataInformerFactory,
		ctx.InformersStarted,
	)
	if err != nil {
		return nil, true, fmt.Errorf("failed to start the generic garbage collector: %v", err)
	}

	// 5、调用 garbageCollector.Run 启动 garbageCollecto
	// Start the garbage collector.
	workers := int(ctx.ComponentConfig.GarbageCollectorController.ConcurrentGCSyncs)
	go garbageCollector.Run(workers, ctx.Stop)

	// 6、监听集群中的`DeletableResources`，当出现新的`DeletableResources`时同步到`monitors`中，确保监控集群中的所有资源
	// Periodically refresh the RESTMapper with new discovery information and sync
	// the garbage collector.
	go garbageCollector.Sync(gcClientset.Discovery(), 30*time.Second, ctx.Stop)

	// 7、注册debug接口，用来提供集群内所有对象的关联关系；
	return garbagecollector.NewDebugHandler(garbageCollector), true, nil
}
```

在`startGarbageCollectorController`中主要调用了四种方法:
- `garbagecollector.NewGarbageCollector`
- `garbageCollector.Run`
- `garbageCollector.Sync`
- `garbagecollector.NewDebugHandler`

下面主要针对这四种方法进行说明。

> `garbagecollector.NewGarbageCollector()`函数解析

- `NewGarbageCollector`的主要功能是初始化`GarbageCollector`和`GraphBuilder`对象，
并调用`gb.syncMonitors`方法初始化`deletableResources`中所有`resource controller`的`informer`。
  - `GarbageCollector`的主要作用是启动`GraphBuilder`以及启动所有的消费者
  - `GraphBuilder`的主要作用是启动所有的生产者。

```go
func NewGarbageCollector(
	metadataClient metadata.Interface,
	mapper resettableRESTMapper,
	deletableResources map[schema.GroupVersionResource]struct{},
	ignoredResources map[schema.GroupResource]struct{},
	sharedInformers controller.InformerFactory,
	informersStarted <-chan struct{},
) (*GarbageCollector, error) {
	attemptToDelete := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "garbage_collector_attempt_to_delete")
	attemptToOrphan := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "garbage_collector_attempt_to_orphan")
	absentOwnerCache := NewUIDCache(500)
	gc := &GarbageCollector{
		metadataClient:   metadataClient,
		restMapper:       mapper,
		attemptToDelete:  attemptToDelete,
		attemptToOrphan:  attemptToOrphan,
		absentOwnerCache: absentOwnerCache,
	}
	gb := &GraphBuilder{
		metadataClient:   metadataClient,
		informersStarted: informersStarted,
		restMapper:       mapper,
		graphChanges:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "garbage_collector_graph_changes"),
		uidToNode: &concurrentUIDToNode{
			uidToNode: make(map[types.UID]*node),
		},
		attemptToDelete:  attemptToDelete,
		attemptToOrphan:  attemptToOrphan,
		absentOwnerCache: absentOwnerCache,
		sharedInformers:  sharedInformers,
		ignoredResources: ignoredResources,
	}
	if err := gb.syncMonitors(deletableResources); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to sync all monitors: %v", err))
	}
	gc.dependencyGraphBuilder = gb

	return gc, nil
}
```

> `gb.syncMonitors()`解析

`syncMonitors`的主要作用是初始化各个资源对象的`informer`，
并调用`gb.controllerFor`为每种资源注册`eventHandler`，
此处每种资源被称为`monitors`，因为为每种资源注册`eventHandler`时，
对于`AddFunc、UpdateFunc`和`DeleteFunc`都会将对应的`event push`到`graphChanges`队列中，
每种资源对象的`informer`都作为生产者

```go
func (gb *GraphBuilder) syncMonitors(resources map[schema.GroupVersionResource]struct{}) error {
	gb.monitorLock.Lock()
	defer gb.monitorLock.Unlock()

	toRemove := gb.monitors
	if toRemove == nil {
		toRemove = monitors{}
	}
	current := monitors{}
	errs := []error{}
	kept := 0
	added := 0
	for resource := range resources {
		if _, ok := gb.ignoredResources[resource.GroupResource()]; ok {
			continue
		}
		if m, ok := toRemove[resource]; ok {
			current[resource] = m
			delete(toRemove, resource)
			kept++
			continue
		}
		kind, err := gb.restMapper.KindFor(resource)
		if err != nil {
			errs = append(errs, fmt.Errorf("couldn't look up resource %q: %v", resource, err))
			continue
		}
		c, s, err := gb.controllerFor(resource, kind)
		if err != nil {
			errs = append(errs, fmt.Errorf("couldn't start monitor for resource %q: %v", resource, err))
			continue
		}
		current[resource] = &monitor{store: s, controller: c}
		added++
	}
	gb.monitors = current

	for _, monitor := range toRemove {
		if monitor.stopCh != nil {
			close(monitor.stopCh)
		}
	}

	klog.V(4).Infof("synced monitors; added %d, kept %d, removed %d", added, kept, len(toRemove))
	// NewAggregate returns nil if errs is 0-length
	return utilerrors.NewAggregate(errs)
}
```



## 参考文献

- [garbage collector controller 源码分析](https://cloud.tencent.com/developer/article/1562130)