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

	"k8s.io/klog/v2"
)

func GetMemory(hnd Handle) (map[int]int64, error) {
	entries, err := ioutil.ReadDir(hnd.SysDevicesNodes())
	if err != nil {
		return nil, err
	}

	memory := map[int]int64{}
	for _, entry := range entries {
		entryName := entry.Name()
		if entry.IsDir() && strings.HasPrefix(entryName, "node") {
			nodeID, err := strconv.Atoi(entryName[4:])
			if err != nil {
				klog.Warningf("cannot detect the node ID for %q", entryName)
				continue
			}
			nodeMemory, err := MemoryForNode(hnd, nodeID)
			if err != nil {
				klog.Warningf("cannot find the memory on NUMA node %d: %v", nodeID, err)
				continue
			}
			memory[nodeID] = nodeMemory
		}
	}
	return memory, nil
}

func MemoryForNode(hnd Handle, nodeID int) (int64, error) {
	path := filepath.Join(
		hnd.SysDevicesNodesNodeNth(nodeID),
		"meminfo",
	)

	value, err := readTotalMemoryFromMeminfo(path)
	if err != nil {
		return -1, err
	}

	return value, nil
}

func readTotalMemoryFromMeminfo(path string) (int64, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return -1, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, "MemTotal") {
			continue
		}

		memTotal := strings.Split(line, ":")
		if len(memTotal) != 2 {
			return -1, fmt.Errorf("MemTotal has unexpected format: %s", line)
		}

		memValue := strings.Trim(memTotal[1], "\t\n kB")
		convertedValue, err := strconv.ParseInt(memValue, 10, 64)
		if err != nil {
			return -1, fmt.Errorf("failed to convert value: %v", memValue)
		}

		return 1024 * convertedValue, nil
	}

	return -1, fmt.Errorf("failed to find MemTotal field under the file %q", path)
}
