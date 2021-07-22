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

## 调用逻辑分析

> 1.运行`kube-controller-manager`服务

通过指定`flags`启动`kube-controller-manager`服务，例：

```shell script
--kube-api-qps=40 --kubeconfig=/root/.kube/config --leader-elect=true \
--node-cidr-mask-size=24 --service-cluster-ip-range=10.68.0.0/16 \
--use-service-account-credentials=true --v=0 --leader-elect=false \
--service-account-private-key-file=/etc/kubernetes/ca-key.pem
```

> 2.解析入参（`flags`），并赋予缺省值

> 3.开启服务监听

开启`kube-controller-manager`服务监听，包含： 

- `http`服务
- `https`服务

> 4.执行选主流程

> 5.启动子控制器

先启动`SA`控制器

再启动其他控制器

> 6.执行子控制器流程






