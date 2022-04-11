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

package nodes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k8stopologyawareschedwg/deployer/pkg/clientutil"

	"github.com/openshift-kni/numaresources-operator/internal/baseload"
)

const (
	LabelRole   = "node-role.kubernetes.io"
	RoleWorker  = "worker"
	RoleMCPTest = "mcp-test"
)

func GetWorkerNodes(cli client.Client) ([]corev1.Node, error) {
	nodes := &corev1.NodeList{}
	selector, err := labels.Parse(fmt.Sprintf("%s/%s=", clientutil.LabelRole, clientutil.RoleWorker))
	if err != nil {
		return nil, err
	}

	err = cli.List(context.TODO(), nodes, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	return nodes.Items, nil
}

func GetLabelRoleWorker() string {
	return fmt.Sprintf("%s/%s", LabelRole, RoleWorker)
}

func GetLabelRoleMCPTest() string {
	return fmt.Sprintf("%s/%s", LabelRole, RoleMCPTest)
}

func GetLoad(k8sCli *kubernetes.Clientset, nodeName string) (baseload.Load, error) {
	nl := baseload.Load{
		Name: nodeName,
	}
	pods, err := k8sCli.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nl, err
	}

	return baseload.FromPods(nodeName, pods.Items), nil
}
