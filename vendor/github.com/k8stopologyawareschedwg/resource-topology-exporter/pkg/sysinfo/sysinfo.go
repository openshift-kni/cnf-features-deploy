/*
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

package sysinfo

import (
	"fmt"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
)

const (
	SysDevicesNode = "/sys/devices/system/node"
)

const (
	HugepageSize2Mi = 2048
	HugepageSize1Gi = 1048576
)

type Handle struct {
	Root string
}

func (hnd Handle) SysDevicesNodes() string {
	return filepath.Join(hnd.Root, SysDevicesNode)
}

func (hnd Handle) SysDevicesNodesNodeNth(nodeID int) string {
	return filepath.Join(hnd.Root, SysDevicesNode, fmt.Sprintf("node%d", nodeID))
}

func GetMemoryResourceCounters(hnd Handle) (map[string]PerNUMACounters, error) {
	memResource := string(v1.ResourceMemory)
	numaCounters := map[string]PerNUMACounters{
		memResource: make(PerNUMACounters),
	}

	hugepages, err := GetHugepages(hnd)
	if err != nil {
		return numaCounters, err
	}

	for _, hpage := range hugepages {
		resourceName := HugepageResourceNameFromSize(hpage.SizeKB)
		numaDevs, ok := numaCounters[resourceName]
		if !ok {
			numaDevs = make(PerNUMACounters)
		}

		numaDevs[hpage.NodeID] = int64(hpage.Total * hpage.SizeKB * 1024)
		numaCounters[resourceName] = numaDevs
	}

	memory, err := GetMemory(hnd)
	if err != nil {
		return numaCounters, err
	}

	for numaID, value := range memory {
		numaCounters[memResource][numaID] = value
	}

	return numaCounters, nil
}
