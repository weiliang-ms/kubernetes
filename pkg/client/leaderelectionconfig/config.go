/*
Copyright 2017 The Kubernetes Authors.

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

package leaderelectionconfig

import (
	"github.com/spf13/pflag"
	componentbaseconfig "k8s.io/component-base/config"
)

// BindFlags binds the LeaderElectionConfiguration struct fields to a flagset
func BindFlags(l *componentbaseconfig.LeaderElectionConfiguration, fs *pflag.FlagSet) {
	// 是否选主，默认否
	fs.BoolVar(&l.LeaderElect, "leader-elect", l.LeaderElect, ""+
		"Start a leader election client and gain leadership before "+
		"executing the main loop. Enable this when running replicated "+
		"components for high availability.")
	// 当leader-elect设置为true生效，选举过程中非leader候选等待选举的时间间隔（default 15s）
	fs.DurationVar(&l.LeaseDuration.Duration, "leader-elect-lease-duration", l.LeaseDuration.Duration, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled.")
	// leader选举过程中在停止leading，再次renew时间间隔，小于或者等于leader-elect-lease-duration duration，
	// 也是leader-elect设置为true生效（默认10s）
	fs.DurationVar(&l.RenewDeadline.Duration, "leader-elect-renew-deadline", l.RenewDeadline.Duration, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled.")
	// 当leader-elect设置为true生效，获取leader或者重新选举的等待间隔（默认 2s）
	fs.DurationVar(&l.RetryPeriod.Duration, "leader-elect-retry-period", l.RetryPeriod.Duration, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled.")
	// 选主期间用于锁定的资源对象的类型，支持的选项是 endpoints 、configmaps、leases、endpointsleases(默认)、configmapsleases
	fs.StringVar(&l.ResourceLock, "leader-elect-resource-lock", l.ResourceLock, ""+
		"The type of resource object that is used for locking during "+
		"leader election. Supported options are 'endpoints', 'configmaps', "+
		"'leases', 'endpointsleases' and 'configmapsleases'.")
	// 选主期间用于锁定的资源对象的名称，默认kube-controller-manager
	fs.StringVar(&l.ResourceName, "leader-elect-resource-name", l.ResourceName, ""+
		"The name of resource object that is used for locking during "+
		"leader election.")
	// 选主期间用于锁定的命名空间的名称，默认kube-system
	fs.StringVar(&l.ResourceNamespace, "leader-elect-resource-namespace", l.ResourceNamespace, ""+
		"The namespace of resource object that is used for locking during "+
		"leader election.")
}
