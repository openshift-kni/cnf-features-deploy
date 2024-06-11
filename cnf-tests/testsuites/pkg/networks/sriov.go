package networks

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/gomega"

	sriovk8sv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	sriovnetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
)

var (
	MlxVendorID                 = "15b3"
	IntelVendorID               = "8086"
	waitingTime   time.Duration = 20 * time.Minute

	sriovclient *sriovtestclient.ClientSet
)

func init() {
	waitingEnv := os.Getenv("SRIOV_WAITING_TIME")
	newTime, err := strconv.Atoi(waitingEnv)
	if err == nil && newTime != 0 {
		waitingTime = time.Duration(newTime) * time.Minute
	}

	sriovclient = sriovtestclient.New("")
}

// IsSriovOperatorInstalled returns true if SriovOperator related Custom Resources are available
// in the cluster, false otherwise.
func IsSriovOperatorInstalled() bool {
	_, err := sriovclient.SriovNetworkNodeStates(namespaces.SRIOVOperator).
		List(context.Background(), metav1.ListOptions{})
	if k8serrors.IsNotFound(err) {
		return false
	}

	Expect(err).ToNot(HaveOccurred())
	return true
}

// CleanSriov cleans SriovNetworks and SriovNetworkNodePolicies with the prefix of `test-`, that are in the `openshift-sriov-network-operator`
func CleanSriov(sriovclient *sriovtestclient.ClientSet) {
	err := sriovnamespaces.CleanNetworks(namespaces.SRIOVOperator, sriovclient)
	Expect(err).ToNot(HaveOccurred())

	if !discovery.Enabled() {
		err = sriovnamespaces.CleanPolicies(namespaces.SRIOVOperator, sriovclient)
		Expect(err).ToNot(HaveOccurred())
	}
	WaitStable(sriovclient)
}

func CreateSriovPolicyAndNetwork(sriovclient *sriovtestclient.ClientSet, namespace, networkName, resourceName, metaPluginsConfig string) {
	numVfs := 4
	sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
	Expect(err).ToNot(HaveOccurred())
	Expect(sriovInfos).ToNot(BeNil())

	nodes, err := nodes.MatchingOptionalSelectorByName(sriovInfos.Nodes)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(nodes)).To(BeNumerically(">", 0))

	node := nodes[0]
	sriovDevice, err := sriovInfos.FindOneSriovDevice(node)
	Expect(err).ToNot(HaveOccurred())

	_, err = sriovnetwork.CreateSriovPolicy(sriovclient, "test-policy", namespaces.SRIOVOperator, sriovDevice.Name, node, numVfs, resourceName, "netdevice")
	Expect(err).ToNot(HaveOccurred())
	WaitStable(sriovclient)

	Eventually(func() int64 {
		testedNode, err := sriovclient.Nodes().Get(context.Background(), node, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/"+resourceName)]
		capacity, _ := resNum.AsInt64()
		return capacity
	}, 10*time.Minute, time.Second).Should(Equal(int64(numVfs)))

	CreateSriovNetwork(sriovclient, sriovDevice, networkName, namespace, namespaces.SRIOVOperator, resourceName, metaPluginsConfig)
}

func CreateSriovNetwork(sriovclient *sriovtestclient.ClientSet, sriovDevice *sriovv1.InterfaceExt, sriovNetworkName, sriovNetworkNamespace, operatorNamespace, resourceName, metaPluginsConfig string) {
	CreateSriovNetworkWithVlan(sriovclient, sriovDevice, sriovNetworkName, sriovNetworkNamespace, operatorNamespace, resourceName, metaPluginsConfig, 0)
}

func CreateSriovNetworkWithVlan(sriovclient *sriovtestclient.ClientSet, sriovDevice *sriovv1.InterfaceExt, sriovNetworkName, sriovNetworkNamespace, operatorNamespace, resourceName, metaPluginsConfig string, vlan int) {
	ipam := `{"type": "host-local","ranges": [[{"subnet": "1.1.1.0/24"}]],"dataDir": "/run/my-orchestrator/container-ipam-state"}`
	err := sriovnetwork.CreateSriovNetwork(sriovclient, sriovDevice, sriovNetworkName, sriovNetworkNamespace, operatorNamespace, resourceName, ipam, func(network *sriovv1.SriovNetwork) {
		if metaPluginsConfig != "" {
			network.Spec.MetaPluginsConfig = metaPluginsConfig
		}
		network.Spec.Vlan = vlan
	})
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		netAttDef := &sriovk8sv1.NetworkAttachmentDefinition{}
		return sriovclient.Get(context.Background(), goclient.ObjectKey{Name: sriovNetworkName, Namespace: sriovNetworkNamespace}, netAttDef)
	}, time.Minute, 5*time.Second).ShouldNot(HaveOccurred())
}

func GetSupportedSriovNics() (map[string]string, error) {
	supportedNicsConfigMap := &corev1.ConfigMap{}

	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: utils.SriovSupportedNicsCM, Namespace: namespaces.SRIOVOperator}, supportedNicsConfigMap)
	if err != nil {
		return nil, fmt.Errorf("cannot get supportedNicsConfigMap: %w", err)
	}

	return supportedNicsConfigMap.Data, nil
}

// if the sriov is not able in the kernel for intel nic the totalVF will be 0 so we skip the device
// That is not the case for Mellanox devices that will report 0 until we configure the sriov interfaces
// with the mstconfig package
func IsIntelDisabledNic(iface sriovv1.InterfaceExt) bool {
	if iface.Vendor == IntelVendorID && iface.TotalVfs == 0 {
		return true
	}
	return false
}

func CreateSriovPolicyAndNetworkDPDKOnlyWithVhost(dpdkResourceName, workerCnfLabelSelector string) {
	createSriovPolicyAndNetwork(dpdkResourceName, workerCnfLabelSelector, true)
}

func CreateSriovPolicyAndNetworkDPDKOnly(dpdkResourceName, workerCnfLabelSelector string) {
	createSriovPolicyAndNetwork(dpdkResourceName, workerCnfLabelSelector, false)
}

func createSriovPolicyAndNetwork(dpdkResourceName, workerCnfLabelSelector string, needVhostNet bool) {
	sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
	Expect(err).ToNot(HaveOccurred())
	Expect(sriovInfos).NotTo(BeNil())

	nn, err := nodes.MatchingCustomSelectorByName(sriovInfos.Nodes, workerCnfLabelSelector)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(nn)).To(BeNumerically(">", 0))

	sriovDevice, err := sriovInfos.FindOneSriovDevice(nn[0])
	Expect(err).ToNot(HaveOccurred())

	CreatePoliciesDPDKOnly(sriovDevice, nn[0], dpdkResourceName, needVhostNet)
	CreateSriovNetwork(sriovclient, sriovDevice, "test-dpdk-network", namespaces.DpdkTest, namespaces.SRIOVOperator, dpdkResourceName, "")
}

func CreatePoliciesDPDKOnly(sriovDevice *sriovv1.InterfaceExt, testNode string, dpdkResourceName string, needVhostNet bool) {
	CreateDpdkPolicy(sriovDevice, testNode, dpdkResourceName, "", 5, needVhostNet)
	WaitStable(sriovclient)

	Eventually(func() int64 {
		testedNode, err := sriovclient.Nodes().Get(context.Background(), testNode, metav1.GetOptions{})
		if err != nil {
			return -1
		}
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/"+dpdkResourceName)]
		capacity, _ := resNum.AsInt64()
		return capacity
	}, 10*time.Minute, time.Second).Should(Equal(int64(5)))
}

func CreateDpdkPolicy(sriovDevice *sriovv1.InterfaceExt, testNode, dpdkResourceName, pfPartition string, vfsNum int, needVhostNet bool) {
	dpdkPolicy := &sriovv1.SriovNetworkNodePolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-dpdkpolicy-",
			Namespace:    namespaces.SRIOVOperator,
		},
		Spec: sriovv1.SriovNetworkNodePolicySpec{
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": testNode,
			},
			NumVfs:       vfsNum,
			ResourceName: dpdkResourceName,
			Priority:     99,
			NicSelector: sriovv1.SriovNetworkNicSelector{
				PfNames: []string{sriovDevice.Name + pfPartition},
			},
			DeviceType:   "netdevice",
			NeedVhostNet: needVhostNet,
		},
	}

	// Mellanox device
	if sriovDevice.Vendor == MlxVendorID {
		dpdkPolicy.Spec.IsRdma = true
	}

	// Intel device
	if sriovDevice.Vendor == IntelVendorID {
		dpdkPolicy.Spec.DeviceType = "vfio-pci"
	}
	err := sriovclient.Create(context.Background(), dpdkPolicy)
	Expect(err).ToNot(HaveOccurred())
}

// WaitStable waits for the sriov setup to be stable after
// configuration modification.
func WaitStable(sriovclient *sriovtestclient.ClientSet) {
	var snoTimeoutMultiplier time.Duration = 1
	isSNO, err := nodes.IsSingleNodeCluster()
	Expect(err).ToNot(HaveOccurred())
	if isSNO {
		snoTimeoutMultiplier = 2
	}
	// This used to be to check for sriov not to be stable first,
	// then stable. The issue is that if no configuration is applied, then
	// the status won't never go to not stable and the test will fail.
	// TODO: find a better way to handle this scenario
	time.Sleep(15 * time.Second)
	Eventually(func() bool {
		res, _ := sriovcluster.SriovStable("openshift-sriov-network-operator", sriovclient)
		// ignoring the error for the disconnected cluster scenario
		return res
	}, waitingTime*snoTimeoutMultiplier, 1*time.Second).Should(BeTrue())

	Eventually(func() bool {
		isClusterReady, _ := sriovcluster.IsClusterStable(sriovclient)
		// ignoring the error for the disconnected cluster scenario
		return isClusterReady
	}, waitingTime*snoTimeoutMultiplier, 1*time.Second).Should(BeTrue())
}
