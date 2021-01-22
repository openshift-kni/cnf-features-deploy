package vrf

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovClean "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/clean"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovNamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	sriovNetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"
	client "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/discovery"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/sriov"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	resourceNameVRF = "sriovnicvrf"
	testNetworkRed  = "test-vrf-sriov-network-red"
	testNetworkBlue = "test-vrf-sriov-network-blue"
)

var _ = Describe("[sriov] VRF integration", func() {
	describe := func(ipStack string) string {

		VRFParameters, err := newVRFTestParameters(ipStack)
		if err != nil {
			return fmt.Sprintf("error in parameters: node=%s, ipStack=%s", "Same Node", ipStack)
		}
		params, err := json.Marshal(VRFParameters)
		if err != nil {
			return fmt.Sprintf("error in parameters: node=%s, ipStack=%s", "Same Node", ipStack)
		}

		return string(params)
	}

	apiclient := client.New("")
	sriovclient := sriovtestclient.New("")
	var nodesList []string

	execute.BeforeAll(func() {
		// TODO: We should add support of discovery mode to sriov+vrf integration tests
		if discovery.Enabled() {
			Skip("Discovery is not supported.")
		}
		err := sriovClean.All()
		Expect(err).ToNot(HaveOccurred())
		sriov.WaitStable(sriovclient)
		err = namespaces.Create(sriovNamespaces.Test, apiclient)
		Expect(err).ToNot(HaveOccurred())
		sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
		Expect(err).ToNot(HaveOccurred())
		Expect(sriovInfos).ToNot(BeNil())
		nodesList, err = nodes.MatchingOptionalSelectorByName(sriovInfos.Nodes)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(nodesList)).To(BeNumerically(">", 0))
		sriovDevice, err := sriovInfos.FindOneSriovDevice(nodesList[0])
		Expect(err).ToNot(HaveOccurred())
		_, err = sriovNetwork.CreateSriovPolicy(sriovclient, "test-policy-", namespaces.SRIOVOperator, sriovDevice.Name, nodesList[0], 5, resourceNameVRF)
		Expect(err).ToNot(HaveOccurred())
		sriov.WaitStable(sriovclient)

		ipam := `{"type": "static"}`
		err = sriovNetwork.CreateSriovNetwork(sriovclient, sriovDevice, testNetworkRed, sriovNamespaces.Test,
			namespaces.SRIOVOperator, resourceNameVRF, ipam, defineSriovNetworkMetaPluginsVRFConfig(VRFRedName))
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			netAttDef := &netattdefv1.NetworkAttachmentDefinition{}
			return sriovclient.Get(context.Background(), runtimeclient.ObjectKey{Name: testNetworkRed, Namespace: sriovNamespaces.Test}, netAttDef)
		}, 60*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		err = sriovNetwork.CreateSriovNetwork(sriovclient, sriovDevice, testNetworkBlue, sriovNamespaces.Test,
			namespaces.SRIOVOperator, resourceNameVRF, ipam, defineSriovNetworkMetaPluginsVRFConfig(VRFBlueName))
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			netAttDef := &netattdefv1.NetworkAttachmentDefinition{}
			return sriovclient.Get(context.Background(), runtimeclient.ObjectKey{Name: testNetworkBlue, Namespace: sriovNamespaces.Test}, netAttDef)
		}, 60*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	})
	AfterEach(func() {
		err := namespaces.CleanPods(sriovNamespaces.Test, apiclient)
		Expect(err).ToNot(HaveOccurred())
		//TODO: We need to remove this block and add cleanup in to sriov-network-operator project
		err = sriovClean.All()
		Expect(err).ToNot(HaveOccurred())
		sriov.WaitStable(sriovclient)
	})

	Context("", func() {
		// OCP-36303
		DescribeTable("Integration: SRIOV, IPAM: static, Interfaces: 1, Scheme: 2 Pods 2 VRFs OCP Primary network overlap",
			func(ipStack string) {
				testVRFScenario(apiclient, sriovNamespaces.Test, nodesList[0], testNetworkBlue, testNetworkRed, ipStack)
			},
			Entry(describe, ipStackIPv4),
		)
	})
})

func defineSriovNetworkMetaPluginsVRFConfig(VRFName string) func(network *sriovv1.SriovNetwork) {
	return func(network *sriovv1.SriovNetwork) {
		network.Spec.MetaPluginsConfig = fmt.Sprintf(`{"type": "vrf", "vrfname": "%s"}`, VRFName)
	}
}
