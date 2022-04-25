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

package baseload

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/openshift-kni/numaresources-operator/internal/resourcelist"
)

// we don't use corev1.ResourceList because we need values for CPU and Memory
// so we can just use a struct at this point
type Load struct {
	Name   string
	CPU    resource.Quantity
	Memory resource.Quantity
}

func FromPods(nodeName string, pods []corev1.Pod) Load {
	nl := Load{
		Name: nodeName,
	}

	cpu := &resource.Quantity{}
	mem := &resource.Quantity{}

	for _, pod := range pods {
		// TODO: we assume a steady state - aka we ignore InitContainers
		for _, cnt := range pod.Spec.Containers {
			for resName, resQty := range cnt.Resources.Requests {
				switch resName {
				case corev1.ResourceCPU:
					cpu.Add(resQty)
				case corev1.ResourceMemory:
					mem.Add(resQty)
				}
			}
		}
	}

	nl.CPU = *cpu
	nl.Memory = *mem
	return nl.Round()
}

func (nl Load) Round() Load {
	// get full cpus, and always take even number of CPUs
	// we round the CPU consumption as expressed in millicores (not entire cores)
	// in order to (try to) avoid bugs related to integer division
	// int64(2900 / 1000) -> 2 -> roundUp(2, 2) -> 2 (correct, but unexpected!)
	// OTOH
	// roundUp(2900, 2000) -> 4000 -> 4000/1000 -> 4 (intended behaviour).
	// Value() round up the millis and roundUp rounds it up to multiples of 2 if needed.
	cpu, mem := resourcelist.RoundUpCoreResources(nl.CPU, nl.Memory)
	return Load{
		Name:   nl.Name,
		CPU:    cpu,
		Memory: mem,
	}
}

func (nl Load) String() string {
	return fmt.Sprintf("load for node %q: CPU=%s Memory=%s", nl.Name, nl.CPU.String(), nl.Memory.String())
}

// Apply adjust the given ResourceList with the current node load by mutating
// the parameter in place
func (nl Load) Apply(res corev1.ResourceList) {
	resourcelist.AddCoreResources(res, nl.CPU, nl.Memory)
}

// Deduct subtract the current node load from the given ResourceList by mutating
// the parameter in place
func (nl Load) Deduct(res corev1.ResourceList) {
	adjustedCPU := res.Cpu()
	adjustedCPU.Sub(nl.CPU)
	res[corev1.ResourceCPU] = *adjustedCPU

	adjustedMemory := res.Memory()
	adjustedMemory.Sub(nl.Memory)
	res[corev1.ResourceMemory] = *adjustedMemory
}
