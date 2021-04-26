package fec

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	sriovfecv1 "github.com/open-ness/openshift-operator/sriov-fec/api/v1"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
)

const (
	// sriovFecClusterConfigName contains the name of the SriovFecCluster config policies allow by the operator
	// https://github.com/open-ness/openshift-operator/blob/main/sriov-fec/controllers/sriovfecclusterconfig_controller.go#L87
	sriovFecClusterConfigName = "config"
	// acc100DeviceID contains the deviceID of the Acc100 accelerator card
	acc100DeviceID     = "0d5c"
	acc100ResourceName = "intel.com/intel_fec_acc100"
)

var _ = Describe("fec", func() {
	var nodeName string
	var pciAddress string
	var err error
	numVfs := 2

	BeforeEach(func() {
		if discovery.Enabled() {
			// TODO: change this to read the sriovFecClusterConfig and validate all the nodes
			Skip("sriov-fec test doesn't support discovery mode")
		}

		nodeName, pciAddress, err = getAcc100Device()
		err = createSriovFecClusterObject(nodeName, pciAddress, numVfs)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := restoreDefaultConfig()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Expose resource on the node", func() {
		It("should show resources under the node", func() {
			Eventually(func() int64 {
				testedNode, err := client.Client.Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName(acc100ResourceName)]
				allocatable, _ := resNum.AsInt64()
				return allocatable
			}, 10*time.Minute, time.Second).Should(Equal(int64(numVfs)))
		})
	})
})

func restoreDefaultConfig() error {
	sriovFecCluster := &sriovfecv1.SriovFecClusterConfig{}
	err := client.Client.Get(context.TODO(), runtimeClient.ObjectKey{Name: sriovFecClusterConfigName, Namespace: namespaces.IntelOperator}, sriovFecCluster)
	if err != nil {
		return nil
	}

	sriovFecCluster.Spec.Nodes[0].PhysicalFunctions[0].VFAmount = 0

	err = client.Client.Update(context.TODO(), sriovFecCluster)
	if err != nil {
		return nil
	}

	Eventually(func() int64 {
		testedNode, err := client.Client.Nodes().Get(context.Background(), sriovFecCluster.Spec.Nodes[0].NodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName(acc100ResourceName)]
		allocatable, _ := resNum.AsInt64()
		return allocatable
	}, 10*time.Minute, time.Second).Should(Equal(int64(0)))

	return nil
}

func getAcc100Device() (string, string, error) {
	nodesWithAcc100, err := getSriovFecNodes()
	nn, err := nodes.MatchingOptionalSelectorByName(nodesWithAcc100)
	if err != nil {
		return "", "", err
	}

	if len(nn) == 0 {
		return "", "", fmt.Errorf("0 nodes with ACC100 accelerator found")
	}

	pci, err := getAcc100PciFromNode(nn[0])
	if err != nil {
		return "", "", err
	}

	return nn[0], pci, nil
}

func createSriovFecClusterObject(nodeName string, pciAddress string, numVfs int) error {
	queueGroupConfig := sriovfecv1.QueueGroupConfig{
		AqDepthLog2:     4,
		NumAqsPerGroups: 16,
		NumQueueGroups:  2,
	}

	sriovFecClusterConfig := &sriovfecv1.SriovFecClusterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: sriovFecClusterConfigName, Namespace: namespaces.IntelOperator},
		Spec: sriovfecv1.SriovFecClusterConfigSpec{
			Nodes: []sriovfecv1.NodeConfig{
				{
					NodeName: nodeName,
					PhysicalFunctions: []sriovfecv1.PhysicalFunctionConfig{
						{
							PCIAddress: pciAddress,
							PFDriver:   "pci-pf-stub",
							VFAmount:   numVfs,
							VFDriver:   "vfio-pci",
							BBDevConfig: sriovfecv1.BBDevConfig{
								ACC100: &sriovfecv1.ACC100BBDevConfig{
									Downlink4G:   queueGroupConfig,
									Downlink5G:   queueGroupConfig,
									Uplink4G:     queueGroupConfig,
									Uplink5G:     queueGroupConfig,
									PFMode:       false,
									MaxQueueSize: 1024,
									NumVfBundles: 16,
								},
							},
						},
					},
				},
			},
		}}

	err := client.Client.Create(context.TODO(), sriovFecClusterConfig, &runtimeClient.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func getSriovFecNodes() ([]string, error) {
	nodesWithAcc100 := []string{}

	sriovFecNodeList := &sriovfecv1.SriovFecNodeConfigList{}
	err := client.Client.List(context.TODO(), sriovFecNodeList, &runtimeClient.ListOptions{Namespace: namespaces.IntelOperator})
	if err != nil {
		return nil, err
	}

	for _, sriovFecNode := range sriovFecNodeList.Items {
		for _, accelerator := range sriovFecNode.Status.Inventory.SriovAccelerators {
			if accelerator.DeviceID == acc100DeviceID {
				nodesWithAcc100 = append(nodesWithAcc100, sriovFecNode.Name)
			}
		}
	}

	return nodesWithAcc100, nil
}

func getAcc100PciFromNode(nodeName string) (string, error) {
	sriovFecNodeConfig := &sriovfecv1.SriovFecNodeConfig{}
	err := client.Client.Get(context.TODO(), runtimeClient.ObjectKey{Name: nodeName, Namespace: namespaces.IntelOperator}, sriovFecNodeConfig)
	if err != nil {
		return "", err
	}

	for _, accelerator := range sriovFecNodeConfig.Status.Inventory.SriovAccelerators {
		if accelerator.DeviceID == acc100DeviceID {
			return accelerator.PCIAddress, nil
		}
	}

	return "", fmt.Errorf("acc100 card not found under node %s", nodeName)
}

func Clean() {
	sriovFecCluster := &sriovfecv1.SriovFecClusterConfig{}
	err := client.Client.Get(context.TODO(), runtimeClient.ObjectKey{Name: sriovFecClusterConfigName, Namespace: namespaces.IntelOperator}, sriovFecCluster)
	if meta.IsNoMatchError(err) || errors.IsNotFound(err) {
		return
	}
	Expect(err).ToNot(HaveOccurred())

	err = client.Client.Delete(context.TODO(), sriovFecCluster)
	Expect(err).ToNot(HaveOccurred())
}
