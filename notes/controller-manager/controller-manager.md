# kube-controller-manager

## 简介

`Controller Manager`是`Kubernetes`的核心组件之一。
`Kubernetes`对集群的管理采用的是控制器模式，即针对各种资源运行多个`controller`（控制器）。
控制器的逻辑是运行永不结束的循环，通过`apiserver`组件时刻获取集群某种资源的状态，并确保资源的当前状态与期望的状态相符合。

## 整体调用逻辑

> 主函数内容

```shell script
package main

import (
    "math/rand"
    "os"
    "time"

    "k8s.io/component-base/logs"
    _ "k8s.io/component-base/metrics/prometheus/clientgo" // load all the prometheus client-go plugin
    _ "k8s.io/component-base/metrics/prometheus/version"  // for version metric registration
    "k8s.io/kubernetes/cmd/kube-controller-manager/app"
)

func main() {
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
```

> 初始化随机数种子

设置随机数种子，加上这行代码，可以保证每次随机都是随机的

```shell script
rand.Seed(time.Now().UnixNano())
```

> 初始化`ControllerManagerCommand`

`ControllerManagerCommand`为核心命令，通过`flags`入参配置`controller-manager`服务

```shell script
command := app.NewControllerManagerCommand()
```

> 初始化日志

```shell script
logs.InitLogs()
defer logs.FlushLogs()
```

> 运行`ControllerManagerCommand`，执行启动逻辑

```shell script
if err := command.Execute(); err != nil {
    os.Exit(1)
}
```

## 初始化`ControllerManagerCommand`逻辑

### 初始化`kube-controller-manager`默认配置





