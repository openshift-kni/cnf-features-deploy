package clean

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift/ptp-operator/test/utils"
	"github.com/openshift/ptp-operator/test/utils/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// All removes any configuration applied by ptp tests.
func All() error {
	err := Configs()
	if err != nil {
		return err
	}

	nodeList, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", utils.PtpGrandmasterNodeLabel)})
	if err != nil {
		return fmt.Errorf("clean.All: Failed to retrieve grandmaster node list %v", err)
	}
	for _, node := range nodeList.Items {
		delete(node.Labels, utils.PtpGrandmasterNodeLabel)
		_, err = client.Client.Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
	}

	nodeList, err = client.Client.Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", utils.PtpSlaveNodeLabel)})
	if err != nil {
		return fmt.Errorf("clean.All: Failed to retrieve slave node list %v", err)
	}
	for _, node := range nodeList.Items {
		delete(node.Labels, utils.PtpSlaveNodeLabel)
		_, err = client.Client.Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("clean.All: Failed to remove label from %s %v", node.Name, err)
		}
	}
	return nil
}

func Configs() error {
	ptpconfigList, err := client.Client.PtpConfigs(utils.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("clean.All: Failed to retrieve ptp config list %v", err)
	}

	for _, ptpConfig := range ptpconfigList.Items {
		if ptpConfig.Name == utils.PtpGrandMasterPolicyName || ptpConfig.Name == utils.PtpSlavePolicyName {
			err = client.Client.PtpConfigs(utils.PtpLinuxDaemonNamespace).Delete(context.Background(), ptpConfig.Name, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("clean.All: Failed to delete ptp config %s %v", ptpConfig.Name, err)
			}
		}
	}
	for i := 0; i < 20; i++ {
		ptpconfigList, err = client.Client.PtpConfigs(utils.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("clean.All: Failed to list ptp config  %v", err)
		}
		if len(ptpconfigList.Items) == 0 {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("clean.All: Failed to list ptp config  %v", err)
}
