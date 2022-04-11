/*
Copyright 2021.

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

package machineconfigpools

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	nropv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	mcpfind "github.com/openshift-kni/numaresources-operator/pkg/machineconfigpools/find"
)

func GetNodeGroupsMCPs(ctx context.Context, cli client.Client, nodeGroups []nropv1alpha1.NodeGroup) ([]*mcov1.MachineConfigPool, error) {
	mcps := &mcov1.MachineConfigPoolList{}
	if err := cli.List(ctx, mcps); err != nil {
		return nil, err
	}
	return mcpfind.NodeGroupsMCPs(mcps, nodeGroups)
}

func GetNodeListFromMachineConfigPool(ctx context.Context, cli client.Client, mcp mcov1.MachineConfigPool) ([]corev1.Node, error) {
	sel, err := metav1.LabelSelectorAsSelector(mcp.Spec.MachineConfigSelector)
	if err != nil {
		return nil, err
	}

	nodeList := &corev1.NodeList{}
	err = cli.List(ctx, nodeList, &client.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}

	return nodeList.Items, nil
}
