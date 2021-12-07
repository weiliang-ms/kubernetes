// +build linux

/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oom

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	"github.com/google/cadvisor/utils/oomparser"
)

type streamer interface {
	StreamOoms(chan<- *oomparser.OomInstance)
}

var _ streamer = &oomparser.OomParser{}

type realWatcher struct {
	recorder    record.EventRecorder
	oomStreamer streamer
}

var _ Watcher = &realWatcher{}

// NewWatcher creates and initializes a OOMWatcher backed by Cadvisor as
// the oom streamer.
func NewWatcher(recorder record.EventRecorder) (Watcher, error) {
	oomStreamer, err := oomparser.New()
	if err != nil {
		return nil, err
	}

	watcher := &realWatcher{
		recorder:    recorder,
		oomStreamer: oomStreamer,
	}

	return watcher, nil
}

const (
	systemOOMEvent           = "SystemOOM"
	recordEventContainerName = "/"
)

// Start watches for system oom's and records an event for every system oom encountered.
func (ow *realWatcher) Start(ref *v1.ObjectReference) error {
	outStream := make(chan *oomparser.OomInstance, 10)

	// cAdvisor模块 从/dev/kmsg读取数据解析出oom容器信息：宿主机pid、时间戳、容器名称、容器内进程名称
	go ow.oomStreamer.StreamOoms(outStream)

	// 解析信息，记录到node event内
	// 通过kubectl describe node <node-name>查看
	/*
		Events:
	  Type     Reason     Age    From            Message
	  ----     ------     ----   ----            -------
	  Warning  SystemOOM  13m    kubelet, node1  System OOM encountered, victim process: nginx, pid: 60867
	  Warning  SystemOOM  13m    kubelet, node1  System OOM encountered, victim process: nginx, pid: 61438
	  Warning  SystemOOM  13m    kubelet, node1  System OOM encountered, victim process: nginx, pid: 61746
	  Warning  SystemOOM  12m    kubelet, node1  System OOM encountered, victim process: runc:[2:INIT], pid: 71899
	  Warning  SystemOOM  12m    kubelet, node1  System OOM encountered, victim process: nginx, pid: 82162
	  Warning  SystemOOM  11m    kubelet, node1  System OOM encountered, victim process: nginx, pid: 96195
	  Warning  SystemOOM  10m    kubelet, node1  System OOM encountered, victim process: runc:[2:INIT], pid: 12707
	  Warning  SystemOOM  7m33s  kubelet, node1  System OOM encountered, victim process: nginx, pid: 87657
	  Warning  SystemOOM  2m32s  kubelet, node1  System OOM encountered, victim process: runc:[2:INIT], pid: 98756
	*/
	go func() {
		defer runtime.HandleCrash()

		for event := range outStream {
			if event.ContainerName == recordEventContainerName {
				klog.V(1).Infof("Got sys oom event: %v", event)
				eventMsg := "System OOM encountered"
				if event.ProcessName != "" && event.Pid != 0 {
					eventMsg = fmt.Sprintf("%s, victim process: %s, pid: %d", eventMsg, event.ProcessName, event.Pid)
				}
				ow.recorder.Eventf(ref, v1.EventTypeWarning, systemOOMEvent, eventMsg)
			}
		}
		klog.Errorf("Unexpectedly stopped receiving OOM notifications")
	}()
	return nil
}
