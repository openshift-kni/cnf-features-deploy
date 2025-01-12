package machineconfigpool

import (
	"context"
	"fmt"
	"time"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoScheme "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

// WaitForCondition waits until the machine config pool will have specified condition type with the expected status
func WaitForCondition(
	cs *testclient.ClientSet,
	mcp *mcov1.MachineConfigPool,
	conditionType mcov1.MachineConfigPoolConditionType,
	conditionStatus corev1.ConditionStatus,
	timeout time.Duration,
) error {
	klog.Infof("Waiting for MCP %s: %s == %s", mcp.Name, conditionType, conditionStatus)
	return wait.PollImmediate(3*time.Second, timeout, func() (bool, error) {
		mcpUpdated, err := cs.MachineConfigPools().Get(context.Background(), mcp.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		if mcpHasCondition(mcpUpdated, conditionType, conditionStatus) {
			klog.Infof("Condition met for MCP %s: %s == %s", mcp.Name, conditionType, conditionStatus)
			return true, nil
		}

		return false, nil
	})
}

// FindNodeSelectorByMCLabel returns the node selector of the mcp that targets machine configs with mcLabel
func FindNodeSelectorByMCLabel(mcLabel string) (string, error) {
	mcp, err := FindMCPByMCLabel(mcLabel)
	if err != nil {
		return "", err
	}

	if mcp.Spec.MachineConfigSelector.MatchExpressions != nil {
		for _, lsr := range mcp.Spec.MachineConfigSelector.MatchExpressions {
			for _, value := range lsr.Values {
				if value == mcLabel {
					for key, label := range mcp.Spec.NodeSelector.MatchLabels {
						newNodeSelector := key + "=" + label
						return newNodeSelector, nil
					}
				}
			}
		}
	}

	if mcp.Spec.MachineConfigSelector.MatchLabels != nil {
		for _, v := range mcp.Spec.MachineConfigSelector.MatchLabels {
			if v == mcLabel {
				for key, label := range mcp.Spec.NodeSelector.MatchLabels {
					newNodeSelector := key + "=" + label
					return newNodeSelector, nil
				}
			}
		}

	}

	return "", fmt.Errorf("cannot find MCP that targets MC with label: %s", mcLabel)
}

// FindMCPByMCLabel returns the MCP that targets machine configs with mcLabel
func FindMCPByMCLabel(mcLabel string) (mcov1.MachineConfigPool, error) {
	mcpList, err := testclient.Client.MachineConfigPools().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return mcov1.MachineConfigPool{}, err
	}

	for _, mcp := range mcpList.Items {
		for _, lsr := range mcp.Spec.MachineConfigSelector.MatchExpressions {
			for _, value := range lsr.Values {
				if value == mcLabel {
					return mcp, nil
				}
			}
		}
		for _, v := range mcp.Spec.MachineConfigSelector.MatchLabels {
			if v == mcLabel {
				return mcp, nil
			}
		}
	}

	return mcov1.MachineConfigPool{}, fmt.Errorf("cannot find MCP that targets MC with label: %s", mcLabel)
}

// WaitForMCPStable waits until the mcp is updating and then waits
// for mcp to be stable again. Former wait is useful to avoid returning
// from this function before the operator is working.
func WaitForMCPStable(mcp mcov1.MachineConfigPool) error {
	err := WaitForCondition(
		testclient.Client,
		&mcp,
		mcov1.MachineConfigPoolUpdating,
		corev1.ConditionTrue,
		2*time.Minute)

	if err != nil {
		return err
	}

	return WaitForMCPUpdated(mcp)
}

// WaitForMCPUpdated waits for the MCP to be in the updated state.
func WaitForMCPUpdated(mcp mcov1.MachineConfigPool) error {
	// We need to wait a long time here for the nodes to reboot
	return WaitForCondition(
		testclient.Client,
		&mcp,
		mcov1.MachineConfigPoolUpdated,
		corev1.ConditionTrue,
		time.Duration(30*mcp.Status.MachineCount)*time.Minute)
}

// WaitForAtLeastOneMCPUpdating waits until at least one MachineConfigPools in the cluster is updating
func WaitForAtLeastOneMCPUpdating() error {
	return wait.PollImmediate(3*time.Second, 2*time.Minute, func() (bool, error) {
		mcpList, err := testclient.Client.MachineConfigPools().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false, nil
		}

		for _, mcp := range mcpList.Items {
			if mcpHasCondition(&mcp, mcov1.MachineConfigPoolUpdating, corev1.ConditionTrue) {
				klog.Infof("MCP %s is updating", mcp.Name)
				return true, nil
			}
		}

		return false, nil
	})

}

// WaitForAllMCPStable waits until all MachineConfigPools in the cluster are not updating (i.e. Updated or Degraded)
func WaitForAllMCPStable() error {
	mcpList, err := testclient.Client.MachineConfigPools().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("can't list MachineConfigPools: %w", err)
	}

	for _, mcp := range mcpList.Items {
		err = WaitForCondition(
			testclient.Client,
			&mcp,
			mcov1.MachineConfigPoolUpdating,
			corev1.ConditionFalse,
			20*time.Minute)

		if err != nil {
			return err
		}
	}

	return nil

}

// DecodeMCYaml decodes a MachineConfig YAML to a MachineConfig struct
func DecodeMCYaml(mcyaml string) (*mcov1.MachineConfig, error) {
	decode := mcoScheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(mcyaml), nil, nil)
	if err != nil {
		return nil, err
	}
	mc, ok := obj.(*mcov1.MachineConfig)
	if !ok {
		return nil, fmt.Errorf("couldnt create MC object from mcyaml")
	}

	return mc, err
}

// ApplyKubeletConfigToNode creates a KubeletConfig, a MachineConfigPool and a
// `node-role.kubernetes.io/<name>` label in order to target a single node in the cluster.
// The role label is applied to the target node after removing any provious `node-role.kubernetes.io/` label,
// as MachineConfigOperator doesn't support multiple roles.
// Return value is a function that can be used to revert the node labeling.
func ApplyKubeletConfigToNode(node *corev1.Node, name string, spec *mcov1.KubeletConfigSpec) (func() error, error) {
	nilFn := func() error { return nil }

	newNodeRole := name
	newNodeRoleSelector := map[string]string{
		"node-role.kubernetes.io/" + newNodeRole: "",
	}

	mcp := mcov1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": name,
			},
		},

		Spec: mcov1.MachineConfigPoolSpec{
			MachineConfigSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "machineconfiguration.openshift.io/role",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{name, "worker"},
				}},
			},
			Paused: false,
			NodeSelector: &metav1.LabelSelector{
				MatchLabels: newNodeRoleSelector,
			},
		},
	}

	kubeletConfig := &mcov1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       *spec.DeepCopy(),
	}

	// Link the KubeletConfig to the MCP
	kubeletConfig.Spec.MachineConfigPoolSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"machineconfiguration.openshift.io/role": name},
	}

	// Create the KubeletConfig
	_, err := controllerutil.CreateOrUpdate(context.Background(), testclient.Client, kubeletConfig, func() error { return nil })
	if err != nil {
		return nilFn, err
	}
	klog.Infof("Created KubeletConfig %s", kubeletConfig.Name)

	// Create MCP
	_, err = controllerutil.CreateOrUpdate(context.Background(), testclient.Client, &mcp, func() error { return nil })
	if err != nil {
		return nilFn, err
	}
	klog.Infof("Created MachineConfigPool %s", mcp.Name)

	// Following wait ensure the node is rebooted only once, as if we apply the MCP to
	// the node before the KubeletConfig has been rendered, the node will reboot twice.
	klog.Infof("Waiting for KubeletConfig to be rendered to MCP")
	err = waitUntilKubeletConfigHasUpdatedTheMCP(name)
	if err != nil {
		return nilFn, err
	}

	// Move the node role to the new one
	previousNodeRole := nodes.FindRoleLabel(node)
	if previousNodeRole != "" {
		err = nodes.RemoveRoleFrom(node.Name, previousNodeRole)
		if err != nil {
			return nilFn, err
		}
		klog.Infof("Removed role[%s] from node %s", previousNodeRole, node.Name)
	}

	err = nodes.AddRoleTo(node.Name, newNodeRole)
	if err != nil {
		return func() error {
			cleanupErr := nodes.AddRoleTo(node.Name, previousNodeRole)
			if cleanupErr != nil {
				return cleanupErr
			}
			klog.Infof("Restored role[%s] on node %s", previousNodeRole, node.Name)
			return nil
		}, err
	}
	klog.Infof("Added role[%s] to node %s", newNodeRole, node.Name)

	err = WaitForMCPStable(mcp)
	if err != nil {
		return func() error {
			cleanupErr := nodes.RemoveRoleFrom(node.Name, newNodeRole)
			if cleanupErr != nil {
				return cleanupErr
			}
			cleanupErr = nodes.AddRoleTo(node.Name, previousNodeRole)
			if cleanupErr != nil {
				return cleanupErr
			}

			klog.Infof("Moved back node role from [%s] to [%s] on %s", newNodeRole, previousNodeRole, node.Name)
			return nil
		}, err
	}

	return func() error {
		cleanupErr := nodes.RemoveRoleFrom(node.Name, newNodeRole)
		if cleanupErr != nil {
			return cleanupErr
		}
		cleanupErr = nodes.AddRoleTo(node.Name, previousNodeRole)
		if cleanupErr != nil {
			return cleanupErr
		}

		klog.Infof("Moved back node role from [%s] to [%s] on %s", newNodeRole, previousNodeRole, node.Name)

		// We don't know which MCP the node belonged to, so wait for at least one MCP to rollout.
		cleanupErr = WaitForAtLeastOneMCPUpdating()
		if cleanupErr != nil {
			return cleanupErr
		}

		return WaitForAllMCPStable()
	}, nil
}

func waitUntilKubeletConfigHasUpdatedTheMCP(name string) error {
	return wait.Poll(10*time.Second, 3*time.Minute, func() (bool, error) {

		mcp, err := testclient.Client.MachineConfigPools().Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			klog.Warningf("Error while waiting for MachineConfigPool[%s] to be updated: %v", name, err)
			return false, nil
		}

		expectedSource := fmt.Sprintf("99-%s-generated-kubelet", name)

		for _, source := range mcp.Spec.Configuration.Source {
			if source.Name == expectedSource {
				return true, nil
			}
		}

		return false, nil
	})
}

func mcpHasCondition(mcp *mcov1.MachineConfigPool,
	conditionType mcov1.MachineConfigPoolConditionType,
	conditionStatus corev1.ConditionStatus) bool {
	for _, c := range mcp.Status.Conditions {
		if c.Type == conditionType && c.Status == conditionStatus {
			return true
		}
	}

	return false
}
