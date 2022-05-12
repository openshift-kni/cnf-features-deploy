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

package tests

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/taints"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"

	e2ereslist "github.com/openshift-kni/numaresources-operator/internal/resourcelist"

	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	e2enrt "github.com/openshift-kni/numaresources-operator/test/utils/noderesourcetopologies"
	e2enodes "github.com/openshift-kni/numaresources-operator/test/utils/nodes"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
	e2ewait "github.com/openshift-kni/numaresources-operator/test/utils/objects/wait"
	e2epadder "github.com/openshift-kni/numaresources-operator/test/utils/padder"
)

var _ = Describe("[serial][disruptive][scheduler] numaresources workload placement considering taints", func() {
	var fxt *e2efixture.Fixture
	var padder *e2epadder.Padder
	var nrtList nrtv1alpha1.NodeResourceTopologyList
	var nrts []nrtv1alpha1.NodeResourceTopology

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-workload-placement")
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")

		padder, err = e2epadder.New(fxt.Client, fxt.Namespace.Name)
		Expect(err).ToNot(HaveOccurred())

		err = fxt.Client.List(context.TODO(), &nrtList)
		Expect(err).ToNot(HaveOccurred())

		// we're ok with any TM policy as long as the updater can handle it,
		// we use this as proxy for "there is valid NRT data for at least X nodes
		policies := []nrtv1alpha1.TopologyManagerPolicy{
			nrtv1alpha1.SingleNUMANodeContainerLevel,
			nrtv1alpha1.SingleNUMANodePodLevel,
		}
		nrts = e2enrt.FilterByPolicies(nrtList.Items, policies)
		if len(nrts) < 2 {
			Skip(fmt.Sprintf("not enough nodes with valid policy - found %d", len(nrts)))
		}

		// Note that this test, being part of "serial", expects NO OTHER POD being scheduled
		// in between, so we consider this information current and valid when the It()s run.
	})

	AfterEach(func() {
		err := padder.Clean()
		Expect(err).NotTo(HaveOccurred())
		err = e2efixture.Teardown(fxt)
		Expect(err).NotTo(HaveOccurred())
	})

	// note we hardcode the values we need here and when we pad node.
	// This is ugly, but automatically computing the values is not straightforward
	// and will we want to start lean and mean.

	When("cluster has two feasible nodes with taint but only one has the requested resources on a single NUMA zone", func() {
		timeout := 5 * time.Minute
		BeforeEach(func() {
			numOfNodeToBePadded := len(nrts) - 1

			rl := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3"),
				corev1.ResourceMemory: resource.MustParse("8G"),
			}
			By("padding the nodes before test start")
			err := padder.Nodes(numOfNodeToBePadded).UntilAvailableIsResourceList(rl).Pad(timeout, e2epadder.PaddingOptions{})
			Expect(err).ToNot(HaveOccurred())

			nodes, err := e2enodes.GetWorkerNodes(fxt.Client)
			Expect(err).ToNot(HaveOccurred())

			t, _, err := taints.ParseTaints([]string{testTaint()})
			Expect(err).ToNot(HaveOccurred())

			for i := range nodes {
				node := &nodes[i]
				updatedNode, updated, err := taints.AddOrUpdateTaint(node, &t[0])
				Expect(err).ToNot(HaveOccurred())
				if updated {
					node = updatedNode
					klog.Infof("adding taint: %q to node: %q", t[0].String(), node.Name)
					err = fxt.Client.Update(context.TODO(), node)
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})

		AfterEach(func() {
			nodes, err := e2enodes.GetWorkerNodes(fxt.Client)
			Expect(err).ToNot(HaveOccurred())

			t, _, err := taints.ParseTaints([]string{testTaint()})
			Expect(err).ToNot(HaveOccurred())

			By("removing taints from the nodes")
			// TODO: remove taints in parallel
			for i := range nodes {
				Eventually(func() error {
					node := &corev1.Node{}
					err := fxt.Client.Get(context.TODO(), client.ObjectKeyFromObject(&nodes[i]), node)
					Expect(err).ToNot(HaveOccurred())

					updatedNode, updated, err := taints.RemoveTaint(node, &t[0])
					Expect(err).ToNot(HaveOccurred())
					if !updated {
						return nil
					}

					klog.Infof("removing taint: %q from node: %q", t[0].String(), updatedNode.Name)
					return fxt.Client.Update(context.TODO(), updatedNode)
				}, time.Minute, time.Second*5).ShouldNot(HaveOccurred())
			}

			By("unpadding the nodes after test finish")
			err = padder.Clean()
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:47594][tier1] should make a pod with a toleration land on a node with enough resources on a specific NUMA zone", func() {
			hostsRequired := 2
			paddedNodes := padder.GetPaddedNodes()
			paddedNodesSet := sets.NewString(paddedNodes...)

			nrtInitialList, err := e2enrt.GetUpdated(fxt.Client, nrtv1alpha1.NodeResourceTopologyList{}, time.Second*10)
			Expect(err).ToNot(HaveOccurred())

			// so we can't support ATM zones > 2. HW with zones > 2 is rare anyway, so not to big of a deal now.
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", 2))
			nrtCandidates := e2enrt.FilterZoneCountEqual(nrtInitialList.Items, 2)
			if len(nrtCandidates) < hostsRequired {
				Skip(fmt.Sprintf("not enough nodes with 2 NUMA Zones: found %d", len(nrtCandidates)))
			}

			singleNUMAPolicyNrts := e2enrt.FilterByPolicies(nrtInitialList.Items, []nrtv1alpha1.TopologyManagerPolicy{nrtv1alpha1.SingleNUMANodePodLevel, nrtv1alpha1.SingleNUMANodeContainerLevel})
			nodesNameSet := e2enrt.AccumulateNames(singleNUMAPolicyNrts)

			// the only node which was not padded is the targetedNode
			// since we know exactly how the test setup looks like we expect only targeted node here
			targetNodeNameSet := nodesNameSet.Difference(paddedNodesSet)
			Expect(targetNodeNameSet.Len()).To(Equal(1))

			targetNodeName, ok := targetNodeNameSet.PopAny()
			Expect(ok).To(BeTrue())

			testPod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			pSpec := &testPod.Spec
			if pSpec.Tolerations == nil {
				pSpec.Tolerations = []corev1.Toleration{}
			}
			pSpec.Tolerations = append(pSpec.Tolerations, testToleration()...)

			testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			cnt := &testPod.Spec.Containers[0]
			requiredRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			}
			cnt.Resources.Requests = requiredRes
			cnt.Resources.Limits = requiredRes

			err = fxt.Client.Create(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			updatedPod, err := e2ewait.ForPodPhase(fxt.Client, testPod.Namespace, testPod.Name, corev1.PodRunning, timeout)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("checking the pod landed on the target node %q vs %q", updatedPod.Spec.NodeName, targetNodeName))
			Expect(updatedPod.Spec.NodeName).To(Equal(targetNodeName),
				"node landed on %q instead of on %v", updatedPod.Spec.NodeName, targetNodeName)

			By(fmt.Sprintf("checking the pod was scheduled with the topology aware scheduler %q", serialconfig.Config.SchedulerName))
			schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)

			nrtPostCreateList, err := e2enrt.GetUpdated(fxt.Client, nrtInitialList, time.Second*10)
			Expect(err).ToNot(HaveOccurred())

			rl := e2ereslist.FromGuaranteedPod(*updatedPod)

			nrtInitial, err := e2enrt.FindFromList(nrtInitialList.Items, updatedPod.Spec.NodeName)
			Expect(err).ToNot(HaveOccurred())

			nrtPostCreate, err := e2enrt.FindFromList(nrtPostCreateList.Items, updatedPod.Spec.NodeName)
			Expect(err).ToNot(HaveOccurred())

			_, err = e2enrt.CheckZoneConsumedResourcesAtLeast(*nrtInitial, *nrtPostCreate, rl)
			Expect(err).ToNot(HaveOccurred())

			By("deleting the test pod")
			if err := fxt.Client.Delete(context.TODO(), updatedPod); err != nil {
				if !apierrors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			}

			By("checking the test pod is removed")
			err = e2ewait.ForPodDeleted(fxt.Client, updatedPod.Namespace, testPod.Name, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			// the NRT updaters MAY be slow to react for a number of reasons including factors out of our control
			// (kubelet, runtime). This is a known behaviour. We can only tolerate some delay in reporting on pod removal.
			Eventually(func() bool {
				By(fmt.Sprintf("checking the resources are restored as expected on %q", updatedPod.Spec.NodeName))

				nrtListPostDelete, err := e2enrt.GetUpdated(fxt.Client, nrtPostCreateList, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())

				nrtPostDelete, err := e2enrt.FindFromList(nrtListPostDelete.Items, updatedPod.Spec.NodeName)
				Expect(err).ToNot(HaveOccurred())

				ok, err := e2enrt.CheckEqualAvailableResources(*nrtInitial, *nrtPostDelete)
				Expect(err).ToNot(HaveOccurred())
				return ok
			}, time.Minute, time.Second*5).Should(BeTrue(), "resources not restored on %q", updatedPod.Spec.NodeName)
		})
	})
})
