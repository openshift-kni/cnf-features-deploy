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

func ForDeploymentComplete(cli client.Client, dp *appsv1.Deployment, pollInterval, pollTimeout time.Duration) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		updatedDp := &appsv1.Deployment{}
		key := client.ObjectKeyFromObject(dp)
		err := cli.Get(context.TODO(), key, updatedDp)
		if err != nil {
			klog.Warningf("failed to get the deployment %#v: %v", key, err)
			return false, err
		}

		if !IsDeploymentComplete(dp, &updatedDp.Status) {
			klog.Warningf("deployment %#v not yet complete", key)
			return false, nil
		}

		klog.Infof("deployment %#v complete", key)
		return true, nil
	})
}

func AreDeploymentReplicasAvailable(newStatus *appsv1.DeploymentStatus, replicas int32) bool {
	return newStatus.UpdatedReplicas == replicas &&
		newStatus.Replicas == replicas &&
		newStatus.AvailableReplicas == replicas
}

func IsDeploymentComplete(dp *appsv1.Deployment, newStatus *appsv1.DeploymentStatus) bool {
	return AreDeploymentReplicasAvailable(newStatus, *(dp.Spec.Replicas)) &&
		newStatus.ObservedGeneration >= dp.Generation
}
