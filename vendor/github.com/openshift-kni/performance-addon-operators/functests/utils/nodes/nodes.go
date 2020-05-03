package nodes

import (
	"context"
	"fmt"
	"path"
	"strings"

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
func GetByRole(role string) ([]corev1.Node, error) {
	selector, err := labels.Parse(fmt.Sprintf("%s/%s=", testutils.LabelRole, role))
	if err != nil {
		return nil, err
	}
	return GetBySelector(selector)
}

// GetBySelector returns all nodes with the specified selector
func GetBySelector(selector labels.Selector) ([]corev1.Node, error) {
	nodes := &corev1.NodeList{}
	if err := testclient.Client.List(context.TODO(), nodes, &client.ListOptions{LabelSelector: selector}); err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

// GetNonRTWorkers returns list of nodes with no worker-cnf label
func GetNonRTWorkers() ([]corev1.Node, error) {
	nonRTWorkerNodes := []corev1.Node{}

	workerNodes, err := GetByRole(testutils.RoleWorker)
	for _, node := range workerNodes {
		if _, ok := node.Labels[fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerCNF)]; ok {
			continue
		}
		nonRTWorkerNodes = append(nonRTWorkerNodes, node)
	}
	return nonRTWorkerNodes, err
}

// GetMachineConfigDaemonByNode returns the machine-config-daemon pod that runs on the specified node
func GetMachineConfigDaemonByNode(node *corev1.Node) (*corev1.Pod, error) {
	listOptions := &client.ListOptions{
		Namespace:     testutils.NamespaceMachineConfigOperator,
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}),
		LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": "machine-config-daemon"}),
	}

	mcds := &corev1.PodList{}
	if err := testclient.Client.List(context.TODO(), mcds, listOptions); err != nil {
		return nil, err
	}

	if len(mcds.Items) < 1 {
		return nil, fmt.Errorf("failed to get machine-config-daemon pod for the node %q", node.Name)
	}
	return &mcds.Items[0], nil
}

// ExecCommandOnMachineConfigDaemon returns the output of the command execution on the machine-config-daemon pod that runs on the specified node
func ExecCommandOnMachineConfigDaemon(node *corev1.Node, command []string) ([]byte, error) {
	mcd, err := GetMachineConfigDaemonByNode(node)
	if err != nil {
		return nil, err
	}
	klog.Infof("found mcd %s for node %s", mcd.Name, node.Name)

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
	return testutils.ExecAndLogCommand("oc", initialArgs...)
}

// ExecCommandOnNode executes given command on given node and returns the result
func ExecCommandOnNode(cmd []string, node *corev1.Node) (string, error) {
	out, err := ExecCommandOnMachineConfigDaemon(node, cmd)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(out), "\n"), nil
}

// GetKubeletConfig returns KubeletConfiguration loaded from the node /etc/kubernetes/kubelet.conf
func GetKubeletConfig(node *corev1.Node) (*kubeletconfigv1beta1.KubeletConfiguration, error) {
	command := []string{"cat", path.Join("/rootfs", testutils.FilePathKubeletConfig)}
	kubeletBytes, err := ExecCommandOnMachineConfigDaemon(node, command)
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
