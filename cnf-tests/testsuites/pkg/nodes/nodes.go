package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"

	ptpv1 "github.com/openshift/ptp-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/exec"
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
	nodes, err := cs.Nodes().List(context.Background(), metav1.ListOptions{
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
	mcds, err := cs.Pods(testutils.NamespaceMachineConfigOperator).List(context.Background(), metav1.ListOptions{
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

// GetSriovConfigDaemonByNode returns the sriov-config-daemon pod that runs on the specified node
func GetSriovConfigDaemonByNode(cs *testclient.ClientSet, node *corev1.Node) (*corev1.Pod, error) {
	labelSelector := "app=sriov-network-config-daemon"
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", node.Name)
	srds, err := cs.Pods(namespaces.SRIOVOperator).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}

	if len(srds.Items) < 1 {
		return nil, fmt.Errorf("failed to find sriov-config-daemon with label selector %q and field selector %q", labelSelector, fieldSelector)
	}
	return &srds.Items[0], nil
}

// ExecCommandOnMachineConfigDaemon returns the output of the command execution on the machine-config-daemon pod that runs on the specified node
func ExecCommandOnMachineConfigDaemon(cs *testclient.ClientSet, node *corev1.Node, command []string) ([]byte, error) {
	mcd, err := GetMachineConfigDaemonByNode(cs, node)
	if err != nil {
		return nil, err
	}

	ret, err := pods.ExecCommandInContainer(cs, *mcd, testutils.ContainerMachineConfigDaemon, command)
	return ret.Bytes(), err
}

// ExecCommandOnMachineConfigDaemon returns the output of the command execution on the machine-config-daemon pod that runs on the specified node
func ExecCommandOnNodeViaSriovDaemon(cs *testclient.ClientSet, node *corev1.Node, command []string) ([]byte, error) {
	srd, err := GetSriovConfigDaemonByNode(cs, node)
	if err != nil {
		return nil, err
	}

	ret, err := pods.ExecCommandInContainer(cs, *srd, testutils.ContainerSriovConfigDaemon, command)
	return ret.Bytes(), err
}

// GetOvsPodByNode returns the ovs-node pod that runs on the specified node
func GetOvnkubePodByNode(cs *testclient.ClientSet, node *corev1.Node) (*corev1.Pod, error) {
	labelSelector := "app=ovnkube-node"
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", node.Name)
	pods, err := cs.Pods(testutils.NamespaceOvn).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("failed to find ovnkube-node pod with label selector %q and field selector %q", labelSelector, fieldSelector)
	}
	return &pods.Items[0], nil
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

// PtpEnabled returns the topology of a given node, filtering using the given selector.
func PtpEnabled(client *client.ClientSet) ([]NodeTopology, error) {
	nodeDevicesList, err := client.NodePtpDevices(ptpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(nodeDevicesList.Items) == 0 {
		return nil, fmt.Errorf("Zero nodes found")
	}

	nodeTopologyList := []NodeTopology{}

	nodesList, err := MatchingOptionalSelectorPTP(nodeDevicesList.Items)
	for _, node := range nodesList {
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
	NodeObject, err := client.Client.Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	NodeObject.Labels[key] = value
	NodeObject, err = client.Client.Nodes().Update(context.Background(), NodeObject, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return NodeObject, nil
}

// LabeledNodesCount return the number of nodes with the given label.
func LabeledNodesCount(label string) (int, error) {
	nodeList, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", label)})
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
	toMatch, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
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

	return MatchingCustomSelectorByName(toFilter, NodesSelector)
}

// MatchingCustomSelectorByName filter the given slice with only the nodes matching the given custom selector.
// The nodesSelector must be in the form of label=value.
// For example: nodesSelector="sctp=true"
func MatchingCustomSelectorByName(toFilter []string, nodesSelector string) ([]string, error) {
	toMatch, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: nodesSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("Error in getting nodes matching %s, %v", nodesSelector, err)
	}
	if len(toMatch.Items) == 0 {
		return nil, fmt.Errorf("Failed to get nodes matching %s, %v", nodesSelector, err)
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
		return nil, fmt.Errorf("Failed to find matching nodes with %s", nodesSelector)
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

// MatchingOptionalSelectorPTP filter the given slice with only the nodes matching the optional selector.
// If no selector is set, it returns the same list.
// The NODES_SELECTOR must be set with a labelselector expression.
// For example: NODES_SELECTOR="sctp=true"
func MatchingOptionalSelectorPTP(toFilter []ptpv1.NodePtpDevice) ([]ptpv1.NodePtpDevice, error) {
	if NodesSelector == "" {
		return toFilter, nil
	}
	toMatch, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: NodesSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("Error in getting nodes matching the %s label selector, %v", NodesSelector, err)
	}
	if len(toMatch.Items) == 0 {
		return nil, fmt.Errorf("Failed to get nodes matching %s label selector", NodesSelector)
	}

	res := make([]ptpv1.NodePtpDevice, 0)
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

// HavingSCTPEnabled takes a list node names and return the same list with only
// the nodes that has SCTP enabled.
func HavingSCTPEnabled(inputNodeNames []string) ([]string, error) {
	allNodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes while searching for SCTP: %w", err)
	}

	ret := make([]string, 0)
	inputNodeNamesAsString := strings.Join(inputNodeNames, " ")
	for _, node := range allNodes.Items {
		if !strings.Contains(inputNodeNamesAsString, node.Name) {
			continue
		}

		nodeHasSctp, err := HasSCTPEnabled(&node)
		if err != nil {
			return ret, err
		}

		if nodeHasSctp {
			ret = append(ret, node.Name)
		}
	}

	return ret, nil
}

func HasSCTPEnabled(node *corev1.Node) (bool, error) {
	cmd := []string{"chroot", "/rootfs", "bash", "-c", "lsmod | grep sctp"}
	out, err := ExecCommandOnMachineConfigDaemon(client.Client, node, cmd)

	if err != nil {
		var exitError exec.ExitError
		if errors.As(err, &exitError) {
			// grep exits with code 1 in case of no output lines
			if exitError.ExitStatus() == 1 {
				return false, nil
			}
		}

		return false, fmt.Errorf(
			"can't determine if node [%s] has SCTP enabled. cmd [%s] exited with code [%d], output:[%s]: %w",
			node.Name, cmd, exitError.ExitStatus(), string(out), err,
		)
	}

	return true, nil
}

// AvailableForSelector returns nodes available for given nodeSelector
func AvailableForSelector(nodeSelector map[string]string) ([]corev1.Node, error) {
	nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: nodeSelectorAsString(nodeSelector),
	})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

// SelectorUnion returns a union of 2 node selectors
func SelectorUnion(nodeSelector1 map[string]string, nodeSelector2 map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range nodeSelector1 {
		result[k] = v
	}
	for k, v := range nodeSelector2 {
		result[k] = v
	}
	return result
}

func nodeSelectorAsString(nodeSelector map[string]string) string {
	result := ""
	first := true
	for k, v := range nodeSelector {
		if !first {
			first = false
			result = result + ", "
		}
		result = result + fmt.Sprintf("%s=%s", k, v)

	}
	return result
}

func IsSingleNodeCluster() (bool, error) {
	nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	return len(nodes.Items) == 1, nil
}

// FindRoleLabel loops over node labels and return the first with key like
// "node-role.kubernetest.io/*", except "node-role.kubernetest.io/worker".
//
// Consider that a node is suppose to have only one "custom role" (role != "worker"). If a node
// has two or more custom roles, MachineConfigOperato stops managing that node.
func FindRoleLabel(node *corev1.Node) string {
	for label := range node.Labels {
		if !strings.HasPrefix(label, "node-role.kubernetes.io/") {
			continue
		}

		if label == "node-role.kubernetes.io/worker" {
			continue
		}

		return strings.TrimPrefix(label, "node-role.kubernetes.io/")
	}

	return ""
}

// AddRoleTo adds the "node-role.kubernetes.io/<role>" to the given node
func AddRoleTo(nodeName, role string) error {
	return setLabel(nodeName, "node-role.kubernetes.io/"+role, "")
}

// RemoveRoleFrom removes the "node-role.kubernetes.io/<role>" from the given node
func RemoveRoleFrom(nodeName, role string) error {
	return setLabel(nodeName, "node-role.kubernetes.io/"+role, nil)
}

func setLabel(nodeName, label string, value any) error {
	patch := struct {
		Metadata map[string]any `json:"metadata"`
	}{
		Metadata: map[string]any{
			"labels": map[string]any{
				label: value,
			},
		},
	}

	patchData, err := json.Marshal(&patch)
	if err != nil {
		return fmt.Errorf("can't marshal patch data[%v] to label node[%s]: %w", patch, nodeName, err)
	}

	_, err = client.Client.Nodes().Patch(context.Background(), nodeName, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("can't patch labels[%s] of node[%s]: %w", string(patchData), nodeName, err)
	}

	return nil
}
