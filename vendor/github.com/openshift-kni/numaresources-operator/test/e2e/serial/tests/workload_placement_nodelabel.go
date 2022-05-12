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
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	e2ereslist "github.com/openshift-kni/numaresources-operator/internal/resourcelist"
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

type getNodeAffinityFunc func(labelName string, labelValue []string, selectOperator corev1.NodeSelectorOperator) *corev1.Affinity

var _ = Describe("[serial][disruptive][scheduler] numaresources workload placement considering node selector", func() {
	var fxt *e2efixture.Fixture
	var padder *e2epadder.Padder
	var nrtList nrtv1alpha1.NodeResourceTopologyList
	var nrts []nrtv1alpha1.NodeResourceTopology

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-workload-placement-nodesel")
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
	Context("with two labeled nodes with two NUMA zones", func() {
		labelName := "size"
		labelValueMedium := "medium"
		labelValueLarge := "large"
		var targetNodeName, alternativeNodeName string
		var requiredRes corev1.ResourceList
		var nrtCandidates []nrtv1alpha1.NodeResourceTopology
		var targetNodeNRTInitial *nrtv1alpha1.NodeResourceTopology
		BeforeEach(func() {
			requiredNUMAZones := 2
			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", requiredNUMAZones))
			nrtCandidates = e2enrt.FilterZoneCountEqual(nrts, requiredNUMAZones)

			neededNodes := 2
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with %d NUMA Zones: found %d, needed %d", requiredNUMAZones, len(nrtCandidates), neededNodes))
			}

			// TODO: this should be >= 5x baseload
			requiredRes = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("16"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			}
			// WARNING: This should be calculated as 3/4 of requiredRes
			paddingRes := corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("12"),
				corev1.ResourceMemory: resource.MustParse("12Gi"),
			}

			By("filtering available nodes with allocatable resources on at least one NUMA zone that can match request")
			nrtCandidates = e2enrt.FilterAnyZoneMatchingResources(nrtCandidates, requiredRes)
			if len(nrtCandidates) < neededNodes {
				Skip(fmt.Sprintf("not enough nodes with NUMA zones each of them can match requests: found %d, needed: %d, request: %v", len(nrtCandidates), neededNodes, requiredRes))
			}
			nrtCandidateNames := e2enrt.AccumulateNames(nrtCandidates)

			var ok bool
			targetNodeName, ok = nrtCandidateNames.PopAny()
			Expect(ok).To(BeTrue(), "cannot select a target node among %#v", nrtCandidateNames.List())
			By(fmt.Sprintf("selecting target node we expect the pod will be scheduled into: %q", targetNodeName))

			alternativeNodeName, ok = nrtCandidateNames.PopAny()
			Expect(ok).To(BeTrue(), "cannot select an alternative target node among %#v", nrtCandidateNames.List())
			By(fmt.Sprintf("selecting alternative node candidate for the scheduling: %q", alternativeNodeName))

			// we need to also pad one of the labeled nodes.
			nrtToPadNames := append(nrtCandidateNames.List(), alternativeNodeName)
			By(fmt.Sprintf("Padding all other candidate nodes: %v", nrtToPadNames))

			var paddingPods []*corev1.Pod
			for nIdx, nodeName := range nrtToPadNames {

				nrtInfo, err := e2enrt.FindFromList(nrtCandidates, nodeName)
				Expect(err).NotTo(HaveOccurred(), "missing NRT info for %q", nodeName)

				baseload, err := nodes.GetLoad(fxt.K8sClient, nodeName)
				Expect(err).NotTo(HaveOccurred(), "cannot get the base load for %q", nodeName)

				for zIdx, zone := range nrtInfo.Zones {
					zoneRes := paddingRes.DeepCopy() // to be extra safe
					if zIdx == 0 {                   // any zone is fine
						baseload.Apply(zoneRes)
					}

					podName := fmt.Sprintf("padding%d-%d", nIdx, zIdx)
					padPod, err := makePaddingPod(fxt.Namespace.Name, podName, zone, zoneRes)
					Expect(err).NotTo(HaveOccurred(), "unable to create padding pod %q on zone %q", podName, zone.Name)

					padPod, err = pinPodTo(padPod, nodeName, zone.Name)
					Expect(err).NotTo(HaveOccurred(), "unable to pin pod %q to zone %q", podName, zone.Name)

					err = fxt.Client.Create(context.TODO(), padPod)
					Expect(err).NotTo(HaveOccurred(), "unable to create pod %q on zone %q", podName, zone.Name)

					paddingPods = append(paddingPods, padPod)
				}
			}
			By("Waiting for padding pods to be ready")
			failedPodIds := e2ewait.ForPaddingPodsRunning(fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			var err error
			targetNodeNRTInitial, err = e2enrt.FindFromList(nrtCandidates, targetNodeName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("[test_id:47598][tier2] should place the pod in the node with available resources in one NUMA zone and fulfilling node selector", func() {
			By(fmt.Sprintf("Labeling nodes %q and %q with label %q:%q", targetNodeName, alternativeNodeName, labelName, labelValueMedium))

			unlabelTarget, err := labelNodeWithValue(fxt.Client, labelName, labelValueMedium, targetNodeName)
			Expect(err).NotTo(HaveOccurred(), "unable to label node %q", targetNodeName)
			defer func() {
				err := unlabelTarget()
				if err != nil {
					klog.Errorf("Error while trying to unlabel node %q. %v", targetNodeName, err)
				}
			}()

			unlabelAlternative, err := labelNodeWithValue(fxt.Client, labelName, labelValueMedium, alternativeNodeName)
			Expect(err).NotTo(HaveOccurred(), "unable to label node %q", alternativeNodeName)
			defer func() {
				err := unlabelAlternative()
				if err != nil {
					klog.Errorf("Error while trying to unlabel node %q. %v", alternativeNodeName, err)
				}
			}()
			By("Scheduling the testing pod")
			pod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			pod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			pod.Spec.Containers[0].Resources.Limits = requiredRes
			pod.Spec.NodeSelector = map[string]string{
				labelName: labelValueMedium,
			}

			err = fxt.Client.Create(context.TODO(), pod)
			Expect(err).NotTo(HaveOccurred(), "unable to create pod %q", pod.Name)

			By("waiting for pod to be running")
			updatedPod, err := e2ewait.ForPodPhase(fxt.Client, pod.Namespace, pod.Name, corev1.PodRunning, 1*time.Minute)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name)
			}
			Expect(err).NotTo(HaveOccurred())

			By("checking the pod has been scheduled in the proper node")
			Expect(updatedPod.Spec.NodeName).To(Equal(targetNodeName))

			By(fmt.Sprintf("checking the pod was scheduled with the topology aware scheduler %q", serialconfig.Config.SchedulerName))
			schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)

			By("Verifing the NRT statistics are updated")
			targetNodeNRTCurrent, err := e2enrt.FindFromList(nrtCandidates, targetNodeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(e2enrt.CheckEqualAvailableResources(*targetNodeNRTInitial, *targetNodeNRTCurrent)).To(BeTrue(), "target node %q initial resources and current resources are different", targetNodeName)
		})

		Context("label two nodes with different label values but both matching the node affinity of the deployment pod of the test", func() {
			var unlabelTarget, unlabelAlternative func() error
			nodesUnlabeled := false
			BeforeEach(func() {
				By(fmt.Sprintf("Labeling target node %q with label %q:%q and the alternative node %q with label %q:%q", targetNodeName, labelName, labelValueLarge, alternativeNodeName, labelName, labelValueMedium))

				var err error
				unlabelTarget, err = labelNodeWithValue(fxt.Client, labelName, labelValueLarge, targetNodeName)
				Expect(err).NotTo(HaveOccurred(), "unable to label node %q", targetNodeName)

				unlabelAlternative, err = labelNodeWithValue(fxt.Client, labelName, labelValueMedium, alternativeNodeName)
				Expect(err).NotTo(HaveOccurred(), "unable to label node %q", alternativeNodeName)
			})

			AfterEach(func() {
				if !nodesUnlabeled {
					/*if we are here this means one of these:
					1. the test failed before getting to the step where it removes the labels
					2. the test failed to remove the labels during the test's check so try again here
					Note that unlabeling an already unlabeled node will not result in an error,
					so this condition is only to avoid extra minor operations
					*/
					err := unlabelTarget()
					if err != nil {
						klog.Errorf("Error while trying to unlabel node %q. %v", targetNodeName, err)
					}
					err = unlabelAlternative()
					if err != nil {
						klog.Errorf("Error while trying to unlabel node %q. %v", alternativeNodeName, err)
					}
				}
			})

			DescribeTable("[tier2] a guaranteed deployment pod with nodeAffinity should be scheduled on one NUMA zone on a matching labeled node with enough resources",
				func(getNodeAffFunc getNodeAffinityFunc) {
					affinity := getNodeAffFunc(labelName, []string{labelValueLarge, labelValueMedium}, corev1.NodeSelectorOpIn)
					By(fmt.Sprintf("create a deployment with one guaranteed pod with node affinity property: %+v ", affinity.NodeAffinity))
					deploymentName := "test-dp"
					var replicas int32 = 1

					podLabels := map[string]string{
						"test": "test-dp",
					}
					deployment := objects.NewTestDeployment(replicas, podLabels, nil, fxt.Namespace.Name, deploymentName, objects.PauseImage, []string{objects.PauseCommand}, []string{})
					deployment.Spec.Template.Spec.SchedulerName = serialconfig.Config.SchedulerName
					deployment.Spec.Template.Spec.Containers[0].Resources.Limits = requiredRes
					deployment.Spec.Template.Spec.Affinity = affinity
					klog.Infof("create the test deployment with requests %s", e2ereslist.ToString(requiredRes))
					err := fxt.Client.Create(context.TODO(), deployment)
					Expect(err).NotTo(HaveOccurred(), "unable to create deployment %q", deployment.Name)

					By("waiting for deployment to be up & running")
					dpRunningTimeout := 1 * time.Minute
					dpRunningPollInterval := 10 * time.Second
					err = e2ewait.ForDeploymentComplete(fxt.Client, deployment, dpRunningPollInterval, dpRunningTimeout)
					Expect(err).NotTo(HaveOccurred(), "Deployment %q not up & running after %v", deployment.Name, dpRunningTimeout)

					By(fmt.Sprintf("checking deployment pods have been scheduled with the topology aware scheduler %q and in the proper node %q", serialconfig.Config.SchedulerName, targetNodeName))
					pods, err := schedutils.ListPodsByDeployment(fxt.Client, *deployment)
					Expect(err).NotTo(HaveOccurred(), "Unable to get pods from Deployment %q: %v", deployment.Name, err)
					for _, pod := range pods {
						Expect(pod.Spec.NodeName).To(Equal(targetNodeName), "pod %s/%s is scheduled on node %q but expected to be on the target node %q", pod.Namespace, pod.Name, targetNodeName)
						schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
						Expect(err).ToNot(HaveOccurred())
						Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
					}

					By("Verifing the NRT statistics are updated")
					targetNodeNRTCurrent, err := e2enrt.FindFromList(nrtCandidates, targetNodeName)
					Expect(err).NotTo(HaveOccurred())
					Expect(e2enrt.CheckEqualAvailableResources(*targetNodeNRTInitial, *targetNodeNRTCurrent)).To(BeTrue(), "target node %q initial resources and current resources are different", targetNodeName)

					By("unlabel nodes during execution and check that the test's pod was not evicted due to shaked matching criteria")
					nodesUnlabeled = true
					err = unlabelTarget()
					//if at least on of the unlabling failed, set nodesUnlabeled to false to try again in afterEach
					if err != nil {
						nodesUnlabeled = false
						klog.Errorf("Error while trying to unlabel node %q. %v", targetNodeName, err)
					}
					err = unlabelAlternative()
					if err != nil {
						nodesUnlabeled = false
						klog.Errorf("Error while trying to unlabel node %q. %v", alternativeNodeName, err)
					}

					//check that it didn't stop running for some time
					By(fmt.Sprintf("ensuring the deployment %q keep being ready", deployment.Name))
					Eventually(func() bool {
						updatedDp := &appsv1.Deployment{}
						err := fxt.Client.Get(context.TODO(), client.ObjectKeyFromObject(deployment), updatedDp)
						Expect(err).ToNot(HaveOccurred())
						return e2ewait.IsDeploymentComplete(deployment, &updatedDp.Status)
					}, time.Second*30, time.Second*5).Should(BeTrue(), "deployment %q became unready", deployment.Name)
				},
				Entry("[test_id:47597] should be able to schedule pod with affinity property requiredDuringSchedulingIgnoredDuringExecution on the available node with feasible numa zone", createNodeAffinityRequiredDuringSchedulingIgnoredDuringExecution),
				Entry("[test_id:49843] should be able to schedule pod with affinity property prefferdDuringSchedulingIgnoredDuringExecution on the available node with feasible numa zone", createNodeAffinityPreferredDuringSchedulingIgnoredDuringExecution),
			)
		})
	})
})

func createNodeAffinityRequiredDuringSchedulingIgnoredDuringExecution(labelName string, labelValue []string, selectOperator corev1.NodeSelectorOperator) *corev1.Affinity {
	nodeSelReq := &corev1.NodeSelectorRequirement{
		Key:      labelName,
		Operator: selectOperator,
		Values:   labelValue,
	}

	nodeSelTerm := &corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{*nodeSelReq},
		MatchFields:      []corev1.NodeSelectorRequirement{},
	}

	aff := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{*nodeSelTerm},
			},
		},
	}
	return aff
}

func createNodeAffinityPreferredDuringSchedulingIgnoredDuringExecution(labelName string, labelValue []string, selectOperator corev1.NodeSelectorOperator) *corev1.Affinity {
	nodeSelReq := &corev1.NodeSelectorRequirement{
		Key:      labelName,
		Operator: selectOperator,
		Values:   labelValue,
	}

	nodeSelTerm := &corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{*nodeSelReq},
		MatchFields:      []corev1.NodeSelectorRequirement{},
	}

	prefTerm := &corev1.PreferredSchedulingTerm{
		Weight:     1,
		Preference: *nodeSelTerm,
	}

	aff := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{*prefTerm},
		},
	}
	return aff
}
