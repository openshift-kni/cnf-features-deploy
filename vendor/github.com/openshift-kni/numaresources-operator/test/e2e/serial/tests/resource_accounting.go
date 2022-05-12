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
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"

	e2ereslist "github.com/openshift-kni/numaresources-operator/internal/resourcelist"

	schedutils "github.com/openshift-kni/numaresources-operator/test/e2e/sched/utils"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	e2enrt "github.com/openshift-kni/numaresources-operator/test/utils/noderesourcetopologies"
	"github.com/openshift-kni/numaresources-operator/test/utils/nodes"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
	e2ewait "github.com/openshift-kni/numaresources-operator/test/utils/objects/wait"
	e2epadder "github.com/openshift-kni/numaresources-operator/test/utils/padder"

	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
)

var _ = Describe("[serial][disruptive][scheduler] numaresources workload resource accounting", func() {
	var fxt *e2efixture.Fixture
	var padder *e2epadder.Padder
	var nrtList nrtv1alpha1.NodeResourceTopologyList
	var nrts []nrtv1alpha1.NodeResourceTopology

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-resource-accounting")
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

	Context("cluster with at least a worker node suitable", func() {
		var nrtTwoZoneCandidates []nrtv1alpha1.NodeResourceTopology
		BeforeEach(func() {
			const requiredNumaZones int = 2
			const requiredNodeNumber int = 1
			// TODO: we need AT LEAST 2 (so 4, 8 is fine...) but we hardcode the padding logic to keep the test simple,
			// so we can't support ATM zones > 2. HW with zones > 2 is rare anyway, so not to big of a deal now.
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", requiredNumaZones))
			nrtTwoZoneCandidates = e2enrt.FilterZoneCountEqual(nrts, requiredNumaZones)
			if len(nrtTwoZoneCandidates) < requiredNodeNumber {
				Skip(fmt.Sprintf("not enough nodes with 2 NUMA Zones: found %d", len(nrtTwoZoneCandidates)))
			}
		})

		It("[placement][test_id:49068][tier2] should keep the pod pending if not enough resources available, then schedule when resources are freed", func() {
			// make sure this is > 1 and LESS than required Res!
			unsuitableFreeRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			}

			requiredRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			}

			By(fmt.Sprintf("creating test pod, total resources required %s", e2ereslist.ToString(requiredRes)))

			By("filtering available nodes with allocatable resources on each NUMA zone that can match request")
			nrtCandidates := e2enrt.FilterAnyZoneMatchingResources(nrtTwoZoneCandidates, requiredRes)
			if len(nrtCandidates) < 1 {
				Skip(fmt.Sprintf("not enough nodes with NUMA zones each of them can match requests: found %d", len(nrtCandidates)))
			}

			candidateNodeNames := e2enrt.AccumulateNames(nrtCandidates)
			// nodes we have now are all equal for our purposes. Pick one at random
			// TODO: make sure we can control this randomness using ginkgo seed or any other way
			targetNodeName, ok := candidateNodeNames.PopAny()
			Expect(ok).To(BeTrue(), "cannot select a target node among %#v", candidateNodeNames.List())
			unsuitableNodeNames := candidateNodeNames.List()

			By(fmt.Sprintf("selecting target node %q and unsuitable nodes %#v (random pick)", targetNodeName, unsuitableNodeNames))
			var targetPaddingPods []*corev1.Pod
			var paddingPods []*corev1.Pod

			By(fmt.Sprintf("preparing target node %q to fit the test case", targetNodeName))
			// first, let's make sure that ONLY the required res can fit in either zone on the target node
			nrtInfo, err := e2enrt.FindFromList(nrtList.Items, targetNodeName)
			Expect(err).ToNot(HaveOccurred(), "missing NRT info for %q", targetNodeName)

			for _, zone := range nrtInfo.Zones {
				By(fmt.Sprintf("padding node %q zone %q", nrtInfo.Name, zone.Name))
				padPod, err := makePaddingPod(fxt.Namespace.Name, "target", zone, requiredRes)
				Expect(err).ToNot(HaveOccurred())

				padPod, err = pinPodTo(padPod, nrtInfo.Name, zone.Name)
				Expect(err).ToNot(HaveOccurred())

				err = fxt.Client.Create(context.TODO(), padPod)
				Expect(err).ToNot(HaveOccurred())
				paddingPods = append(paddingPods, padPod)
			}

			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			for _, zone := range nrtInfo.Zones {
				By(fmt.Sprintf("making node %q zone %q unsuitable with a placeholder pod", nrtInfo.Name, zone.Name))
				Expect(err).ToNot(HaveOccurred(), "cannot detect the zone ID from %q", zone.Name)
				// now put a minimal pod (1 cpu 1Gi) on both zones. Now the target node as whole will still have the
				// required resources, but no NUMA zone individually will
				targetedPaddingPod := objects.NewTestPodPause(fxt.Namespace.Name, fmt.Sprintf("tgtpadpod-%s", zone.Name))
				targetedPaddingPod.Spec.NodeName = nrtInfo.Name
				targetedPaddingPod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				}

				targetedPaddingPod, err = pinPodTo(targetedPaddingPod, nrtInfo.Name, zone.Name)
				Expect(err).ToNot(HaveOccurred())

				err = fxt.Client.Create(context.TODO(), targetedPaddingPod)
				Expect(err).ToNot(HaveOccurred())
				targetPaddingPods = append(targetPaddingPods, targetedPaddingPod)
			}

			By("Waiting for padding pods to be ready")
			failedPodIds = e2ewait.ForPaddingPodsRunning(fxt, targetPaddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			By("saturating nodes we want to be unsuitable")
			for idx, unsuitableNodeName := range unsuitableNodeNames {
				nrtInfo, err := e2enrt.FindFromList(nrtList.Items, unsuitableNodeName)
				Expect(err).ToNot(HaveOccurred(), "missing NRT info for %q", unsuitableNodeName)

				for _, zone := range nrtInfo.Zones {
					name := fmt.Sprintf("unsuitable%d", idx)
					By(fmt.Sprintf("saturating node %q -> %q zone %q", nrtInfo.Name, name, zone.Name))
					padPod, err := makePaddingPod(fxt.Namespace.Name, name, zone, unsuitableFreeRes)
					Expect(err).ToNot(HaveOccurred())

					padPod, err = pinPodTo(padPod, nrtInfo.Name, zone.Name)
					Expect(err).ToNot(HaveOccurred())

					err = fxt.Client.Create(context.TODO(), padPod)
					Expect(err).ToNot(HaveOccurred())
					paddingPods = append(paddingPods, padPod)
				}
			}

			allPaddingPods := append([]*corev1.Pod{}, paddingPods...)
			allPaddingPods = append(allPaddingPods, targetPaddingPods...)

			By("Waiting for padding pods to be ready")
			failedPodIds = e2ewait.ForPaddingPodsRunning(fxt, allPaddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			// TODO: smarter cooldown
			time.Sleep(18 * time.Second)
			for _, unsuitableNodeName := range unsuitableNodeNames {
				dumpNRTForNode(fxt.Client, unsuitableNodeName, "unsuitable")
			}
			dumpNRTForNode(fxt.Client, targetNodeName, "targeted")

			By(fmt.Sprintf("running the test pod requiring: %s", e2ereslist.ToString(requiredRes)))
			pod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			pod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			pod.Spec.Containers[0].Resources.Limits = requiredRes
			pod.Spec.NodeSelector = map[string]string{
				serialconfig.MultiNUMALabel: "2",
			}
			err = fxt.Client.Create(context.TODO(), pod)
			Expect(err).ToNot(HaveOccurred())

			By("check the pod is still pending")
			// TODO: lacking better ways, let's monitor the pod "long enough" and let's check it stays Pending
			// if it stays Pending "long enough" it still means little, but OTOH if it goes Running or Failed we
			// can tell for sure something's wrong
			err = e2ewait.WhileInPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodPending, 10*time.Second, 3)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			By("deleting a placeholder pod pod") // any pod is fine
			targetPaddingPod := targetPaddingPods[0]
			err = fxt.Client.Delete(context.TODO(), targetPaddingPod)
			Expect(err).ToNot(HaveOccurred())

			By("checking the test pod is removed")
			err = e2ewait.ForPodDeleted(fxt.Client, targetPaddingPod.Namespace, targetPaddingPod.Name, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the pod to be scheduled")
			updatedPod, err := e2ewait.ForPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodRunning, 3*time.Minute)
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
		})
	})

	Context("cluster with node/s having two numa zones, and there are enough resources on one node but not in any numa zone when trying to schedule a deployment with burstable pods", func() {
		var nrtCandidates []nrtv1alpha1.NodeResourceTopology
		var targetNodeName string
		var targetNodeNRTInitial *nrtv1alpha1.NodeResourceTopology
		var deployment *appsv1.Deployment
		var reqResources corev1.ResourceList
		var err error

		/*
			1. choose a target node on which the test's burstable pod will run
			2. fully pad the non-target nodes
			3. test step: create a workload with burstable pod
		*/
		BeforeEach(func() {
			const requiredNUMAZones = 2
			By(fmt.Sprintf("filtering available nodes with %d NUMA zones", requiredNUMAZones))
			nrtCandidates = e2enrt.FilterZoneCountEqual(nrts, requiredNUMAZones)

			const neededNodes = 1
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with at least %d NUMA Zones: found %d, needed %d", requiredNUMAZones, len(nrtCandidates), neededNodes))
			}

			nrtCandidateNames := e2enrt.AccumulateNames(nrtCandidates)

			var ok bool
			targetNodeName, ok = nrtCandidateNames.PopAny()
			Expect(ok).To(BeTrue(), "cannot select a node among %#v", nrtCandidateNames.List())
			By(fmt.Sprintf("selecting node to schedule the test pod: %q", targetNodeName))

			targetNodeNRTInitial, err = e2enrt.FindFromList(nrtCandidates, targetNodeName)
			Expect(err).NotTo(HaveOccurred())

			//calculate base load on the target node
			baseload, err := nodes.GetLoad(fxt.K8sClient, targetNodeName)
			Expect(err).ToNot(HaveOccurred(), "missing node load info for %q", targetNodeName)
			klog.Infof(fmt.Sprintf("computed base load: %s", baseload))

			//get maximum available node CPU and Memory
			reqResources := corev1.ResourceList{
				corev1.ResourceCPU:    availableResourceType(*targetNodeNRTInitial, corev1.ResourceCPU),
				corev1.ResourceMemory: availableResourceType(*targetNodeNRTInitial, corev1.ResourceMemory),
			}

			By("prepare the test's pod resources as maximum available resources on the target node with the baselaod deducted")
			baseload.Deduct(reqResources)

			By("padding all other candidate nodes leaving room for the baseload only")
			var paddingPods []*corev1.Pod
			for _, nodeName := range nrtCandidateNames.List() {
				node := &corev1.Node{}
				nodeKey := client.ObjectKey{Name: nodeName}
				err = fxt.Client.Get(context.TODO(), nodeKey, node)
				Expect(err).NotTo(HaveOccurred())

				//calculate base load on the node
				baseload, err := nodes.GetLoad(fxt.K8sClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), "missing node load info for %q", nodeName)
				klog.Infof(fmt.Sprintf("computed base load: %s", baseload))

				//get nrt info of the node
				klog.Infof(fmt.Sprintf("preparing node %q to fit the test case", nodeName))
				nrtInfo, err := e2enrt.FindFromList(nrtCandidates, nodeName)
				Expect(err).ToNot(HaveOccurred(), "missing NRT info for %q", nodeName)

				baseloadRes := corev1.ResourceList{
					corev1.ResourceCPU:    baseload.CPU,
					corev1.ResourceMemory: baseload.Memory,
				}

				paddingRes, err := e2enrt.SaturateNodeUntilLeft(*nrtInfo, baseloadRes)
				Expect(err).ToNot(HaveOccurred(), "could not get padding resources for node %q", nrtInfo.Name)

				for _, zone := range nrtInfo.Zones {
					By(fmt.Sprintf("fully padding node %q zone %q ", nrtInfo.Name, zone.Name))
					padPod := newPaddingPod(nrtInfo.Name, zone.Name, fxt.Namespace.Name, paddingRes[zone.Name])

					padPod, err = pinPodTo(padPod, nrtInfo.Name, zone.Name)
					Expect(err).ToNot(HaveOccurred(), "unable to pin pod %q to zone %q", padPod.Name, zone.Name)

					err = fxt.Client.Create(context.TODO(), padPod)
					Expect(err).ToNot(HaveOccurred())
					paddingPods = append(paddingPods, padPod)
				}
			}

			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")
		})

		It("[test_id:47618][tier2] should properly schedule deployment with burstable pod with no changes in NRTs", func() {
			By("create a deployment with one burstable pod")
			deploymentName := "test-dp"
			var replicas int32 = 1

			podLabels := map[string]string{
				"test": "test-dp",
			}
			nodeSelector := map[string]string{}
			deployment = objects.NewTestDeployment(replicas, podLabels, nodeSelector, fxt.Namespace.Name, deploymentName, objects.PauseImage, []string{objects.PauseCommand}, []string{})
			deployment.Spec.Template.Spec.SchedulerName = serialconfig.Config.SchedulerName
			// make it burstable
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests = reqResources

			klog.Infof("create the bustable test deployment with requests %s", e2ereslist.ToString(reqResources))
			err = fxt.Client.Create(context.TODO(), deployment)
			Expect(err).NotTo(HaveOccurred(), "unable to create deployment %q", deployment.Name)

			By("waiting for deployment to be up&running")
			dpRunningTimeout := 1 * time.Minute
			dpRunningPollInterval := 10 * time.Second
			err = e2ewait.ForDeploymentComplete(fxt.Client, deployment, dpRunningPollInterval, dpRunningTimeout)
			Expect(err).NotTo(HaveOccurred(), "Deployment %q not up&running after %v", deployment.Name, dpRunningTimeout)

			By(fmt.Sprintf("checking deployment pods have been scheduled with the topology aware scheduler %q and in the proper node %q", serialconfig.Config.SchedulerName, targetNodeName))
			pods, err := schedutils.ListPodsByDeployment(fxt.Client, *deployment)
			Expect(err).NotTo(HaveOccurred(), "Unable to get pods from Deployment %q: %v", deployment.Name, err)
			for _, pod := range pods {
				Expect(pod.Spec.NodeName).To(Equal(targetNodeName), "pod %s/%s is scheduled on node %q but expected to be on the target node %q", pod.Namespace, pod.Name, targetNodeName)
				schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
				Expect(err).ToNot(HaveOccurred())
				Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
			}

			targetNodeNRTCurrent, err := e2enrt.FindFromList(nrtCandidates, targetNodeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(e2enrt.CheckEqualAvailableResources(*targetNodeNRTInitial, *targetNodeNRTCurrent)).To(BeTrue())
		})

	})

})
