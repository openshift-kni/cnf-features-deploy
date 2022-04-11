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

package tests

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	schedutils "github.com/openshift-kni/numaresources-operator/test/e2e/sched/utils"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	e2enrt "github.com/openshift-kni/numaresources-operator/test/utils/noderesourcetopologies"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
	e2ewait "github.com/openshift-kni/numaresources-operator/test/utils/objects/wait"
	e2epadder "github.com/openshift-kni/numaresources-operator/test/utils/padder"

	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
)

var _ = Describe("[serial][disruptive][scheduler] numaresources workload overhead", func() {
	var fxt *e2efixture.Fixture
	var padder *e2epadder.Padder
	var nrtList nrtv1alpha1.NodeResourceTopologyList
	var nrts []nrtv1alpha1.NodeResourceTopology

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-workload-overhead")
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

		When("a RuntimeClass exist in the cluster", func() {
			var rtClass *nodev1.RuntimeClass
			BeforeEach(func() {
				rtClass = &nodev1.RuntimeClass{
					TypeMeta: metav1.TypeMeta{
						Kind:       "RuntimeClass",
						APIVersion: "node.k8s.io/vi",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-rtclass",
					},
					Handler: "runc",
					Overhead: &nodev1.Overhead{
						PodFixed: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("500M"),
						},
					},
				}

				err := fxt.Client.Create(context.TODO(), rtClass)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				if rtClass != nil {
					err := fxt.Client.Delete(context.TODO(), rtClass)
					if err != nil {
						klog.Errorf("Unable to delete RuntimeClass %q", rtClass.Name)
					}
				}
			})
			It("[test_id:47582][tier2] schedule a guaranteed Pod in a single NUMA zone and check overhead is not accounted in NRT", func() {

				skipUnlessEnvVar("E2E_SERIAL_STAGING", "FIXME: NRT filter clashes with the noderesources fit plugin")

				podResources := corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				}

				minRes := corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				}

				// need a zone with resources for overhead, pod and a little bit more to avoid zone saturation
				zoneRequiredResources := rtClass.Overhead.PodFixed.DeepCopy()
				zoneRequiredResources.Cpu().Add(*podResources.Cpu())
				zoneRequiredResources.Cpu().Add(*minRes.Cpu())

				zoneRequiredResources.Memory().Add(*podResources.Memory())
				zoneRequiredResources.Memory().Add(*minRes.Memory())

				nrtCandidates := e2enrt.FilterAnyZoneMatchingResources(nrtTwoZoneCandidates, zoneRequiredResources)
				const minCandidates int = 1
				if len(nrtCandidates) < minCandidates {
					Skip(fmt.Sprintf("There should be at least %d nodes with at least %v resources: found %d", minCandidates, zoneRequiredResources, len(nrtCandidates)))
				}

				candidateNodeNames := e2enrt.AccumulateNames(nrtCandidates)
				targetNodeName, ok := candidateNodeNames.PopAny()
				Expect(ok).To(BeTrue(), "cannot select a target node among %#v", candidateNodeNames.List())

				By("padding non-target nodes")
				var paddingPods []*corev1.Pod
				for _, nodeName := range candidateNodeNames.List() {

					nrtInfo, err := e2enrt.FindFromList(nrtCandidates, nodeName)
					Expect(err).NotTo(HaveOccurred(), "missing NRT Info for node %q", nodeName)

					for _, zone := range nrtInfo.Zones {
						padPod, err := makePaddingPod(fxt.Namespace.Name, nodeName, zone, minRes)
						Expect(err).NotTo(HaveOccurred())

						pinnedPadPod, err := pinPodTo(padPod, nodeName, zone.Name)
						Expect(err).NotTo(HaveOccurred())

						err = fxt.Client.Create(context.TODO(), pinnedPadPod)
						Expect(err).NotTo(HaveOccurred())

						paddingPods = append(paddingPods, pinnedPadPod)
					}

				}

				failedPods := e2ewait.ForPodListAllRunning(fxt.Client, paddingPods)
				for _, failedPod := range failedPods {
					_ = objects.LogEventsForPod(fxt.K8sClient, failedPod.Namespace, failedPod.Name)
				}
				Expect(failedPods).To(BeEmpty())

				By("checking the resource allocation as the test starts")
				nrtListInitial, err := e2enrt.GetUpdated(fxt.Client, nrtList, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Scheduling the testing deployment with RuntimeClass=%q", rtClass.Name))
				var deploymentName string = "test-dp"
				var replicas int32 = 1

				podLabels := map[string]string{
					"test": "test-dp",
				}
				nodeSelector := map[string]string{}
				deployment := objects.NewTestDeployment(replicas, podLabels, nodeSelector, fxt.Namespace.Name, deploymentName, objects.PauseImage, []string{objects.PauseCommand}, []string{})
				deployment.Spec.Template.Spec.SchedulerName = serialconfig.Config.SchedulerName
				deployment.Spec.Template.Spec.Containers[0].Resources.Limits = podResources
				deployment.Spec.Template.Spec.RuntimeClassName = &rtClass.Name

				err = fxt.Client.Create(context.TODO(), deployment)
				Expect(err).NotTo(HaveOccurred(), "unable to create deployment %q", deployment.Name)

				By("waiting for deployment to be up&running")
				dpRunningTimeout := 1 * time.Minute
				dpRunningPollInterval := 10 * time.Second
				err = e2ewait.ForDeploymentComplete(fxt.Client, deployment, dpRunningPollInterval, dpRunningTimeout)
				Expect(err).NotTo(HaveOccurred(), "Deployment %q not up&running after %v", deployment.Name, dpRunningTimeout)

				nrtListPostCreate, err := e2enrt.GetUpdated(fxt.Client, nrtListInitial, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("checking deployment pods have been scheduled with the topology aware scheduler %q and in the proper node %q", serialconfig.Config.SchedulerName, targetNodeName))
				pods, err := schedutils.ListPodsByDeployment(fxt.Client, *deployment)
				Expect(err).NotTo(HaveOccurred(), "Unable to get pods from Deployment %q:  %v", deployment.Name, err)

				podResourcesWithOverhead := podResources.DeepCopy()
				podResourcesWithOverhead.Cpu().Add(*rtClass.Overhead.PodFixed.Cpu())
				podResourcesWithOverhead.Memory().Add(*rtClass.Overhead.PodFixed.Memory())

				for _, pod := range pods {
					Expect(pod.Spec.NodeName).To(Equal(targetNodeName))
					schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)
					Expect(err).ToNot(HaveOccurred())
					Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", pod.Namespace, pod.Name, serialconfig.Config.SchedulerName)

					By(fmt.Sprintf("checking the resources are accounted as expected on %q", pod.Spec.NodeName))
					nrtInitial, err := e2enrt.FindFromList(nrtListInitial.Items, pod.Spec.NodeName)
					Expect(err).ToNot(HaveOccurred())
					nrtPostCreate, err := e2enrt.FindFromList(nrtListPostCreate.Items, pod.Spec.NodeName)
					Expect(err).ToNot(HaveOccurred())

					_, err = e2enrt.CheckZoneConsumedResourcesAtLeast(*nrtInitial, *nrtPostCreate, podResources)
					Expect(err).ToNot(HaveOccurred())

					// Here error have to Ocurr because pod overhead resources should not be count
					_, err = e2enrt.CheckZoneConsumedResourcesAtLeast(*nrtInitial, *nrtPostCreate, podResourcesWithOverhead)
					Expect(err).To(HaveOccurred())
				}

			})
		})
	})
})
