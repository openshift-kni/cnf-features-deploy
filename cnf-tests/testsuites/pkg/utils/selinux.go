package utils

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
)

func IsContainerUseDevicesSEBooleanDisabled(node string) (bool, error) {
	mcd, err := getMachineConfigDaemon(node)
	if err != nil {
		return false, err
	}

	output, err := pods.ExecCommand(client.Client, *mcd, []string{"sh", "-c", "nsenter --mount=/proc/1/ns/mnt -- sh -c 'getsebool container_use_devices'"})
	if err != nil {
		return false, err
	}

	return strings.Contains(output.String(), "off"), nil
}
func SetContainerUseDevicesSEBoolean(node string) error {
	mcd, err := getMachineConfigDaemon(node)
	if err != nil {
		return err
	}

	_, err = pods.ExecCommand(client.Client, *mcd, []string{"sh", "-c", "nsenter --mount=/proc/1/ns/mnt -- sh -c 'setsebool container_use_devices 1'"})
	return err
}

func UnsetContainerUseDevicesSEBoolean(node string) error {
	mcd, err := getMachineConfigDaemon(node)
	if err != nil {
		return err
	}

	_, err = pods.ExecCommand(client.Client, *mcd, []string{"sh", "-c", "nsenter --mount=/proc/1/ns/mnt -- sh -c 'setsebool container_use_devices 0'"})
	return err
}

func getMachineConfigDaemon(node string) (*corev1.Pod, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node}).String(),
		LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": "machine-config-daemon"}).String(),
	}

	mcds, err := client.Client.Pods(utils.NamespaceMachineConfigOperator).List(context.Background(), listOptions)
	if err != nil {
		return nil, err
	}

	if len(mcds.Items) == 0 {
		return nil, fmt.Errorf("cluster machines are not managed by the machine operator")
	}

	return &mcds.Items[0], nil
}
