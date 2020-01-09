package machineconfigpool

import (
	"time"

	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

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
		mcpUpdated, err := cs.MachineConfigPools().Get(mcp.Name, metav1.GetOptions{})
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
