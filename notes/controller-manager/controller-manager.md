# kube-controller-manager

## 简介

`Controller Manager`是`Kubernetes`的核心组件之一。
`Kubernetes`对集群的管理采用的是控制器模式，即针对各种资源运行多个`controller`（控制器）。
控制器的逻辑是运行永不结束的循环，通过`apiserver`组件时刻获取集群某种资源的状态，并确保资源的当前状态与期望的状态相符合。

## 源码分析

### 整体调用逻辑

> 主函数`controller-manager.go`

实质为初始化[cobra](https://github.com/spf13/cobra) 类型一条命令执行，并记录日志

    func main() {
        // 生成随机数
    	rand.Seed(time.Now().UnixNano())
    
    	command := app.NewControllerManagerCommand()
    
    	// TODO: once we switch everything over to Cobra commands, we can go back to calling
    	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
    	// normalize func and add the go flag set by hand.
    	// utilflag.InitFlags()
    	logs.InitLogs()
    	defer logs.FlushLogs()
    
    	if err := command.Execute(); err != nil {
    		os.Exit(1)
    	}
    }

> `NewControllerManagerCommand()`方法实体分析


    func NewControllerManagerCommand() *cobra.Command {
        // 1.初始化一份KubeControllerManager默认配置，包含：
            // a.服务监听端口
            // b.子控制器（job-controller等）默认配置
            // c.开启认证、鉴权，针对'/healthz' path允许访问
            // d.存在内存中的ca证书
            // e.配置GarbageCollectorController的gc忽略`events`资源对象（{Group: "", Resource: "events"}: {}）
            // f.配置选主资源对象：kube-controller-manager，选主时kube-system命名空间及kube-controller-manager服务加锁
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
    			verflag.PrintAndExitIfRequested()
    			utilflag.PrintFlags(cmd.Flags())
    
    			c, err := s.Config(KnownControllers(), ControllersDisabledByDefault.List())
    			if err != nil {
    				fmt.Fprintf(os.Stderr, "%v\n", err)
    				os.Exit(1)
    			}
    
    			if err := Run(c.Complete(), wait.NeverStop); err != nil {
    				fmt.Fprintf(os.Stderr, "%v\n", err)
    				os.Exit(1)
    			}
    		},
    	}
    
    	fs := cmd.Flags()
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
    	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
    		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
    	})
    
    	return cmd
    }

### 
