/*
Copyright 2022 The Kubernetes Authors.

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

package wait

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ForDaemonSetReady(cli client.Client, ds *appsv1.DaemonSet, pollInterval, pollTimeout time.Duration) (*appsv1.DaemonSet, error) {
	updatedDs := &appsv1.DaemonSet{}
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		key := client.ObjectKeyFromObject(ds)
		err := cli.Get(context.TODO(), key, updatedDs)
		if err != nil {
			klog.Warningf("failed to get the daemonset %#v: %v", key, err)
			return false, err
		}

		if !AreDaemonSetPodsReady(&updatedDs.Status) {
			klog.Warningf("daemonset %#v desired %d scheduled %d ready %d",
				key,
				updatedDs.Status.DesiredNumberScheduled,
				updatedDs.Status.CurrentNumberScheduled,
				updatedDs.Status.NumberReady)
			return false, nil
		}

		klog.Infof("daemonset %#v ready", key)
		return true, nil
	})
	return updatedDs, err
}

func AreDaemonSetPodsReady(newStatus *appsv1.DaemonSetStatus) bool {
	return newStatus.DesiredNumberScheduled > 0 &&
		newStatus.DesiredNumberScheduled == newStatus.NumberReady
}
