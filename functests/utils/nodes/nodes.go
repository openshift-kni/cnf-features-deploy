package nodes

import (
	"fmt"
	"os/exec"
	"path"

	"github.com/ghodss/yaml"
	testutils "github.com/openshift-kni/cnf-features-deploy/functests/utils"
	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
)

// GetByRole returns all nodes with the specified role
func GetByRole(cs *testclient.ClientSet, role string) ([]corev1.Node, error) {
	nodes, err := cs.Nodes().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s/%s=", testutils.LabelRole, role),
	})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

// FilterByResource returns all nodes with the specified allocated resource greater than 0
func FilterByResource(cs *testclient.ClientSet, nodes []corev1.Node, resource corev1.ResourceName) []corev1.Node {
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
func GetMachineConfigDaemonByNode(cs *testclient.ClientSet, node *corev1.Node) (*corev1.Pod, error) {
	labelSelector := "k8s-app=machine-config-daemon"
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", node.Name)
	mcds, err := cs.Pods(testutils.NamespaceMachineConfigOperator).List(metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}

	if len(mcds.Items) < 1 {
		return nil, fmt.Errorf("failed to find machine-config-daemon with label selector %q and field selector %q", labelSelector, fieldSelector)
	}
	return &mcds.Items[0], nil
}

// ExecCommandOnMachineConfigDaemon returns the output of the command execution on the machine-config-daemon pod that runs on the specified node
func ExecCommandOnMachineConfigDaemon(cs *testclient.ClientSet, node *corev1.Node, command []string) ([]byte, error) {
	mcd, err := GetMachineConfigDaemonByNode(cs, node)
	if err != nil {
		return nil, err
	}

	initialArgs := []string{
		"rsh",
		"-n", testutils.NamespaceMachineConfigOperator,
		"-c", testutils.ContainerMachineConfigDaemon,
		"--timeout", "30",
		mcd.Name,
	}
	initialArgs = append(initialArgs, command...)
	return exec.Command("oc", initialArgs...).CombinedOutput()
}

// GetKubeletConfig returns KubeletConfiguration loaded from the node /etc/kubernetes/kubelet.conf
func GetKubeletConfig(cs *testclient.ClientSet, node *corev1.Node) (*kubeletconfigv1beta1.KubeletConfiguration, error) {
	command := []string{"cat", path.Join("/rootfs", testutils.FilePathKubeletConfig)}
	kubeletBytes, err := ExecCommandOnMachineConfigDaemon(cs, node, command)
	if err != nil {
		return nil, err
	}

	kubeletConfig := &kubeletconfigv1beta1.KubeletConfiguration{}
	if err := yaml.Unmarshal(kubeletBytes, kubeletConfig); err != nil {
		return nil, err
	}
	return kubeletConfig, err
}
