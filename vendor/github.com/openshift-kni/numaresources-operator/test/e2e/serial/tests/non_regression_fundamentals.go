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

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"

	"github.com/openshift-kni/numaresources-operator/internal/wait"

	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"

	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
)

var _ = Describe("[serial][fundamentals][scheduler] numaresources fundamentals non-regression", func() {
	var fxt *e2efixture.Fixture
	var nrtList nrtv1alpha1.NodeResourceTopologyList

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-non-regression-fundamentals")
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")

		err = fxt.Client.List(context.TODO(), &nrtList)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := e2efixture.Teardown(fxt)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("using the NUMA-aware scheduler without NRT data", func() {
		var testPod *corev1.Pod

		BeforeEach(func() {
			if len(nrtList.Items) > 0 {
				Skip("this test require empty NRT data")
			}
		})

		AfterEach(func() {
			if testPod == nil {
				return
			}

			err := fxt.Client.Delete(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			By("checking the test pod is removed")
			err = wait.ForPodDeleted(fxt.Client, testPod.Namespace, testPod.Name, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[tier1] should run a best-effort pod", func() {
			testPod = objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName

			By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
			err := fxt.Client.Create(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			timeout := 5 * time.Minute
			updatedPod, err := wait.ForPodPhase(fxt.Client, testPod.Namespace, testPod.Name, corev1.PodRunning, timeout)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
		})

		It("[tier1] should run a burstable pod", func() {
			testPod = objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			testPod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			}

			By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
			err := fxt.Client.Create(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			timeout := 5 * time.Minute
			updatedPod, err := wait.ForPodPhase(fxt.Client, testPod.Namespace, testPod.Name, corev1.PodRunning, timeout)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
		})

		It("[tier1][test_id:47611] should run a guaranteed pod", func() {
			testPod = objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			testPod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			}

			By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
			err := fxt.Client.Create(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			timeout := 5 * time.Minute
			updatedPod, err := wait.ForPodPhase(fxt.Client, testPod.Namespace, testPod.Name, corev1.PodRunning, timeout)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
		})
	})
})
