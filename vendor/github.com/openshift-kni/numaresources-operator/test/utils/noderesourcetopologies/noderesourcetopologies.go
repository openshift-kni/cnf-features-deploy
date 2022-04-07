/*
Copyright 2022 The Kubernetes Authors.

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

package noderesourcetopologies

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"

	e2ereslist "github.com/openshift-kni/numaresources-operator/internal/resourcelist"
)

// ErrNotEnoughResources means a NUMA zone or a node has not enough resouces to reserve
var ErrNotEnoughResources = errors.New("nrt: Not enough resources")

func GetZoneIDFromName(zoneName string) (int, error) {
	for _, prefix := range []string{
		"node-",
	} {
		if !strings.HasPrefix(zoneName, prefix) {
			continue
		}
		return strconv.Atoi(zoneName[len(prefix):])
	}
	return strconv.Atoi(zoneName)
}

func GetUpdated(cli client.Client, ref nrtv1alpha1.NodeResourceTopologyList, timeout time.Duration) (nrtv1alpha1.NodeResourceTopologyList, error) {
	var updatedNrtList nrtv1alpha1.NodeResourceTopologyList
	err := wait.Poll(1*time.Second, timeout, func() (bool, error) {
		err := cli.List(context.TODO(), &updatedNrtList)
		if err != nil {
			klog.Errorf("cannot get the NRT List: %v", err)
			return false, err
		}
		klog.Infof("NRT List current ResourceVersion %s reference %s", updatedNrtList.ListMeta.ResourceVersion, ref.ListMeta.ResourceVersion)
		return (updatedNrtList.ListMeta.ResourceVersion != ref.ListMeta.ResourceVersion), nil
	})
	return updatedNrtList, err
}

func CheckEqualAvailableResources(nrtInitial, nrtUpdated nrtv1alpha1.NodeResourceTopology) (bool, error) {
	for idx := 0; idx < len(nrtInitial.Zones); idx++ {
		zoneInitial := &nrtInitial.Zones[idx] // shortcut
		zoneUpdated, err := findZoneByName(nrtUpdated, zoneInitial.Name)
		if err != nil {
			klog.Errorf("missing updated zone %q: %v", zoneInitial.Name, err)
			return false, err
		}
		ok, what, err := checkEqualResourcesInfo(nrtInitial.Name, zoneInitial.Name, zoneInitial.Resources, zoneUpdated.Resources)
		if err != nil {
			klog.Errorf("error checking zone %q: %v", zoneInitial.Name, err)
			return false, err
		}
		if !ok {
			klog.Infof("node %q zone %q resource %q is different", nrtInitial.Name, zoneInitial.Name, what)
			return false, nil
		}
	}
	return true, nil
}

func CheckZoneConsumedResourcesAtLeast(nrtInitial, nrtUpdated nrtv1alpha1.NodeResourceTopology, required corev1.ResourceList) (string, error) {
	for idx := 0; idx < len(nrtInitial.Zones); idx++ {
		zoneInitial := &nrtInitial.Zones[idx] // shortcut
		zoneUpdated, err := findZoneByName(nrtUpdated, zoneInitial.Name)
		if err != nil {
			klog.Errorf("missing updated zone %q: %v", zoneInitial.Name, err)
			return "", err
		}
		ok, err := checkConsumedResourcesAtLeast(zoneInitial.Resources, zoneUpdated.Resources, required)
		if err != nil {
			klog.Errorf("error checking zone %q: %v", zoneInitial.Name, err)
			return "", err
		}
		if ok {
			klog.Infof("match for zone %q", zoneInitial.Name)
			return zoneInitial.Name, nil
		}
	}
	return "", nil
}

func SaturateZoneUntilLeft(zone nrtv1alpha1.Zone, requiredRes corev1.ResourceList) (corev1.ResourceList, error) {
	paddingRes := make(corev1.ResourceList)
	for resName, resQty := range requiredRes {
		zoneQty, ok := FindResourceAvailableByName(zone.Resources, string(resName))
		if !ok {
			return nil, fmt.Errorf("resource %q not found in zone %q", string(resName), zone.Name)
		}

		if zoneQty.Cmp(resQty) < 0 {
			klog.Errorf("resource %q already too scarce in zone %q (target %v amount %v)", resName, zone.Name, resQty, zoneQty)
			return nil, ErrNotEnoughResources
		}
		klog.Infof("zone %q resource %q available %s allocation target %s", zone.Name, resName, zoneQty.String(), resQty.String())
		paddingQty := zoneQty.DeepCopy()
		paddingQty.Sub(resQty)
		paddingRes[resName] = paddingQty
	}

	return paddingRes, nil
}

func SaturateNodeUntilLeft(nrtInfo nrtv1alpha1.NodeResourceTopology, requiredRes corev1.ResourceList) (map[string]corev1.ResourceList, error) {
	//TODO: support splitting the requiredRes on multiple numas
	//corrently the function deducts the requiredRes from the first Numa

	paddingRes := make(map[string]corev1.ResourceList)

	zeroRes := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("0"),
		corev1.ResourceMemory: resource.MustParse("0"),
	}
	var zonePadRes corev1.ResourceList
	var err error
	for ind, zone := range nrtInfo.Zones {
		if ind == 0 {
			zonePadRes, err = SaturateZoneUntilLeft(zone, zeroRes)
		} else {
			zonePadRes, err = SaturateZoneUntilLeft(zone, requiredRes)
		}
		if err != nil {
			klog.Errorf(fmt.Sprintf("could not make padding pod for zone %q leaving 0 resources available.", zone.Name))
			return nil, err
		}
		klog.Infof("Padding resources for zone %q: %s", zone.Name, e2ereslist.ToString(zonePadRes))
		paddingRes[zone.Name] = zonePadRes
	}

	return paddingRes, nil
}

func checkEqualResourcesInfo(nodeName, zoneName string, resourcesInitial, resourcesUpdated []nrtv1alpha1.ResourceInfo) (bool, string, error) {
	for _, res := range resourcesInitial {
		initialQty := res.Available
		updatedQty, ok := FindResourceAvailableByName(resourcesUpdated, res.Name)
		if !ok {
			return false, res.Name, fmt.Errorf("resource %q not found in the updated set", res.Name)
		}
		if initialQty.Cmp(updatedQty) != 0 {
			klog.Infof("node %q zone %q resource %q initial=%v updated=%v", nodeName, zoneName, res.Name, initialQty, updatedQty)
			return false, res.Name, nil
		}
	}
	return true, "", nil
}

func checkConsumedResourcesAtLeast(resourcesInitial, resourcesUpdated []nrtv1alpha1.ResourceInfo, required corev1.ResourceList) (bool, error) {
	for resName, resQty := range required {
		initialQty, ok := FindResourceAvailableByName(resourcesInitial, string(resName))
		if !ok {
			return false, fmt.Errorf("resource %q not found in the initial set", string(resName))
		}
		updatedQty, ok := FindResourceAvailableByName(resourcesUpdated, string(resName))
		if !ok {
			return false, fmt.Errorf("resource %q not found in the updated set", string(resName))
		}
		expectedQty := initialQty.DeepCopy()
		expectedQty.Sub(resQty)
		ret := updatedQty.Cmp(expectedQty)
		if ret > 0 {
			return false, nil
		}
	}
	return true, nil
}

func AccumulateNames(nrts []nrtv1alpha1.NodeResourceTopology) sets.String {
	nodeNames := sets.NewString()
	for _, nrt := range nrts {
		nodeNames.Insert(nrt.Name)
	}
	return nodeNames
}

func FilterTopologyManagerPolicy(nrts []nrtv1alpha1.NodeResourceTopology, tmPolicy nrtv1alpha1.TopologyManagerPolicy) []nrtv1alpha1.NodeResourceTopology {
	ret := []nrtv1alpha1.NodeResourceTopology{}
	for _, nrt := range nrts {
		if !contains(nrt.TopologyPolicies, string(tmPolicy)) {
			klog.Warningf("SKIP: node %q doesn't support topology manager policy %q", nrt.Name, string(tmPolicy))
			continue
		}
		klog.Infof("ADD : node %q supports topology manager policy %q", nrt.Name, string(tmPolicy))
		ret = append(ret, nrt)
	}
	return ret
}

func FilterZoneCountEqual(nrts []nrtv1alpha1.NodeResourceTopology, count int) []nrtv1alpha1.NodeResourceTopology {
	ret := []nrtv1alpha1.NodeResourceTopology{}
	for _, nrt := range nrts {
		if len(nrt.Zones) != count {
			klog.Warningf("SKIP: node %q has %d zones (desired %d)", nrt.Name, len(nrt.Zones), count)
			continue
		}
		klog.Infof("ADD : node %q has %d zones (desired %d)", nrt.Name, len(nrt.Zones), count)
		ret = append(ret, nrt)
	}
	return ret
}

func FilterAnyZoneMatchingResources(nrts []nrtv1alpha1.NodeResourceTopology, requests corev1.ResourceList) []nrtv1alpha1.NodeResourceTopology {
	reqStr := e2ereslist.ToString(requests)
	ret := []nrtv1alpha1.NodeResourceTopology{}
	for _, nrt := range nrts {
		matches := 0
		for _, zone := range nrt.Zones {
			klog.Infof(" ----> node %q zone %q provides %s requrst %s", nrt.Name, zone.Name, e2ereslist.ToString(AvailableFromZone(zone)), reqStr)
			if !ZoneResourcesMatchesRequest(zone.Resources, requests) {
				continue
			}
			matches++
		}
		if matches == 0 {
			klog.Warningf("SKIP: node %q can't provide %s", nrt.Name, reqStr)
			continue
		}
		klog.Infof("ADD : node %q provides at least %s", nrt.Name, reqStr)
		ret = append(ret, nrt)
	}
	return ret
}

func FindFromList(nrts []nrtv1alpha1.NodeResourceTopology, name string) (*nrtv1alpha1.NodeResourceTopology, error) {
	for idx := 0; idx < len(nrts); idx++ {
		if nrts[idx].Name == name {
			return &nrts[idx], nil
		}
	}
	return nil, fmt.Errorf("failed to find NRT for %q", name)
}

// AvailableFromZone returns a ResourceList of all available resources under the zone
func AvailableFromZone(z nrtv1alpha1.Zone) corev1.ResourceList {
	rl := corev1.ResourceList{}

	for _, ri := range z.Resources {
		rl[corev1.ResourceName(ri.Name)] = ri.Available
	}
	return rl
}

func ZoneResourcesMatchesRequest(resources []nrtv1alpha1.ResourceInfo, requests corev1.ResourceList) bool {
	for resName, resQty := range requests {
		zoneQty, ok := FindResourceAvailableByName(resources, string(resName))
		if !ok {
			return false
		}
		if zoneQty.Cmp(resQty) < 0 {
			return false
		}
	}
	return true
}

func FilterByPolicies(list []nrtv1alpha1.NodeResourceTopology, policies []nrtv1alpha1.TopologyManagerPolicy) []nrtv1alpha1.NodeResourceTopology {
	var filteredNrts []nrtv1alpha1.NodeResourceTopology
	for _, policy := range policies {
		nrts := FilterTopologyManagerPolicy(list, policy)
		filteredNrts = append(filteredNrts, nrts...)
	}
	return filteredNrts
}

func findZoneByName(nrt nrtv1alpha1.NodeResourceTopology, zoneName string) (*nrtv1alpha1.Zone, error) {
	for idx := 0; idx < len(nrt.Zones); idx++ {
		if nrt.Zones[idx].Name == zoneName {
			return &nrt.Zones[idx], nil
		}
	}
	return nil, fmt.Errorf("cannot find zone %q", zoneName)
}

func FindResourceAvailableByName(resources []nrtv1alpha1.ResourceInfo, name string) (resource.Quantity, bool) {
	for _, resource := range resources {
		if resource.Name != name {
			continue
		}
		return resource.Available, true
	}
	return *resource.NewQuantity(0, resource.DecimalSI), false
}

func contains(items []string, st string) bool {
	for _, item := range items {
		if item == st {
			return true
		}
	}
	return false
}
