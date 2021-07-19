# 配置控制器

## 函数主体

```shell script
c, err := s.Config(KnownControllers(), ControllersDisabledByDefault.List())
```

## KnownControllers()解析

> 函数主体

```shell script
// KnownControllers returns all known controllers's name
func KnownControllers() []string {
	ret := sets.StringKeySet(NewControllerInitializers(IncludeCloudLoops))

	// add "special" controllers that aren't initialized normally.  These controllers cannot be initialized
	// using a normal function.  The only known special case is the SA token controller which *must* be started
	// first to ensure that the SA tokens for future controllers will exist.  Think very carefully before adding
	// to this list.
	ret.Insert(
		saTokenControllerName,
	)

	return ret.List()
}
```

> 控制器列表

```shell script
{
"endpoint",
"endpointslice",
"replicationcontroller",
"podgc",
"resourcequota",
"namespace",
"serviceaccount",
"garbagecollector",
"daemonset",
"job",
"deployment",
"replicaset",
"horizontalpodautoscaling",
"disruption",
"statefulset",
"cronjob",
"csrsigning",
"csrapproving",
"csrcleaner",
"ttl",
"bootstrapsigner",
"tokencleaner",
"nodeipam",
"nodelifecycle",
"persistentvolume-binder",
"attachdetach",
"persistentvolume-expander",
"clusterrole-aggregation",
"pvc-protection",
"pv-protection",
"ttl-after-finished",
"root-ca-cert-publisher",
"serviceaccount-token"
}

如果云环境下,还包含以下控制器
if loopMode == IncludeCloudLoops {
    controllers["service"] = startServiceController
    controllers["route"] = startRouteController
    controllers["cloud-node-lifecycle"] = startCloudNodeLifecycleController
    // TODO: volume controller into the IncludeCloudLoops only set.
}
```
## 


