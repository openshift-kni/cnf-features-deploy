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

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"

	e2ereslist "github.com/openshift-kni/numaresources-operator/internal/resourcelist"

	schedutils "github.com/openshift-kni/numaresources-operator/test/e2e/sched/utils"
	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	e2enrt "github.com/openshift-kni/numaresources-operator/test/utils/noderesourcetopologies"
	"github.com/openshift-kni/numaresources-operator/test/utils/nodes"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
	e2ewait "github.com/openshift-kni/numaresources-operator/test/utils/objects/wait"
)

type paddingInfo struct {
	pod                 *corev1.Pod
	targetNodeName      string
	unsuitableNodeNames []string
	unsuitableFreeRes   []corev1.ResourceList
}

type setupPaddingFunc func(fxt *e2efixture.Fixture, nrtList nrtv1alpha1.NodeResourceTopologyList, padInfo paddingInfo) []*corev1.Pod

var _ = Describe("[serial][disruptive][scheduler] numaresources workload placement considering TM policy", func() {
	var fxt *e2efixture.Fixture
	var nrtList nrtv1alpha1.NodeResourceTopologyList

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-workload-placement-tmpol")
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")

		err = fxt.Client.List(context.TODO(), &nrtList)
		Expect(err).ToNot(HaveOccurred())

		// Note that this test, being part of "serial", expects NO OTHER POD being scheduled
		// in between, so we consider this information current and valid when the It()s run.
	})

	AfterEach(func() {
		err := e2efixture.Teardown(fxt)
		Expect(err).NotTo(HaveOccurred())
	})

	// note we hardcode the values we need here and when we pad node.
	// This is ugly, but automatically computing the values is not straightforward
	// and will we want to start lean and mean.

	Context("with at least two nodes suitable", func() {
		var targetNodeName string
		var nrtCandidates []nrtv1alpha1.NodeResourceTopology

		setupCluster := func(requiredRes, paddingRes corev1.ResourceList) {
			requiredNUMAZones := 2
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", requiredNUMAZones))
			nrtCandidates = e2enrt.FilterZoneCountEqual(nrtList.Items, requiredNUMAZones)

			neededNodes := 2
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with 2 NUMA Zones: found %d, needed %d", len(nrtCandidates), neededNodes))
			}

			By("filtering available nodes with allocatable resources on at least one NUMA zone that can match request")
			nrtCandidates = e2enrt.FilterAnyZoneMatchingResources(nrtCandidates, requiredRes)
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with NUMA zones each of them can match requests: found %d, needed: %d", len(nrtCandidates), neededNodes))
			}
			nrtCandidateNames := e2enrt.AccumulateNames(nrtCandidates)

			var ok bool
			targetNodeName, ok = nrtCandidateNames.PopAny()
			ExpectWithOffset(1, ok).To(BeTrue(), "cannot select a target node among %#v", nrtCandidateNames.List())
			By(fmt.Sprintf("selecting node to schedule the pod: %q", targetNodeName))
			// need to prepare all the other nodes so they cannot have any one NUMA zone with enough resources
			// but have enough allocatable resources at node level to shedule the pod on it.
			// If we pad each zone with a pod with 3/4 of the required resources, as those nodes have at least
			// 2 NUMA zones, they will have enogh allocatable resources at node level to accomodate the required
			// resources but they won't have enough resources in only one NUMA zone.

			By("Padding all other candidate nodes")

			var paddingPods []*corev1.Pod
			for nIdx, nodeName := range nrtCandidateNames.List() {

				nrtInfo, err := e2enrt.FindFromList(nrtCandidates, nodeName)
				ExpectWithOffset(1, err).NotTo(HaveOccurred(), "missing NRT info for %q", nodeName)

				for zIdx, zone := range nrtInfo.Zones {
					podName := fmt.Sprintf("padding-%d-%d", nIdx, zIdx)
					padPod, err := makePaddingPod(fxt.Namespace.Name, podName, zone, paddingRes)
					ExpectWithOffset(1, err).NotTo(HaveOccurred(), "unable to create padding pod %q on zone %q", podName, zone.Name)

					padPod, err = pinPodTo(padPod, nodeName, zone.Name)
					ExpectWithOffset(1, err).NotTo(HaveOccurred(), "unable to pin pod %q to zone %q", podName, zone.Name)

					err = fxt.Client.Create(context.TODO(), padPod)
					ExpectWithOffset(1, err).NotTo(HaveOccurred(), "unable to create pod %q on zone %q", podName, zone.Name)

					paddingPods = append(paddingPods, padPod)
				}
			}

			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			ExpectWithOffset(1, failedPodIds).To(BeEmpty(), "some padding pods have failed to run")
		}

		// FIXME: this is a slight abuse of DescribeTable, but we need to run
		// the same code which a different test_id per tmscope
		DescribeTable("[tier1] a guaranteed pod with one container should be scheduled into one NUMA zone",
			func(tmPolicy nrtv1alpha1.TopologyManagerPolicy, requiredRes, paddingRes corev1.ResourceList) {
				setupCluster(requiredRes, paddingRes)

				nrts := e2enrt.FilterTopologyManagerPolicy(nrtCandidates, tmPolicy)
				if len(nrts) != len(nrtCandidates) {
					Skip(fmt.Sprintf("not enough nodes with policy %q - found %d", string(tmPolicy), len(nrts)))
				}

				By("Scheduling the testing pod")
				pod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
				pod.Spec.SchedulerName = serialconfig.Config.SchedulerName
				pod.Spec.Containers[0].Resources.Limits = requiredRes

				err := fxt.Client.Create(context.TODO(), pod)
				Expect(err).NotTo(HaveOccurred(), "unable to create pod %q", pod.Name)

				By("waiting for node to be up&running")
				podRunningTimeout := 1 * time.Minute
				updatedPod, err := e2ewait.ForPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodRunning, podRunningTimeout)
				if err != nil {
					_ = objects.LogEventsForPod(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name)
				}
				Expect(err).NotTo(HaveOccurred(), "Pod %q not up&running after %v", pod.Name, podRunningTimeout)

				By("checking the pod has been scheduled in the proper node")
				Expect(updatedPod.Spec.NodeName).To(Equal(targetNodeName))

				By(fmt.Sprintf("checking the pod was scheduled with the topology aware scheduler %q", serialconfig.Config.SchedulerName))
				schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
				Expect(err).ToNot(HaveOccurred())
				Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
			},
			Entry("[test_id:48713][tmscope:cnt]",
				nrtv1alpha1.SingleNUMANodeContainerLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("3"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				},
			),
			Entry("[tmscope:pod]",
				nrtv1alpha1.SingleNUMANodePodLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("3"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				},
			),
			Entry("[tmscope:cnt][hugepages]",
				nrtv1alpha1.SingleNUMANodeContainerLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("4"),
					corev1.ResourceMemory:                resource.MustParse("4Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("256Mi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("3"),
					corev1.ResourceMemory:                resource.MustParse("3Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("192Mi"),
				},
			),
			Entry("[tmscope:pod][hugepages]",
				nrtv1alpha1.SingleNUMANodePodLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("4"),
					corev1.ResourceMemory:                resource.MustParse("4Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("256Mi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("3"),
					corev1.ResourceMemory:                resource.MustParse("3Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("192Mi"),
				},
			),
		)

		// FIXME: this is a slight abuse of DescribeTable, but we need to run
		// the same code which a different test_id per tmscope
		DescribeTable("[tier1] a deployment with a guaranteed pod with one container should be scheduled into one NUMA zone",
			func(tmPolicy nrtv1alpha1.TopologyManagerPolicy, requiredRes, paddingRes corev1.ResourceList) {
				setupCluster(requiredRes, paddingRes)

				nrts := e2enrt.FilterTopologyManagerPolicy(nrtCandidates, tmPolicy)
				if len(nrts) != len(nrtCandidates) {
					Skip(fmt.Sprintf("not enough nodes with policy %q - found %d", string(tmPolicy), len(nrts)))
				}

				By("Scheduling the testing deployment")
				var deploymentName string = "test-dp"
				var replicas int32 = 1

				podLabels := map[string]string{
					"test": "test-dp",
				}
				nodeSelector := map[string]string{}
				deployment := objects.NewTestDeployment(replicas, podLabels, nodeSelector, fxt.Namespace.Name, deploymentName, objects.PauseImage, []string{objects.PauseCommand}, []string{})
				deployment.Spec.Template.Spec.SchedulerName = serialconfig.Config.SchedulerName
				deployment.Spec.Template.Spec.Containers[0].Resources.Limits = requiredRes

				err := fxt.Client.Create(context.TODO(), deployment)
				Expect(err).NotTo(HaveOccurred(), "unable to create deployment %q", deployment.Name)

				By("waiting for deployment to be up&running")
				dpRunningTimeout := 1 * time.Minute
				dpRunningPollInterval := 10 * time.Second
				err = e2ewait.ForDeploymentComplete(fxt.Client, deployment, dpRunningPollInterval, dpRunningTimeout)
				Expect(err).NotTo(HaveOccurred(), "Deployment %q not up&running after %v", deployment.Name, dpRunningTimeout)

				By(fmt.Sprintf("checking deployment pods have been scheduled with the topology aware scheduler %q and in the proper node %q", serialconfig.Config.SchedulerName, targetNodeName))
				pods, err := schedutils.ListPodsByDeployment(fxt.Client, *deployment)
				Expect(err).NotTo(HaveOccurred(), "Unable to get pods from Deployment %q:  %v", deployment.Name, err)

				for _, pod := range pods {
					Expect(pod.Spec.NodeName).To(Equal(targetNodeName))
					schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
					Expect(err).ToNot(HaveOccurred())
					Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
				}
			},
			Entry("[test_id:47583][tmscope:cnt]",
				nrtv1alpha1.SingleNUMANodeContainerLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("3"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				},
			),
			Entry("[tmscope:pod]",
				nrtv1alpha1.SingleNUMANodePodLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("3"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				},
			),
			Entry("[tmscope:cnt][hugepages]",
				nrtv1alpha1.SingleNUMANodeContainerLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("4"),
					corev1.ResourceMemory:                resource.MustParse("4Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("256Mi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("3"),
					corev1.ResourceMemory:                resource.MustParse("3Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("192Mi"),
				},
			),
			Entry("[tmscope:pod][hugepages]",
				nrtv1alpha1.SingleNUMANodePodLevel,
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("4"),
					corev1.ResourceMemory:                resource.MustParse("4Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("256Mi"),
				},
				corev1.ResourceList{
					corev1.ResourceCPU:                   resource.MustParse("3"),
					corev1.ResourceMemory:                resource.MustParse("3Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("192Mi"),
				},
			),
		)
	})

	DescribeTable("[placement] cluster with multiple worker nodes suitable",
		func(tmPolicy nrtv1alpha1.TopologyManagerPolicy, setupPadding setupPaddingFunc, podRes, unsuitableFreeRes []corev1.ResourceList) {

			hostsRequired := 2

			nrts := e2enrt.FilterTopologyManagerPolicy(nrtList.Items, tmPolicy)
			if len(nrts) < hostsRequired {
				Skip(fmt.Sprintf("not enough nodes with policy %q - found %d", string(tmPolicy), len(nrts)))
			}

			Expect(len(unsuitableFreeRes)).To(Equal(hostsRequired), "mismatch unsuitable resource declarations expected %d items, but found %d", hostsRequired, len(unsuitableFreeRes))

			pod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			pod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			pod.Spec.NodeSelector = map[string]string{
				serialconfig.MultiNUMALabel: "2",
			}
			pod.Spec.Containers = append(pod.Spec.Containers, pod.Spec.Containers[0])
			pod.Spec.Containers[0].Name = "testcnt-0"
			pod.Spec.Containers[0].Resources.Limits = podRes[0]
			pod.Spec.Containers[1].Name = "testcnt-1"
			pod.Spec.Containers[1].Resources.Limits = podRes[1]

			requiredRes := e2ereslist.FromGuaranteedPod(*pod)

			numaZonesRequired := 2

			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", numaZonesRequired))
			nrtCandidates := e2enrt.FilterZoneCountEqual(nrts, numaZonesRequired)
			if len(nrtCandidates) < hostsRequired {
				Skip(fmt.Sprintf("not enough nodes with %d NUMA Zones: found %d", numaZonesRequired, len(nrtCandidates)))
			}
			By("filtering available nodes with allocatable resources on each NUMA zone that can match request")
			nrtCandidates = e2enrt.FilterAnyZoneMatchingResources(nrtCandidates, requiredRes)
			if len(nrtCandidates) < hostsRequired {
				Skip(fmt.Sprintf("not enough nodes with NUMA zones each of them can match requests: found %d", len(nrtCandidates)))
			}

			candidateNodeNames := e2enrt.AccumulateNames(nrtCandidates)
			// nodes we have now are all equal for our purposes. Pick one at random
			// TODO: make sure we can control this randomness using ginkgo seed or any other way
			targetNodeName, ok := candidateNodeNames.PopAny()
			Expect(ok).To(BeTrue(), "cannot select a target node among %#v", candidateNodeNames.List())
			unsuitableNodeNames := candidateNodeNames.List()

			By(fmt.Sprintf("selecting target node %q and unsuitable nodes %#v (random pick)", targetNodeName, unsuitableNodeNames))

			padInfo := paddingInfo{
				pod:                 pod,
				targetNodeName:      targetNodeName,
				unsuitableNodeNames: unsuitableNodeNames,
				unsuitableFreeRes:   unsuitableFreeRes,
			}

			By("Padding nodes to create the test workload scenario")
			paddingPods := setupPadding(fxt, nrtList, padInfo)

			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			// TODO: smarter cooldown
			klog.Infof("cooling down")
			time.Sleep(18 * time.Second)

			for _, unsuitableNodeName := range unsuitableNodeNames {
				dumpNRTForNode(fxt.Client, unsuitableNodeName, "unsuitable")
			}
			dumpNRTForNode(fxt.Client, targetNodeName, "target")

			By("checking the resource allocation as the test starts")
			nrtListInitial, err := e2enrt.GetUpdated(fxt.Client, nrtList, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			By("running the test pod")
			if data, err := yaml.Marshal(pod); err != nil {
				klog.Infof("Pod:\n%s", data)
			}

			By("running the test pod")
			err = fxt.Client.Create(context.TODO(), pod)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the pod to be scheduled")
			updatedPod, err := e2ewait.ForPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodRunning, 2*time.Minute)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("checking the pod landed on the target node %q vs %q", updatedPod.Spec.NodeName, targetNodeName))
			Expect(updatedPod.Spec.NodeName).To(Equal(targetNodeName),
				"pod landed on %q instead of on %v", updatedPod.Spec.NodeName, targetNodeName)

			By(fmt.Sprintf("checking the pod was scheduled with the topology aware scheduler %q", serialconfig.Config.SchedulerName))
			schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)

			By(fmt.Sprintf("checking the resources are accounted as expected on %q", updatedPod.Spec.NodeName))
			nrtListPostCreate, err := e2enrt.GetUpdated(fxt.Client, nrtList, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			nrtInitial, err := e2enrt.FindFromList(nrtListInitial.Items, updatedPod.Spec.NodeName)
			Expect(err).ToNot(HaveOccurred())
			nrtPostCreate, err := e2enrt.FindFromList(nrtListPostCreate.Items, updatedPod.Spec.NodeName)
			Expect(err).ToNot(HaveOccurred())

			// TODO: this is only partially correct. We should check with NUMA zone granularity (not with NODE granularity)
			_, err = e2enrt.CheckZoneConsumedResourcesAtLeast(*nrtInitial, *nrtPostCreate, requiredRes)
			Expect(err).ToNot(HaveOccurred())

			By("deleting the test pod")
			err = fxt.Client.Delete(context.TODO(), updatedPod)
			Expect(err).ToNot(HaveOccurred())

			By("checking the test pod is removed")
			err = e2ewait.ForPodDeleted(fxt.Client, updatedPod.Namespace, updatedPod.Name, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			// the NRT updaters MAY be slow to react for a number of reasons including factors out of our control
			// (kubelet, runtime). This is a known behaviour. We can only tolerate some delay in reporting on pod removal.
			Eventually(func() bool {
				By(fmt.Sprintf("checking the resources are restored as expected on %q", updatedPod.Spec.NodeName))

				nrtListPostDelete, err := e2enrt.GetUpdated(fxt.Client, nrtListPostCreate, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())

				nrtPostDelete, err := e2enrt.FindFromList(nrtListPostDelete.Items, updatedPod.Spec.NodeName)
				Expect(err).ToNot(HaveOccurred())

				ok, err := e2enrt.CheckEqualAvailableResources(*nrtInitial, *nrtPostDelete)
				Expect(err).ToNot(HaveOccurred())
				return ok
			}, time.Minute, time.Second*5).Should(BeTrue(), "resources not restored on %q", updatedPod.Spec.NodeName)
		},

		Entry("[test_id:47575][tmscope:cnt][tier1] should make a pod with two gu cnt land on a node with enough resources on a specific NUMA zone, each cnt on a different zone",
			nrtv1alpha1.SingleNUMANodeContainerLevel,
			setupPaddingContainerLevel,
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("6"),
					corev1.ResourceMemory: resource.MustParse("6Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("12"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
			// make sure the sum is equal to the sum of the requirement of the test pod,
			// so the *node* total free resources are equal between the target node and
			// the unsuitable nodes
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("16"),
					corev1.ResourceMemory: resource.MustParse("12Gi"),
				},
			},
		),
		Entry("[test_id:47577][tmscope:pod][tier1] should make a pod with two gu cnt land on a node with enough resources on a specific NUMA zone, all cnt on the same zone",
			nrtv1alpha1.SingleNUMANodePodLevel,
			setupPaddingPodLevel,
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("6"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("12Gi"),
				},
			},
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("14"),
					corev1.ResourceMemory: resource.MustParse("10Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("10"),
					corev1.ResourceMemory: resource.MustParse("16Gi"),
				},
			},
		),
		Entry("[tmscope:cnt][hugepages] should make a pod with two gu cnt land on a node with enough resources on a specific NUMA zone, each cnt on a different zone",
			nrtv1alpha1.SingleNUMANodeContainerLevel,
			setupPaddingContainerLevel,
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:                   resource.MustParse("6"),
					corev1.ResourceMemory:                resource.MustParse("6Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("96Mi"),
				},
				{
					corev1.ResourceCPU:                   resource.MustParse("12"),
					corev1.ResourceMemory:                resource.MustParse("8Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("128Mi"),
				},
			},
			// make sure the sum is equal to the sum of the requirement of the test pod,
			// so the *node* total free resources are equal between the target node and
			// the unsuitable nodes
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				{
					corev1.ResourceCPU:                   resource.MustParse("16"),
					corev1.ResourceMemory:                resource.MustParse("12Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("192Mi"),
				},
			},
		),
		Entry("[tmscope:pod][hugepages] should make a pod with two gu cnt land on a node with enough resources on a specific NUMA zone, all cnt on the same zone",
			nrtv1alpha1.SingleNUMANodePodLevel,
			setupPaddingPodLevel,
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:                   resource.MustParse("6"),
					corev1.ResourceMemory:                resource.MustParse("4Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("32Mi"),
				},
				{
					corev1.ResourceCPU:                   resource.MustParse("8"),
					corev1.ResourceMemory:                resource.MustParse("12Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("128Mi"),
				},
			},
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:                   resource.MustParse("14"),
					corev1.ResourceMemory:                resource.MustParse("10Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("32Mi"),
				},
				{
					corev1.ResourceCPU:                   resource.MustParse("10"),
					corev1.ResourceMemory:                resource.MustParse("16Gi"),
					corev1.ResourceName("hugepages-2Mi"): resource.MustParse("144Mi"),
				},
			},
		),
	)
})

func setupPaddingPodLevel(fxt *e2efixture.Fixture, nrtList nrtv1alpha1.NodeResourceTopologyList, padInfo paddingInfo) []*corev1.Pod {
	baseload, err := nodes.GetLoad(fxt.K8sClient, padInfo.targetNodeName)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "missing node load info for %q", padInfo.targetNodeName)
	By(fmt.Sprintf("computed base load: %s", baseload))

	By(fmt.Sprintf("preparing target node %q to fit the test case", padInfo.targetNodeName))
	// first, let's make sure that ONLY the required res can fit in either zone on the target node
	nrtInfo, err := e2enrt.FindFromList(nrtList.Items, padInfo.targetNodeName)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "missing NRT info for %q", padInfo.targetNodeName)

	// if we get this far we can now depend on the fact that len(nrt.Zones) == len(pod.Spec.Containers) == 2

	paddingPods := []*corev1.Pod{}

	// as low as you can get, that's why it's hardcoded
	targetUnsuitableRes := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("2"),
		corev1.ResourceMemory: resource.MustParse("2Gi"),
	}

	var zone nrtv1alpha1.Zone
	// fix target node
	zone = nrtInfo.Zones[0]
	By(fmt.Sprintf("padding node %q zone %q to fit only %s", nrtInfo.Name, zone.Name, e2ereslist.ToString(targetUnsuitableRes)))
	padPod, err := makePaddingPod(fxt.Namespace.Name, "target", zone, targetUnsuitableRes)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	padPod, err = pinPodTo(padPod, nrtInfo.Name, zone.Name)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	err = fxt.Client.Create(context.TODO(), padPod)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	paddingPods = append(paddingPods, padPod)

	podTotRes := e2ereslist.FromGuaranteedPod(*padInfo.pod)
	By(fmt.Sprintf("testpod resource requests (vanilla): %s", e2ereslist.ToString(podTotRes)))
	baseload.Apply(podTotRes)
	By(fmt.Sprintf("testpod resource requests (adjusted): %s", e2ereslist.ToString(podTotRes)))

	zone = nrtInfo.Zones[1]
	By(fmt.Sprintf("padding node %q zone %q to fit only %s", nrtInfo.Name, zone.Name, e2ereslist.ToString(podTotRes)))
	padPod, err = makePaddingPod(fxt.Namespace.Name, "target", zone, podTotRes)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	padPod, err = pinPodTo(padPod, nrtInfo.Name, zone.Name)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	err = fxt.Client.Create(context.TODO(), padPod)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	paddingPods = append(paddingPods, padPod)

	paddingPodsUnsuitable := setupPaddingForUnsuitableNodes(2, fxt, nrtList, padInfo)
	return append(paddingPods, paddingPodsUnsuitable...)
}

func setupPaddingContainerLevel(fxt *e2efixture.Fixture, nrtList nrtv1alpha1.NodeResourceTopologyList, padInfo paddingInfo) []*corev1.Pod {
	baseload, err := nodes.GetLoad(fxt.K8sClient, padInfo.targetNodeName)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "missing node load info for %q", padInfo.targetNodeName)
	By(fmt.Sprintf("computed base load: %s", baseload))

	By(fmt.Sprintf("preparing target node %q to fit the test case", padInfo.targetNodeName))
	// first, let's make sure that ONLY the required res can fit in either zone on the target node
	nrtInfo, err := e2enrt.FindFromList(nrtList.Items, padInfo.targetNodeName)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "missing NRT info for %q", padInfo.targetNodeName)

	// if we get this far we can now depend on the fact that len(nrt.Zones) == len(pod.Spec.Containers) == 2

	numCnts := len(padInfo.pod.Spec.Containers)
	paddingPods := []*corev1.Pod{}

	for idx := 0; idx < numCnts; idx++ {
		numaIdx := idx % 2
		zone := nrtInfo.Zones[numaIdx]
		cnt := padInfo.pod.Spec.Containers[idx] // TODO: reverse

		// TODO: apply baseload

		By(fmt.Sprintf("padding node %q zone %q to fit only %s", nrtInfo.Name, zone.Name, e2ereslist.ToString(cnt.Resources.Limits)))
		padPod, err := makePaddingPod(fxt.Namespace.Name, "target", zone, cnt.Resources.Limits)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		padPod, err = pinPodTo(padPod, nrtInfo.Name, zone.Name)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		err = fxt.Client.Create(context.TODO(), padPod)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		paddingPods = append(paddingPods, padPod)
	}

	paddingPodsUnsuitable := setupPaddingForUnsuitableNodes(2, fxt, nrtList, padInfo)
	return append(paddingPods, paddingPodsUnsuitable...)
}

func setupPaddingForUnsuitableNodes(offset int, fxt *e2efixture.Fixture, nrtList nrtv1alpha1.NodeResourceTopologyList, padInfo paddingInfo) []*corev1.Pod {
	paddingPods := []*corev1.Pod{}

	// still working under the assumption that len(nrt.Zones) == len(pod.Spec.Containers) == 2
	for nodeIdx, unsuitableNodeName := range padInfo.unsuitableNodeNames {
		nrtInfo, err := e2enrt.FindFromList(nrtList.Items, unsuitableNodeName)
		ExpectWithOffset(offset, err).ToNot(HaveOccurred(), "missing NRT info for %q", unsuitableNodeName)

		baseload, err := nodes.GetLoad(fxt.K8sClient, unsuitableNodeName)
		ExpectWithOffset(offset, err).ToNot(HaveOccurred(), "missing node load info for %q", unsuitableNodeName)
		By(fmt.Sprintf("computed base load: %s", baseload))

		for zoneIdx, zone := range nrtInfo.Zones {
			padRes := padInfo.unsuitableFreeRes[zoneIdx]
			name := fmt.Sprintf("unsuitable%d", nodeIdx)

			By(fmt.Sprintf("saturating node %q -> %q zone %q to fit only (vanilla) %s", nrtInfo.Name, name, zone.Name, e2ereslist.ToString(padRes)))
			if zoneIdx == 0 { // any random zone is actually fine
				baseload.Apply(padRes)
				By(fmt.Sprintf("saturating node %q -> %q zone %q to fit only (adjusted) %s", nrtInfo.Name, name, zone.Name, e2ereslist.ToString(padRes)))
			}

			padPod, err := makePaddingPod(fxt.Namespace.Name, name, zone, padRes)
			ExpectWithOffset(offset, err).ToNot(HaveOccurred())

			padPod, err = pinPodTo(padPod, nrtInfo.Name, zone.Name)
			ExpectWithOffset(offset, err).ToNot(HaveOccurred())

			err = fxt.Client.Create(context.TODO(), padPod)
			ExpectWithOffset(offset, err).ToNot(HaveOccurred())
			paddingPods = append(paddingPods, padPod)
		}
	}

	return paddingPods
}
