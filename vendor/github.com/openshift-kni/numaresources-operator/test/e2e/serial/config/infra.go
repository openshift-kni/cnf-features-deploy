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

package config

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/client"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"

	nropv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	"github.com/openshift-kni/numaresources-operator/pkg/machineconfigpools"
	"github.com/openshift-kni/numaresources-operator/pkg/objectnames"

	numacellmanifests "github.com/openshift-kni/numaresources-operator/test/deviceplugin/pkg/numacell/manifests"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	"github.com/openshift-kni/numaresources-operator/test/utils/images"
	e2ewait "github.com/openshift-kni/numaresources-operator/test/utils/objects/wait"
)

func SetupInfra(fxt *e2efixture.Fixture, nroOperObj *nropv1alpha1.NUMAResourcesOperator, nrtList nrtv1alpha1.NodeResourceTopologyList) {
	setupNUMACell(fxt, nroOperObj.Spec.NodeGroups, 3*time.Minute)
	LabelNodes(fxt.Client, nrtList)
}

func TeardownInfra(fxt *e2efixture.Fixture, nrtList nrtv1alpha1.NodeResourceTopologyList) {
	UnlabelNodes(fxt.Client, nrtList)
}

func setupNUMACell(fxt *e2efixture.Fixture, nodeGroups []nropv1alpha1.NodeGroup, timeout time.Duration) {
	klog.Infof("e2e infra setup begin")

	Expect(nodeGroups).ToNot(BeEmpty(), "cannot autodetect the TAS node groups from the cluster")

	mcps, err := machineconfigpools.GetNodeGroupsMCPs(context.TODO(), fxt.Client, nodeGroups)
	Expect(err).ToNot(HaveOccurred())

	klog.Infof("setting e2e infra for %d MCPs", len(mcps))

	sa := numacellmanifests.ServiceAccount(fxt.Namespace.Name, numacellmanifests.Prefix)
	err = fxt.Client.Create(context.TODO(), sa)
	Expect(err).ToNot(HaveOccurred(), "cannot create the numacell serviceaccount %q in the namespace %q", sa.Name, sa.Namespace)

	ro := numacellmanifests.Role(fxt.Namespace.Name, numacellmanifests.Prefix)
	err = fxt.Client.Create(context.TODO(), ro)
	Expect(err).ToNot(HaveOccurred(), "cannot create the numacell role %q in the namespace %q", sa.Name, sa.Namespace)

	rb := numacellmanifests.RoleBinding(fxt.Namespace.Name, numacellmanifests.Prefix)
	err = fxt.Client.Create(context.TODO(), rb)
	Expect(err).ToNot(HaveOccurred(), "cannot create the numacell rolebinding %q in the namespace %q", sa.Name, sa.Namespace)

	var dss []*appsv1.DaemonSet
	for _, mcp := range mcps {
		if mcp.Spec.NodeSelector == nil {
			klog.Warningf("the machine config pool %q does not have node selector", mcp.Name)
			continue
		}

		dsName := objectnames.GetComponentName(numacellmanifests.Prefix, mcp.Name)
		klog.Infof("setting e2e infra for %q: daemonset %q", mcp.Name, dsName)

		pullSpec := getNUMACellDevicePluginPullSpec()
		ds := numacellmanifests.DaemonSet(mcp.Spec.NodeSelector.MatchLabels, fxt.Namespace.Name, dsName, sa.Name, pullSpec)
		err = fxt.Client.Create(context.TODO(), ds)
		Expect(err).ToNot(HaveOccurred(), "cannot create the numacell daemonset %q in the namespace %q", ds.Name, ds.Namespace)

		dss = append(dss, ds)
	}

	klog.Infof("daemonsets created (%d)", len(dss))

	var wg sync.WaitGroup
	for _, ds := range dss {
		wg.Add(1)
		go func(ds *appsv1.DaemonSet) {
			defer GinkgoRecover()
			defer wg.Done()

			klog.Infof("waiting for daemonset %q to be ready", ds.Name)

			// TODO: what if timeout < period?
			ds, err := e2ewait.ForDaemonSetReady(fxt.Client, ds, 10*time.Second, timeout)
			Expect(err).ToNot(HaveOccurred(), "DaemonSet %q failed to go running", ds.Name)
		}(ds)
	}
	wg.Wait()

	klog.Infof("e2e infra setup completed")
}

func getNUMACellDevicePluginPullSpec() string {
	if pullSpec, ok := os.LookupEnv("E2E_NROP_URL_NUMACELL_DEVICE_PLUGIN"); ok {
		return pullSpec
	}
	// backward compatibility
	if pullSpec, ok := os.LookupEnv("E2E_NUMACELL_DEVICE_PLUGIN_URL"); ok {
		return pullSpec
	}
	return images.NUMACellDevicePluginTestImageCI
}

func LabelNodes(cli client.Client, nrtList nrtv1alpha1.NodeResourceTopologyList) {
	for _, nrt := range nrtList.Items {
		node := corev1.Node{}
		err := cli.Get(context.TODO(), client.ObjectKey{Name: nrt.Name}, &node)
		Expect(err).ToNot(HaveOccurred())
		labelValue := fmt.Sprintf("%d", len(nrt.Zones))
		node.Labels[MultiNUMALabel] = labelValue

		klog.Infof("labeling node %q with %s: %s", nrt.Name, MultiNUMALabel, labelValue)
		err = cli.Update(context.TODO(), &node)
		Expect(err).ToNot(HaveOccurred())
	}
}

func UnlabelNodes(cli client.Client, nrtList nrtv1alpha1.NodeResourceTopologyList) {
	for _, nrt := range nrtList.Items {
		node := corev1.Node{}
		err := cli.Get(context.TODO(), client.ObjectKey{Name: nrt.Name}, &node)
		Expect(err).ToNot(HaveOccurred())

		klog.Infof("unlabeling node %q removing label %s", nrt.Name, MultiNUMALabel)
		delete(node.Labels, MultiNUMALabel)
		err = cli.Update(context.TODO(), &node)
		Expect(err).ToNot(HaveOccurred())
	}
}
