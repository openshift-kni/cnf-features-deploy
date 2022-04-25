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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	schedutils "github.com/openshift-kni/numaresources-operator/test/e2e/sched/utils"
	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	e2enrt "github.com/openshift-kni/numaresources-operator/test/utils/noderesourcetopologies"
	"github.com/openshift-kni/numaresources-operator/test/utils/nodes"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
	e2ewait "github.com/openshift-kni/numaresources-operator/test/utils/objects/wait"
	e2epadder "github.com/openshift-kni/numaresources-operator/test/utils/padder"
)

var _ = Describe("[serial][disruptive][scheduler] numaresources workload unschedulable", func() {
	var fxt *e2efixture.Fixture
	var padder *e2epadder.Padder
	var nrtList nrtv1alpha1.NodeResourceTopologyList
	var nrts []nrtv1alpha1.NodeResourceTopology

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())

		var err error
		fxt, err = e2efixture.Setup("e2e-test-workload-unschedulable")
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

	Context("with no suitable node", func() {
		var requiredRes corev1.ResourceList
		BeforeEach(func() {
			requiredNUMAZones := 2
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", requiredNUMAZones))
			nrtCandidates := e2enrt.FilterZoneCountEqual(nrts, requiredNUMAZones)

			neededNodes := 1
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with 2 NUMA Zones: found %d, needed %d", len(nrtCandidates), neededNodes))
			}

			nrtCandidateNames := e2enrt.AccumulateNames(nrtCandidates)

			//TODO: we should calculate requiredRes from NUMA zones in cluster nodes instead.
			requiredRes = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("16"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			}

			By("Padding selected node")
			// TODO This should be calculated as 3/4 of requiredRes
			paddingRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("12"),
				corev1.ResourceMemory: resource.MustParse("12Gi"),
			}

			var paddingPods []*corev1.Pod
			for _, nodeName := range nrtCandidateNames.List() {

				nrtInfo, err := e2enrt.FindFromList(nrtCandidates, nodeName)
				Expect(err).NotTo(HaveOccurred(), "missing NRT info for %q", nodeName)

				baseload, err := nodes.GetLoad(fxt.K8sClient, nodeName)
				Expect(err).NotTo(HaveOccurred(), "cannot get base load for %q", nodeName)

				for idx, zone := range nrtInfo.Zones {
					zoneRes := paddingRes.DeepCopy() // extra safety
					if idx == 0 {                    // any zone is fine
						baseload.Apply(zoneRes)
					}

					podName := fmt.Sprintf("padding%s-%d", nodeName, idx)
					padPod, err := makePaddingPod(fxt.Namespace.Name, podName, zone, zoneRes)
					Expect(err).NotTo(HaveOccurred(), "unable to create padding pod %q on zone", podName, zone.Name)

					padPod, err = pinPodTo(padPod, nodeName, zone.Name)
					Expect(err).NotTo(HaveOccurred(), "unable to pin pod %q to zone", podName, zone.Name)

					err = fxt.Client.Create(context.TODO(), padPod)
					Expect(err).NotTo(HaveOccurred(), "unable to create pod %q on zone", podName, zone.Name)

					paddingPods = append(paddingPods, padPod)
				}
			}

			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")
		})

		It("[test_id:47617][tier2][unsched] workload requests guaranteed pod resources available on one node but not on a single numa", func() {

			By("Scheduling the testing pod")
			pod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			pod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			pod.Spec.Containers[0].Resources.Limits = requiredRes

			err := fxt.Client.Create(context.TODO(), pod)
			Expect(err).NotTo(HaveOccurred(), "unable to create pod %q", pod.Name)

			err = e2ewait.WhileInPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodPending, 10*time.Second, 3)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("checking the pod was scheduled with the topology aware scheduler %q", serialconfig.Config.SchedulerName))
			isFailed, err := nrosched.CheckPODSchedulingFailedForAlignment(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(isFailed).To(BeTrue(), "pod %s/%s with scheduler %s did NOT fail", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
		})

		It("[test_id:48963][tier2][unsched] a deployment with a guaranteed pod resources available on one node but not on a single numa", func() {

			By("Scheduling the testing deployment")
			deploymentName := "test-dp"
			var replicas int32 = 1

			podLabels := map[string]string{
				"test": "test-deployment",
			}
			nodeSelector := map[string]string{}
			deployment := objects.NewTestDeployment(replicas, podLabels, nodeSelector, fxt.Namespace.Name, deploymentName, objects.PauseImage, []string{objects.PauseCommand}, []string{})
			deployment.Spec.Template.Spec.SchedulerName = serialconfig.Config.SchedulerName
			deployment.Spec.Template.Spec.Containers[0].Resources.Limits = requiredRes

			err := fxt.Client.Create(context.TODO(), deployment)
			Expect(err).NotTo(HaveOccurred(), "unable to create deployment %q", deployment.Name)

			By(fmt.Sprintf("checking deployment pods have been scheduled with the topology aware scheduler %q ", serialconfig.Config.SchedulerName))
			pods, err := schedutils.ListPodsByDeployment(fxt.Client, *deployment)
			Expect(err).NotTo(HaveOccurred(), "Unable to get pods from Deployment %q:  %v", deployment.Name, err)

			for _, pod := range pods {
				isFailed, err := nrosched.CheckPODSchedulingFailedForAlignment(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
				if err != nil {
					_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(isFailed).To(BeTrue(), "pod %s/%s with scheduler %s did NOT fail", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
			}
		})

		It("[test_id:48962][tier2][unsched] a daemonset with a guaranteed pod resources available on one node but not on a single numa", func() {

			By("Scheduling the testing daemonset")
			dsName := "test-ds"

			podLabels := map[string]string{
				"test": "test-daemonset",
			}
			nodeSelector := map[string]string{}
			ds := objects.NewTestDaemonset(podLabels, nodeSelector, fxt.Namespace.Name, dsName, objects.PauseImage, []string{objects.PauseCommand}, []string{})
			ds.Spec.Template.Spec.SchedulerName = serialconfig.Config.SchedulerName
			ds.Spec.Template.Spec.Containers[0].Resources.Limits = requiredRes

			err := fxt.Client.Create(context.TODO(), ds)
			Expect(err).NotTo(HaveOccurred(), "unable to create deployment %q", ds.Name)

			By(fmt.Sprintf("checking daemonset pods have been scheduled with the topology aware scheduler %q ", serialconfig.Config.SchedulerName))
			pods, err := schedutils.ListPodsByDaemonset(fxt.Client, *ds)
			Expect(err).ToNot(HaveOccurred(), "Unable to get pods from daemonset %q:  %v", ds.Name, err)

			for _, pod := range pods {
				isFailed, err := nrosched.CheckPODSchedulingFailedForAlignment(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
				if err != nil {
					_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(isFailed).To(BeTrue(), "pod %s/%s with scheduler %s did NOT fail", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
			}
		})
	})
	Context("with at least two nodes with two numa zones and enough resources in one numazone", func() {
		It("[test_id:47592][tier2][unsched] a daemonset with a guaranteed pod resources available on one node/one single numa zone but not in any other node", func() {
			requiredNUMAZones := 2
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", requiredNUMAZones))
			nrtCandidates := e2enrt.FilterZoneCountEqual(nrts, requiredNUMAZones)

			neededNodes := 2
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with 2 NUMA Zones: found %d, needed %d", len(nrtCandidates), neededNodes))
			}

			nrtCandidateNames := e2enrt.AccumulateNames(nrtCandidates)

			targetNodeName, ok := nrtCandidateNames.PopAny()
			Expect(ok).To(BeTrue(), "unable to get targetNodeName")

			//TODO: we should calculate requiredRes from NUMA zones in cluster nodes instead.
			requiredRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("16"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			}

			By("Padding non selected nodes")
			// TODO This should be calculated as 3/4 of requiredRes
			paddingRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("12"),
				corev1.ResourceMemory: resource.MustParse("12Gi"),
			}

			var paddingPods []*corev1.Pod
			for nodeIdx, nodeName := range nrtCandidateNames.List() {

				nrtInfo, err := e2enrt.FindFromList(nrtCandidates, nodeName)
				Expect(err).NotTo(HaveOccurred(), "missing NRT info for %q", nodeName)

				baseload, err := nodes.GetLoad(fxt.K8sClient, nodeName)
				Expect(err).NotTo(HaveOccurred(), "cannot get base load for %q", nodeName)

				for zoneIdx, zone := range nrtInfo.Zones {
					zoneRes := paddingRes.DeepCopy() // extra safety
					if zoneIdx == 0 {                // any zone is fine
						baseload.Apply(zoneRes)
					}

					podName := fmt.Sprintf("padding%d-%d", nodeIdx, zoneIdx)
					padPod, err := makePaddingPod(fxt.Namespace.Name, podName, zone, zoneRes)
					Expect(err).NotTo(HaveOccurred(), "unable to create padding pod %q on zone", podName, zone.Name)

					padPod, err = pinPodTo(padPod, nodeName, zone.Name)
					Expect(err).NotTo(HaveOccurred(), "unable to pin pod %q to zone", podName, zone.Name)

					err = fxt.Client.Create(context.TODO(), padPod)
					Expect(err).NotTo(HaveOccurred(), "unable to create pod %q on zone", podName, zone.Name)

					paddingPods = append(paddingPods, padPod)
				}
			}

			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			By("Scheduling the testing daemonset")
			dsName := "test-ds"

			podLabels := map[string]string{
				"test": "test-daemonset",
			}
			nodeSelector := map[string]string{}
			ds := objects.NewTestDaemonset(podLabels, nodeSelector, fxt.Namespace.Name, dsName, objects.PauseImage, []string{objects.PauseCommand}, []string{})
			ds.Spec.Template.Spec.SchedulerName = serialconfig.Config.SchedulerName
			ds.Spec.Template.Spec.Containers[0].Resources.Limits = requiredRes

			err := fxt.Client.Create(context.TODO(), ds)
			Expect(err).NotTo(HaveOccurred(), "unable to create deployment %q", ds.Name)

			By(fmt.Sprintf("checking daemonset pods have been scheduled with the topology aware scheduler %q ", serialconfig.Config.SchedulerName))
			pods, err := schedutils.ListPodsByDaemonset(fxt.Client, *ds)
			Expect(err).ToNot(HaveOccurred(), "Unable to get pods from daemonset %q:  %v", ds.Name, err)

			//TODO: should not we wait until pods have at least been scheduled? how?

			By(fmt.Sprintf("checking only daemonset pod in targetNode:%q is up and running", targetNodeName))
			podRunningTimeout := 3 * time.Minute
			for _, pod := range pods {
				if pod.Spec.NodeName == targetNodeName {
					scheduledWithTAS, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
					if err != nil {
						_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
					}
					Expect(err).ToNot(HaveOccurred())
					Expect(scheduledWithTAS).To(BeTrue(), "pod %s/%s was NOT scheduled with  %s", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)

					_, err = e2ewait.ForPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodRunning, podRunningTimeout)
					Expect(err).ToNot(HaveOccurred(), "unable to get pod %s/%s to be Running after %v", pod.Namespace, pod.Name, podRunningTimeout)

				} else {
					isFailed, err := nrosched.CheckPODSchedulingFailedForAlignment(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
					if err != nil {
						_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
					}
					Expect(err).ToNot(HaveOccurred())
					Expect(isFailed).To(BeTrue(), "pod %s/%s with scheduler %s did NOT fail", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
				}

			}
		})
	})
	Context("with at least one node", func() {
		It("[test_id:47616][tier2][tmscope:pod] pod with two containers each on one numa zone can NOT be scheduled", func() {

			// Requirements:
			// Need at least this nodes
			neededNodes := 1
			// with at least this number of numa zones
			requiredNUMAZones := 2
			// and with this policy
			tmPolicy := nrtv1alpha1.SingleNUMANodePodLevel

			// filter by policy
			nrtCandidates := e2enrt.FilterTopologyManagerPolicy(nrtList.Items, tmPolicy)
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with policy %q - found %d", string(tmPolicy), len(nrtCandidates)))
			}

			// Filter by number of numa zones
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", requiredNUMAZones))
			nrtCandidates = e2enrt.FilterZoneCountEqual(nrtCandidates, requiredNUMAZones)
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with 2 NUMA Zones: found %d, needed %d", len(nrtCandidates), neededNodes))
			}

			// filter by resources on each numa zone
			requiredResCnt1 := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			}

			requiredResCnt2 := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("5"),
				corev1.ResourceMemory: resource.MustParse("5Gi"),
			}

			By("filtering available nodes with allocatable resources on at least one NUMA zone that can match request")
			nrtCandidates = filterNRTsEachRequestOnADifferentZone(nrtCandidates, requiredResCnt1, requiredResCnt2)
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with NUMA zones each of them can match requests: found %d, needed: %d", len(nrtCandidates), neededNodes))
			}

			// After filter get one of the candidate nodes left
			nrtCandidateNames := e2enrt.AccumulateNames(nrtCandidates)
			targetNodeName, ok := nrtCandidateNames.PopAny()
			Expect(ok).To(BeTrue(), "cannot select a target node among %#v", nrtCandidateNames.List())
			By(fmt.Sprintf("selecting node to schedule the pod: %q", targetNodeName))

			By("Padding all other candidate nodes")
			var paddingPods []*corev1.Pod
			for nodeIdx, nodeName := range nrtCandidateNames.List() {

				nrtInfo, err := e2enrt.FindFromList(nrtCandidates, nodeName)
				Expect(err).NotTo(HaveOccurred(), "missing NRT info for %q", nodeName)

				baseload, err := nodes.GetLoad(fxt.K8sClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), "missing node load info for %q", nodeName)

				baseloadResources := corev1.ResourceList{
					corev1.ResourceCPU:    baseload.CPU,
					corev1.ResourceMemory: baseload.Memory,
				}

				paddingResources, err := e2enrt.SaturateNodeUntilLeft(*nrtInfo, baseloadResources)
				Expect(err).ToNot(HaveOccurred(), "could not get padding resources for node %q", nrtInfo.Name)

				for zoneIdx, zone := range nrtInfo.Zones {
					podName := fmt.Sprintf("padding%d-%d", nodeIdx, zoneIdx)
					padPod := newPaddingPod(nodeName, zone.Name, fxt.Namespace.Name, paddingResources[zone.Name])

					padPod, err = pinPodTo(padPod, nodeName, zone.Name)
					Expect(err).NotTo(HaveOccurred(), "unable to pin pod %q to zone %q", podName, zone.Name)

					err = fxt.Client.Create(context.TODO(), padPod)
					Expect(err).NotTo(HaveOccurred(), "unable to create pod %q on zone %q", podName, zone.Name)

					paddingPods = append(paddingPods, padPod)
				}
			}

			By("Padding target node")
			//calculate base load on the target node
			targetNodeBaseLoad, err := nodes.GetLoad(fxt.K8sClient, targetNodeName)
			Expect(err).ToNot(HaveOccurred(), "missing node load info for %q", targetNodeName)

			// Pad the zones so no one could handle both containers
			// so we should left enought resources on each zone to accomodate
			// the "biggest" ( in term of resources) container but not the
			// sum of both
			paddingResources := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("6"),
				corev1.ResourceMemory: resource.MustParse("6Gi"),
			}

			targetNrt, err := e2enrt.FindFromList(nrtCandidates, targetNodeName)
			Expect(err).NotTo(HaveOccurred(), "missing NRT info for targetNode %q", targetNodeName)

			for zoneIdx, zone := range targetNrt.Zones {
				zoneRes := paddingResources.DeepCopy()
				if zoneIdx == 0 { //any zone would do it, we just choose one.
					targetNodeBaseLoad.Apply(zoneRes)
				}

				paddingRes, err := e2enrt.SaturateZoneUntilLeft(zone, zoneRes)
				Expect(err).NotTo(HaveOccurred(), "could not get padding resources for node %q", targetNrt.Name)

				podName := fmt.Sprintf("padding-tgt-%d", zoneIdx)
				padPod := newPaddingPod(targetNodeName, zone.Name, fxt.Namespace.Name, paddingRes)

				padPod, err = pinPodTo(padPod, targetNodeName, zone.Name)
				Expect(err).NotTo(HaveOccurred(), "unable to pin pod %q to zone %q", podName, zone.Name)

				err = fxt.Client.Create(context.TODO(), padPod)
				Expect(err).NotTo(HaveOccurred(), "unable to create pod %q on zone %q", podName, zone.Name)

				paddingPods = append(paddingPods, padPod)
			}

			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			By("Scheduling the testing pod")
			pod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			pod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			pod.Spec.Containers = append(pod.Spec.Containers, pod.Spec.Containers[0])
			pod.Spec.Containers[0].Name = pod.Name + "-cnt-0"
			pod.Spec.Containers[0].Resources.Limits = requiredResCnt1
			pod.Spec.Containers[1].Name = pod.Name + "cnt-1"
			pod.Spec.Containers[1].Resources.Limits = requiredResCnt2

			err = fxt.Client.Create(context.TODO(), pod)
			Expect(err).NotTo(HaveOccurred(), "unable to create pod %q", pod.Name)

			interval := 10 * time.Second
			By(fmt.Sprintf("Checking pod %q keeps in %q state for at least %v seconds ...", pod.Name, string(corev1.PodPending), interval*3))
			err = e2ewait.WhileInPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodPending, interval, 3)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("checking the pod was scheduled with the topology aware scheduler %q", serialconfig.Config.SchedulerName))
			isFailed, err := nrosched.CheckPODSchedulingFailedForAlignment(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			if !isFailed {
				_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
			}
			Expect(isFailed).To(BeTrue(), "pod %s/%s with scheduler %s did NOT fail", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
		})
	})

	// other than the other tests, here we expect all the worker nodes (including none-bm hosts) to be padded
	Context("with zero suitable nodes", func() {
		It("[test_id:47615][tier2][unsched] a deployment with multiple guaranteed pods resources that doesn't fit at the NUMA level", func() {
			requiredNUMAZones := 2
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", requiredNUMAZones))
			nrtCandidates := e2enrt.FilterZoneCountEqual(nrts, requiredNUMAZones)

			neededNodes := 1
			numOfnrtCandidates := len(nrtCandidates)
			if numOfnrtCandidates < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with 2 NUMA Zones: found %d, needed %d", len(nrtCandidates), neededNodes))
			}
			By("padding all the nodes")
			requiredRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			}

			padUntilRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			}
			err := padder.Nodes(len(nrts)).UntilAvailableIsResourceList(padUntilRes).Pad(time.Minute*2, e2epadder.PaddingOptions{})
			Expect(err).ToNot(HaveOccurred())

			nrtInitialList, err := e2enrt.GetUpdated(fxt.Client, nrtv1alpha1.NodeResourceTopologyList{}, time.Second*10)
			Expect(err).ToNot(HaveOccurred())

			By("creating a deployment")
			dpName := "test-dp"
			schedulerName := nrosched.NROSchedulerName
			replicas := int32(6)
			podLabels := map[string]string{
				"test": "test-dp",
			}

			podSpec := corev1.PodSpec{
				SchedulerName: schedulerName,
				Containers: []corev1.Container{
					{
						Name:    dpName + "-cnt1",
						Image:   objects.PauseImage,
						Command: []string{objects.PauseCommand},
						Resources: corev1.ResourceRequirements{
							Requests: requiredRes,
							Limits:   requiredRes,
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyAlways,
			}
			dp := objects.NewTestDeploymentWithPodSpec(replicas, podLabels, map[string]string{}, fxt.Namespace.Name, dpName, podSpec)

			err = fxt.Client.Create(context.TODO(), dp)
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("checking deployment pods have been scheduled with the topology aware scheduler %q ", schedulerName))
			pods, err := schedutils.ListPodsByDeployment(fxt.Client, *dp)
			Expect(err).ToNot(HaveOccurred(), "unable to get pods from deployment %q:  %v", dp.Name, err)

			allFailed := true
			for _, pod := range pods {
				isFailed, err := nrosched.CheckPODSchedulingFailed(fxt.K8sClient, pod.Namespace, pod.Name, schedulerName)
				if err != nil {
					_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
				}
				Expect(err).ToNot(HaveOccurred())
				allFailed = allFailed && isFailed
				if !isFailed {
					klog.Warningf("pod %s/%s with scheduler %s did NOT fail", pod.Namespace, pod.Name, schedulerName)
					continue
				}
			}
			Expect(allFailed).To(BeTrue(), "some pods are running, but we expect all of them to fail")

			By("updating deployment in such way that some pods will fit into NUMA nodes")
			err = fxt.Client.Get(context.TODO(), client.ObjectKeyFromObject(dp), dp)
			Expect(err).ToNot(HaveOccurred())

			// 6 pods in total (replica is 6)
			// we should expect 'expectedReadyReplicas' out of 6 pods to be running
			expectedReadyReplicas := calcExpectedReadyReplicas(numOfnrtCandidates, len(nrts)-numOfnrtCandidates)
			klog.Infof("expecting %d out of %d to be running", expectedReadyReplicas, replicas)

			numaLevelFitRequiredRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			}
			cnt := &dp.Spec.Template.Spec.Containers[0]
			cnt.Resources.Limits = numaLevelFitRequiredRes
			cnt.Resources.Requests = numaLevelFitRequiredRes

			err = fxt.Client.Update(context.TODO(), dp)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for some of the pods to be running")
			dpKey := client.ObjectKeyFromObject(dp)
			Eventually(func() bool {
				err = fxt.Client.Get(context.TODO(), dpKey, dp)
				Expect(err).ToNot(HaveOccurred())

				if dp.Status.ReadyReplicas != expectedReadyReplicas {
					klog.Warningf("deployment: %q has a wrong number of ready replicas, expected: %d got %d", dpKey.String(), expectedReadyReplicas, dp.Status.ReadyReplicas)
					return false
				}
				return true
			}, time.Minute*5, time.Second*30).Should(BeTrue(), "deployment %q failed to have %d running replicas within the defined period", dpKey.String(), expectedReadyReplicas)

			By("checking NRT objects updated accordingly")
			nrtPostDpCreateList, err := e2enrt.GetUpdated(fxt.Client, nrtInitialList, time.Second*10)
			Expect(err).ToNot(HaveOccurred())

			for _, initialNrt := range nrtInitialList.Items {
				nrtPostDpCreate, err := e2enrt.FindFromList(nrtPostDpCreateList.Items, initialNrt.Name)
				Expect(err).ToNot(HaveOccurred())

				_, err = e2enrt.CheckZoneConsumedResourcesAtLeast(initialNrt, *nrtPostDpCreate, numaLevelFitRequiredRes)
				Expect(err).ToNot(HaveOccurred())
			}
		})
	})
})

// Return only those NRTs where each request could fit into a different zone.
func filterNRTsEachRequestOnADifferentZone(nrts []nrtv1alpha1.NodeResourceTopology, r1, r2 corev1.ResourceList) []nrtv1alpha1.NodeResourceTopology {
	ret := []nrtv1alpha1.NodeResourceTopology{}
	for _, nrt := range nrts {
		if nrtCanAccomodateEachRequestOnADifferentZone(nrt, r1, r2) {
			ret = append(ret, nrt)
		}
	}
	return ret
}

// returns true if nrt can accomodate r1 and r2 in one of its two first zones.
func nrtCanAccomodateEachRequestOnADifferentZone(nrt nrtv1alpha1.NodeResourceTopology, r1, r2 corev1.ResourceList) bool {
	if len(nrt.Zones) < 2 {
		return false
	}
	return eachRequestFitsOnADifferentZone(nrt.Zones[0], nrt.Zones[1], r1, r2)
}

//returns true if r1 fits on z1 AND r2 on z2 or the other way around
func eachRequestFitsOnADifferentZone(z1, z2 nrtv1alpha1.Zone, r1, r2 corev1.ResourceList) bool {
	return (e2enrt.ZoneResourcesMatchesRequest(z1.Resources, r1) && e2enrt.ZoneResourcesMatchesRequest(z2.Resources, r2)) ||
		(e2enrt.ZoneResourcesMatchesRequest(z1.Resources, r2) && e2enrt.ZoneResourcesMatchesRequest(z2.Resources, r1))
}

func calcExpectedReadyReplicas(numOfMultiNUMACandidates, numOfSingleNUMACandidates int) int32 {
	// each NUMA should hold a single pod, so we should expect the number of replicas to be equal to number of available NUMAs
	var expectedReadyReplicas int32
	// multiNUMACandidates nodes has 2 NUMAs each
	expectedReadyReplicas += int32(numOfMultiNUMACandidates * 2)
	// multiNUMACandidates nodes has 1 NUMA each
	expectedReadyReplicas += int32(numOfSingleNUMACandidates)

	return expectedReadyReplicas
}
