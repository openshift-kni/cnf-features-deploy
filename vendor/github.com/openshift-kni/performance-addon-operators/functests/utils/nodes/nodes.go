package nodes

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

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

// GetByLabels returns all nodes with the specified labels
func GetByLabels(nodeLabels map[string]string) ([]corev1.Node, error) {
	selector := labels.SelectorFromSet(nodeLabels)
	return GetBySelector(selector)
}

// GetNonPerformancesWorkers returns list of nodes with non matching perfomance profile labels
func GetNonPerformancesWorkers(nodeSelectorLabels map[string]string) ([]corev1.Node, error) {
	nonPerformanceWorkerNodes := []corev1.Node{}
	workerNodes, err := GetByRole(testutils.RoleWorker)
	for _, node := range workerNodes {
		for label := range nodeSelectorLabels {
			if _, ok := node.Labels[label]; !ok {
				nonPerformanceWorkerNodes = append(nonPerformanceWorkerNodes, node)
				break
			}
		}
	}
	return nonPerformanceWorkerNodes, err
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

// MatchingOptionalSelector filter the given slice with only the nodes matching the optional selector.
// If no selector is set, it returns the same list.
// The NODES_SELECTOR must be set with a labelselector expression.
// For example: NODES_SELECTOR="sctp=true"
// Inspired from: https://github.com/fedepaol/sriov-network-operator/blob/master/test/util/nodes/nodes.go
func MatchingOptionalSelector(toFilter []corev1.Node) ([]corev1.Node, error) {
	if testutils.NodesSelector == "" {
		return toFilter, nil
	}

	selector, err := labels.Parse(testutils.NodesSelector)
	if err != nil {
		return nil, fmt.Errorf("Error parsing the %s label selector, %v", testutils.NodesSelector, err)
	}

	toMatch, err := GetBySelector(selector)
	if err != nil {
		return nil, fmt.Errorf("Error in getting nodes matching the %s label selector, %v", testutils.NodesSelector, err)
	}
	if len(toMatch) == 0 {
		return nil, fmt.Errorf("Failed to get nodes matching %s label selector", testutils.NodesSelector)
	}

	res := make([]corev1.Node, 0)
	for _, n := range toFilter {
		for _, m := range toMatch {
			if n.Name == m.Name {
				res = append(res, n)
				break
			}
		}
	}

	return res, nil
}

// HasPreemptRTKernel returns no error if the node booted with PREEMPT RT kernel
func HasPreemptRTKernel(node *corev1.Node) error {
	// verify that the kernel-rt-core installed it also means the the machine booted with the RT kernel
	// because the machine-config-daemon uninstalls regular kernel once you install the RT one and
	// on traditional yum systems, rpm -q kernel can be completely different from what you're booted
	// because yum keeps multiple kernels but only one userspace;
	// with rpm-ostree rpm -q is telling you what you're booted into always,
	// because ostree binds together (kernel, userspace) as a single commit.
	cmd := []string{"chroot", "/rootfs", "rpm", "-q", "kernel-rt-core"}
	if _, err := ExecCommandOnNode(cmd, node); err != nil {
		return err
	}

	cmd = []string{"/bin/bash", "-c", "cat /rootfs/sys/kernel/realtime"}
	out, err := ExecCommandOnNode(cmd, node)
	if err != nil {
		return err
	}

	if out != "1" {
		return fmt.Errorf("RT kernel disabled")
	}

	return nil
}

func BannedCPUs(node corev1.Node) (banned cpuset.CPUSet, err error) {
	cmd := []string{"sed", "-n", "s/^IRQBALANCE_BANNED_CPUS=\\(.*\\)/\\1/p", "/rootfs/etc/sysconfig/irqbalance"}
	bannedCPUs, err := ExecCommandOnNode(cmd, &node)
	if err != nil {
		return cpuset.NewCPUSet(), fmt.Errorf("failed to execute %v: %v", cmd, err)
	}

	banned, err = components.CPUMaskToCPUSet(bannedCPUs)
	if err != nil {
		return cpuset.NewCPUSet(), fmt.Errorf("failed to parse the banned CPUs: %v", err)
	}

	return banned, nil
}
