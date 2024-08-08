package bond

import (
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var sriovclient *sriovtestclient.ClientSet

const testNamespace string = "nmstate-sriov-testing"

func init() {
}

var _ = Describe("[sriov] NMState Operator Integration", func() {

	BeforeAll(func() {
		err := namespaces.Create(namespaces.BondTestNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())

		By("CleanSriov...")
		networks.CleanSriov(sriovclient)

		By("Discover SRIOV devices")
		
		Expect(err).ToNot(HaveOccurred())

	})

	Context("when a SriovNetworkNodePolicy and NodeNetworkConfigurationPolicySpec targets the same NIC", func() {

		BeforeAll(func() {
			namespaces.CleanPods(testNamespace, client.Client)
		})

		It("VFs should spawn correctly", func() {

			
			
		})
	})
})
/*
// findUnusedDevice search through all the nodes and NICs to find an SRIOV capable device that
// is not used as primary device for the node.
func findUnusedSRIOVDevice() (string, sriovv1.InterfaceExt) {
	ctx := context.Background()

	sriovCapableNodes, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
	for _, nodeName := range sriovCapableNodes.Nodes {
		sriovDevices, err := sriovCapableNodes.FindSriovDevices(nodeName)
		Expect(err).ToNot(HaveOccurred())

		nodeNetworkState := nmstatev1beta1.NodeNetworkState{}
		err = client.Client.Get(ctx, runtimeclient.ObjectKey{Name: nodeName, Namespace: namespaces.IntelOperator}, nodeNetworkState)
		Expect(err).ToNot(HaveOccurred())

		getNodeStateInterface(nodeNetworkState)
	}
	for nodeName, sriovNetworkNodeState := range sriovCapableNodes.States {
		nic, err := findUnusedDeviceOnNode(nodeName)
	}
}

func getNodeStateInterface(state *nmstatev1beta1.NodeNetworkState) {
	//for _, interface := range state.Status.CurrentState.Interface

}


*/