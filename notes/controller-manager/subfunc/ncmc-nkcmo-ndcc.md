<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [NewDefaultComponentConfig()解析](#newdefaultcomponentconfig%E8%A7%A3%E6%9E%90)
  - [函数`NewDefaultComponentConfig()`主体](#%E5%87%BD%E6%95%B0newdefaultcomponentconfig%E4%B8%BB%E4%BD%93)
  - [相关数据结构](#%E7%9B%B8%E5%85%B3%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84)
    - [`KubeControllerManagerConfiguration`数据结构](#kubecontrollermanagerconfiguration%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84)
    - [`KubeControllerManagerConfiguration.Generic`数据结构](#kubecontrollermanagerconfigurationgeneric%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84)
    - [`ClientConnection`数据结构](#clientconnection%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84)
  - [逐行分析](#%E9%80%90%E8%A1%8C%E5%88%86%E6%9E%90)
    - [初始化结构体](#%E5%88%9D%E5%A7%8B%E5%8C%96%E7%BB%93%E6%9E%84%E4%BD%93)
    - [默认值赋值](#%E9%BB%98%E8%AE%A4%E5%80%BC%E8%B5%8B%E5%80%BC)
    - [对象转换](#%E5%AF%B9%E8%B1%A1%E8%BD%AC%E6%8D%A2)
    - [赋值监听端口](#%E8%B5%8B%E5%80%BC%E7%9B%91%E5%90%AC%E7%AB%AF%E5%8F%A3)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# NewDefaultComponentConfig()解析

## 函数`NewDefaultComponentConfig()`主体

```shell script
// s, err := options.NewKubeControllerManagerOptions()
func NewDefaultComponentConfig(insecurePort int32) (kubectrlmgrconfig.KubeControllerManagerConfiguration, error) {
	versioned := kubectrlmgrconfigv1alpha1.KubeControllerManagerConfiguration{}
	kubectrlmgrconfigscheme.Scheme.Default(&versioned)

	internal := kubectrlmgrconfig.KubeControllerManagerConfiguration{}
	if err := kubectrlmgrconfigscheme.Scheme.Convert(&versioned, &internal, nil); err != nil {
		return internal, err
	}
	internal.Generic.Port = insecurePort
	return internal, nil
}
```

## 相关数据结构

### `KubeControllerManagerConfiguration`数据结构

```shell script
type KubeControllerManagerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Generic holds configuration for a generic controller-manager
	Generic GenericControllerManagerConfiguration
	// KubeCloudSharedConfiguration holds configuration for shared related features
	// both in cloud controller manager and kube-controller manager.
	KubeCloudShared KubeCloudSharedConfiguration

	// AttachDetachControllerConfiguration holds configuration for
	// AttachDetachController related features.
	AttachDetachController AttachDetachControllerConfiguration
	// CSRSigningControllerConfiguration holds configuration for
	// CSRSigningController related features.
	CSRSigningController CSRSigningControllerConfiguration
	// DaemonSetControllerConfiguration holds configuration for DaemonSetController
	// related features.
	DaemonSetController DaemonSetControllerConfiguration
	// DeploymentControllerConfiguration holds configuration for
	// DeploymentController related features.
	DeploymentController DeploymentControllerConfiguration
	// StatefulSetControllerConfiguration holds configuration for
	// StatefulSetController related features.
	StatefulSetController StatefulSetControllerConfiguration
	// DeprecatedControllerConfiguration holds configuration for some deprecated
	// features.
	DeprecatedController DeprecatedControllerConfiguration
	// EndpointControllerConfiguration holds configuration for EndpointController
	// related features.
	EndpointController EndpointControllerConfiguration
	// EndpointSliceControllerConfiguration holds configuration for
	// EndpointSliceController related features.
	EndpointSliceController EndpointSliceControllerConfiguration
	// GarbageCollectorControllerConfiguration holds configuration for
	// GarbageCollectorController related features.
	GarbageCollectorController GarbageCollectorControllerConfiguration
	// HPAControllerConfiguration holds configuration for HPAController related features.
	HPAController HPAControllerConfiguration
	// JobControllerConfiguration holds configuration for JobController related features.
	JobController JobControllerConfiguration
	// NamespaceControllerConfiguration holds configuration for NamespaceController
	// related features.
	NamespaceController NamespaceControllerConfiguration
	// NodeIPAMControllerConfiguration holds configuration for NodeIPAMController
	// related features.
	NodeIPAMController NodeIPAMControllerConfiguration
	// NodeLifecycleControllerConfiguration holds configuration for
	// NodeLifecycleController related features.
	NodeLifecycleController NodeLifecycleControllerConfiguration
	// PersistentVolumeBinderControllerConfiguration holds configuration for
	// PersistentVolumeBinderController related features.
	PersistentVolumeBinderController PersistentVolumeBinderControllerConfiguration
	// PodGCControllerConfiguration holds configuration for PodGCController
	// related features.
	PodGCController PodGCControllerConfiguration
	// ReplicaSetControllerConfiguration holds configuration for ReplicaSet related features.
	ReplicaSetController ReplicaSetControllerConfiguration
	// ReplicationControllerConfiguration holds configuration for
	// ReplicationController related features.
	ReplicationController ReplicationControllerConfiguration
	// ResourceQuotaControllerConfiguration holds configuration for
	// ResourceQuotaController related features.
	ResourceQuotaController ResourceQuotaControllerConfiguration
	// SAControllerConfiguration holds configuration for ServiceAccountController
	// related features.
	SAController SAControllerConfiguration
	// ServiceControllerConfiguration holds configuration for ServiceController
	// related features.
	ServiceController ServiceControllerConfiguration
	// TTLAfterFinishedControllerConfiguration holds configuration for
	// TTLAfterFinishedController related features.
	TTLAfterFinishedController TTLAfterFinishedControllerConfiguration
}
```

### `KubeControllerManagerConfiguration.Generic`数据结构

```shell script
type GenericControllerManagerConfiguration struct {
	// port is the port that the controller-manager's http service runs on.
	Port int32
	// address is the IP address to serve on (set to 0.0.0.0 for all interfaces).
	Address string
	// minResyncPeriod is the resync period in reflectors; will be random between
	// minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration
	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection componentbaseconfigv1alpha1.ClientConnectionConfiguration
	// How long to wait between starting controller managers
	ControllerStartInterval metav1.Duration
	// leaderElection defines the configuration of leader election client.
	LeaderElection componentbaseconfigv1alpha1.LeaderElectionConfiguration
	// Controllers is the list of controllers to enable or disable
	// '*' means "all enabled by default controllers"
	// 'foo' means "enable 'foo'"
	// '-foo' means "disable 'foo'"
	// first item for a particular name wins
	Controllers []string
	// DebuggingConfiguration holds configuration for Debugging related features.
	Debugging componentbaseconfigv1alpha1.DebuggingConfiguration
}
```

### `ClientConnection`数据结构

```shell script
type ClientConnectionConfiguration struct {
	// KubeConfig配置文件路径
	Kubeconfig string `json:"kubeconfig"`
	// acceptContentTypes defines the Accept header sent by clients when connecting to a server, overriding the
	// default value of 'application/json'. This field will control all connections to the server used by a particular
	// client.
	AcceptContentTypes string `json:"acceptContentTypes"`
	// contentType is the content type used when sending data to the server from this client.
	ContentType string `json:"contentType"`
	// qps controls the number of queries per second allowed for this connection.
	QPS float32 `json:"qps"`
	// burst allows extra queries to accumulate when a client is exceeding its rate.
	Burst int32 `json:"burst"`
}
```

## 逐行分析

### 初始化结构体

```shell script
versioned := kubectrlmgrconfigv1alpha1.KubeControllerManagerConfiguration{}
```

### 默认值赋值

> 赋值内容

- 通用配置`KubeControllerManagerConfiguration.GenericControllerManagerConfiguration`

| 配置项                                | 配置描述                          | 默认值                               | 赋值参数 |
| :-----------------------------------:| :-----------------------------: | :---------------------------------: |:----: |
| Address                              | 监听地址                          | 0.0.0.0                             |--bind-address |
| Port                                 | 监听端口                          | 10252                               |--port |
| MinResyncPeriod                      | 重新同步周期                       | 12h0m0s                             |--min-resync-period |
| ClientConnection.Kubeconfig          | 集群配置文件                       | ""                                  | --kubeconfig |
| ClientConnection.AcceptContentTypes  | 接收数据流类型                     | ""                                  |  |
| ClientConnection.ContentType         | 发送数据流类型                     | application/vnd.kubernetes.protobuf | --kubeconfig |
| ClientConnection.QPS                 | 访问api-server服务每秒请求次数      | 20                                  | --kube-api-qps |
| ClientConnection.Burst               | 客户端超过其速率时累积额外的查询次数   | 30                                  | --kube-api-burst |

其他控制器赋值内容,暂不解析

> 赋值流程解析

隐式调用,调用链如下:

1.(隐式调用)`kubectrlmgrconfigscheme.Scheme`调用触发
`k8s.io/kubernetes/pkg/controller/apis/config/scheme/sheme.go`的`init`函数
```shell script
// NewDefaultComponentConfig returns kube-controller manager configuration object.
func NewDefaultComponentConfig(insecurePort int32) (kubectrlmgrconfig.KubeControllerManagerConfiguration, error) {
	versioned := kubectrlmgrconfigv1alpha1.KubeControllerManagerConfiguration{}
	kubectrlmgrconfigscheme.Scheme.Default(&versioned)

	internal := kubectrlmgrconfig.KubeControllerManagerConfiguration{}
	if err := kubectrlmgrconfigscheme.Scheme.Convert(&versioned, &internal, nil); err != nil {
		return internal, err
	}
	internal.Generic.Port = insecurePort
	return internal, nil
}
```

2.(隐式调用)`k8s.io/kubernetes/pkg/controller/apis/config/scheme/sheme.go`的`init`函数,调用
`k8s.io/kubernetes/pkg/controller/apis/config/v1alpha1/register.go`的`AddToScheme`变量,触发`init`函数

```shell script
func init() {
	AddToScheme(Scheme)
}

// AddToScheme registers the API group and adds types to a scheme
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(config.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1alpha1.SchemeGroupVersion))
}
```

3.(显式调用)`k8s.io/kubernetes/pkg/controller/apis/config/v1alpha1/register.go`的`init`函数调用
`k8s.io/kubernetes/pkg/controller/apis/config/v1alpha1/defaults.go`的`addDefaultingFuncs()`函数

```shell script
func init() {
	localSchemeBuilder.Register(addDefaultingFuncs)
}
```

4.(显式调用)`k8s.io/kubernetes/pkg/controller/apis/config/v1alpha1/defaults.go`的`addDefaultingFuncs()`函数
`k8s.io/kubernetes/pkg/controller/apis/config/v1alpha1/zz_generated.defaults.go`的`RegisterDefaults()`函数

```shell script
func addDefaultingFuncs(scheme *kruntime.Scheme) error {
	return RegisterDefaults(scheme)
}
```

5.(显式调用)执行默认值赋值流程
```shell script
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&v1alpha1.KubeControllerManagerConfiguration{}, func(obj interface{}) {
		SetObjectDefaults_KubeControllerManagerConfiguration(obj.(*v1alpha1.KubeControllerManagerConfiguration))
	})
	return nil
}

func SetObjectDefaults_KubeControllerManagerConfiguration(in *v1alpha1.KubeControllerManagerConfiguration) {
	SetDefaults_KubeControllerManagerConfiguration(in)
	SetDefaults_KubeCloudSharedConfiguration(&in.KubeCloudShared)
}
```

### 对象转换

结构体类型转换

```shell script
// kubectrlmgrconfigv1alpha1.KubeControllerManagerConfiguration{}为序列化结构体对象,主要用于赋值
// kubectrlmgrconfig.KubeControllerManagerConfiguration{}为普通结构体对象
// 将序列化类型结构体转化为普通结构体对象
internal := kubectrlmgrconfig.KubeControllerManagerConfiguration{}
if err := kubectrlmgrconfigscheme.Scheme.Convert(&versioned, &internal, nil); err != nil {
    return internal, err
}
```

### 赋值监听端口

10252
```shell script
internal.Generic.Port = insecurePort
```