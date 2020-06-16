package nodes

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	testutils "github.com/openshift-kni/cnf-features-deploy/functests/utils"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
)

// NodesSelector represent the label selector used to filter impacted nodes.
var NodesSelector string

func init() {
	NodesSelector = os.Getenv("NODES_SELECTOR")
}

const ptpLinuxDaemonNamespace = "openshift-ptp"

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
		"exec",
		"-i",
		"-n", testutils.NamespaceMachineConfigOperator,
		"-c", testutils.ContainerMachineConfigDaemon,
		"--timeout", "30",
		mcd.Name,
		"--",
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

// NodeTopology represents a subset of the node topology
// structure.
type NodeTopology struct {
	NodeName      string
	InterfaceList []string
	NodeObject    *corev1.Node
}

// GetNodeTopology returns the topology of a given node.
func GetNodeTopology(client *client.ClientSet) ([]NodeTopology, error) {
	nodeDevicesList, err := client.NodePtpDevices(ptpLinuxDaemonNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(nodeDevicesList.Items) == 0 {
		return nil, fmt.Errorf("Zero nodes found")
	}

	nodeTopologyList := []NodeTopology{}

	for _, node := range nodeDevicesList.Items {
		if len(node.Status.Devices) > 0 {
			interfaceList := []string{}
			for _, iface := range node.Status.Devices {
				interfaceList = append(interfaceList, iface.Name)
			}
			nodeTopology := NodeTopology{NodeName: node.Name, InterfaceList: interfaceList}
			nodeTopologyList = append(nodeTopologyList, nodeTopology)
		}
	}

	return nodeTopologyList, nil
}

// LabelNode labels a node.
func LabelNode(nodeName, key, value string) (*corev1.Node, error) {
	NodeObject, err := client.Client.Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	NodeObject.Labels[key] = value
	NodeObject, err = client.Client.Nodes().Update(NodeObject)
	if err != nil {
		return nil, err
	}

	return NodeObject, nil
}

// LabeledNodesCount return the number of nodes with the given label.
func LabeledNodesCount(label string) (int, error) {
	nodeList, err := client.Client.Nodes().List(metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", label)})
	if err != nil {
		return 0, err
	}
	return len(nodeList.Items), nil
}

// MatchingOptionalSelector filter the given slice with only the nodes matching the optional selector.
// If no selector is set, it returns the same list.
// The NODES_SELECTOR must be set with a labelselector expression.
// For example: NODES_SELECTOR="sctp=true"
func MatchingOptionalSelector(toFilter []corev1.Node) ([]corev1.Node, error) {
	if NodesSelector == "" {
		return toFilter, nil
	}
	toMatch, err := client.Client.Nodes().List(metav1.ListOptions{
		LabelSelector: NodesSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("Error in getting nodes matching the %s label selector, %v", NodesSelector, err)
	}
	if len(toMatch.Items) == 0 {
		return nil, fmt.Errorf("Failed to get nodes matching %s label selector", NodesSelector)
	}

	res := make([]corev1.Node, 0)
	for _, n := range toFilter {
		for _, m := range toMatch.Items {
			if n.Name == m.Name {
				res = append(res, n)
				break
			}
		}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("Failed to find matching nodes with %s label selector", NodesSelector)
	}
	return res, nil
}

// MatchingOptionalSelectorByName filter the given slice with only the nodes matching the optional selector.
// If no selector is set, it returns the same list.
// The NODES_SELECTOR must be in the form of label=value.
// For example: NODES_SELECTOR="sctp=true"
func MatchingOptionalSelectorByName(toFilter []string) ([]string, error) {
	if NodesSelector == "" {
		return toFilter, nil
	}
	toMatch, err := client.Client.Nodes().List(metav1.ListOptions{
		LabelSelector: NodesSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("Error in getting nodes matching %s, %v", NodesSelector, err)
	}
	if len(toMatch.Items) == 0 {
		return nil, fmt.Errorf("Failed to get nodes matching %s, %v", NodesSelector, err)
	}

	res := make([]string, 0)
	for _, n := range toFilter {
		for _, m := range toMatch.Items {
			if n == m.Name {
				res = append(res, n)
			}
		}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("Failed to find matching nodes with %s", NodesSelector)
	}
	return res, nil
}

// PodLabelSelector returns a map based on the optional NODES_SELECTOR variable.
func PodLabelSelector() (map[string]string, bool) {
	if NodesSelector == "" {
		return nil, false
	}
	values := strings.Split(NodesSelector, "=")
	if len(values) != 2 {
		return nil, false
	}
	return map[string]string{
		values[0]: values[1],
	}, true
}
