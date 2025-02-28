// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// derived from
//   https://github.com/google/cadvisor/blob/master/utils/sysfs/sysfs.go @ ef7e64f9
// updated for cache info detection from
//   https://github.com/google/cadvisor/blob/master/utils/sysfs/sysfs.go @ v0.49.1
// as Apache 2.0 license allows.

package machine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
	"k8s.io/utils/cpuset"

	"github.com/google/cadvisor/utils/sysfs"
)

const (
	blockDir     = "/sys/block"
	cacheDir     = "/sys/devices/system/cpu/cpu"
	netDir       = "/sys/class/net"
	dmiDir       = "/sys/class/dmi"
	ppcDevTree   = "/proc/device-tree"
	s390xDevTree = "/etc" // s390/s390x changes

	meminfoFile = "meminfo"

	distanceFile = "distance"

	sysFsCPUTopology = "topology"

	// CPUPhysicalPackageID is a physical package id of cpu#. Typically corresponds to a physical socket number,
	// but the actual value is architecture and platform dependent.
	CPUPhysicalPackageID = "physical_package_id"
	// CPUCoreID is the CPU core ID of cpu#. Typically it is the hardware platform's identifier
	// (rather than the kernel's). The actual value is architecture and platform dependent.
	CPUCoreID = "core_id"

	coreIDFilePath    = "/" + sysFsCPUTopology + "/core_id"
	packageIDFilePath = "/" + sysFsCPUTopology + "/physical_package_id"

	// memory size calculations

	cpuDirPattern  = "cpu*[0-9]"
	nodeDirPattern = "node*[0-9]"

	//HugePagesNrFile name of nr_hugepages file in sysfs
	HugePagesNrFile = "nr_hugepages"
)

var (
	nodeDir = "/sys/devices/system/node/"
)

type relocatableSysFs struct {
	root string
}

func NewRelocatableSysFs(root string) sysfs.SysFs {
	return &relocatableSysFs{
		root: root,
	}
}

func NewRealSysFs(root string) sysfs.SysFs {
	return NewRelocatableSysFs("/")
}

func (fs *relocatableSysFs) GetNodesPaths() ([]string, error) {
	pathPattern := filepath.Join(fs.root, nodeDir, nodeDirPattern)
	return filepath.Glob(pathPattern)
}

func (fs *relocatableSysFs) GetCPUsPaths(cpusPath string) ([]string, error) {
	pathPattern := filepath.Join(fs.root, cpusPath, cpuDirPattern)
	return filepath.Glob(pathPattern)
}

func (fs *relocatableSysFs) GetCoreID(cpuPath string) (string, error) {
	// intentionally not prepending fs.root, because this function
	// is expected to be used with `cpuPath` as returned by
	// GetCPUsPaths
	coreIDFilePath := filepath.Join(cpuPath, coreIDFilePath)
	coreID, err := os.ReadFile(coreIDFilePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(coreID)), err
}

func (fs *relocatableSysFs) GetCPUPhysicalPackageID(cpuPath string) (string, error) {
	// intentionally not prepending fs.root, because this function
	// is expected to be used with `cpuPath` as returned by
	// GetCPUsPaths
	packageIDFilePath := filepath.Join(cpuPath, packageIDFilePath)
	packageID, err := os.ReadFile(packageIDFilePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(packageID)), err
}

func (fs *relocatableSysFs) GetMemInfo(nodePath string) (string, error) {
	meminfoPath := filepath.Join(fs.root, nodePath, meminfoFile)
	meminfo, err := os.ReadFile(meminfoPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(meminfo)), err
}

func (fs *relocatableSysFs) GetDistances(nodePath string) (string, error) {
	distancePath := filepath.Join(fs.root, nodePath, distanceFile)
	distance, err := os.ReadFile(distancePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(distance)), err
}

func (fs *relocatableSysFs) readDir(dirPath string) ([]os.FileInfo, error) {
	var finfos []os.FileInfo
	dents, err := os.ReadDir(filepath.Join(fs.root, dirPath))
	if err != nil {
		return finfos, err
	}
	for _, dent := range dents {
		finfo, err := dent.Info()
		if err != nil {
			return finfos, err
		}
		finfos = append(finfos, finfo)
	}
	return finfos, nil
}

func (fs *relocatableSysFs) GetHugePagesInfo(hugePagesDirectory string) ([]os.FileInfo, error) {
	return fs.readDir(hugePagesDirectory)
}

func (fs *relocatableSysFs) GetHugePagesNr(hugepagesDirectory string, hugePageName string) (string, error) {
	hugePageFilePath := filepath.Join(fs.root, hugepagesDirectory, hugePageName, HugePagesNrFile)
	hugePageFile, err := os.ReadFile(hugePageFilePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(hugePageFile)), err
}

func (fs *relocatableSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	return fs.readDir(blockDir)
}

func (fs *relocatableSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	dev, err := os.ReadFile(filepath.Join(fs.root, blockDir, name, "/dev"))
	if err != nil {
		return "", err
	}
	return string(dev), nil
}

func (fs *relocatableSysFs) GetBlockDeviceScheduler(name string) (string, error) {
	sched, err := os.ReadFile(filepath.Join(fs.root, blockDir, name, "/queue/scheduler"))
	if err != nil {
		return "", err
	}
	return string(sched), nil
}

func (fs *relocatableSysFs) GetBlockDeviceSize(name string) (string, error) {
	size, err := os.ReadFile(filepath.Join(fs.root, blockDir, name, "/size"))
	if err != nil {
		return "", err
	}
	return string(size), nil
}

func (fs *relocatableSysFs) GetNetworkDevices() ([]os.FileInfo, error) {
	files, err := os.ReadDir(filepath.Join(fs.root, netDir))
	if err != nil {
		return nil, err
	}

	// Filter out non-directory & non-symlink files
	var dirs []os.FileInfo
	for _, f := range files {
		if f.Type().Type()|os.ModeSymlink != 0 {
			continue
		}
		if !f.IsDir() {
			continue
		}
		finfo, err := f.Info()
		if err != nil {
			return dirs, err
		}
		dirs = append(dirs, finfo)
	}
	return dirs, nil
}

func (fs *relocatableSysFs) GetNetworkAddress(name string) (string, error) {
	address, err := os.ReadFile(filepath.Join(fs.root, netDir, name, "/address"))
	if err != nil {
		return "", err
	}
	return string(address), nil
}

func (fs *relocatableSysFs) GetNetworkMtu(name string) (string, error) {
	mtu, err := os.ReadFile(filepath.Join(fs.root, netDir, name, "/mtu"))
	if err != nil {
		return "", err
	}
	return string(mtu), nil
}

func (fs *relocatableSysFs) GetNetworkSpeed(name string) (string, error) {
	speed, err := os.ReadFile(filepath.Join(fs.root, netDir, name, "/speed"))
	if err != nil {
		return "", err
	}
	return string(speed), nil
}

func (fs *relocatableSysFs) GetNetworkStatValue(dev string, stat string) (uint64, error) {
	statPath := filepath.Join(fs.root, netDir, dev, "/statistics", stat)
	out, err := os.ReadFile(statPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read stat from %q for device %q", statPath, dev)
	}
	var s uint64
	n, err := fmt.Sscanf(string(out), "%d", &s)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("could not parse value from %q for file %s", string(out), statPath)
	}
	return s, nil
}

func (fs *relocatableSysFs) GetCaches(id int) ([]os.FileInfo, error) {
	cpuPath := filepath.Join(fs.root, fmt.Sprintf("%s%d/cache", cacheDir, id))
	return fs.readDir(cpuPath)
}

func (fs *relocatableSysFs) IsBlockDeviceHidden(name string) (bool, error) {
	return false, nil
}

func toFileInfo(dirs []os.DirEntry) ([]os.FileInfo, error) {
	info := []os.FileInfo{}
	for _, dir := range dirs {
		fI, err := dir.Info()
		if err != nil {
			return nil, err
		}
		info = append(info, fI)
	}
	return info, nil
}

func bitCount(i uint64) (count int) {
	for i != 0 {
		if i&1 == 1 {
			count++
		}
		i >>= 1
	}
	return
}

func getCPUCount(cache string) (count int, err error) {
	out, err := os.ReadFile(filepath.Join(cache, "/shared_cpu_map"))
	if err != nil {
		return 0, err
	}
	masks := strings.Split(string(out), ",")
	for _, mask := range masks {
		// convert hex string to uint64
		m, err := strconv.ParseUint(strings.TrimSpace(mask), 16, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse cpu map %q: %v", string(out), err)
		}
		count += bitCount(m)
	}
	return
}

func (fs *relocatableSysFs) GetCacheInfo(cpu int, name string) (sysfs.CacheInfo, error) {
	cachePath := filepath.Join(fs.root, fmt.Sprintf("%s%d/cache/%s", cacheDir, cpu, name))
	out, err := os.ReadFile(filepath.Join(cachePath, "/id"))
	if err != nil {
		return sysfs.CacheInfo{}, err
	}
	var id int
	n, err := fmt.Sscanf(string(out), "%d", &id)
	if err != nil || n != 1 {
		return sysfs.CacheInfo{}, err
	}

	out, err = os.ReadFile(filepath.Join(cachePath, "/size"))
	if err != nil {
		return sysfs.CacheInfo{}, err
	}
	var size uint64
	n, err = fmt.Sscanf(string(out), "%dK", &size)
	if err != nil || n != 1 {
		return sysfs.CacheInfo{}, err
	}
	// convert to bytes
	size = size * 1024
	out, err = os.ReadFile(filepath.Join(cachePath, "/level"))
	if err != nil {
		return sysfs.CacheInfo{}, err
	}
	var level int
	n, err = fmt.Sscanf(string(out), "%d", &level)
	if err != nil || n != 1 {
		return sysfs.CacheInfo{}, err
	}

	out, err = os.ReadFile(filepath.Join(cachePath, "/type"))
	if err != nil {
		return sysfs.CacheInfo{}, err
	}
	cacheType := strings.TrimSpace(string(out))
	cpuCount, err := getCPUCount(cachePath)
	if err != nil {
		return sysfs.CacheInfo{}, err
	}
	return sysfs.CacheInfo{
		Id:    id,
		Size:  size,
		Level: level,
		Type:  cacheType,
		Cpus:  cpuCount,
	}, nil
}

func (fs *relocatableSysFs) GetSystemUUID() (string, error) {
	if id, err := os.ReadFile(filepath.Join(fs.root, dmiDir, "id", "product_uuid")); err == nil {
		return strings.TrimSpace(string(id)), nil
	} else if id, err = os.ReadFile(filepath.Join(fs.root, ppcDevTree, "system-id")); err == nil {
		return strings.TrimSpace(strings.TrimRight(string(id), "\000")), nil
	} else if id, err = os.ReadFile(filepath.Join(fs.root, ppcDevTree, "vm,uuid")); err == nil {
		return strings.TrimSpace(strings.TrimRight(string(id), "\000")), nil
	} else if id, err = os.ReadFile(filepath.Join(fs.root, s390xDevTree, "machine-id")); err == nil {
		return strings.TrimSpace(string(id)), nil
	} else {
		return "", err
	}
}

func (fs *relocatableSysFs) IsCPUOnline(cpuPath string) bool {
	onlinePath, err := filepath.Abs(filepath.Join(fs.root, cpuPath+"/../online"))
	if err != nil {
		klog.V(1).Infof("Unable to get absolute path for %s", cpuPath)
		return false
	}

	// Quick check to determine if file exists: if it does not then kernel CPU hotplug is disabled and all CPUs are online.
	_, err = os.Stat(onlinePath)
	if err != nil && os.IsNotExist(err) {
		return true
	}
	if err != nil {
		klog.V(1).Infof("Unable to stat %s: %s", onlinePath, err)
	}

	cpuID, err := getCPUID(cpuPath)
	if err != nil {
		klog.V(1).Infof("Unable to get CPU ID from path %s: %s", cpuPath, err)
		return false
	}

	isOnline, err := isCPUOnline(onlinePath, cpuID)
	if err != nil {
		klog.V(1).Infof("Unable to get online CPUs list: %s", err)
		return false
	}
	return isOnline
}

func getCPUID(dir string) (uint16, error) {
	regex := regexp.MustCompile("cpu([0-9]+)")
	matches := regex.FindStringSubmatch(dir)
	if len(matches) == 2 {
		id, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		return uint16(id), nil
	}
	return 0, fmt.Errorf("can't get CPU ID from %s", dir)
}

func isCPUOnline(path string, cpuID uint16) (bool, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if len(fileContent) == 0 {
		return false, fmt.Errorf("%s found to be empty", path)
	}

	cpus, err := cpuset.Parse(strings.TrimSpace(string(fileContent)))
	if err != nil {
		return false, err
	}

	for _, cpu := range cpus.UnsortedList() {
		if uint16(cpu) == cpuID {
			return true, nil
		}
	}
	return false, nil
}
