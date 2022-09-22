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

package resourcelist

import (
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ToString(res corev1.ResourceList) string {
	idx := 0
	resNames := make([]string, len(res))
	for resName := range res {
		resNames[idx] = string(resName)
		idx++
	}
	sort.Strings(resNames)

	items := []string{}
	for _, resName := range resNames {
		resQty := res[corev1.ResourceName(resName)]
		items = append(items, fmt.Sprintf("%s=%s", resName, resQty.String()))
	}
	return strings.Join(items, ", ")
}

func FromReplicaSet(rs appsv1.ReplicaSet) corev1.ResourceList {
	rl := FromContainers(rs.Spec.Template.Spec.Containers)
	replicas := rs.Spec.Replicas
	for resName, resQty := range rl {
		replicaResQty := resQty.DeepCopy()
		// index begins from 1 because we already have resources of one replica
		for i := 1; i < int(*replicas); i++ {
			resQty.Add(replicaResQty)
		}
		rl[resName] = resQty
	}
	return rl
}

func FromGuaranteedPod(pod corev1.Pod) corev1.ResourceList {
	return FromContainers(pod.Spec.Containers)
}

func FromContainers(containers []corev1.Container) corev1.ResourceList {
	res := make(corev1.ResourceList)
	for idx := 0; idx < len(containers); idx++ {
		cnt := &containers[idx] // shortcut
		for resName, resQty := range cnt.Resources.Limits {
			qty := res[resName]
			qty.Add(resQty)
			res[resName] = qty
		}
	}
	return res
}

func AddCoreResources(res, resToAdd corev1.ResourceList) {
	for resName, resQty := range resToAdd {
		qty := res[resName]
		qty.Add(resQty)
		res[resName] = qty
	}
}

func SubCoreResources(res, resToSub corev1.ResourceList) error {
	for resName, resQty := range resToSub {
		if resQty.Cmp(res[resName]) > 0 {
			return fmt.Errorf("cannot substract resource %q because it is not found in the current resources", resName)
		}
		qty := res[resName]
		qty.Sub(resQty)
		res[resName] = qty
	}
	return nil
}

func RoundUpCoreResources(cpu, mem resource.Quantity) (resource.Quantity, resource.Quantity) {
	retCpu := *resource.NewQuantity(roundUp(cpu.Value(), 2), resource.DecimalSI)
	retMem := mem.DeepCopy() // TODO: this is out of over caution
	// FIXME: this rounds to G (1000) not to Gi (1024) which works but is not what we intended
	retMem.RoundUp(resource.Giga)
	return retCpu, retMem
}

func roundUp(num, multiple int64) int64 {
	return ((num + multiple - 1) / multiple) * multiple
}
