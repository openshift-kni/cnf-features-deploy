package dpdk

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/utils/pointer"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	sriovnetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/discovery"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/images"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/machineconfigpool"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/sriov"
)

const (
	LOG_ENTRY              = "Accumulated forward statistics for all ports"
	DEMO_APP_NAMESPACE     = "dpdk"
	SERVER_TESTPMD_COMMAND = "testpmd -l ${CPU} -w ${PCIDEVICE_OPENSHIFT_IO_%s} --iova-mode=va -- -i --portmask=0x1 --nb-cores=2 --forward-mode=mac --port-topology=loop --no-mlockall"
	CLIENT_TESTPMD_COMMAND = "testpmd -l ${CPU} -w ${PCIDEVICE_OPENSHIFT_IO_%s} --iova-mode=va -- -i --portmask=0x1 --nb-cores=2 --eth-peer=0,ff:ff:ff:ff:ff:ff --forward-mode=txonly --no-mlockall"
)

var (
	machineConfigPoolName          string
	performanceProfileName         string
	enforcedPerformanceProfileName string

	resourceName = "dpdknic"

	sriovclient *sriovtestclient.ClientSet

	OriginalSriovPolicies      []*sriovv1.SriovNetworkNodePolicy
	OriginalSriovNetworks      []*sriovv1.SriovNetwork
	OriginalPerformanceProfile *performancev2.PerformanceProfile
)

func init() {
	machineConfigPoolName = os.Getenv("ROLE_WORKER_CNF")
	if machineConfigPoolName == "" {
		machineConfigPoolName = "worker-cnf"
	}

	performanceProfileName = os.Getenv("PERF_TEST_PROFILE")
	if performanceProfileName == "" {
		performanceProfileName = "performance"
	} else {
		enforcedPerformanceProfileName = performanceProfileName
	}

	OriginalSriovPolicies = make([]*sriovv1.SriovNetworkNodePolicy, 0)
	OriginalSriovNetworks = make([]*sriovv1.SriovNetwork, 0)

	// Reuse the sriov client
	// Use the SRIOV test client
	sriovclient = sriovtestclient.New("")
}

var _ = Describe("dpdk", func() {
	var dpdkWorkloadPod *corev1.Pod
	var discoverySuccessful bool
	discoveryFailedReason := "Can not run tests in discovery mode. Failed to discover required resources"

	execute.BeforeAll(func() {
		var exist bool
		dpdkWorkloadPod, exist = tryToFindDPDKPod()
		if exist {
			return
		}
		nodeSelector, _ := nodes.PodLabelSelector()

		if discovery.Enabled() {
			var performanceProfiles []*performancev2.PerformanceProfile
			discoverySuccessful, discoveryFailedReason, performanceProfiles = discoverPerformanceProfiles()

			if !discoverySuccessful {
				discoveryFailedReason = "Could not find a valid performance profile"
				return
			}

			discovered, err := discovery.DiscoverPerformanceProfileAndPolicyWithAvailableNodes(client.Client, sriovclient, namespaces.SRIOVOperator, resourceName, performanceProfiles, nodeSelector)
			if err != nil {
				discoverySuccessful, discoveryFailedReason = false, "Can not run tests in discovery mode. Failed to discover required resources."
				return
			}
			profile, sriovDevice := discovered.Profile, discovered.Device
			resourceName = discovered.Resource

			nodeSelector = nodes.SelectorUnion(nodeSelector, profile.Spec.NodeSelector)
			CreateSriovNetwork(sriovDevice, "test-dpdk-network", resourceName)

		} else {
			findOrOverridePerformanceProfile()
			createSriovPolicyAndNetwork()
		}

		var err error
		dpdkWorkloadPod, err = createDPDKWorkload(nodeSelector,
			strings.ToUpper(resourceName),
			fmt.Sprintf(SERVER_TESTPMD_COMMAND, strings.ToUpper(resourceName)),
			60,
			true)
		Expect(err).ToNot(HaveOccurred())

		_, err = createDPDKWorkload(nodeSelector,
			strings.ToUpper(resourceName),
			fmt.Sprintf(CLIENT_TESTPMD_COMMAND, strings.ToUpper(resourceName)),
			10,
			false)
		Expect(err).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		if discovery.Enabled() && !discoverySuccessful {
			Skip(discoveryFailedReason)
		}
	})

	Context("Validate the build", func() {
		It("Should forward and receive packets from a pod running dpdk base on a image created by building config", func() {
			Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
			var out string
			var err error

			if dpdkWorkloadPod.Spec.Containers[0].Image == images.For(images.Dpdk) {
				Skip("skip test as we can't find a dpdk workload running with a s2i build")
			}

			By("Parsing output from the DPDK application")
			Eventually(func() string {
				out, err = pods.GetLog(dpdkWorkloadPod)
				Expect(err).ToNot(HaveOccurred())
				return out
			}, 8*time.Minute, 1*time.Second).Should(ContainSubstring(LOG_ENTRY),
				"Cannot find accumulated statistics")
			checkRxTx(out)
		})
	})

	Context("Validate a DPDK workload running inside a pod", func() {
		It("Should forward and receive packets", func() {
			Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
			var out string
			var err error

			if dpdkWorkloadPod.Spec.Containers[0].Image != images.For(images.Dpdk) {
				Skip("skip test as we find a dpdk workload running with a s2i build")
			}

			By("Parsing output from the DPDK application")
			Eventually(func() string {
				out, err = pods.GetLog(dpdkWorkloadPod)
				Expect(err).ToNot(HaveOccurred())
				return out
			}, 8*time.Minute, 1*time.Second).Should(ContainSubstring(LOG_ENTRY),
				"Cannot find accumulated statistics")
			checkRxTx(out)
		})
	})

	Context("Validate NUMA aliment", func() {
		var cpuList []string
		BeforeEach(func() {
			Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
		})

		execute.BeforeAll(func() {
			buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
			Expect(err).ToNot(HaveOccurred())
			cpuList, err = getCpuSet(buff.String())
			Expect(err).ToNot(HaveOccurred())
		})

		// 28078
		It("should allocate the requested number of cpus", func() {
			numOfCPU := dpdkWorkloadPod.Spec.Containers[0].Resources.Limits.Cpu().Value()
			Expect(len(cpuList)).To(Equal(int(numOfCPU)))
		})

		// 28432
		It("should allocate all the resources on the same NUMA node", func() {
			By("finding the CPUs numa")
			cpuNumaNode, err := findNUMAForCPUs(dpdkWorkloadPod, cpuList)
			Expect(err).ToNot(HaveOccurred())

			By("finding the pci numa")
			pciNumaNode, err := findNUMAForSRIOV(dpdkWorkloadPod)
			Expect(err).ToNot(HaveOccurred())

			By("expecting cpu and pci to be on the same numa")
			Expect(cpuNumaNode).To(Equal(pciNumaNode))
		})
	})

	Context("Validate HugePages", func() {
		var activeNumberOfFreeHugePages int
		var numaNode int

		BeforeEach(func() {
			Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
			buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
			Expect(err).ToNot(HaveOccurred())
			cpuList, err := getCpuSet(buff.String())
			Expect(err).ToNot(HaveOccurred())
			numaNode, err = findNUMAForCPUs(dpdkWorkloadPod, cpuList)
			Expect(err).ToNot(HaveOccurred())

			buff, err = pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat",
				fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-1048576kB/free_hugepages", numaNode)})
			Expect(err).ToNot(HaveOccurred())
			activeNumberOfFreeHugePages, err = strconv.Atoi(strings.Replace(buff.String(), "\r\n", "", 1))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allocate the amount of hugepages requested", func() {
			Expect(activeNumberOfFreeHugePages).To(BeNumerically(">", 0))

			// In case of nodeselector set, this pod will end up in a compliant node because the selection
			// logic is applied to the workload pod.
			pod := pods.DefineWithHugePages(namespaces.DpdkTest, dpdkWorkloadPod.Spec.NodeName)
			pod, err := client.Client.Pods(namespaces.DpdkTest).Create(context.Background(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat",
					fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-1048576kB/free_hugepages", numaNode)})
				Expect(err).ToNot(HaveOccurred())
				numberOfFreeHugePages, err := strconv.Atoi(strings.Replace(buff.String(), "\r\n", "", 1))
				Expect(err).ToNot(HaveOccurred())

				return numberOfFreeHugePages
			}, 5*time.Minute, 5*time.Second).Should(Equal(activeNumberOfFreeHugePages - 1))

			pod, err = client.Client.Pods(namespaces.DpdkTest).Get(context.Background(), pod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))

			err = client.Client.Pods(namespaces.DpdkTest).Delete(context.Background(), pod.Name, metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				_, err := client.Client.Pods(namespaces.DpdkTest).Get(context.Background(), pod.Name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					return err
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(HaveOccurred())
		})
	})

	// TODO: find a better why to restore the configuration
	// This will not work if we use a random order running
	Context("restoring configuration", func() {
		It("should restore the cluster to the original status", func() {
			if !discovery.Enabled() {
				By(" restore performance profile")
				RestorePerformanceProfile()
			}

			By("cleaning the sriov test configuration")
			CleanSriov()
		})
	})
})

// checkRxTx parses the output from the DPDK test application
// and verifies that packets have passed the NIC TX and RX queues
func checkRxTx(out string) {
	lines := strings.Split(out, "\n")
	Expect(len(lines)).To(BeNumerically(">=", 3))
	for i, line := range lines {
		if strings.Contains(line, LOG_ENTRY) {
			d := getNumberOfPackets(lines[i+1], "RX")
			Expect(d).To(BeNumerically(">", 0), "number of received packets should be greater than 0")
			d = getNumberOfPackets(lines[i+2], "TX")
			Expect(d).To(BeNumerically(">", 0), "number of transferred packets should be greater than 0")
			break
		}
	}
}

// getNumberOfPackets parses the string
// and returns an element representing the number of packets
func getNumberOfPackets(line, firstFieldSubstr string) int {
	r := strings.Fields(line)
	Expect(r[0]).To(ContainSubstring(firstFieldSubstr))
	Expect(len(r)).To(Equal(6), "the slice doesn't contain 6 elements")
	d, err := strconv.Atoi(r[1])
	Expect(err).ToNot(HaveOccurred())
	return d
}

func tryToFindDPDKPod() (*corev1.Pod, bool) {
	pod, exist, err := findDPDKWorkloadPod()
	Expect(err).ToNot(HaveOccurred())

	if exist {
		return pod, true
	}

	return nil, false
}

func discoverPerformanceProfiles() (bool, string, []*performancev2.PerformanceProfile) {
	if enforcedPerformanceProfileName != "" {
		performanceProfile, err := findDefaultPerformanceProfile()
		if err != nil {
			return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
		}
		valid, err := validatePerformanceProfile(performanceProfile)
		if !valid || err != nil {
			return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
		}
		return true, "", []*performancev2.PerformanceProfile{performanceProfile}
	}

	performanceProfileList := &performancev2.PerformanceProfileList{}
	var profiles []*performancev2.PerformanceProfile
	err := client.Client.List(context.TODO(), performanceProfileList)
	if err != nil {
		return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
	}
	for _, performanceProfile := range performanceProfileList.Items {
		valid, err := validatePerformanceProfile(&performanceProfile)
		if valid && err == nil {
			profiles = append(profiles, &performanceProfile)
		}
	}
	if len(profiles) > 0 {
		return true, "", profiles
	}
	return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
}

func findDefaultPerformanceProfile() (*performancev2.PerformanceProfile, error) {
	performanceProfile := &performancev2.PerformanceProfile{}
	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: performanceProfileName}, performanceProfile)
	return performanceProfile, err
}

func findOrOverridePerformanceProfile() {
	var valid = true
	performanceProfile, err := findDefaultPerformanceProfile()
	if err != nil {
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		valid = false
		performanceProfile = nil
	}
	if valid {
		valid, err = validatePerformanceProfile(performanceProfile)
		Expect(err).ToNot(HaveOccurred())
	}
	if !valid {
		if performanceProfile != nil {
			OriginalPerformanceProfile = performanceProfile.DeepCopy()

			// Clean and create a new performance profile for the dpdk application
			err = CleanPerformanceProfiles()
			Expect(err).ToNot(HaveOccurred())

			err = WaitForClusterToBeStable()
			Expect(err).ToNot(HaveOccurred())
		}

		err := CreatePerformanceProfile()
		Expect(err).ToNot(HaveOccurred())
		err = WaitForClusterToBeStable()
		Expect(err).ToNot(HaveOccurred())
	}
}

func createSriovPolicyAndNetwork() {
	// Clean and create a new sriov policy and network for the dpdk application
	CleanSriov()
	sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
	Expect(err).ToNot(HaveOccurred())

	Expect(sriovInfos).ToNot(BeNil())

	nn, err := nodes.MatchingOptionalSelectorByName(sriovInfos.Nodes)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(nn)).To(BeNumerically(">", 0))

	sriovDevice, err := sriovInfos.FindOneSriovDevice(nn[0])
	Expect(err).ToNot(HaveOccurred())

	CreateSriovPolicy(sriovDevice, nn[0], 5, resourceName)
	CreateSriovNetwork(sriovDevice, "test-dpdk-network", resourceName)

	// When the dpdk-testing namespace is created it takes time for the network attachment definition to be created
	// there by the sriov network operator
	Eventually(func() error {
		netattachdef := &netattdefv1.NetworkAttachmentDefinition{}
		return client.Client.Get(context.TODO(), goclient.ObjectKey{Name: "test-dpdk-network", Namespace: namespaces.DpdkTest}, netattachdef)
	}, 20*time.Second, time.Second).ShouldNot(HaveOccurred())
}

func validatePerformanceProfile(performanceProfile *performancev2.PerformanceProfile) (bool, error) {

	// Check we have more then two isolated CPU
	cpuSet, err := cpuset.Parse(string(*performanceProfile.Spec.CPU.Isolated))
	if err != nil {
		return false, err
	}

	cpuSetSlice := cpuSet.ToSlice()
	if len(cpuSetSlice) < 6 {
		return false, nil
	}

	if performanceProfile.Spec.HugePages == nil {
		return false, nil
	}

	if *performanceProfile.Spec.HugePages.DefaultHugePagesSize != "1G" {
		return false, nil
	}

	if len(performanceProfile.Spec.HugePages.Pages) == 0 {
		return false, nil
	}

	if performanceProfile.Spec.HugePages.Pages[0].Count < 5 {
		return false, nil
	}

	if performanceProfile.Spec.HugePages.Pages[0].Size != "1G" {
		return false, nil
	}

	return true, nil
}

func CleanPerformanceProfiles() error {
	performanceProfileList := &performancev2.PerformanceProfileList{}
	err := client.Client.List(context.TODO(), performanceProfileList, &goclient.ListOptions{})
	if err != nil {
		return err
	}

	for _, policy := range performanceProfileList.Items {
		err := client.Client.Delete(context.TODO(), &policy, &goclient.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func WaitForClusterToBeStable() error {
	mcp := &mcv1.MachineConfigPool{}
	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: machineConfigPoolName}, mcp)
	if err != nil {
		return err
	}

	err = machineconfigpool.WaitForCondition(
		client.Client,
		&mcv1.MachineConfigPool{ObjectMeta: metav1.ObjectMeta{Name: machineConfigPoolName}},
		mcv1.MachineConfigPoolUpdating,
		corev1.ConditionTrue,
		2*time.Minute)
	if err != nil {
		return err
	}

	// We need to wait a long time here for the node to reboot
	err = machineconfigpool.WaitForCondition(
		client.Client,
		&mcv1.MachineConfigPool{ObjectMeta: metav1.ObjectMeta{Name: machineConfigPoolName}},
		mcv1.MachineConfigPoolUpdated,
		corev1.ConditionTrue,
		time.Duration(20*mcp.Status.MachineCount)*time.Minute)

	return err
}

func CreatePerformanceProfile() error {
	isolatedCPUSet := performancev2.CPUSet("8-15")
	reservedCPUSet := performancev2.CPUSet("0-7")
	hugepageSize := performancev2.HugePageSize("1G")
	performanceProfile := &performancev2.PerformanceProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: performanceProfileName,
		},
		Spec: performancev2.PerformanceProfileSpec{
			CPU: &performancev2.CPU{
				Isolated: &isolatedCPUSet,
				Reserved: &reservedCPUSet,
			},
			HugePages: &performancev2.HugePages{
				DefaultHugePagesSize: &hugepageSize,
				Pages: []performancev2.HugePage{
					{
						Count: 5,
						Size:  hugepageSize,
						Node:  pointer.Int32Ptr(0),
					},
				},
			},
			NodeSelector: map[string]string{
				fmt.Sprintf("node-role.kubernetes.io/%s", machineConfigPoolName): "",
			},
		},
	}

	return client.Client.Create(context.TODO(), performanceProfile)
}

func ValidateSriovNetwork(resourceName string) (bool, error) {
	sriovNetwork := &sriovv1.SriovNetwork{}
	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: "dpdk-network", Namespace: namespaces.SRIOVOperator}, sriovNetwork)

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func ValidateSriovPolicy() (bool, error) {
	sriovPolicies := &sriovv1.SriovNetworkNodePolicyList{}
	err := client.Client.List(context.TODO(), sriovPolicies, &goclient.ListOptions{Namespace: namespaces.SRIOVOperator})
	if err != nil {
		return false, err
	}

	for _, policy := range sriovPolicies.Items {
		if policy.Spec.ResourceName == resourceName {
			return true, nil
		}
	}

	return false, nil
}

func CreateSriovPolicy(sriovDevice *sriovv1.InterfaceExt, testNode string, numVfs int, resourceName string) {
	nodePolicy := &sriovv1.SriovNetworkNodePolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-policy-",
			Namespace:    namespaces.SRIOVOperator,
		},
		Spec: sriovv1.SriovNetworkNodePolicySpec{
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": testNode,
			},
			NumVfs:       numVfs,
			ResourceName: resourceName,
			Priority:     99,
			NicSelector: sriovv1.SriovNetworkNicSelector{
				PfNames: []string{sriovDevice.Name},
			},
			DeviceType: "netdevice",
		},
	}

	// Mellanox device
	if sriovDevice.Vendor == "15b3" {
		nodePolicy.Spec.IsRdma = true
	}

	// Intel device
	if sriovDevice.Vendor == "8086" {
		nodePolicy.Spec.DeviceType = "vfio-pci"
	}
	err := sriovclient.Create(context.Background(), nodePolicy)
	Expect(err).ToNot(HaveOccurred())
	sriov.WaitStable(sriovclient)

	Eventually(func() int64 {
		testedNode, err := sriovclient.Nodes().Get(context.Background(), testNode, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/"+resourceName)]
		capacity, _ := resNum.AsInt64()
		return capacity
	}, 10*time.Minute, time.Second).Should(Equal(int64(numVfs)))
}

func CreateSriovNetwork(sriovDevice *sriovv1.InterfaceExt, sriovNetworkName string, resourceName string) {
	ipam := `{"type": "host-local","ranges": [[{"subnet": "1.1.1.0/24"}]],"dataDir": "/run/my-orchestrator/container-ipam-state"}`
	err := sriovnetwork.CreateSriovNetwork(sriovclient, sriovDevice, sriovNetworkName, namespaces.DpdkTest, namespaces.SRIOVOperator, resourceName, ipam)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		netAttDef := &netattdefv1.NetworkAttachmentDefinition{}
		return sriovclient.Get(context.Background(), goclient.ObjectKey{Name: sriovNetworkName, Namespace: namespaces.DpdkTest}, netAttDef)
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

}

func createDPDKWorkload(nodeSelector map[string]string, resourceName, testpmdCommand string, runningTime int, isServer bool) (*corev1.Pod, error) {
	resources := map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceName("hugepages-1Gi"): resource.MustParse("2Gi"),
		corev1.ResourceMemory:                resource.MustParse("1Gi"),
		corev1.ResourceCPU:                   resource.MustParse("4"),
	}
	container := corev1.Container{
		Name:  "dpdk",
		Image: images.For(images.Dpdk),
		Command: []string{
			"/bin/bash",
			"-c",
			fmt.Sprintf(`set -ex
export CPU=$(cat /sys/fs/cgroup/cpuset/cpuset.cpus)
echo ${CPU}
echo ${PCIDEVICE_OPENSHIFT_IO_%s}

cat <<EOF >test.sh
spawn %s
set timeout 10000
expect "testpmd>"
send -- "port stop 0\r"
expect "testpmd>"
send -- "port detach 0\r"
expect "testpmd>"
send -- "port attach ${PCIDEVICE_OPENSHIFT_IO_%s}\r"
expect "testpmd>"
send -- "port start 0\r"
expect "testpmd>"
send -- "start\r"
expect "testpmd>"
sleep %d
send -- "stop\r"
expect "testpmd>"
send -- "quit\r"
expect eof
EOF

expect -f test.sh

sleep INF
`, resourceName, testpmdCommand, resourceName, runningTime)},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: pointer.Int64Ptr(0),
			Capabilities: &corev1.Capabilities{
				// Enable NET_RAW is required by mellanox nics as they are using the netdevice driver
				// NET_RAW was removed from the default capabilities
				// https://access.redhat.com/security/cve/cve-2020-14386
				Add: []corev1.Capability{"IPC_LOCK", "SYS_RESOURCE", "NET_RAW"},
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "RUN_TYPE",
				Value: "testpmd",
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: resources,
			Limits:   resources,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "hugepages",
				MountPath: "/mnt/huge",
			},
		},
	}

	dpdkPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dpdk-",
			Namespace:    namespaces.DpdkTest,
			Labels: map[string]string{
				"app": "dpdk",
			},
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": "dpdk-testing/test-dpdk-network",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{container},
			Volumes: []corev1.Volume{
				{
					Name: "hugepages",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumHugePages},
					},
				},
			},
		},
	}

	if len(nodeSelector) > 0 {
		dpdkPod.Spec.NodeSelector = nodeSelector
	}

	if nodeSelector != nil && len(nodeSelector) > 0 {
		if dpdkPod.Spec.NodeSelector == nil {
			dpdkPod.Spec.NodeSelector = make(map[string]string)
		}
		for k, v := range nodeSelector {
			dpdkPod.Spec.NodeSelector[k] = v
		}
	}

	imageStream, err := client.Client.ImageStreams(DEMO_APP_NAMESPACE).Get(context.TODO(), "s2i-dpdk-app", metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
	}

	if len(imageStream.Status.Tags) > 0 && !discovery.Enabled() {
		// Use the command from the image
		if isServer {
			dpdkPod.Spec.Containers[0].Command = nil
		}
		dpdkPod.Spec.Containers[0].Image = "image-registry.openshift-image-registry.svc:5000/dpdk/s2i-dpdk-app:latest"

		_, err = client.Client.RoleBindings(DEMO_APP_NAMESPACE).Get(context.TODO(), "system:image-puller", metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}

			//We need to create a rolebinding to allow the dpdk-testing project to pull image from the dpdk project
			roleBind := rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "system:image-puller", Namespace: DEMO_APP_NAMESPACE},
				RoleRef: rbacv1.RoleRef{Name: "system:image-puller", Kind: "ClusterRole", APIGroup: "rbac.authorization.k8s.io"},
				Subjects: []rbacv1.Subject{
					{Kind: "ServiceAccount", Name: "default", Namespace: namespaces.DpdkTest},
				}}

			_, err = client.Client.RoleBindings(DEMO_APP_NAMESPACE).Create(context.TODO(), &roleBind, metav1.CreateOptions{})
			if err != nil {
				return nil, err
			}
		}
	}

	dpdkPod, err = client.Client.Pods(namespaces.DpdkTest).Create(context.Background(), dpdkPod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	err = pods.WaitForCondition(client.Client, dpdkPod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
	if err != nil {
		return nil, err
	}

	err = client.Client.Get(context.TODO(), goclient.ObjectKey{Name: dpdkPod.Name, Namespace: dpdkPod.Namespace}, dpdkPod)
	if err != nil {
		return nil, err
	}

	return dpdkPod, nil
}

// findDPDKWorkloadPod finds a pod running a DPDK application using a know label
// Label: app="dpdk"
func findDPDKWorkloadPod() (*corev1.Pod, bool, error) {
	return findDPDKWorkloadPodByLabelSelector(labels.SelectorFromSet(labels.Set{"app": "dpdk"}).String(), namespaces.DpdkTest)
}

func findDPDKWorkloadPodByLabelSelector(labelSelector, namespace string) (*corev1.Pod, bool, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	p, err := client.Client.Pods(namespace).List(context.Background(), listOptions)
	if err != nil {
		return nil, false, err
	}

	if len(p.Items) == 0 {
		return nil, false, nil
	}

	var pod corev1.Pod
	podReady := false
	for _, pod = range p.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podReady = true
			break
		}
	}

	if !podReady {
		return nil, false, nil
	}

	err = pods.WaitForCondition(client.Client, &pod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
	if err != nil {
		return nil, false, err
	}

	return &pod, true, nil
}

func getCpuSet(cpuListOutput string) ([]string, error) {
	cpuList := make([]string, 0)
	cpuListOutputClean := strings.Replace(cpuListOutput, "\r\n", "", 1)
	cpuListOutputClean = strings.Replace(cpuListOutputClean, " ", "", -1)
	cpuRangeList := strings.Split(cpuListOutputClean, ",")

	for _, cpuRange := range cpuRangeList {
		cpuSplit := strings.Split(cpuRange, "-")
		if len(cpuSplit) == 1 {
			cpuList = append(cpuList, cpuSplit[0])
			continue
		}

		if len(cpuSplit) != 2 {
			return nil, fmt.Errorf("unexpected cpu list: %s", cpuListOutput)
		}

		idx, err := strconv.Atoi(cpuSplit[0])
		if err != nil {
			return nil, err
		}
		endIdx, err := strconv.Atoi(cpuSplit[1])
		if err != nil {
			return nil, err
		}

		for ; idx <= endIdx; idx++ {
			cpuList = append(cpuList, strconv.Itoa(idx))
		}

	}

	return cpuList, nil
}

// findNUMAForCPUs finds the NUMA node if all the CPUs in the list are in the same one
func findNUMAForCPUs(pod *corev1.Pod, cpuList []string) (int, error) {
	buff, err := pods.ExecCommand(client.Client, *pod, []string{"lscpu"})
	Expect(err).ToNot(HaveOccurred())
	findCPUOnSameNuma := false
	numaNode := -1
	for _, line := range strings.Split(buff.String(), "\r\n") {
		if strings.Contains(line, "CPU(s)") && strings.Contains(line, "NUMA") {
			numaNode++
			numaLine := strings.Split(line, "CPU(s):   ")
			Expect(len(numaLine)).To(Equal(2))
			cpuMap := make(map[string]bool)

			cpuNumaList, err := getCpuSet(numaLine[1])
			Expect(err).ToNot(HaveOccurred())
			for _, cpu := range cpuNumaList {
				cpuMap[cpu] = true
			}

			findCPUs := true
			for _, cpu := range cpuList {
				if _, ok := cpuMap[cpu]; !ok {
					findCPUs = false
					break
				}
			}

			if findCPUs {
				findCPUOnSameNuma = true
				break
			}
		}
	}

	if !findCPUOnSameNuma {
		return numaNode, fmt.Errorf("not all the cpus are in the same numa node")
	}

	return numaNode, nil
}

// findNUMAForSRIOV finds the NUMA node for a give PCI address
func findNUMAForSRIOV(pod *corev1.Pod) (int, error) {
	buff, err := pods.ExecCommand(client.Client, *pod, []string{"env"})
	Expect(err).ToNot(HaveOccurred())
	pciAddress := ""
	pciEnvVariableName := fmt.Sprintf("PCIDEVICE_OPENSHIFT_IO_%s", strings.ToUpper(resourceName))
	for _, line := range strings.Split(buff.String(), "\r\n") {
		if strings.Contains(line, pciEnvVariableName) {
			envSplit := strings.Split(line, "=")
			Expect(len(envSplit)).To(Equal(2))
			pciAddress = envSplit[1]
		}
	}
	Expect(pciAddress).ToNot(BeEmpty())

	buff, err = pods.ExecCommand(client.Client, *pod, []string{"lspci", "-v", "-nn", "-mm", "-s", pciAddress})
	Expect(err).ToNot(HaveOccurred())
	for _, line := range strings.Split(buff.String(), "\r\n") {
		if strings.Contains(line, "NUMANode:") {
			numaSplit := strings.Split(line, "NUMANode:\t")
			Expect(len(numaSplit)).To(Equal(2))
			pciNuma, err := strconv.Atoi(numaSplit[1])
			Expect(err).ToNot(HaveOccurred())
			return pciNuma, err
		}
	}
	return -1, fmt.Errorf("failed to find the numa for pci %s", pciEnvVariableName)
}

func RestorePerformanceProfile() {
	if OriginalPerformanceProfile == nil {
		return
	}

	err := CleanPerformanceProfiles()
	Expect(err).ToNot(HaveOccurred())

	err = WaitForClusterToBeStable()
	Expect(err).ToNot(HaveOccurred())

	name := OriginalPerformanceProfile.Name
	OriginalPerformanceProfile.ObjectMeta = metav1.ObjectMeta{Name: name}
	err = client.Client.Create(context.TODO(), OriginalPerformanceProfile)
	Expect(err).ToNot(HaveOccurred())

	err = WaitForClusterToBeStable()
	Expect(err).ToNot(HaveOccurred())
}

func CleanSriov() {
	// This clean only the policy and networks with the prefix of test
	err := sriovnamespaces.CleanPods(namespaces.DpdkTest, sriovclient)
	Expect(err).ToNot(HaveOccurred())
	err = sriovnamespaces.CleanNetworks(namespaces.SRIOVOperator, sriovclient)
	Expect(err).ToNot(HaveOccurred())

	if !discovery.Enabled() {
		err = sriovnamespaces.CleanPolicies(namespaces.SRIOVOperator, sriovclient)
		Expect(err).ToNot(HaveOccurred())
	}
	sriov.WaitStable(sriovclient)
}
