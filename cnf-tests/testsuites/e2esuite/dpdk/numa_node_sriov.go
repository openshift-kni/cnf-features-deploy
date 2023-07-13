package dpdk

import (
	"context"
	"fmt"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
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
	"math/rand"
	"strings"
	"time"
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

		By("Apply single-numa-node performance profile")
		perfProfile := performanceprofile.DefineSingleNUMANode("single-numa-node-pp", machineConfigPoolName)
		err = performanceprofile.OverridePerformanceProfile("single-numa-node-pp", machineConfigPoolName, perfProfile)
		Expect(err).ToNot(HaveOccurred())

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

		createSriovNetworkAndPolicy(
			withNodeSelector(testingNode),
			withNumVFs(8), withPfNameSelector(numa0Device.Name+"#0-3"),
			withNetworkNameAndNamespace(sriovnamespaces.Test, "test-numa-0-exclude-topology-false"),
			withExcludeTopology(false),
		)

		createSriovNetworkAndPolicy(
			withNodeSelector(testingNode),
			withNumVFs(8), withPfNameSelector(numa0Device.Name+"#4-7"),
			withNetworkNameAndNamespace(sriovnamespaces.Test, "test-numa-0-exclude-topology-true"),
			withExcludeTopology(true),
		)

		createSriovNetworkAndPolicy(
			withNodeSelector(testingNode),
			withNumVFs(8), withPfNameSelector(numa1Device.Name+"#0-3"),
			withNetworkNameAndNamespace(sriovnamespaces.Test, "test-numa-1-exclude-topology-false"),
			withExcludeTopology(false),
		)

		createSriovNetworkAndPolicy(
			withNodeSelector(testingNode),
			withNumVFs(8), withPfNameSelector(numa1Device.Name+"#4-7"),
			withNetworkNameAndNamespace(sriovnamespaces.Test, "test-numa-1-exclude-topology-true"),
			withExcludeTopology(true),
		)

		By("Waiting for SRIOV devices to get configured")
		networks.WaitStable(sriovclient)
	})

	AfterAll(func() {
		By("Cleaning performance profiles")
		err := performanceprofile.RestorePerformanceProfile(machineConfigPoolName)
		Expect(err).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		By("Clean any pods in " + sriovnamespaces.Test + " namespace")
		namespaces.CleanPods(sriovnamespaces.Test, sriovclient)
	})

	It("Validate the creation of a pod with excludeTopology set to False and an SRIOV interface in a different NUMA node than the pod", func() {
		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-exclude-topology-false")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodFailed))
			g.Expect(actualPod.Status.Message).To(ContainSubstring("Resources cannot be allocated with Topology locality"))
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

	It("Validate the creation of a pod with excludeTopology set to True and an SRIOV interface in a different "+
		"NUMA node than the pod", func() {
		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-exclude-topology-true")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodRunning))
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("Create server Pod and run E2E ICMP validation")
		err = validateE2EICMPTraffic(pod, fmt.Sprintf(`[{"name": "test-numa-0-exclude-topology-true","ips":["192.0.2.250/24"]}]`))
		Expect(err).ToNot(HaveOccurred(), "ICMP traffic failed over SRIOV interface pod interface")
	})

	It("Validate the creation of a pod with excludeTopology set to True and an SRIOV interface in a same NUMA node "+
		"than the pod", func() {
		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-1-exclude-topology-true")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodRunning))
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("Create server Pod and run E2E ICMP validation")
		err = validateE2EICMPTraffic(pod, fmt.Sprintf(`[{"name": "test-numa-1-exclude-topology-true","ips":["192.0.2.250/24"]}]`))
		Expect(err).ToNot(HaveOccurred(), "ICMP traffic failed over SRIOV interface pod interface")
	})
})

func findDeviceOnNUMANode(node *corev1.Node, devices []*sriovv1.InterfaceExt, numaNode string) (*sriovv1.InterfaceExt, error) {
	for _, device := range devices {
		out, err := nodes.ExecCommandOnNode([]string{"cat", fmt.Sprintf("/sys/class/net/%s/device/numa_node", device.Name)}, node)
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

func withNodeSelector(node *corev1.Node) func(*sriovv1.SriovNetworkNodePolicy, *sriovv1.SriovNetwork) {
	return func(p *sriovv1.SriovNetworkNodePolicy, n *sriovv1.SriovNetwork) {
		p.Spec.NodeSelector = map[string]string{
			"kubernetes.io/hostname": node.Name,
		}
	}
}

func withPfNameSelector(pfNameSelector string) func(*sriovv1.SriovNetworkNodePolicy, *sriovv1.SriovNetwork) {
	return func(p *sriovv1.SriovNetworkNodePolicy, n *sriovv1.SriovNetwork) {
		p.Spec.NicSelector = sriovv1.SriovNetworkNicSelector{
			PfNames: []string{pfNameSelector},
		}
		p.ObjectMeta.Name = "test-numa-test-policy-" + strings.ReplaceAll(pfNameSelector, "#", "-")
	}
}

func withExcludeTopology(excludeTopology bool) func(*sriovv1.SriovNetworkNodePolicy, *sriovv1.SriovNetwork) {
	return func(p *sriovv1.SriovNetworkNodePolicy, n *sriovv1.SriovNetwork) {
		p.Spec.ExcludeTopology = excludeTopology
	}
}

func withNumVFs(numVFs int) func(*sriovv1.SriovNetworkNodePolicy, *sriovv1.SriovNetwork) {
	return func(p *sriovv1.SriovNetworkNodePolicy, n *sriovv1.SriovNetwork) {
		p.Spec.NumVfs = numVFs
	}
}

func withNetworkNameAndNamespace(namespace, name string) func(*sriovv1.SriovNetworkNodePolicy, *sriovv1.SriovNetwork) {
	return func(p *sriovv1.SriovNetworkNodePolicy, n *sriovv1.SriovNetwork) {
		n.ObjectMeta.Name = name
		n.Spec.NetworkNamespace = namespace
	}
}

func createSriovNetworkAndPolicy(opts ...func(*sriovv1.SriovNetworkNodePolicy, *sriovv1.SriovNetwork)) {
	resourceName := fmt.Sprintf("numaSriovResource%d", rand.Intn(1000000)+1000000)
	policy := &sriovv1.SriovNetworkNodePolicy{
		Spec: sriovv1.SriovNetworkNodePolicySpec{
			ResourceName: resourceName,
			Priority:     99,
			DeviceType:   "netdevice",
		},
	}

	network := &sriovv1.SriovNetwork{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: sriovv1.SriovNetworkSpec{
			ResourceName: resourceName,
			IPAM:         `{ "type": "host-local", "subnet": "192.0.2.0/24" }`,
		}}

	for _, opt := range opts {
		opt(policy, network)
	}

	policy, err := sriovclient.SriovNetworkNodePolicies(namespaces.SRIOVOperator).
		Create(context.Background(), policy, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	network, err = sriovclient.SriovNetworks(namespaces.SRIOVOperator).
		Create(context.Background(), network, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	klog.Infof("created policy[%s] and network[%s]", policy.Name, network.Name)
}

func validateE2EICMPTraffic(pod *corev1.Pod, annotation string) error {
	By("Create SRIOV Network")
	serverPod := pods.DefinePod(sriovnamespaces.Test)
	serverPod = pods.RedefinePodWithNetwork(serverPod, annotation)
	command := []string{"bash", "-c", "ping -I net1 192.0.2.250 -c 5"}
	serverPod, err := client.Client.Pods(sriovnamespaces.Test).
		Create(context.Background(), serverPod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	_, err = pods.ExecCommand(client.Client, *pod, command)

	return err

}
