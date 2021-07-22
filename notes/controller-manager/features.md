# 特性列表

基于`v.18.6`

## 默认开启

- `BETA`
    -`APIListChunking`: 启用`API`客户端以块的形式从`API`服务器检索（`LIST`或`GET`）资源
    -`APIResponseCompression`: 压缩`LIST`或`GET`请求的`API`响应
    - `AllowInsecureBackendProxy`: 当尝试获取`Pod`的日志时，`Kubelet`可能有一个过期的服务证书。开启该特性配置，允许最终用户可以绕过`Kubernetes API Server`的默认行为，并跳过`Kubelet`的`TLS`验证来收集日志。
    - 

## 默认关闭

- `ALPHA`
    - `APIPriorityAndFairness`: 在每个服务器上启用优先级和公平性来管理请求并发
    - `AllAlpha`: 开启所有`Alpha`阶段特性
    - `AllBeta`: 开启所有`Beta`阶段特性

```shell script
AnyVolumeDataSource=true|false (ALPHA - default=false)
AppArmor=true|false (BETA - default=true)
BalanceAttachedNodeVolumes=true|false (ALPHA - default=false)
BoundServiceAccountTokenVolume=true|false (ALPHA - default=false)
CPUManager=true|false (BETA - default=true)
CRIContainerLogRotation=true|false (BETA - default=true)
CSIInlineVolume=true|false (BETA - default=true)
CSIMigration=true|false (BETA - default=true)
CSIMigrationAWS=true|false (BETA - default=false)
CSIMigrationAWSComplete=true|false (ALPHA - default=false)
CSIMigrationAzureDisk=true|false (ALPHA - default=false)
CSIMigrationAzureDiskComplete=true|false (ALPHA - default=false)
CSIMigrationAzureFile=true|false (ALPHA - default=false)
CSIMigrationAzureFileComplete=true|false (ALPHA - default=false)
CSIMigrationGCE=true|false (BETA - default=false)
CSIMigrationGCEComplete=true|false (ALPHA - default=false)
CSIMigrationOpenStack=true|false (BETA - default=false)
CSIMigrationOpenStackComplete=true|false (ALPHA - default=false)
ConfigurableFSGroupPolicy=true|false (ALPHA - default=false)
CustomCPUCFSQuotaPeriod=true|false (ALPHA - default=false)
DefaultIngressClass=true|false (BETA - default=true)
DevicePlugins=true|false (BETA - default=true)
DryRun=true|false (BETA - default=true)
DynamicAuditing=true|false (ALPHA - default=false)
DynamicKubeletConfig=true|false (BETA - default=true)
EndpointSlice=true|false (BETA - default=true)
EndpointSliceProxying=true|false (ALPHA - default=false)
EphemeralContainers=true|false (ALPHA - default=false)
EvenPodsSpread=true|false (BETA - default=true)
ExpandCSIVolumes=true|false (BETA - default=true)
ExpandInUsePersistentVolumes=true|false (BETA - default=true)
ExpandPersistentVolumes=true|false (BETA - default=true)
ExperimentalHostUserNamespaceDefaulting=true|false (BETA - default=false)
HPAScaleToZero=true|false (ALPHA - default=false)
HugePageStorageMediumSize=true|false (ALPHA - default=false)
HyperVContainer=true|false (ALPHA - default=false)
IPv6DualStack=true|false (ALPHA - default=false)
ImmutableEphemeralVolumes=true|false (ALPHA - default=false)
KubeletPodResources=true|false (BETA - default=true)
LegacyNodeRoleBehavior=true|false (ALPHA - default=true)
LocalStorageCapacityIsolation=true|false (BETA - default=true)
LocalStorageCapacityIsolationFSQuotaMonitoring=true|false (ALPHA - default=false)
NodeDisruptionExclusion=true|false (ALPHA - default=false)
NonPreemptingPriority=true|false (ALPHA - default=false)
PodDisruptionBudget=true|false (BETA - default=true)
PodOverhead=true|false (BETA - default=true)
ProcMountType=true|false (ALPHA - default=false)
QOSReserved=true|false (ALPHA - default=false)
RemainingItemCount=true|false (BETA - default=true)
RemoveSelfLink=true|false (ALPHA - default=false)
ResourceLimitsPriorityFunction=true|false (ALPHA - default=false)
RotateKubeletClientCertificate=true|false (BETA - default=true)
RotateKubeletServerCertificate=true|false (BETA - default=true)
RunAsGroup=true|false (BETA - default=true)
RuntimeClass=true|false (BETA - default=true)
SCTPSupport=true|false (ALPHA - default=false)
SelectorIndex=true|false (ALPHA - default=false)
ServerSideApply=true|false (BETA - default=true)
ServiceAccountIssuerDiscovery=true|false (ALPHA - default=false)
ServiceAppProtocol=true|false (ALPHA - default=false)
ServiceNodeExclusion=true|false (ALPHA - default=false)
ServiceTopology=true|false (ALPHA - default=false)
StartupProbe=true|false (BETA - default=true)
StorageVersionHash=true|false (BETA - default=true)
SupportNodePidsLimit=true|false (BETA - default=true)
SupportPodPidsLimit=true|false (BETA - default=true)
Sysctls=true|false (BETA - default=true)
TTLAfterFinished=true|false (ALPHA - default=false)
TokenRequest=true|false (BETA - default=true)
TokenRequestProjection=true|false (BETA - default=true)
TopologyManager=true|false (BETA - default=true)
ValidateProxyRedirects=true|false (BETA - default=true)
VolumeSnapshotDataSource=true|false (BETA - default=true)
WinDSR=true|false (ALPHA - default=false)
WinOverlay=true|false (ALPHA - default=false)
```