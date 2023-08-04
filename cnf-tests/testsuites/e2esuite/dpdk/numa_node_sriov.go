package dpdk

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

	var (
		numa0DeviceList []*sriovv1.InterfaceExt
		numa1DeviceList []*sriovv1.InterfaceExt
	)

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

		numa0DeviceList, err = findDevicesOnNUMANode(testingNode, sriovDevices, "0")
		Expect(len(numa0DeviceList)).To(BeNumerically(">=", 1))
		By("Using NUMA0 device1 " + numa0DeviceList[0].Name)

		numa1DeviceList, err = findDevicesOnNUMANode(testingNode, sriovDevices, "1")
		Expect(len(numa1DeviceList)).To(BeNumerically(">=", 1))
		By("Using NUMA1 device1 " + numa1DeviceList[0].Name)

		// SriovNetworkNodePolicy
		// NUMA node0 device1 excludeTopology = false
		// NUMA node0 device1 excludeTopology = true
		// NUMA node0 device2 excludeTopology = false
		// NUMA node0 device2 excludeTopology = true
		// NUMA node1 device3 excludeTopology = false
		// NUMA node1 device3 excludeTopology = true

		By("Create SRIOV policies and networks")

		ipam := `{ "type": "host-local", "subnet": "192.0.2.0/24" }`

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa0DeviceList[0], "#0-3",
			"test-numa-0-nic1-exclude-topology-false-", testingNode.Name,
			"testNuma0NIC1ExcludeTopoplogyFalse", ipam, false)

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa0DeviceList[0], "#4-7",
			"test-numa-0-nic1-exclude-topology-true-", testingNode.Name,
			"testNuma0NIC1ExcludeTopoplogyTrue", ipam, true)

		if len(numa0DeviceList) > 1 {
			By("Using NUMA0 device2 " + numa0DeviceList[1].Name)

			createSriovNetworkAndPolicyForNumaAffinityTest(8, numa0DeviceList[1], "#0-3",
				"test-numa-0-nic2-exclude-topology-false-", testingNode.Name,
				"testNuma1NIC2ExcludeTopoplogyFalse", ipam, false)

			createSriovNetworkAndPolicyForNumaAffinityTest(8, numa0DeviceList[1], "#4-7",
				"test-numa-0-nic2-exclude-topology-true-", testingNode.Name,
				"testNuma1NIC2ExcludeTopoplogyTrue", ipam, true)
		}

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa1DeviceList[0], "#0-3",
			"test-numa-1-nic1-exclude-topology-false-", testingNode.Name,
			"testNuma1NIC1ExcludeTopoplogyFalse", ipam, false)

		createSriovNetworkAndPolicyForNumaAffinityTest(8, numa1DeviceList[0], "#4-7",
			"test-numa-1-nic1-exclude-topology-true-", testingNode.Name,
			"testNuma1NIC1ExcludeTopoplogyTrue", ipam, true)

		By("Waiting for SRIOV devices to get configured")
		networks.WaitStable(sriovclient)
	})

	BeforeEach(func() {
		By("Clean any pods in " + sriovnamespaces.Test + " namespace")
		namespaces.CleanPods(sriovnamespaces.Test, sriovclient)
	})

	It("Validate the creation of a pod with excludeTopology set to False and an SRIOV interface in a different NUMA node than the pod", func() {
		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-nic1-exclude-topology-false-network")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodFailed))
			g.Expect(actualPod.Status.Reason).To(Equal("TopologyAffinityError"))
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

	It("Validate the creation of a pod with excludeTopology set to True and an SRIOV interface in a same NUMA node "+
		"than the pod", func() {
		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-1-nic1-exclude-topology-true-network")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodRunning))
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("Validate Pod NUMA Node")
		expectPodCPUsAreOnNUMANode(pod, 1)

		By("Create server Pod and run E2E ICMP validation")
		validateE2EICMPTraffic(pod, fmt.Sprintf(`[{"name": "test-numa-1-nic1-exclude-topology-true-network","ips":["192.0.2.250/24"]}]`))
	})

	It("Validate the creation of a pod with excludeTopology set to True and an SRIOV interface in a different NUMA node "+
		"than the pod", func() {
		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-nic1-exclude-topology-true-network")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodRunning))
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("Validate Pod NUMA Node")
		expectPodCPUsAreOnNUMANode(pod, 1)

		By("Create server Pod and run E2E ICMP validation")
		validateE2EICMPTraffic(pod, fmt.Sprintf(`[{"name": "test-numa-0-nic1-exclude-topology-true-network","ips":["192.0.2.250/24"]}]`))
	})

	It("Validate the creation of a pod with two sriovnetworknodepolicies one with excludeTopology False and the "+
		"second true each interface is in different NUMA as the pod", func() {

		if len(numa0DeviceList) < 2 {
			testSkip := "There are not enough Interfaces in NUMA Node 0 to complete this test"
			Skip(testSkip)
		}

		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-nic1-exclude-topology-true-network, "+
			"test-numa-0-nic2-exclude-topology-false-network")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodFailed))
			g.Expect(actualPod.Status.Reason).To(Equal("TopologyAffinityError"))
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

	It("Validate the creation of a pod with excludeTopology set to True and multiple SRIOV interfaces located in "+
		"different NUMA nodes than the pod", func() {

		if len(numa0DeviceList) < 2 {
			testSkip := "There are not enough Interfaces in NUMA Node 0 to complete this test"
			Skip(testSkip)
		}

		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-nic1-exclude-topology-true-network, "+
			"test-numa-0-nic2-exclude-topology-true-network")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodRunning))
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("Validate Pod NUMA Node")
		expectPodCPUsAreOnNUMANode(pod, 1)

		By("Create server Pod and run E2E ICMP validation")
		validateE2EICMPTraffic(pod, fmt.Sprintf(`[{"name": "test-numa-0-nic1-exclude-topology-true-network","ips":["192.0.2.250/24"]}]`))
	})

	It("Validate the creation of a pod with excludeTopology set to False and each interface is "+
		"in the different NUMA as the pod", func() {

		if len(numa0DeviceList) < 2 {
			testSkip := "There are not enough Interfaces in NUMA Node 0 to complete this test"
			Skip(testSkip)
		}

		pod := pods.DefinePod(sriovnamespaces.Test)
		pods.RedefineWithGuaranteedQoS(pod, "2", "500Mi")
		pod = pods.RedefinePodWithNetwork(pod, "test-numa-0-nic1-exclude-topology-false-network, "+
			"test-numa-0-nic2-exclude-topology-false-network")

		pod, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), pod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodFailed))
			g.Expect(actualPod.Status.Message).To(ContainSubstring("Resources cannot be allocated with Topology locality"))
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

	It("Utilize all available VFs then create a pod with guaranteed CPU and excludeTopology set to True", func() {
		barePod := pods.DefinePod(sriovnamespaces.Test)
		podWithQos := pods.RedefineWithGuaranteedQoS(barePod, "2", "500Mi")

		numVFs := 4

		By("Verifies a pod can consume all the available VFs")
		useAllVFsNetworkSpec := []string{}
		for vf := 0; vf < numVFs; vf++ {
			useAllVFsNetworkSpec = append(useAllVFsNetworkSpec, "test-numa-0-nic1-exclude-topology-true-network")
		}
		podWithAllVfs := pods.RedefinePodWithNetwork(podWithQos.DeepCopy(), strings.Join(useAllVFsNetworkSpec, ","))

		podWithAllVfs, err := client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), podWithAllVfs, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), podWithAllVfs.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodRunning))
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("A pod that uses a VF should not go to Running state")
		podWithOneVf := pods.RedefinePodWithNetwork(podWithQos.DeepCopy(), "test-numa-0-nic1-exclude-topology-true-network")
		podWithOneVf, err = client.Client.Pods(sriovnamespaces.Test).
			Create(context.Background(), podWithOneVf, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(pods.GetStringEventsForPodFn(client.Client, podWithOneVf), 30*time.Second, 1*time.Second).
			Should(ContainSubstring("Insufficient openshift.io/testNuma0NIC1ExcludeTopoplogyTrue"))

		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), podWithOneVf.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodPending))
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("Release all VFs by deleting the running pod")
		err = client.Client.Pods(sriovnamespaces.Test).
			Delete(context.Background(), podWithAllVfs.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("The pod with one VF should start")
		Eventually(func(g Gomega) {
			actualPod, err := client.Client.Pods(sriovnamespaces.Test).Get(context.Background(), podWithOneVf.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(actualPod.Status.Phase).To(Equal(corev1.PodRunning))
			g.Expect(actualPod.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed))
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

})

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
		withExcludeTopology(excludeTopology),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	err = sriovnetwork.CreateSriovNetwork(sriovclient, intf, policyGeneratedName+"network",
		sriovnamespaces.Test, namespaces.SRIOVOperator, resourceName, ipam)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

}

func validateE2EICMPTraffic(pod *corev1.Pod, annotation string) {
	serverPod := pods.DefinePod(sriovnamespaces.Test)
	serverPod = pods.RedefinePodWithNetwork(serverPod, annotation)
	command := []string{"bash", "-c", "ping -I net1 192.0.2.250 -c 5"}
	serverPod, err := client.Client.Pods(sriovnamespaces.Test).
		Create(context.Background(), serverPod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(func(g Gomega) error {
		_, err = pods.ExecCommand(client.Client, *pod, command)
		return err
	}, 30*time.Second, 1*time.Second).Should(Succeed(), "ICMP traffic failed over SRIOV interface pod interface")
}

func findDevicesOnNUMANode(node *corev1.Node, devices []*sriovv1.InterfaceExt, numaNode string) ([]*sriovv1.InterfaceExt, error) {
	listOfDevices := []*sriovv1.InterfaceExt{}

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
			listOfDevices = append(listOfDevices, device)
		}
	}

	return listOfDevices, nil
}

func expectPodCPUsAreOnNUMANode(pod *corev1.Pod, expectedCPUsNUMA int) {

	buff, err := pods.ExecCommand(client.Client, *pod, []string{"cat", "/sys/fs/cgroup/cpuset.cpus"})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	cpuList, err := getCpuSet(buff.String())
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	numaNode, err := findNUMAForCPUs(pod, cpuList)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	ExpectWithOffset(1, numaNode).To(Equal(expectedCPUsNUMA))
}
