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

package padder

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	k8swait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	numacellapi "github.com/openshift-kni/numaresources-operator/test/deviceplugin/pkg/numacell/api"
	"github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	nrtutil "github.com/openshift-kni/numaresources-operator/test/utils/noderesourcetopologies"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects/wait"
)

// This package allows to control the amount of available allocationTarget under the nodes.
// it's doing that by "padding" the nodes with guaranteed pods until node reaches the desired amount of available allocationTarget.
//
// Usage example:
// pad := New(client, "test-ns")
// pad.Nodes(2).UntilAvailableIsResource(corev1.ResourceCPU, "3").UntilAvailableIsResource(corev1.ResourceMemory, "100Mi").Pad(timeout)
// or
// rl := corev1.ResourceList {
//		corev1.ResourceCPU: "10",
//		corev1.ResourceMemory: "100Mi"
//	}
// pad.Nodes(2).UntilAvailableIsResourceList(rl).Pad(timeout)

const PadderLabel = "nrop-test-pad-pod"

type Padder struct {
	// Client defines the API client to run CRUD operations, that will be used for testing
	Client      client.Client
	paddedNodes []string
	namespace   string
	*padRequest
}

// PaddingOptions is some configuration that modifies options for a pad request.
type PaddingOptions struct {
	// LabelSelector pad nodes with a given label.
	// the number of padded nodes is still limit by padRequest.nNodes value
	LabelSelector labels.Selector
}

type padRequest struct {
	nNodes           int
	allocationTarget corev1.ResourceList
}

// New return new padder object
// ns is where padding pods will get deployed
func New(cli client.Client, ns string) (*Padder, error) {
	return &Padder{
		Client:     cli,
		namespace:  ns,
		padRequest: &padRequest{allocationTarget: make(corev1.ResourceList)},
	}, nil
}

// Nodes number of nodes to pad
func (p *Padder) Nodes(n int) *Padder {
	p.padRequest.nNodes = n
	return p
}

// UntilAvailableIsResource will pad pods into nodes until reach the desired available allocationTarget
func (p *Padder) UntilAvailableIsResource(resName corev1.ResourceName, quantity string) *Padder {
	quan := resource.MustParse(quantity)
	p.allocationTarget[resName] = quan
	return p
}

// UntilAvailableIsResourceList is like UntilAvailableIsResource but with a complete ResourceList as a parameter
func (p *Padder) UntilAvailableIsResourceList(resources corev1.ResourceList) *Padder {
	p.allocationTarget = resources
	return p
}

// Pad will create guaranteed pad pods on each NUMA zone
// in order to align the nodes
// with the requested amount of available allocationTarget
// and wait until timeout to see if nodes got updated
func (p *Padder) Pad(timeout time.Duration, options PaddingOptions) error {
	if p.nNodes == 0 {
		klog.Warningf("no nodes for padding were found. please specify at least one node")
		return nil
	}

	var opts []client.ListOption
	if options.LabelSelector != nil {
		opts = []client.ListOption{
			client.MatchingLabelsSelector{Selector: options.LabelSelector},
		}
	}
	nodeList := &corev1.NodeList{}
	err := p.Client.List(context.TODO(), nodeList, opts...)
	if err != nil {
		return err
	}

	nrtList, err := nrtutil.GetUpdated(p.Client, nrtv1alpha1.NodeResourceTopologyList{}, time.Second*10)
	if err != nil {
		return err
	}

	// since there is a relation of 1 : 1 between nodes and NRTs we can filter by the nodes` name
	nrts := filterNrtByNodeName(nrtList.Items, nodeList.Items)

	singleNumaNrt := nrtutil.FilterByPolicies(nrts, []nrtv1alpha1.TopologyManagerPolicy{nrtv1alpha1.SingleNUMANodePodLevel, nrtv1alpha1.SingleNUMANodeContainerLevel})
	if p.nNodes > len(singleNumaNrt) {
		return fmt.Errorf("not enough nodes were found for padding. requested: %d, got: %d", p.nNodes, len(singleNumaNrt))
	}

	nNodes := p.nNodes
	var pods []*corev1.Pod
	candidateNodes := nrtutil.AccumulateNames(singleNumaNrt)

	for nNodes > 0 {
		nodePadded := false

		// select one node randomly
		nodeName, ok := candidateNodes.PopAny()
		if !ok {
			return fmt.Errorf("cannot select a node to be padded among %#v", candidateNodes.List())
		}

		nrt, err := nrtutil.FindFromList(singleNumaNrt, nodeName)
		if err != nil {
			return err
		}

		for _, zone := range nrt.Zones {
			// check that zone has at least the amount of allocationTarget that needed
			if nrtutil.ZoneResourcesMatchesRequest(zone.Resources, p.allocationTarget) {
				availResList := nrtutil.AvailableFromZone(zone)
				diffList, err := diffAvailableToExpected(availResList, p.allocationTarget)
				if err != nil {
					return err
				}

				// create a pod that asks for exact amount of allocationTarget as the diff
				// in order to reach to desired amount of available allocationTarget in the node
				padPod := objects.NewTestPodPause(p.namespace, fixture.RandomizeName("padder"))

				// label the pod with a pad label, so it will be easier to identify later
				labelPod(padPod, map[string]string{PadderLabel: ""})

				cnt := &padPod.Spec.Containers[0]
				cnt.Resources.Limits = diffList
				cnt.Resources.Requests = diffList

				padPod, err = pinPodTo(padPod, zone, nrt.Name)
				if err != nil {
					return err
				}

				if err := p.Client.Create(context.TODO(), padPod); err != nil {
					return err
				}
				pods = append(pods, padPod)
				nodePadded = true
			} else {
				klog.Warningf("node: %q zone: %q, doesn't have enough available allocationTarget", nrt.Name, zone.Name)
			}
		}
		if nodePadded {
			nNodes--
			// store the node name, so we could check it's corresponding NRT later
			// or in order to return it to the user for further use later
			p.paddedNodes = append(p.paddedNodes, nodeName)
		}
	}

	if failedPods := wait.ForPodListAllRunning(p.Client, pods); len(failedPods) > 0 {
		var asStrings []string
		for _, pod := range failedPods {
			asStrings = append(asStrings, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
		return fmt.Errorf("pad pods are not running: %s", strings.Join(asStrings, ", "))
	}

	success, err := p.waitForUpdatedNRTs(timeout)
	if err != nil {
		return err
	}
	if !success {
		return fmt.Errorf("noderesourcestopologies are not updated with correct amount of available resources")
	}

	return nil
}

// Clean can be called after test finished
// in order to clean all the padding pod in an easier way
func (p *Padder) Clean() error {
	pod := &corev1.Pod{}
	if err := p.Client.DeleteAllOf(
		context.TODO(),
		pod,
		[]client.DeleteAllOfOption{
			client.InNamespace(p.namespace),
			client.MatchingLabels{PadderLabel: ""},
			client.GracePeriodSeconds(5),
		}...); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	podList := &corev1.PodList{}
	if err := p.Client.List(
		context.TODO(),
		podList,
		[]client.ListOption{
			client.InNamespace(p.namespace),
			client.MatchingLabels{PadderLabel: ""},
		}...); err != nil {
		return err
	}

	var errLock sync.Mutex
	var deletionErrors []string

	var wg sync.WaitGroup
	for _, padPod := range podList.Items {
		wg.Add(1)
		go func(pod corev1.Pod) {
			defer wg.Done()

			klog.Infof("waiting for pod %q to get deleted", pod.Name)
			if err := wait.ForPodDeleted(p.Client, p.namespace, pod.Name, time.Minute); err != nil {
				errLock.Lock()
				deletionErrors = append(deletionErrors, err.Error())
				errLock.Unlock()
			}
		}(padPod)
	}
	wg.Wait()
	if deletionErrors != nil {
		return fmt.Errorf("failed to wait for pad pods deletion. errors: %s", strings.Join(deletionErrors, ", "))
	}

	p.paddedNodes = []string{}
	return nil
}

func (p *Padder) GetPaddedNodes() []string {
	return p.paddedNodes
}

func (p *Padder) waitForUpdatedNRTs(timeout time.Duration) (bool, error) {
	NRTUpdated := false
	err := k8swait.PollImmediate(time.Second, timeout, func() (bool, error) {
		nrtList, err := nrtutil.GetUpdated(p.Client, nrtv1alpha1.NodeResourceTopologyList{}, time.Second*10)
		if err != nil {
			klog.Warningf("failed to get updated noderesourcestopologies objects")
			return false, err
		}

		for _, nodeName := range p.paddedNodes {
			nrt, err := nrtutil.FindFromList(nrtList.Items, nodeName)
			if err != nil {
				klog.Warningf("failed to get find noderesourcestopologies with name: %q", nodeName)
				return false, err
			}

			for _, zone := range nrt.Zones {
				if !isZoneMeetAllocationTarget(zone, p.allocationTarget) {
					klog.Warningf("node: %q zone: %q does not meet allocationTarget: %v", nodeName, zone.Name, p.allocationTarget)
					return false, nil
				}
			}
		}
		NRTUpdated = true
		return true, nil
	})

	return NRTUpdated, err
}

// return a resourceList of differences between the available and expected amount of resources
// it will return an error if requested allocationTarget are more than available
func diffAvailableToExpected(availableResLst corev1.ResourceList, expectedResLst corev1.ResourceList) (corev1.ResourceList, error) {
	// this list will be requested by the pad pod
	// in order to align the nodes resources with the expectedResLst
	resourceList := corev1.ResourceList{}

	for n, q1 := range availableResLst {
		if q2, ok := expectedResLst[n]; ok {
			// resource exist in expected list
			// calc the diff
			q1.Sub(q2)
			if q1.Sign() == -1 {
				return resourceList, fmt.Errorf("expected resource: %s=%v is higher than available: %s=%v", n, q2.String(), n, q1.String())
			}
			resourceList[n] = q1
		} else {
			// resource does not exist in the expected list,
			// hence we don't expect any changes on that specific resource
			resourceList[n] = resource.MustParse("0")
		}
	}

	return resourceList, nil
}

func labelPod(pod *corev1.Pod, labelMap map[string]string) {
	if pod.Labels == nil {
		pod.Labels = labelMap
		return
	}
	for k, v := range labelMap {
		pod.Labels[k] = v
	}
}

func isZoneMeetAllocationTarget(zone nrtv1alpha1.Zone, target corev1.ResourceList) bool {
	available := nrtutil.AvailableFromZone(zone)
	for res, targetQuan := range target {
		availQuan := available.Name(res, resource.DecimalSI)
		// we expect target to be the upper limit of the available resources
		if targetQuan.Cmp(*availQuan) < 0 {
			klog.Infof("expected target: %q to be greater or equal to available: %q",
				fmt.Sprintf("%s=%s", res, targetQuan.String()),
				fmt.Sprintf("%s=%s", res, availQuan.String()))
			return false
		}
	}
	return true
}

func pinPodTo(pod *corev1.Pod, zone nrtv1alpha1.Zone, nodeName string) (*corev1.Pod, error) {
	klog.Infof("forcing affinity to [%s: %s]", "kubernetes.io/hostname", nodeName)
	pod.Spec.NodeSelector = map[string]string{
		"kubernetes.io/hostname": nodeName,
	}

	// try to pin to specific zone only if the NUMA cell resources exists
	if numaCellResourceFound(zone) {
		zoneID, err := nrtutil.GetZoneIDFromName(zone.Name)
		if err != nil {
			return nil, err
		}
		klog.Infof("creating padding pod for node %q zone %d", nodeName, zoneID)

		cnt := &pod.Spec.Containers[0] // shortcut
		cnt.Resources.Limits[numacellapi.MakeResourceName(zoneID)] = resource.MustParse("1")
	}
	return pod, nil
}

func numaCellResourceFound(zone nrtv1alpha1.Zone) bool {
	for _, res := range zone.Resources {
		if strings.HasPrefix(res.Name, numacellapi.NUMACellResourceNamespace) {
			return true
		}
	}
	return false
}

func filterNrtByNodeName(nrts []nrtv1alpha1.NodeResourceTopology, nodes []corev1.Node) []nrtv1alpha1.NodeResourceTopology {
	var filtered []nrtv1alpha1.NodeResourceTopology
	for _, node := range nodes {
		if nrt, err := nrtutil.FindFromList(nrts, node.Name); err == nil {
			filtered = append(filtered, *nrt)
		}
	}
	return filtered
}
