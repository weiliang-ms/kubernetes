## gernic类型

    // 是否应在云提供商上分配和设置Pod的CIDR
    --allocate-node-cidrs
    // CIDR分配器的类型 (default "RangeAllocator")
    --cidr-allocator-type string
            Type of CIDR allocator to use (default "RangeAllocator")
    --cloud-config string
            The path to the cloud provider configuration file. Empty string for no configuration file.
    --cloud-provider string
            The provider for cloud services. Empty string for no provider.
    --cluster-cidr string
            CIDR Range for Pods in cluster. Requires --allocate-node-cidrs to be true
    --cluster-name string
            The instance prefix for the cluster. (default "kubernetes")
    --configure-cloud-routes
            Should CIDRs allocated by allocate-node-cidrs be configured on the cloud provider. (default true)
    --controller-start-interval duration
            Interval between starting controller managers.
    --controllers strings
            A list of controllers to enable. '*' enables all on-by-default controllers, 'foo' enables the controller named 'foo', '-foo' disables the
            controller named 'foo'.
            All controllers: attachdetach, bootstrapsigner, cloud-node-lifecycle, clusterrole-aggregation, cronjob, csrapproving, csrcleaner,
            csrsigning, daemonset, deployment, disruption, endpoint, endpointslice, garbagecollector, horizontalpodautoscaling, job, namespace,
            nodeipam, nodelifecycle, persistentvolume-binder, persistentvolume-expander, podgc, pv-protection, pvc-protection, replicaset,
            replicationcontroller, resourcequota, root-ca-cert-publisher, route, service, serviceaccount, serviceaccount-token, statefulset,
            tokencleaner, ttl, ttl-after-finished
            Disabled-by-default controllers: bootstrapsigner, tokencleaner (default [*])
    --external-cloud-volume-plugin string
            The plugin to use when cloud provider is set to external. Can be empty, should only be set when cloud-provider is external. Currently
            used to allow node and volume controllers to work for in tree cloud providers.
    --feature-gates mapStringBool
            A set of key=value pairs that describe feature gates for alpha/experimental features. Options are:
            APIListChunking=true|false (BETA - default=true)
            APIPriorityAndFairness=true|false (ALPHA - default=false)
            APIResponseCompression=true|false (BETA - default=true)
            AllAlpha=true|false (ALPHA - default=false)
            AllBeta=true|false (BETA - default=false)
            AllowInsecureBackendProxy=true|false (BETA - default=true)
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
    --kube-api-burst int32
            Burst to use while talking with kubernetes apiserver. (default 30)
    --kube-api-content-type string
            Content type of requests sent to apiserver. (default "application/vnd.kubernetes.protobuf")
    --kube-api-qps float32
            QPS to use while talking with kubernetes apiserver. (default 20)
    --leader-elect
            Start a leader election client and gain leadership before executing the main loop. Enable this when running replicated components for
            high availability. (default true)
    --leader-elect-lease-duration duration
            The duration that non-leader candidates will wait after observing a leadership renewal until attempting to acquire leadership of a led
            but unrenewed leader slot. This is effectively the maximum duration that a leader can be stopped before it is replaced by another
            candidate. This is only applicable if leader election is enabled. (default 15s)
    --leader-elect-renew-deadline duration
            The interval between attempts by the acting master to renew a leadership slot before it stops leading. This must be less than or equal to
            the lease duration. This is only applicable if leader election is enabled. (default 10s)
    --leader-elect-resource-lock endpoints
            The type of resource object that is used for locking during leader election. Supported options are endpoints (default) and `configmaps`.
            (default "endpointsleases")
    --leader-elect-resource-name string
            The name of resource object that is used for locking during leader election. (default "kube-controller-manager")
    --leader-elect-resource-namespace string
            The namespace of resource object that is used for locking during leader election. (default "kube-system")
    --leader-elect-retry-period duration
            The duration the clients should wait between attempting acquisition and renewal of a leadership. This is only applicable if leader
            election is enabled. (default 2s)
    --min-resync-period duration
            The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod. (default 12h0m0s)
    --node-monitor-period duration
            The period for syncing NodeStatus in NodeController. (default 5s)
    --route-reconciliation-period duration
            The period for reconciling routes created for Nodes by cloud provider. (default 10s)
    --use-service-account-credentials
            If true, use individual service account credentials for each controller.

