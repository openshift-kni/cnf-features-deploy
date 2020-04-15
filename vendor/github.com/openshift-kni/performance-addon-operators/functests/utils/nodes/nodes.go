package nodes

import (
	"context"
	"fmt"
	"os/exec"
	"path"

	"github.com/ghodss/yaml"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetByRole returns all nodes with the specified role
func GetByRole(c client.Client, role string) ([]corev1.Node, error) {
	selector, err := labels.Parse(fmt.Sprintf("%s/%s=", testutils.LabelRole, role))
	if err != nil {
		return nil, err
	}

	nodes := &corev1.NodeList{}
	if err := c.List(context.TODO(), nodes, &client.ListOptions{LabelSelector: selector}); err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

// GetNonRTWorkers returns list of nodes with no worker-rt label
func GetNonRTWorkers() ([]corev1.Node, error) {
	nonRTWorkerNodes := []corev1.Node{}

	workerNodes, err := GetByRole(testclient.Client, testutils.RoleWorker)
	for _, node := range workerNodes {
		if _, ok := node.Labels[fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT)]; ok {
			continue
		}
		nonRTWorkerNodes = append(nonRTWorkerNodes, node)
	}
	return nonRTWorkerNodes, err
}

// FilterByResource returns all nodes with the specified allocated resource greater than 0
func FilterByResource(nodes []corev1.Node, resource corev1.ResourceName) []corev1.Node {
	nodesWithResource := []corev1.Node{}
	for _, node := range nodes {
		for name, quantity := range node.Status.Allocatable {
			if name == testutils.ResourceSRIOV && !quantity.IsZero() {
				nodesWithResource = append(nodesWithResource, node)
			}
		}
	}
	return nodesWithResource
}

// GetMachineConfigDaemonByNode returns the machine-config-daemon pod that runs on the specified node
func GetMachineConfigDaemonByNode(c client.Client, node *corev1.Node) (*corev1.Pod, error) {
	listOptions := &client.ListOptions{
		Namespace:     testutils.NamespaceMachineConfigOperator,
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}),
		LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": "machine-config-daemon"}),
	}

	mcds := &corev1.PodList{}
	if err := c.List(context.TODO(), mcds, listOptions); err != nil {
		return nil, err
	}

	if len(mcds.Items) < 1 {
		return nil, fmt.Errorf("failed to get machine-config-daemon pod for the node %q", node.Name)
	}
	return &mcds.Items[0], nil
}

// ExecCommandOnMachineConfigDaemon returns the output of the command execution on the machine-config-daemon pod that runs on the specified node
func ExecCommandOnMachineConfigDaemon(c client.Client, node *corev1.Node, command []string) ([]byte, error) {
	mcd, err := GetMachineConfigDaemonByNode(c, node)
	if err != nil {
		return nil, err
	}

	initialArgs := []string{
		"exec",
		"-i",
		"-n", testutils.NamespaceMachineConfigOperator,
		"-c", testutils.ContainerMachineConfigDaemon,
		"--request-timeout", "30",
		mcd.Name,
		"--",
	}
	initialArgs = append(initialArgs, command...)
	return exec.Command("oc", initialArgs...).CombinedOutput()
}

// GetKubeletConfig returns KubeletConfiguration loaded from the node /etc/kubernetes/kubelet.conf
func GetKubeletConfig(c client.Client, node *corev1.Node) (*kubeletconfigv1beta1.KubeletConfiguration, error) {
	command := []string{"cat", path.Join("/rootfs", testutils.FilePathKubeletConfig)}
	kubeletBytes, err := ExecCommandOnMachineConfigDaemon(c, node, command)
	if err != nil {
		return nil, err
	}

	klog.Infof("command output: %s", string(kubeletBytes))
	kubeletConfig := &kubeletconfigv1beta1.KubeletConfiguration{}
	if err := yaml.Unmarshal(kubeletBytes, kubeletConfig); err != nil {
		return nil, err
	}
	return kubeletConfig, err
}
