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

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nropv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	"github.com/openshift-kni/numaresources-operator/internal/wait"

	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"

	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
)

var _ = Describe("[serial][disruptive][scheduler] numaresources scheduler removal on a live cluster", func() {
	var fxt *e2efixture.Fixture

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-sched-remove")
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")

		nrosched.CheckNROSchedulerAvailable(fxt.Client, serialconfig.Config.NROSchedObj.Name)
	})

	AfterEach(func() {
		restoreScheduler(fxt, serialconfig.Config.NROSchedObj)
		nrosched.CheckNROSchedulerAvailable(fxt.Client, serialconfig.Config.NROSchedObj.Name)

		err := e2efixture.Teardown(fxt)
		Expect(err).NotTo(HaveOccurred())
	})

	When("removing the topology aware scheduler from a live cluster", func() {
		It("[case:1][test_id:47593][tier1] should keep existing workloads running", func() {
			var err error

			dp := createDeploymentSync(fxt, "testdp", serialconfig.Config.SchedulerName)

			By(fmt.Sprintf("deleting the NRO Scheduler object: %s", serialconfig.Config.NROSchedObj.Name))
			err = fxt.Client.Delete(context.TODO(), serialconfig.Config.NROSchedObj)
			Expect(err).ToNot(HaveOccurred())

			maxStep := 3
			for step := 0; step < maxStep; step++ {
				time.Sleep(10 * time.Second)

				By(fmt.Sprintf("ensuring the deployment %q keep being ready %d/%d", dp.Name, step+1, maxStep))

				updatedDp := &appsv1.Deployment{}
				err = fxt.Client.Get(context.TODO(), client.ObjectKeyFromObject(dp), updatedDp)
				Expect(err).ToNot(HaveOccurred())

				Expect(wait.IsDeploymentComplete(dp, &updatedDp.Status)).To(BeTrue(), "deployment %q become unready", dp.Name)
			}
		})

		It("[case:2][test_id:49093][tier1] should keep new scheduled workloads pending", func() {
			var err error

			By(fmt.Sprintf("deleting the NRO Scheduler object: %s", serialconfig.Config.NROSchedObj.Name))
			err = fxt.Client.Delete(context.TODO(), serialconfig.Config.NROSchedObj)
			Expect(err).ToNot(HaveOccurred())

			dp := createDeployment(fxt, "testdp", serialconfig.Config.SchedulerName)

			maxStep := 3
			for step := 0; step < maxStep; step++ {
				time.Sleep(10 * time.Second)

				By(fmt.Sprintf("ensuring the deployment %q keep being pending %d/%d", dp.Name, step+1, maxStep))

				updatedDp := &appsv1.Deployment{}
				err = fxt.Client.Get(context.TODO(), client.ObjectKeyFromObject(dp), updatedDp)
				Expect(err).ToNot(HaveOccurred())

				Expect(wait.IsDeploymentComplete(dp, &updatedDp.Status)).To(BeFalse(), "deployment %q become ready", dp.Name)
			}
		})
	})
})

var _ = Describe("[serial][disruptive][scheduler] numaresources scheduler restart on a live cluster", func() {
	var fxt *e2efixture.Fixture
	var nroSchedObj *nropv1alpha1.NUMAResourcesScheduler
	var schedulerName string

	BeforeEach(func() {
		var err error
		fxt, err = e2efixture.Setup("e2e-test-sched-remove")
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")

		nroSchedObj = &nropv1alpha1.NUMAResourcesScheduler{}
		err = fxt.Client.Get(context.TODO(), client.ObjectKey{Name: nrosched.NROSchedObjectName}, nroSchedObj)
		Expect(err).ToNot(HaveOccurred(), "cannot get %q in the cluster", nrosched.NROSchedObjectName)

		schedulerName = nroSchedObj.Status.SchedulerName
		Expect(schedulerName).ToNot(BeEmpty(), "cannot autodetect the TAS scheduler name from the cluster")

		nrosched.CheckNROSchedulerAvailable(fxt.Client, nroSchedObj.Name)
	})

	AfterEach(func() {
		err := e2efixture.Teardown(fxt)
		Expect(err).NotTo(HaveOccurred())
	})

	When("restarting the topology aware scheduler in a live cluster", func() {
		It("[case:1][test_id:48069][tier2] should schedule any pending workloads submitted while the scheduler was unavailable", func() {
			var err error

			dps, err := getDPOwnedBy(fxt.Client, nroSchedObj.ObjectMeta)
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("deleting the NRO Scheduler object: %s", nroSchedObj.Name))
			err = fxt.Client.Delete(context.TODO(), nroSchedObj)
			Expect(err).ToNot(HaveOccurred())

			// make sure scheduler deployment is gone
			Eventually(func() bool {
				for _, dp := range dps {
					if err := fxt.Client.Get(context.TODO(), client.ObjectKeyFromObject(dp), dp); err != nil {
						if apierrors.IsNotFound(err) {
							return true
						}
						klog.Warningf("failed to get Deployment %s/%s; err: %v", dp.Namespace, dp.Name, err)
						return false
					}
					klog.Warningf("Deployment %s/%s is still exists", dp.Namespace, dp.Name)
					return false
				}
				return true
			}, time.Minute, time.Second*10).Should(BeTrue())

			dp := createDeployment(fxt, "testdp", schedulerName)

			updatedDp := &appsv1.Deployment{}
			maxStep := 3
			for step := 0; step < maxStep; step++ {
				time.Sleep(10 * time.Second)

				By(fmt.Sprintf("ensuring the deployment %q keep being pending %d/%d", dp.Name, step+1, maxStep))

				err = fxt.Client.Get(context.TODO(), client.ObjectKeyFromObject(dp), updatedDp)
				Expect(err).ToNot(HaveOccurred())

				Expect(wait.IsDeploymentComplete(dp, &updatedDp.Status)).To(BeFalse(), "deployment %q become ready", dp.Name)
			}

			restoreScheduler(fxt, nroSchedObj)
			nrosched.CheckNROSchedulerAvailable(fxt.Client, nroSchedObj.Name)

			By(fmt.Sprintf("waiting for the test deployment %q to become complete and ready", updatedDp.Name))
			_, err = wait.ForDeploymentComplete(fxt.Client, updatedDp, 2*time.Second, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func restoreScheduler(fxt *e2efixture.Fixture, nroSchedObj *nropv1alpha1.NUMAResourcesScheduler) {
	By(fmt.Sprintf("re-creating the NRO Scheduler object: %s", nroSchedObj.Name))
	nroSched := &nropv1alpha1.NUMAResourcesScheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NUMAResourcesScheduler",
			APIVersion: nropv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nroSchedObj.Name,
		},
		Spec: nroSchedObj.Spec,
	}

	err := fxt.Client.Create(context.TODO(), nroSched)
	Expect(err).NotTo(HaveOccurred())
}

func createDeployment(fxt *e2efixture.Fixture, name, schedulerName string) *appsv1.Deployment {
	var err error
	var replicas int32 = 2

	podLabels := map[string]string{
		"test": "test-dp",
	}
	nodeSelector := map[string]string{}
	dp := objects.NewTestDeployment(replicas, podLabels, nodeSelector, fxt.Namespace.Name, name, objects.PauseImage, []string{objects.PauseCommand}, []string{})
	dp.Spec.Template.Spec.SchedulerName = schedulerName

	By(fmt.Sprintf("creating a test deployment %q", name))
	err = fxt.Client.Create(context.TODO(), dp)
	Expect(err).ToNot(HaveOccurred())

	return dp
}

func createDeploymentSync(fxt *e2efixture.Fixture, name, schedulerName string) *appsv1.Deployment {
	dpRunningTimeout := time.Minute
	dpRunningPollInterval := 10 * time.Second
	dp := createDeployment(fxt, name, schedulerName)
	By(fmt.Sprintf("waiting for the test deployment %q to be complete and ready", name))

	_, err := wait.ForDeploymentComplete(fxt.Client, dp, dpRunningPollInterval, dpRunningTimeout)
	Expect(err).ToNot(HaveOccurred(), "Deployment %q is not up & running after %v", dp.Name, dpRunningTimeout)
	return dp
}

func getDPOwnedBy(cli client.Client, objMeta metav1.ObjectMeta) ([]*appsv1.Deployment, error) {
	dpList := &appsv1.DeploymentList{}
	if err := cli.List(context.TODO(), dpList); err != nil {
		return nil, err
	}

	var dps []*appsv1.Deployment
	for i := range dpList.Items {
		if objects.IsOwnedBy(dpList.Items[i].ObjectMeta, objMeta) {
			dps = append(dps, &dpList.Items[i])
		}
	}
	return dps, nil
}
