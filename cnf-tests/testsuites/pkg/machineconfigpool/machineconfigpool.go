package machineconfigpool

import (
	"context"
	"fmt"
	"time"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoScheme "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/scheme"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitForCondition waits until the machine config pool will have specified condition type with the expected status
func WaitForCondition(
	cs *testclient.ClientSet,
	mcp *mcov1.MachineConfigPool,
	conditionType mcov1.MachineConfigPoolConditionType,
	conditionStatus corev1.ConditionStatus,
	timeout time.Duration,
) error {
	return wait.PollImmediate(10*time.Second, timeout, func() (bool, error) {
		mcpUpdated, err := cs.MachineConfigPools().Get(context.Background(), mcp.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		for _, c := range mcpUpdated.Status.Conditions {
			if c.Type == conditionType && c.Status == conditionStatus {
				return true, nil
			}
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

// WaitForMCPStable waits until the mcp is stable
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

	// We need to wait a long time here for the nodes to reboot
	err = WaitForCondition(
		testclient.Client,
		&mcp,
		mcov1.MachineConfigPoolUpdated,
		corev1.ConditionTrue,
		time.Duration(30*mcp.Status.MachineCount)*time.Minute)

	return err
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
