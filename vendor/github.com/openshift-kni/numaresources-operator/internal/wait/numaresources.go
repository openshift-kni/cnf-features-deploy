/*
 * Copyright 2022 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wait

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nropv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
)

func ForNUMAResourcesOperatorDeleted(cli client.Client, nrop *nropv1alpha1.NUMAResourcesOperator, pollInterval, pollTimeout time.Duration) error {
	err := wait.Poll(pollInterval, pollTimeout, func() (bool, error) {
		updatedNrop := nropv1alpha1.NUMAResourcesOperator{}
		key := ObjectKeyFromObject(nrop)
		err := cli.Get(context.TODO(), key.AsKey(), &updatedNrop)
		return deletionStatusFromError("NUMAResourcesOperator", key, err)
	})
	return err
}

func ForNUMAResourcesSchedulerDeleted(cli client.Client, nrSched *nropv1alpha1.NUMAResourcesScheduler, pollInterval, pollTimeout time.Duration) error {
	err := wait.Poll(pollInterval, pollTimeout, func() (bool, error) {
		updatedNROSched := nropv1alpha1.NUMAResourcesScheduler{}
		key := ObjectKeyFromObject(nrSched)
		err := cli.Get(context.TODO(), key.AsKey(), &updatedNROSched)
		return deletionStatusFromError("NUMAResourcesScheduler", key, err)
	})
	return err
}

func ForDaemonsetInNUMAResourcesOperatorStatus(cli client.Client, nroObj *nropv1alpha1.NUMAResourcesOperator, interval time.Duration, timeout time.Duration) (*nropv1alpha1.NUMAResourcesOperator, error) {
	updatedNRO := nropv1alpha1.NUMAResourcesOperator{}
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		key := ObjectKeyFromObject(nroObj)
		err := cli.Get(context.TODO(), key.AsKey(), &updatedNRO)
		if err != nil {
			klog.Warningf("failed to get the NUMAResourcesOperator %s: %v", key.String(), err)
			return false, err
		}

		if len(updatedNRO.Status.DaemonSets) == 0 {
			klog.Warningf("failed to get the DaemonSet from NUMAResourcesOperator %s", key.String())
			return false, nil
		}
		klog.Infof("Daemonset info %s/%s ready in NUMAResourcesOperator %s", updatedNRO.Status.DaemonSets[0].Namespace, updatedNRO.Status.DaemonSets[0].Name, key.String())
		return true, nil
	})
	return &updatedNRO, err
}
