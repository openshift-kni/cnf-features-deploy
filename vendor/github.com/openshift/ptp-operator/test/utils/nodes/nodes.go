package nodes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift/ptp-operator/test/utils"
	"github.com/openshift/ptp-operator/test/utils/client"
)

type NodeTopology struct {
	NodeName      string
	InterfaceList []string
	NodeObject    *corev1.Node
}

func GetNodeTopology(client *client.ClientSet) ([]NodeTopology, error) {
	nodeDevicesList, err := client.NodePtpDevices(PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{})
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
