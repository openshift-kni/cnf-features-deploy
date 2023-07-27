package dpdk

import (
	"context"
	"fmt"
	"path/filepath"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	sriovnetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	utilNodes "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/performanceprofile"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	"github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils/nodes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var _ = Describe("[sriov] NUMA node alignment", Ordered, func() {

	BeforeAll(func() {
		if discovery.Enabled() {
			Skip("Discovery mode not supported")
		}

		isSNO, err := utilNodes.IsSingleNodeCluster()
		Expect(err).ToNot(HaveOccurred())
		if isSNO {
			Skip("Single Node openshift not yet supported")
		}

		perfProfile, err := performanceprofile.FindDefaultPerformanceProfile(performanceProfileName)
		Expect(err).ToNot(HaveOccurred())
		if !performanceprofile.IsSingleNUMANode(perfProfile) {
			Skip("SR-IOV NUMA test suite expects a performance profile with 'single-numa-node' to be present")
		}

		err = namespaces.Create(sriovnamespaces.Test, client.Client)
		Expect(err).ToNot(HaveOccurred())

		By("Clean SRIOV policies and networks")
		networks.CleanSriov(sriovclient)

		By("Discover SRIOV devices")
		sriovCapableNodes, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(sriovCapableNodes.Nodes)).To(BeNumerically(">", 0))
		testingNode, err := nodes.GetByName(sriovCapableNodes.Nodes[0])
		Expect(err).ToNot(HaveOccurred())
		By("Using node " + testingNode.Name)

		sriovDevices, err := sriovCapableNodes.FindSriovDevices(testingNode.Name)
		Expect(err).ToNot(HaveOccurred())

		numa0Device, err := findDeviceOnNUMANode(testingNode, sriovDevices, "0")
		Expect(err).ToNot(HaveOccurred())
		By("Using NUMA0 device " + numa0Device.Name)

		numa1Device, err := findDeviceOnNUMANode(testingNode, sriovDevices, "1")
		Expect(err).ToNot(HaveOccurred())
		By("Using NUMA1 device " + numa1Device.Name)

		// SriovNetworkNodePolicy
		// NUMA node0 device excludeTopology = true
		// NUMA node0 device excludeTopology = false
		// NUMA node1 device excludeTopology = true
		// NUMA node1 device excludeTopology = false

		By("Create SRIOV policies and networks")

		ipam := `{ "type": "host-local", "subnet": "192.0.2.0/24" }`

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa0Device, "#0-3",
			"test-numa-0-exclude-topology-false-", testingNode.Name,
			"testNuma0ExcludeTopoplogyFalse", ipam, false)

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa0Device, "#4-7",
			"test-numa-0-exclude-topology-true-", testingNode.Name,
			"testNuma0ExcludeTopoplogyTrue", ipam, true)

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa1Device, "#0-3",
			"test-numa-1-exclude-topology-true-", testingNode.Name,
			"testNuma1ExcludeTopoplogyFalse", ipam, false)

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa1Device, "#4-7",
			"test-numa-1-exclude-topology-true-", testingNode.Name,
			"testNuma1ExcludeTopoplogyTrue", ipam, true)

		By("Waiting for SRIOV devices to get configured")
		networks.WaitStable(sriovclient)
	})

	BeforeEach(func() {
		By("Clean any pods in " + sriovnamespaces.Test + " namespace")
		namespaces.CleanPods(sriovnamespaces.Test, sriovclient)
	})

	It("Validate the creation of a pod with excludeTopology set to False and an SRIOV interface in a different NUMA node than the pod", func() {
		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "1", "100m")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-exclude-topology-false")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodFailed))
			g.Expect(actualPod.Status.Reason).To(Equal("TopologyAffinityError"))
		}).Should(Succeed())
	})
})

func findDeviceOnNUMANode(node *corev1.Node, devices []*sriovv1.InterfaceExt, numaNode string) (*sriovv1.InterfaceExt, error) {
	for _, device := range devices {
		out, err := nodes.ExecCommandOnNode([]string{
			"cat",
			filepath.Clean(filepath.Join("/sys/class/net/", device.Name, "/device/numa_node")),
		}, node)
		if err != nil {
			klog.Warningf("can't get device [%s] NUMA node: out(%s) err(%s)", device.Name, string(out), err.Error())
			continue
		}

		if out == numaNode {
			return device, nil
		}
	}

	return nil, fmt.Errorf("can't find any SR-IOV device on NUMA [%s] for node [%s]. Available devices: %+v", numaNode, node.Name, devices)
}

func withExcludeTopology(excludeTopology bool) func(*sriovv1.SriovNetworkNodePolicy) {
	return func(p *sriovv1.SriovNetworkNodePolicy) {
		p.Spec.ExcludeTopology = excludeTopology
	}
}

func createSriovNetworkAndPolicyForNumaAffinityTest(numVFs int, intf *sriovv1.InterfaceExt, vfSelector, policyGeneratedName, nodeName, resourceName, ipam string, excludeTopology bool) {
	_, err := sriovnetwork.CreateSriovPolicy(
		sriovclient, policyGeneratedName, namespaces.SRIOVOperator,
		intf.Name+vfSelector, nodeName, numVFs,
		resourceName, "netdevice",
		withExcludeTopology(false),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	sriovnetwork.CreateSriovNetwork(sriovclient, intf, "test-numa-0-exclude-topology-false",
		sriovnamespaces.Test, namespaces.SRIOVOperator, "testNuma0ExcludeTopoplogyFalse", ipam)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

}
