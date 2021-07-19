# NewKubeControllerManagerOptions()解析

初始化

## 函数主体

```shell script
// NewKubeControllerManagerOptions creates a new KubeControllerManagerOptions with a default config.
func NewKubeControllerManagerOptions() (*KubeControllerManagerOptions, error) {
	// 初始化控制器默认配置
	componentConfig, err := NewDefaultComponentConfig(ports.InsecureKubeControllerManagerPort)
	if err != nil {
		return nil, err
	}

	// 初始化配置
	s := KubeControllerManagerOptions{
		Generic:         cmoptions.NewGenericControllerManagerConfigurationOptions(&componentConfig.Generic),
		KubeCloudShared: cmoptions.NewKubeCloudSharedOptions(&componentConfig.KubeCloudShared),
		ServiceController: &cmoptions.ServiceControllerOptions{
			ServiceControllerConfiguration: &componentConfig.ServiceController,
		},
		AttachDetachController: &AttachDetachControllerOptions{
			&componentConfig.AttachDetachController,
		},
		CSRSigningController: &CSRSigningControllerOptions{
			&componentConfig.CSRSigningController,
		},
		DaemonSetController: &DaemonSetControllerOptions{
			&componentConfig.DaemonSetController,
		},
		DeploymentController: &DeploymentControllerOptions{
			&componentConfig.DeploymentController,
		},
		StatefulSetController: &StatefulSetControllerOptions{
			&componentConfig.StatefulSetController,
		},
		DeprecatedFlags: &DeprecatedControllerOptions{
			&componentConfig.DeprecatedController,
		},
		EndpointController: &EndpointControllerOptions{
			&componentConfig.EndpointController,
		},
		EndpointSliceController: &EndpointSliceControllerOptions{
			&componentConfig.EndpointSliceController,
		},
		GarbageCollectorController: &GarbageCollectorControllerOptions{
			&componentConfig.GarbageCollectorController,
		},
		HPAController: &HPAControllerOptions{
			&componentConfig.HPAController,
		},
		JobController: &JobControllerOptions{
			&componentConfig.JobController,
		},
		NamespaceController: &NamespaceControllerOptions{
			&componentConfig.NamespaceController,
		},
		NodeIPAMController: &NodeIPAMControllerOptions{
			&componentConfig.NodeIPAMController,
		},
		NodeLifecycleController: &NodeLifecycleControllerOptions{
			&componentConfig.NodeLifecycleController,
		},
		PersistentVolumeBinderController: &PersistentVolumeBinderControllerOptions{
			&componentConfig.PersistentVolumeBinderController,
		},
		PodGCController: &PodGCControllerOptions{
			&componentConfig.PodGCController,
		},
		ReplicaSetController: &ReplicaSetControllerOptions{
			&componentConfig.ReplicaSetController,
		},
		ReplicationController: &ReplicationControllerOptions{
			&componentConfig.ReplicationController,
		},
		ResourceQuotaController: &ResourceQuotaControllerOptions{
			&componentConfig.ResourceQuotaController,
		},
		SAController: &SAControllerOptions{
			&componentConfig.SAController,
		},
		TTLAfterFinishedController: &TTLAfterFinishedControllerOptions{
			&componentConfig.TTLAfterFinishedController,
		},
		SecureServing: apiserveroptions.NewSecureServingOptions().WithLoopback(),
		InsecureServing: (&apiserveroptions.DeprecatedInsecureServingOptions{
			BindAddress: net.ParseIP(componentConfig.Generic.Address),
			BindPort:    int(componentConfig.Generic.Port),
			BindNetwork: "tcp",
		}).WithLoopback(),
		Authentication: apiserveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:  apiserveroptions.NewDelegatingAuthorizationOptions(),
	}

	// 开启认证、鉴权
	s.Authentication.RemoteKubeConfigFileOptional = true
	s.Authorization.RemoteKubeConfigFileOptional = true
	// 开放/healthz path（无需鉴权）
	s.Authorization.AlwaysAllowPaths = []string{"/healthz"}

	// Set the PairName but leave certificate directory blank to generate in-memory by default
	// 设置ca证书（基于内存）
	s.SecureServing.ServerCert.CertDirectory = ""
	s.SecureServing.ServerCert.PairName = "kube-controller-manager"
	s.SecureServing.BindPort = ports.KubeControllerManagerPort

	// 配置GarbageCollectorController的gc策略，不回收events类型
	gcIgnoredResources := make([]garbagecollectorconfig.GroupResource, 0, len(garbagecollector.DefaultIgnoredResources()))
	for r := range garbagecollector.DefaultIgnoredResources() {
		gcIgnoredResources = append(gcIgnoredResources, garbagecollectorconfig.GroupResource{Group: r.Group, Resource: r.Resource})
	}

	s.GarbageCollectorController.GCIgnoredResources = gcIgnoredResources
	// 设置选主资源名称（选主时，该控制器资源对象所属命名空间及实例加锁）
	s.Generic.LeaderElection.ResourceName = "kube-controller-manager"
	s.Generic.LeaderElection.ResourceNamespace = "kube-system"

	return &s, nil
}
```

## 流程解析

### 1.[初始化控制器默认配置](ncmc-nkcmo-ndcc.md)

```shell script
componentConfig, err := NewDefaultComponentConfig(ports.InsecureKubeControllerManagerPort)
if err != nil {
    return nil, err
}
```

### 2.初始化控制器

```shell script
// 初始化配置
	s := KubeControllerManagerOptions{
		Generic:         cmoptions.NewGenericControllerManagerConfigurationOptions(&componentConfig.Generic),
		KubeCloudShared: cmoptions.NewKubeCloudSharedOptions(&componentConfig.KubeCloudShared),
		ServiceController: &cmoptions.ServiceControllerOptions{
			ServiceControllerConfiguration: &componentConfig.ServiceController,
		},
		AttachDetachController: &AttachDetachControllerOptions{
			&componentConfig.AttachDetachController,
		},
		CSRSigningController: &CSRSigningControllerOptions{
			&componentConfig.CSRSigningController,
		},
		DaemonSetController: &DaemonSetControllerOptions{
			&componentConfig.DaemonSetController,
		},
		DeploymentController: &DeploymentControllerOptions{
			&componentConfig.DeploymentController,
		},
		StatefulSetController: &StatefulSetControllerOptions{
			&componentConfig.StatefulSetController,
		},
		DeprecatedFlags: &DeprecatedControllerOptions{
			&componentConfig.DeprecatedController,
		},
		EndpointController: &EndpointControllerOptions{
			&componentConfig.EndpointController,
		},
		EndpointSliceController: &EndpointSliceControllerOptions{
			&componentConfig.EndpointSliceController,
		},
		GarbageCollectorController: &GarbageCollectorControllerOptions{
			&componentConfig.GarbageCollectorController,
		},
		HPAController: &HPAControllerOptions{
			&componentConfig.HPAController,
		},
		JobController: &JobControllerOptions{
			&componentConfig.JobController,
		},
		NamespaceController: &NamespaceControllerOptions{
			&componentConfig.NamespaceController,
		},
		NodeIPAMController: &NodeIPAMControllerOptions{
			&componentConfig.NodeIPAMController,
		},
		NodeLifecycleController: &NodeLifecycleControllerOptions{
			&componentConfig.NodeLifecycleController,
		},
		PersistentVolumeBinderController: &PersistentVolumeBinderControllerOptions{
			&componentConfig.PersistentVolumeBinderController,
		},
		PodGCController: &PodGCControllerOptions{
			&componentConfig.PodGCController,
		},
		ReplicaSetController: &ReplicaSetControllerOptions{
			&componentConfig.ReplicaSetController,
		},
		ReplicationController: &ReplicationControllerOptions{
			&componentConfig.ReplicationController,
		},
		ResourceQuotaController: &ResourceQuotaControllerOptions{
			&componentConfig.ResourceQuotaController,
		},
		SAController: &SAControllerOptions{
			&componentConfig.SAController,
		},
		TTLAfterFinishedController: &TTLAfterFinishedControllerOptions{
			&componentConfig.TTLAfterFinishedController,
		},
		SecureServing: apiserveroptions.NewSecureServingOptions().WithLoopback(),
		InsecureServing: (&apiserveroptions.DeprecatedInsecureServingOptions{
			BindAddress: net.ParseIP(componentConfig.Generic.Address),
			BindPort:    int(componentConfig.Generic.Port),
			BindNetwork: "tcp",
		}).WithLoopback(),
		Authentication: apiserveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:  apiserveroptions.NewDelegatingAuthorizationOptions(),
	}
```
### 3.开启认证鉴权

```shell script
// 开启认证、鉴权
s.Authentication.RemoteKubeConfigFileOptional = true
s.Authorization.RemoteKubeConfigFileOptional = true
// 开放/healthz path（无需鉴权）
s.Authorization.AlwaysAllowPaths = []string{"/healthz"}

// Set the PairName but leave certificate directory blank to generate in-memory by default
// 设置ca证书（基于内存）
s.SecureServing.ServerCert.CertDirectory = ""
s.SecureServing.ServerCert.PairName = "kube-controller-manager"
s.SecureServing.BindPort = ports.KubeControllerManagerPort
```

### 4.配置`GarbageCollectorController`的`gc`策略

```shell script
// 配置GarbageCollectorController的gc策略，不回收events类型
gcIgnoredResources := make([]garbagecollectorconfig.GroupResource, 0, len(garbagecollector.DefaultIgnoredResources()))
for r := range garbagecollector.DefaultIgnoredResources() {
    gcIgnoredResources = append(gcIgnoredResources, garbagecollectorconfig.GroupResource{Group: r.Group, Resource: r.Resource})
}

s.GarbageCollectorController.GCIgnoredResources = gcIgnoredResources
```

### 5.设置选主资源名称

```shell script
// 选主时，该控制器资源对象所属命名空间及实例加锁
s.Generic.LeaderElection.ResourceName = "kube-controller-manager"
s.Generic.LeaderElection.ResourceNamespace = "kube-system"
```

### 6.初始化默认特性列表

隐式初始化, [options.go](../../../cmd/kube-controller-manager/app/options/options.go) 内引用`k8s.io/kubernetes/pkg/features`

触发[init方法](../../../pkg/features/kube_features.go) 调用,从而初始化默认特性列表

```shell script
import(
...
_ "k8s.io/kubernetes/pkg/features"
...
)
```