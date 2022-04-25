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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

func ForMachineConfigPoolDeleted(cli client.Client, mcp *machineconfigv1.MachineConfigPool, pollInterval, pollTimeout time.Duration) error {
	err := wait.Poll(pollInterval, pollTimeout, func() (bool, error) {
		updatedMcp := machineconfigv1.MachineConfigPool{}
		key := client.ObjectKeyFromObject(mcp)
		err := cli.Get(context.TODO(), key, &updatedMcp)
		return deletionStatusFromError("MachineConfigPool", key, err)
	})
	return err
}

func ForKubeletConfigDeleted(cli client.Client, kc *machineconfigv1.KubeletConfig, pollInterval, pollTimeout time.Duration) error {
	err := wait.Poll(pollInterval, pollTimeout, func() (bool, error) {
		updatedKc := machineconfigv1.KubeletConfig{}
		key := client.ObjectKeyFromObject(kc)
		err := cli.Get(context.TODO(), key, &updatedKc)
		return deletionStatusFromError("KubeletConfig", key, err)
	})
	return err
}

func ForMachineConfigPoolCondition(cli client.Client, mcp *machineconfigv1.MachineConfigPool, condType machineconfigv1.MachineConfigPoolConditionType, pollInterval, pollTimeout time.Duration) error {
	err := wait.Poll(pollInterval, pollTimeout, func() (bool, error) {
		updatedMcp := machineconfigv1.MachineConfigPool{}
		key := client.ObjectKeyFromObject(mcp)
		err := cli.Get(context.TODO(), key, &updatedMcp)
		if err != nil {
			return false, err
		}
		for _, cond := range updatedMcp.Status.Conditions {
			if cond.Type == condType {
				if cond.Status == corev1.ConditionTrue {
					return true, nil
				} else {
					klog.Infof("mcp: %q condition type: %q status is: %q expected status: %q", updatedMcp.Name, cond.Type, cond.Status, corev1.ConditionTrue)
					return false, nil
				}
			}
		}
		klog.Infof("mcp: %q condition type: %q was not found", updatedMcp.Name, condType)
		return false, nil
	})
	return err
}
