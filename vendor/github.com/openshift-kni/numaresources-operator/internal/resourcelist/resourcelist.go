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
