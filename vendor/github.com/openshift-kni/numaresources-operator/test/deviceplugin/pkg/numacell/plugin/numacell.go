/*
Copyright 2020.

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

package plugin

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/jaypipes/ghw/pkg/topology"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"

	"github.com/openshift-kni/numaresources-operator/test/deviceplugin/pkg/numacell/api"
	numacellapi "github.com/openshift-kni/numaresources-operator/test/deviceplugin/pkg/numacell/api"
)

// NUMACellLister is the object responsible for discovering initial pool of devices and their allocation.
type NUMACellLister struct {
	topoInfo    *topology.Info
	nameToID    map[string]int64
	deviceCount int
}

func NewNUMACellLister(topoInfo *topology.Info, deviceCount int) NUMACellLister {
	if deviceCount <= 0 {
		klog.Warningf("invalid devices count, reset to %d", api.NUMACellDefaultDeviceCount)
		deviceCount = api.NUMACellDefaultDeviceCount
	}
	klog.Infof("NUMACell: %d devices per NUMA cell", deviceCount)
	return NUMACellLister{
		topoInfo:    topoInfo,
		nameToID:    make(map[string]int64),
		deviceCount: deviceCount,
	}
}

type message struct{}

// NUMACellDevicePlugin is an implementation of DevicePlugin that is capable of exposing devices to containers.
type NUMACellDevicePlugin struct {
	deviceID    string
	numacellID  int64
	deviceCount int
	update      chan message
}

func (ncl NUMACellLister) GetResourceNamespace() string {
	return numacellapi.NUMACellResourceNamespace
}

// Discovery discovers all NUMA cells within the system.
func (ncl NUMACellLister) Discover(pluginListCh chan dpm.PluginNameList) {
	for _, node := range ncl.topoInfo.Nodes {
		deviceID := numacellapi.MakeDeviceID(node.ID)
		ncl.nameToID[deviceID] = int64(node.ID)
		pluginListCh <- dpm.PluginNameList{deviceID}
	}
}

// NewPlugin initializes new device plugin with NUMACell specific attributes.
func (ncl NUMACellLister) NewPlugin(deviceID string) dpm.PluginInterface {
	numacellID, found := ncl.nameToID[deviceID]
	klog.Infof("Creating device plugin %s -> %d (%v)", deviceID, numacellID, found)
	return &NUMACellDevicePlugin{
		deviceID:    deviceID,
		numacellID:  numacellID,
		update:      make(chan message),
		deviceCount: ncl.deviceCount,
	}
}

func (dpi *NUMACellDevicePlugin) device(idx int) *pluginapi.Device {
	return &pluginapi.Device{
		ID:     fmt.Sprintf("%s-%03d", dpi.deviceID, idx),
		Health: pluginapi.Healthy,
		Topology: &pluginapi.TopologyInfo{
			Nodes: []*pluginapi.NUMANode{
				{
					ID: int64(dpi.numacellID),
				},
			},
		},
	}
}

func (dpi *NUMACellDevicePlugin) devices() []*pluginapi.Device {
	devs := []*pluginapi.Device{}
	for cnt := 0; cnt < dpi.deviceCount; cnt++ {
		devs = append(devs, dpi.device(cnt))
	}
	return devs
}

// ListAndWatch sends gRPC stream of devices.
func (dpi *NUMACellDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	devs := dpi.devices()

	// Send initial list of devices
	resp := new(pluginapi.ListAndWatchResponse)
	resp.Devices = devs
	klog.Infof("send devices %v\n", resp)

	if err := s.Send(resp); err != nil {
		klog.Errorf("failed to list NUMA cells: %v\n", err)
		return err
	}

	// TODO handle signals like sriovdp does
	for range dpi.update {
		err := s.Send(&pluginapi.ListAndWatchResponse{Devices: devs})
		if err != nil {
			klog.Errorf("error sending ListAndWatchResponse: %v", err)
			return err
		}
	}
	return nil
}

// Allocate allocates a set of devices to be used by container runtime environment.
func (dpi *NUMACellDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var response pluginapi.AllocateResponse

	dpi.update <- message{}

	klog.Infof("Allocate() called with %+v", r)
	for _, container := range r.ContainerRequests {
		if len(container.DevicesIDs) != 1 {
			return nil, fmt.Errorf("can't allocate more than 1 numacell")
		}
		if !strings.HasPrefix(container.DevicesIDs[0], numacellapi.NUMACellResourceName) {
			return nil, fmt.Errorf("cannot allocate numacell %q", container.DevicesIDs[0])
		}

		dev := new(pluginapi.DeviceSpec)
		dev.HostPath = numacellapi.NUMACellDevicePath      // TODO
		dev.ContainerPath = numacellapi.NUMACellDevicePath // TODO
		dev.Permissions = "rw"

		containerResp := new(pluginapi.ContainerAllocateResponse)
		containerResp.Devices = []*pluginapi.DeviceSpec{dev}
		// this is only meant to improve debuggability
		containerResp.Envs = map[string]string{
			numacellapi.NUMACellEnvironVarName: fmt.Sprintf("%d", dpi.numacellID),
		}

		response.ContainerResponses = append(response.ContainerResponses, containerResp)
	}
	klog.Infof("AllocateResponse send: %+v", response)
	return &response, nil
}

// GetDevicePluginOptions returns options to be communicated with Device
// Manager
func (NUMACellDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return nil, nil
}

// GetPreferredAllocation returns a preferred set of devices to allocate
// from a list of available ones. The resulting preferred allocation is not
// guaranteed to be the allocation ultimately performed by the
// devicemanager. It is only designed to help the devicemanager make a more
// informed allocation decision when possible.
func (NUMACellDevicePlugin) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return nil, nil
}

// PreStartContainer is called, if indicated by Device Plugin during registeration phase,
// before each container start. Device plugin can run device specific operations
// such as reseting the device before making devices available to the container
func (NUMACellDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return nil, nil
}
