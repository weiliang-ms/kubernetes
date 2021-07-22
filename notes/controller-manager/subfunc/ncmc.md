# NewControllerManagerCommand()
## 函数主体

```shell script
// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand() *cobra.Command {
	s, err := options.NewKubeControllerManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

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

	// 获取flags集合
	fs := cmd.Flags()
	//
	namedFlagSets := s.Flags(KnownControllers(), ControllersDisabledByDefault.List())

	verflag.AddFlags(namedFlagSets.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name())
	registerLegacyGlobalFlags(namedFlagSets)
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())

	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})

	// 设置帮助指令
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	return cmd
}
``` 

## 调用分析

> [1.初始化控制器](ncmc-ncmo.md)

初始化`kube-controller-manager`，附带默认配置（`flags`)

```shell script
s, err := options.NewKubeControllerManagerOptions()
```

> 2.命令参数绑定

```shell script
// 获取flags集合
fs := cmd.Flags()
//
namedFlagSets := s.Flags(KnownControllers(), ControllersDisabledByDefault.List())

verflag.AddFlags(namedFlagSets.FlagSet("global"))
globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name())
registerLegacyGlobalFlags(namedFlagSets)
for _, f := range namedFlagSets.FlagSets {
    fs.AddFlagSet(f)
}
usageFmt := "Usage:\n  %s\n"
cols, _, _ := term.TerminalSize(cmd.OutOrStdout())

cmd.SetUsageFunc(func(cmd *cobra.Command) error {
    fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
    cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
    return nil
})

// 设置帮助指令
cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
    cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
})
```

> 3.执行命令主体逻辑
```shell script
Run: func(cmd *cobra.Command, args []string) {
    // 输出版本
    verflag.PrintAndExitIfRequested()
    // 输出可选标识
    utilflag.PrintFlags(cmd.Flags())

    // 配置kube-controller-manager
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