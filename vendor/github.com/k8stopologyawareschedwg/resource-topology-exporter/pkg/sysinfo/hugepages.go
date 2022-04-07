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
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

// TODO review
type Hugepages struct {
	NodeID int
	SizeKB int
	Total  int
}

type PerNUMACounters map[int]int64

func GetHugepages(hnd Handle) ([]*Hugepages, error) {
	entries, err := ioutil.ReadDir(hnd.SysDevicesNodes())
	if err != nil {
		return nil, err
	}

	hugepages := []*Hugepages{}
	for _, entry := range entries {
		entryName := entry.Name()
		if entry.IsDir() && strings.HasPrefix(entryName, "node") {
			nodeID, err := strconv.Atoi(entryName[4:])
			if err != nil {
				klog.Warningf("cannot detect the node ID for %q", entryName)
				continue
			}
			nodeHugepages, err := HugepagesForNode(hnd, nodeID)
			if err != nil {
				klog.Warningf("cannot find the hugepages on NUMA node %d: %v", nodeID, err)
				continue
			}
			hugepages = append(hugepages, nodeHugepages...)
		}
	}
	return hugepages, nil
}

func HugepagesForNode(hnd Handle, nodeID int) ([]*Hugepages, error) {
	path := filepath.Join(
		hnd.SysDevicesNodesNodeNth(nodeID),
		"hugepages",
	)
	hugepages := []*Hugepages{}

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		entryName := entry.Name()
		entryPath := filepath.Join(path, entryName)
		var hugepageSizeKB int
		if n, err := fmt.Sscanf(entryName, "hugepages-%dkB", &hugepageSizeKB); n != 1 || err != nil {
			klog.Warningf("malformed hugepages entry %q", entryName)
			continue
		}

		totalCount, err := readIntFromFile(filepath.Join(entryPath, "nr_hugepages"))
		if err != nil {
			klog.Warningf("cannot read from %q: %v", entryPath, err)
			continue
		}

		hugepages = append(hugepages, &Hugepages{
			NodeID: nodeID,
			SizeKB: hugepageSizeKB,
			Total:  totalCount,
		})
	}

	return hugepages, nil
}

func readIntFromFile(path string) (int, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func HugepageResourceNameFromSize(sizeKB int) string {
	qty := resource.NewQuantity(int64(sizeKB*1024), resource.BinarySI)
	return "hugepages-" + qty.String()
}
