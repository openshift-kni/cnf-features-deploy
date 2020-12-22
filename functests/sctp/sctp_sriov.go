package sctp

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovv1 "github.com/openshift/sriov-network-operator/api/v1"
	sriovtestclient "github.com/openshift/sriov-network-operator/test/util/client"
	sriovcluster "github.com/openshift/sriov-network-operator/test/util/cluster"
	sriovdiscovery "github.com/openshift/sriov-network-operator/test/util/discovery"
	sriovnamespaces "github.com/openshift/sriov-network-operator/test/util/namespaces"
	sriovnetwork "github.com/openshift/sriov-network-operator/test/util/network"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/discovery"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/sriov"
)

const (
	testNetwork = "test-sctp-sriov-network"
)

var _ = Describe("[sriov] SCTP integration", func() {
	sriovclient := sriovtestclient.New("")
	discoveryFailed := false
	var testNode string

	execute.BeforeAll(func() {
		err := namespaces.Create(TestNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := client.Client.ServiceAccounts(TestNamespace).Get(context.Background(), "default", metav1.GetOptions{})
			return err
		}, 1*time.Minute, 5*time.Second).Should(Not(HaveOccurred()))

		err = namespaces.Clean(TestNamespace, "testsctp-", client.Client)
		Expect(err).ToNot(HaveOccurred())

		selector := sctpNodeSelector
		if selector == "" {
			selector, err = findSCTPNodeSelector()
			Expect(err).ToNot(HaveOccurred())
		}
		sriovSctpNodes := discoverSRIOVNodes(sriovclient, selector)

		if !discovery.Enabled() {
			err := sriovnamespaces.Clean(namespaces.SRIOVOperator, TestNamespace, sriovclient, false)
			Expect(err).ToNot(HaveOccurred())
			createSRIOVNetworkPolicy(sriovclient, sriovSctpNodes.Nodes[0], sriovSctpNodes, "sctptestres")
			sriov.WaitStable(sriovclient)
			Eventually(func() int64 {
				testedNode, err := client.Client.Nodes().Get(context.Background(), sriovSctpNodes.Nodes[0], metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				resNum, _ := testedNode.Status.Allocatable["openshift.io/sctptestres"]
				capacity, _ := resNum.AsInt64()
				return capacity
			}, 10*time.Minute, time.Second).Should(Equal(int64(5)))

		} else {
			err := sriovnamespaces.CleanNetworks(namespaces.SRIOVOperator, sriovclient)
			Expect(err).ToNot(HaveOccurred())
			err = sriovnamespaces.CleanPods(TestNamespace, sriovclient)
			Expect(err).ToNot(HaveOccurred())
		}

		node, resourceName, numVfs, sriovDevice, err := sriovdiscovery.DiscoveredResources(sriovclient,
			sriovSctpNodes, namespaces.SRIOVOperator, func(policy sriovv1.SriovNetworkNodePolicy) bool {
				if policy.Spec.DeviceType != "netdevice" {
					return false
				}
				return true
			},
			func(node string, sriovDeviceList []*sriovv1.InterfaceExt) (*sriovv1.InterfaceExt, bool) {
				if len(sriovDeviceList) == 0 {
					return nil, false
				}
				return sriovDeviceList[0], true
			},
		)
		Expect(err).ToNot(HaveOccurred())

		discoveryFailed = node == "" || resourceName == "" || numVfs < 2
		if discoveryFailed {
			return
		}
		ipam := `{"type": "host-local","ranges": [[{"subnet": "1.1.1.0/24"}]],"dataDir": "/run/my-orchestrator/container-ipam-state"}`
		err = sriovnetwork.CreateSriovNetwork(sriovclient, sriovDevice, testNetwork, TestNamespace, namespaces.SRIOVOperator, resourceName, ipam)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			netAttDef := &netattdefv1.NetworkAttachmentDefinition{}
			return sriovclient.Get(context.Background(), runtimeclient.ObjectKey{Name: testNetwork, Namespace: TestNamespace}, netAttDef)
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		testNode = node
	})

	var _ = Describe("Test Connectivity", func() {
		Context("Connectivity between client and server", func() {
			BeforeEach(func() {
				namespaces.Clean(TestNamespace, "testsctp-", client.Client)
				if discoveryFailed {
					Skip("Discovery failed, failed to find a valid node with SCTP and SRIOV enabled")
				}
			})
			AfterEach(func() {
				// TODO: This is ugly and works only because this is the only
				// test in this context. To be removed and replaced with a clean on top
				// of sriov tests generic / no policy
				if !discovery.Enabled() {
					err := sriovnamespaces.Clean(namespaces.SRIOVOperator, TestNamespace, sriovclient, false)
					Expect(err).ToNot(HaveOccurred())
					sriov.WaitStable(sriovclient)
				}
			})

			It("Should work over a SR-IOV device", func() {
				By("Starting the server")
				serverPod := startServerPod(testNode, TestNamespace, testNetwork)
				ips, err := sriovnetwork.GetSriovNicIPs(serverPod, "net1")
				Expect(err).ToNot(HaveOccurred())
				Expect(ips).NotTo(BeNil(), "No sriov network interface found.")

				testClientServerConnection(client.Client, TestNamespace, ips[0],
					30101, testNode, serverPod.Name, true, testNetwork)
			})
		})
	})
})

func discoverSRIOVNodes(client *sriovtestclient.ClientSet, sctpSelector string) *sriovcluster.EnabledNodes {
	sriovInfos, err := sriovcluster.DiscoverSriov(client, namespaces.SRIOVOperator)
	Expect(err).ToNot(HaveOccurred())
	Expect(sriovInfos).ToNot(BeNil())

	var sriovSCTPInfos sriovcluster.EnabledNodes
	sriovSCTPInfos.Nodes = make([]string, 0)
	sriovSCTPInfos.States = make(map[string]sriovv1.SriovNetworkNodeState, 0)

	sctpNodeSelector, err = findSCTPNodeSelector()
	Expect(err).ToNot(HaveOccurred())

	sctpNodes := getSCTPNodes(sctpNodeSelector)
	Expect(len(sctpNodes)).To(BeNumerically(">", 0))

	for _, sctpNode := range sctpNodes {
		for _, sriovNode := range sriovInfos.Nodes {
			if sctpNode.Name == sriovNode {
				sriovSCTPInfos.Nodes = append(sriovSCTPInfos.Nodes, sriovNode)
				sriovSCTPInfos.States[sriovNode] = sriovInfos.States[sriovNode]
				break
			}
		}
	}
	Expect(len(sriovSCTPInfos.Nodes)).To(BeNumerically(">", 0))
	return &sriovSCTPInfos
}

func createSRIOVNetworkPolicy(client *sriovtestclient.ClientSet, node string, sriovInfos *sriovcluster.EnabledNodes, resourceName string) {
	// For the context of tests is better to use a Mellanox card
	// as they support all the virtual function flags
	// if we don't find a Mellanox card we fall back to any sriov
	// capability interface and skip the rate limit test.
	intf, err := sriovInfos.FindOneMellanoxSriovDevice(node)
	if err != nil {
		intf, err = sriovInfos.FindOneSriovDevice(node)
		Expect(err).ToNot(HaveOccurred())
	}

	config := &sriovv1.SriovNetworkNodePolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-sctppolicy",
			Namespace:    namespaces.SRIOVOperator,
		},

		Spec: sriovv1.SriovNetworkNodePolicySpec{
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": node,
			},
			NumVfs:       5,
			ResourceName: resourceName,
			Priority:     99,
			NicSelector: sriovv1.SriovNetworkNicSelector{
				PfNames: []string{intf.Name},
			},
			DeviceType: "netdevice",
		},
	}
	err = client.Create(context.Background(), config)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() sriovv1.Interfaces {
		nodeState, err := client.SriovNetworkNodeStates(namespaces.SRIOVOperator).Get(context.Background(), node, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return nodeState.Spec.Interfaces
	}, 1*time.Minute, 1*time.Second).Should(ContainElement(MatchFields(
		IgnoreExtras,
		Fields{
			"Name":   Equal(intf.Name),
			"NumVfs": Equal(5),
		})))
}
