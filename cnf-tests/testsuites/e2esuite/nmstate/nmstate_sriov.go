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
		//sriovCapableNodes, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
		//Expect(err).ToNot(HaveOccurred())

	})

	Context("when a SriovNetworkNodePolicy and NodeNetworkConfigurationPolicySpec targets the same NIC", func() {

		BeforeAll(func() {
			namespaces.CleanPods(testNamespace, client.Client)
		})

		It("VFs should spawn correctly", func() {

			
			
		})
	})
})
