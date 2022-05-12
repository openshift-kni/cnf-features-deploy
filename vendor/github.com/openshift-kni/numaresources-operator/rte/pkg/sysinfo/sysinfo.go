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
	"sort"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/jaypipes/ghw/pkg/pci"
	"github.com/jaypipes/ghw/pkg/topology"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	rtesysinfo "github.com/k8stopologyawareschedwg/resource-topology-exporter/pkg/sysinfo"
)

const (
	SysDevicesOnlineCPUs = "/sys/devices/system/cpu/online"
)

type Config struct {
	ReservedCPUs string `json:"reservedCpus,omitempty"`
	// vendor:device -> resourcename
	ResourceMapping map[string]string `json:"resourceMapping,omitempty"`
	// numa zone -> reserved amount
	ReservedMemory map[int]int64 `json:"reservedMemory,omitempty"`
}

func (cfg Config) ToYAML() ([]byte, error) {
	return yaml.Marshal(cfg)
}

func ResourceMappingFromString(s string) map[string]string {
	// comma-separated 'vendor:device=resourcename'
	rmap := make(map[string]string)
	for _, keyvalue := range strings.Split(strings.TrimSpace(s), ",") {
		if len(keyvalue) == 0 {
			continue
		}
		items := strings.SplitN(keyvalue, "=", 2)
		if len(items) != 2 {
			klog.Warningf("malformed resource mapping item %q, skipped", keyvalue)
			continue
		}
		rmap[strings.TrimSpace(items[0])] = strings.TrimSpace(items[1])
	}
	return rmap
}

func ResourceMappingToString(rmap map[string]string) string {
	var keys []string
	for key := range rmap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var items []string
	for _, key := range keys {
		items = append(items, fmt.Sprintf("%s=%s", key, rmap[key]))
	}
	return strings.Join(items, ",")
}

func ReservedMemoryFromString(s string) map[int]int64 {
	// comma-separated 'numaID=amount'")
	rmap := make(map[int]int64)
	for _, keyvalue := range strings.Split(strings.TrimSpace(s), ",") {
		if len(keyvalue) == 0 {
			continue
		}
		items := strings.SplitN(keyvalue, "=", 2)
		if len(items) != 2 {
			klog.Warningf("malformed resource mapping item %q, skipped", keyvalue)
			continue
		}
		numaID, err := strconv.Atoi(strings.TrimSpace(items[0]))
		if err != nil {
			klog.Warningf("cannot parse NUMA identifier %q: %v - skipped", items[0], err)
			continue
		}

		res, err := resource.ParseQuantity(strings.TrimSpace(items[1]))
		if err != nil {
			klog.Warningf("cannot parse NUMA memory amount %q: %v - skipped", items[1], err)
			continue
		}
		val, ok := res.AsInt64()
		if !ok {
			klog.Warningf("NUMA memory amount %q representation error: %v - skipped", items[1], err)
			continue
		}
		rmap[numaID] = val
	}
	return rmap
}

func ReservedMemoryToString(rmap map[int]int64) string {
	var keys []int
	for key := range rmap {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	var items []string
	for _, key := range keys {
		items = append(items, fmt.Sprintf("%d=%d", key, rmap[key]))
	}
	return strings.Join(items, ",")
}

func (cfg Config) IsEmpty() bool {
	return cfg.ReservedCPUs == "" && len(cfg.ResourceMapping) == 0
}

func (cfg Config) ToYAMLString() string {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "<MALFORMED>"
	}
	return string(data)
}

// NUMA Cell -> deviceIDs
type PerNUMADevices map[int][]string

// NUMA Cell -> counter
type PerNUMACounters map[int]int64

type SysInfo struct {
	CPUs cpuset.CPUSet
	// resource name -> devices
	Resources map[string]PerNUMADevices
	// memory type -> counters
	Memory map[string]PerNUMACounters
}

func magnitude(order int) string {
	orders := []string{
		"Ki",
		"Mi",
		"Gi",
		"Ti",
		"Pi",
		"Ei",
	}
	if order < 0 || order >= len(orders) {
		return ""
	}
	return orders[order]
}

func FormatSize(v int64) string {
	var k int64 = 1024
	if v < k {
		return fmt.Sprintf("%d", v)
	}
	m := 0
	n := v / k
	for n >= k {
		n /= k
		k *= k
		m++
	}
	return fmt.Sprintf("%d%s", n, magnitude(m))
}

func (si SysInfo) String() string {
	b := strings.Builder{}
	fmt.Fprintf(&b, "cpus: allocatable %q\n", si.CPUs.String())
	for memoryType, numaMem := range si.Memory {
		fmt.Fprintf(&b, "%s:\n", memoryType)
		for numaNode, amount := range numaMem {
			fmt.Fprintf(&b, "  numa cell %d -> %s\n", numaNode, FormatSize(amount))
		}
	}
	for resourceName, numaDevs := range si.Resources {
		fmt.Fprintf(&b, "resource %q:\n", resourceName)
		for numaNode, devs := range numaDevs {
			fmt.Fprintf(&b, "  numa cell %d -> %v\n", numaNode, devs)
		}
	}
	return b.String()
}

func NewSysinfo(conf Config) (SysInfo, error) {
	var err error
	var sysinfo SysInfo

	sysinfo.CPUs, err = GetCPUResources(conf.ReservedCPUs, GetOnlineCPUs)
	if err != nil {
		return sysinfo, err
	}
	if sysinfo.CPUs.Size() == 0 {
		return sysinfo, fmt.Errorf("no allocatable cpus")
	}

	sysinfo.Resources, err = GetPCIResources(conf.ResourceMapping, GetPCIDevices)
	if err != nil {
		return sysinfo, err
	}

	sysinfo.Memory, err = GetMemoryResources(conf.ReservedMemory, GetAvailableMemory)
	if err != nil {
		return sysinfo, err
	}
	return sysinfo, nil
}

func GetCPUResources(resCPUs string, getCPUs func() (cpuset.CPUSet, error)) (cpuset.CPUSet, error) {
	reservedCPUs, err := cpuset.Parse(resCPUs)
	if err != nil {
		return cpuset.CPUSet{}, err
	}
	klog.Infof("cpus: reserved %q", reservedCPUs.String())

	cpus, err := getCPUs()
	if err != nil {
		return cpuset.CPUSet{}, err
	}
	klog.Infof("cpus: online %q", cpus.String())

	return cpus.Difference(reservedCPUs), nil
}

func GetPCIResources(resourceMap map[string]string, getPCIs func() ([]*pci.Device, error)) (map[string]PerNUMADevices, error) {
	numaResources := make(map[string]PerNUMADevices)
	devices, err := getPCIs()
	if err != nil {
		return numaResources, err
	}

	for _, dev := range devices {
		resourceName, ok := ResourceNameForDevice(dev, resourceMap)
		if !ok {
			continue
		}

		numaDevs, ok := numaResources[resourceName]
		if !ok {
			numaDevs = make(PerNUMADevices)
		}

		nodeID := -1
		if dev.Node != nil {
			nodeID = dev.Node.ID
		}
		numaDevs[nodeID] = append(numaDevs[nodeID], dev.Address)
		numaResources[resourceName] = numaDevs
	}

	return numaResources, nil
}

// TODO: support hugepages reservation
func GetMemoryResources(reservedMemory map[int]int64, getAvailableMemory func() ([]*topology.Node, []*rtesysinfo.Hugepages, error)) (map[string]PerNUMACounters, error) {
	numaMemory := make(map[string]PerNUMACounters)
	nodes, hugepages, err := getAvailableMemory()
	if err != nil {
		return numaMemory, err
	}

	counters := make(PerNUMACounters)
	for _, node := range nodes {
		counters[node.ID] += node.Memory.TotalUsableBytes
	}
	memCounters := make(PerNUMACounters)
	for numaID, amount := range counters {
		reserved := reservedMemory[numaID]
		if reserved > amount {
			// TODO log
			memCounters[numaID] = 0
		}
		memCounters[numaID] = amount - reserved
	}
	numaMemory[string(corev1.ResourceMemory)] = memCounters

	for _, hp := range hugepages {
		name := rtesysinfo.HugepageResourceNameFromSize(hp.SizeKB)
		hpCounters, ok := numaMemory[name]
		if !ok {
			hpCounters = make(PerNUMACounters)
		}
		hpCounters[hp.NodeID] += int64(hp.Total)
		numaMemory[name] = hpCounters
	}

	return numaMemory, nil
}

func ResourceNameForDevice(dev *pci.Device, resourceMap map[string]string) (string, bool) {
	devID := fmt.Sprintf("%s:%s", dev.Vendor.ID, dev.Product.ID)
	if resourceName, ok := resourceMap[devID]; ok {
		klog.Infof("devs: resource for %s is %q", devID, resourceName)
		return resourceName, true
	}
	if resourceName, ok := resourceMap[dev.Vendor.ID]; ok {
		klog.Infof("devs: resource for %s is %q", dev.Vendor.ID, resourceName)
		return resourceName, true
	}
	return "", false
}

func GetOnlineCPUs() (cpuset.CPUSet, error) {
	data, err := ioutil.ReadFile(SysDevicesOnlineCPUs)
	if err != nil {
		return cpuset.CPUSet{}, err
	}
	cpus := strings.TrimSpace(string(data))
	return cpuset.Parse(cpus)
}

func GetPCIDevices() ([]*pci.Device, error) {
	info, err := pci.New()
	if err != nil {
		return nil, err
	}
	return info.Devices, nil
}

func GetAvailableMemory() ([]*topology.Node, []*rtesysinfo.Hugepages, error) {
	hugepages, err := rtesysinfo.GetHugepages(rtesysinfo.Handle{})
	if err != nil {
		return nil, nil, err
	}
	info, err := topology.New()
	if err != nil {
		return nil, nil, err
	}
	return info.Nodes, hugepages, nil
}
