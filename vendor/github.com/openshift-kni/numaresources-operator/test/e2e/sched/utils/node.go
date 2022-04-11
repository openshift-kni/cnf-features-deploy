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

package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform"
	"github.com/openshift-kni/numaresources-operator/test/utils/configuration"
)

const (
	// LabelMasterRole contains the key for the role label
	LabelMasterRole = "node-role.kubernetes.io/master"

	// LabelControlPlane contains the key for the control-plane role label
	LabelControlPlane = "node-role.kubernetes.io/control-plane"
)

func ListMasterNodes(aclient client.Client) ([]corev1.Node, error) {
	nodeList := &corev1.NodeList{}
	labels := metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelMasterRole: "",
		},
	}
	if configuration.Platform == platform.Kubernetes {
		labels.MatchLabels[LabelControlPlane] = ""
	}

	selNodes, err := metav1.LabelSelectorAsSelector(&labels)
	if err != nil {
		return nil, err
	}

	err = aclient.List(context.TODO(), nodeList, &client.ListOptions{LabelSelector: selNodes})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func GetNodeNames(nodes []corev1.Node) []string {
	var nodeNames []string
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
	}

	return nodeNames
}
