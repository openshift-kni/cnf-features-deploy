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

package find

import (
	"context"
	"fmt"

	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nropv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
)

func NodeGroupsMCPs(mcps *mcov1.MachineConfigPoolList, nodeGroups []nropv1alpha1.NodeGroup) ([]*mcov1.MachineConfigPool, error) {
	var result []*mcov1.MachineConfigPool
	for _, nodeGroup := range nodeGroups {
		found := false

		// handled by validation
		if nodeGroup.MachineConfigPoolSelector == nil {
			continue
		}

		for i := range mcps.Items {
			mcp := &mcps.Items[i]

			selector, err := metav1.LabelSelectorAsSelector(nodeGroup.MachineConfigPoolSelector)
			// handled by validation
			if err != nil {
				klog.Errorf("bad node group machine config pool selector %q", nodeGroup.MachineConfigPoolSelector.String())
				continue
			}

			mcpLabels := labels.Set(mcp.Labels)
			if selector.Matches(mcpLabels) {
				found = true
				result = append(result, mcp)
			}
		}

		if !found {
			return nil, fmt.Errorf("failed to find MachineConfigPool for the node group with the selector %q", nodeGroup.MachineConfigPoolSelector.String())
		}
	}

	return result, nil
}

func MCPBySelector(mcps []*mcov1.MachineConfigPool, sel *metav1.LabelSelector) (*mcov1.MachineConfigPool, error) {
	if sel == nil {
		return nil, fmt.Errorf("no MCP selector for selector %v", sel)

	}

	selector, err := metav1.LabelSelectorAsSelector(sel)
	if err != nil {
		return nil, err
	}

	for _, mcp := range mcps {
		if selector.Matches(labels.Set(mcp.Labels)) {
			return mcp, nil
		}
	}
	return nil, fmt.Errorf("cannot find MCP related to the selector %v", sel)
}

func GetMCPsFromMCOKubeletConfig(ctx context.Context, cli client.Client, mcoKubeletConfig mcov1.KubeletConfig) ([]*mcov1.MachineConfigPool, error) {
	mcps := &mcov1.MachineConfigPoolList{}
	var result []*mcov1.MachineConfigPool
	if err := cli.List(ctx, mcps); err != nil {
		return nil, err
	}
	for index := range mcps.Items {
		mcp := &mcps.Items[index]

		sel, err := metav1.LabelSelectorAsSelector(mcoKubeletConfig.Spec.MachineConfigPoolSelector)
		if err != nil {
			return nil, err
		}
		mcpLabels := labels.Set(mcp.Labels)
		if sel.Matches(mcpLabels) {
			result = append(result, mcp)
		}

	}
	return result, nil
}
