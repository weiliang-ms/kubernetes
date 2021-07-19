# 初始化kube-controller-manager指令

## 函数主体

```shell script
  cmd := &cobra.Command{
		Use: "kube-controller-manager",
		Long: `The Kubernetes controller manager is a daemon that embeds
the core control loops shipped with Kubernetes. In applications of robotics and
automation, a control loop is a non-terminating loop that regulates the state of
the system. In Kubernetes, a controller is a control loop that watches the shared
state of the cluster through the apiserver and makes changes attempting to move the
current state towards the desired state. Examples of controllers that ship with
Kubernetes today are the replication controller, endpoints controller, namespace
controller, and serviceaccounts controller.`,
		Run: func(cmd *cobra.Command, args []string) {
			// 输出版本
			verflag.PrintAndExitIfRequested()
			// 输出可选标识
			utilflag.PrintFlags(cmd.Flags())

			// 配置kube-controller-manager
			/*
				// 返回控制器名称列表，并排序
				KnownControllers() -> []string{"etcd","namespace",...}
			*/
			c, err := s.Config(KnownControllers(), ControllersDisabledByDefault.List())
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			// c.Complete() -> api-server对控制器服务认证授权
			// 执行控制器启动流程
			if err := Run(c.Complete(), wait.NeverStop); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
```

## Run方法解析

```shell script
Run: func(cmd *cobra.Command, args []string) {
			// 输出版本
			verflag.PrintAndExitIfRequested()
			// 输出可选标识
			utilflag.PrintFlags(cmd.Flags())

			// 配置kube-controller-manager
			/*
				// 返回控制器名称列表，并排序
				KnownControllers() -> []string{"etcd","namespace",...}
			*/
			c, err := s.Config(KnownControllers(), ControllersDisabledByDefault.List())
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			// c.Complete() -> api-server对控制器服务认证授权
			// 执行控制器启动流程
			if err := Run(c.Complete(), wait.NeverStop); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
```

> 1.
> 2.
> 3.配置控制器

```shell script
c, err := s.Config(KnownControllers(), ControllersDisabledByDefault.List())
```