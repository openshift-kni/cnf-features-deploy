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

func FromGuaranteedPod(pod corev1.Pod) corev1.ResourceList {
	res := make(corev1.ResourceList)
	for idx := 0; idx < len(pod.Spec.Containers); idx++ {
		cnt := &pod.Spec.Containers[idx] // shortcut
		for resName, resQty := range cnt.Resources.Limits {
			qty := res[resName]
			qty.Add(resQty)
			res[resName] = qty
		}
	}
	return res
}

func AddCoreResources(res corev1.ResourceList, cpu, mem resource.Quantity) {
	adjustedCPU := res.Cpu()
	adjustedCPU.Add(cpu)
	res[corev1.ResourceCPU] = *adjustedCPU

	adjustedMemory := res.Memory()
	adjustedMemory.Add(mem)
	res[corev1.ResourceMemory] = *adjustedMemory
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
