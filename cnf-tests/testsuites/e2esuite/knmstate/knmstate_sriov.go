package knmstate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	"github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"
	nmstateshared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var sriovclient *sriovtestclient.ClientSet

const testNamespace string = "test-knmstate-sriov"

const nmstateDesiredStateTemplateStaticIP string = `interfaces:
- name: %s
  description: test for SR-IOV network operator integration
  type: ethernet
  state: up
  ipv4:
    dhcp: false
    address:
    - ip: 192.0.2.2
      prefix-length: 24
    enabled: true`

func init() {
	sriovclient = sriovtestclient.New("")
}

var _ = Describe("[knmstate] SR-IOV Network Operator Integration", func() {

	var sriovInfos *cluster.EnabledNodes

	BeforeEach(func() {
		if !networks.IsSriovOperatorInstalled() {
			Skip("SR-IOV operator not installed on the cluster.")
		}

		if !isKnmstateOperatorInstalled() {
			Skip("Kubernetes NMState operator not installed on the cluster.")
		}

		disableNMStateFn := enableNMState()
		DeferCleanup(disableNMStateFn)

		err := namespaces.Create(testNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())

		By("CleanSriov...")
		networks.CleanSriov(sriovclient)

		By("Discover SRIOV devices")
		sriovInfos, err = cluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(sriovInfos.Nodes)).ToNot(BeZero(), "no node with SR-IOV NICs found")
	})

	Context("when a SriovNetworkNodePolicy and NodeNetworkConfigurationPolicySpec target the same NIC", func() {

		BeforeEach(func() {
			namespaces.CleanPods(testNamespace, client.Client)
		})

		It("VFs should spawn correctly", func() {
			node, devices, err := findNodeWithNonPrimarySriovNIC(sriovInfos)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(devices)).ToNot(BeZero())

			By("Using node " + node)

			testDevice := devices[0]
			By("Using device " + testDevice.Name)

			By("Configuring an IPv4 address on Physical Function via NMState")
			nodeNetworkConfigPolicy := nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-sriov-device-" + testDevice.Name},
				Spec: nmstateshared.NodeNetworkConfigurationPolicySpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": node,
					},
					DesiredState: nmstateshared.State{
						Raw: nmstateshared.RawState(fmt.Sprintf(nmstateDesiredStateTemplateStaticIP, testDevice.Name)),
					},
				},
			}
			err = client.Client.Create(context.Background(), &nodeNetworkConfigPolicy)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(client.Client.Delete, context.Background(), &nodeNetworkConfigPolicy)

			waitForNMStatePolicyToBeStable(nodeNetworkConfigPolicy.Name)

			By("Configuring VF on device via SriovNetworkNodePolicy")
			sriovNetworkNodePolicy, err := sriovnetwork.CreateSriovPolicy(sriovclient, "test-nmstate-", namespaces.SRIOVOperator, testDevice.Name, node, 10, "testnmstateresource", "netdevice")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(sriovclient.Delete, context.Background(), sriovNetworkNodePolicy)

			By("Verifying Virtual Functions have been correctly configured")
			Eventually(func() int64 {
				testedNode, err := sriovclient.Nodes().Get(context.Background(), node, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				resNum := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/testnmstateresource")]
				capacity, _ := resNum.AsInt64()
				return capacity
			}).
				WithPolling(5*time.Second).
				WithTimeout(10*time.Minute).
				Should(Equal(int64(10)), "the number of the observed VFs is not that same as the one sriov created")

			By("Verifying Physical Function IP address has been correctly configured")
			Eventually(func(g Gomega) {
				out, err := ipAddrShow(node, testDevice.Name)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(out).To(ContainSubstring("192.0.2.2"))
			}).
				WithPolling(5 * time.Second).
				WithTimeout(1 * time.Minute).
				Should(Succeed())
		})
	})
})

func isKnmstateOperatorInstalled() bool {
	list := nmstatev1.NMStateList{}
	err := client.Client.List(context.Background(), &list)
	if k8serrors.IsNotFound(err) {
		return false
	}

	Expect(err).ToNot(HaveOccurred())
	return true
}

func ipAddrShow(node, intf string) (string, error) {
	nodeObj, err := client.Client.Nodes().Get(context.Background(), node, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("can't get node[%s]: %w", node, err)
	}

	cmd := []string{"ip", "-brief", "address", "show", "dev", intf}
	buf, err := nodes.ExecCommandOnMachineConfigDaemon(client.Client, nodeObj, cmd)
	if err != nil {
		return "", fmt.Errorf("command[%v] failed with output[%s] on node[%s]: %w", cmd, string(buf), node, err)
	}
	return string(buf), err
}

// findNodeWithNonPrimarySriovNIC returns a node name with its SR-IOV devices. The device list does not include
// the one used for primary networking, if it is an SR-IOV device.
func findNodeWithNonPrimarySriovNIC(n *cluster.EnabledNodes) (string, []*sriovv1.InterfaceExt, error) {
	var returnError error

	for _, node := range n.Nodes {
		devices, err := n.FindSriovDevices(node)
		if err != nil {
			returnError = errors.Join(returnError, err)
			continue
		}

		mainDeviceName, err := getPrimaryNICForNode(node)
		if err != nil {
			returnError = errors.Join(returnError, err)
			continue
		}

		nonPrimaryDevices := []*sriovv1.InterfaceExt{}
		for _, device := range devices {
			if device.Name == mainDeviceName {
				continue
			}
			nonPrimaryDevices = append(nonPrimaryDevices, device)
		}

		if len(nonPrimaryDevices) > 0 {
			return node, nonPrimaryDevices, nil
		}
	}

	return "", nil, fmt.Errorf("can't find any SR-IOV devices in cluster's nodes: %w", returnError)
}

func getPrimaryNICForNode(node string) (string, error) {
	nodeObj, err := client.Client.Nodes().Get(context.Background(), node, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	buf, err := nodes.ExecCommandOnMachineConfigDaemon(client.Client, nodeObj,
		[]string{"/bin/bash", "-c", "chroot /rootfs /usr/bin/ovs-vsctl list-ports br-ex | /usr/bin/grep -v patch"})
	if err != nil {
		return "", err
	}

	exp, err := regexp.Compile("\r\n")
	if err != nil {
		return "", err
	}

	return exp.ReplaceAllString(string(buf), ""), nil
}

func waitForNMStatePolicyToBeStable(nmstatePolicyName string) {
	key := runtimeclient.ObjectKey{Name: nmstatePolicyName}
	Eventually(func(g Gomega) {
		nmstatePolicy := nmstatev1.NodeNetworkConfigurationPolicy{}
		err := client.Client.Get(context.Background(), key, &nmstatePolicy)
		g.Expect(err).ToNot(HaveOccurred())
		availableCondition := nmstatePolicy.Status.Conditions.
			Find(nmstateshared.NodeNetworkConfigurationPolicyConditionAvailable)
		g.Expect(availableCondition).ToNot(BeNil())
		g.Expect(availableCondition.Status).
			To(Equal(corev1.ConditionTrue))
	}).
		WithOffset(1).
		WithPolling(10 * time.Second).
		WithTimeout(1 * time.Minute).
		Should(Succeed())
}

// enableNMState activates the kubernetes nmstate operator by creating the nmstate.io/NMState
// resource if it's missing. It return a function to restore the cluster to its previous state.
// If kNMState is already running, this function does nothing and the returned callback is harmless too.
func enableNMState() func() {
	key := runtimeclient.ObjectKey{Name: "nmstate"}
	nmstate := nmstatev1.NMState{}
	err := client.Client.Get(context.Background(), key, &nmstate)
	if err == nil {
		By("NMState resource already present")
		return func() {}
	}

	Expect(k8serrors.IsNotFound(err)).To(BeTrue())

	By("Creating NMState resource")
	nmstate = nmstatev1.NMState{ObjectMeta: metav1.ObjectMeta{Name: "nmstate"}}
	err = client.Client.Create(context.Background(), &nmstate)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func(g Gomega) {
		nodeNetworkStates := nmstatev1beta1.NodeNetworkStateList{}
		err := client.Client.List(context.Background(), &nodeNetworkStates)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(len(nodeNetworkStates.Items)).ToNot(BeZero())
	}).
		WithPolling(5 * time.Second).
		WithTimeout(1 * time.Minute).
		Should(Succeed())

	return func() {
		By("Disabling kNMState operator")
		err := client.Client.Delete(context.Background(), &nmstate)
		Expect(err).ToNot(HaveOccurred())
	}
}
