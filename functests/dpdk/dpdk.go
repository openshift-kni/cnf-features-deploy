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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/utils/pointer"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	perfv1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	sriovk8sv1 "github.com/openshift/sriov-network-operator/pkg/apis/k8s/v1"
	sriovv1 "github.com/openshift/sriov-network-operator/pkg/apis/sriovnetwork/v1"
	sriovtestclient "github.com/openshift/sriov-network-operator/test/util/client"
	sriovcluster "github.com/openshift/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/openshift/sriov-network-operator/test/util/namespaces"
	sriovnetwork "github.com/openshift/sriov-network-operator/test/util/network"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/images"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/machineconfigpool"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
)

const (
	SRIOV_OPERATOR_NAMESPACE = "openshift-sriov-network-operator"
	LOG_ENTRY                = "Accumulated forward statistics for all ports"
	PCI_ENV_VARIABLE_NAME    = "PCIDEVICE_OPENSHIFT_IO_DPDKNIC"
	DEMO_APP_NAMESPACE       = "dpdk"
)

var (
	machineConfigPoolName  string
	performanceProfileName string

	sriovclient *sriovtestclient.ClientSet

	OriginalSriovPolicies      []*sriovv1.SriovNetworkNodePolicy
	OriginalSriovNetworks      []*sriovv1.SriovNetwork
	OriginalPerformanceProfile *perfv1alpha1.PerformanceProfile
)

func init() {
	machineConfigPoolName = os.Getenv("ROLE_WORKER_RT")
	if machineConfigPoolName == "" {
		machineConfigPoolName = "worker-cnf"
	}

	performanceProfileName = os.Getenv("PERF_TEST_PROFILE")
	if performanceProfileName == "" {
		performanceProfileName = "performance"
	}

	OriginalSriovPolicies = make([]*sriovv1.SriovNetworkNodePolicy, 0)
	OriginalSriovNetworks = make([]*sriovv1.SriovNetwork, 0)

	// Reuse the sriov client
	// Use the SRIOV test client
	sriovclient = sriovtestclient.New("", func(scheme *runtime.Scheme) {
		sriovv1.AddToScheme(scheme)
	})
}

var _ = Describe("dpdk", func() {
	var dpdkWorkloadPod *corev1.Pod

	execute.BeforeAll(func() {
		var exist bool
		dpdkWorkloadPod, exist = tryToFindDPDKPod()
		if exist {
			return
		}

		findOrOverridePerformanceProfile()
		findOrOverrideSriovNetwork()
		dpdkWorkloadPod = createPod()
	})

	Context("Validate the build and deployment configuration", func() {
		It("Should forward and receive packets from a pod running dpdk base on a image created by building config", func() {
			var out string
			var err error
			_, exist, err := findDPDKDeploymentConfigWorkloadPod()
			Expect(err).ToNot(HaveOccurred())

			if !exist {
				Skip("skip test as we can't find a dpdk workload created by a deployment config object")
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
			var out string
			var err error
			_, exist, err := findDPDKDeploymentConfigWorkloadPod()
			Expect(err).ToNot(HaveOccurred())

			if exist {
				Skip("skip test as we find a dpdk workload created by a deployment config object")
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
			buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
			Expect(err).ToNot(HaveOccurred())
			cpuList = strings.Split(strings.Replace(buff.String(), "\r\n", "", 1), ",")
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
			buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
			Expect(err).ToNot(HaveOccurred())
			cpuList := strings.Split(strings.Replace(buff.String(), "\r\n", "", 1), ",")
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

			pod := pods.DefineWithHugePages(namespaces.DpdkTest, dpdkWorkloadPod.Spec.NodeName, "200000000")
			pod, err := client.Client.Pods(namespaces.DpdkTest).Create(pod)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat",
					fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-1048576kB/free_hugepages", numaNode)})
				Expect(err).ToNot(HaveOccurred())
				numberOfFreeHugePages, err := strconv.Atoi(strings.Replace(buff.String(), "\r\n", "", 1))
				Expect(err).ToNot(HaveOccurred())

				// the created pod is going to use 3 hugepages so we validate the number of hugepages is equal
				// to the number before we start this pod less 3
				return numberOfFreeHugePages
			}, 5*time.Minute, 5*time.Second).Should(Equal(activeNumberOfFreeHugePages - 3))

			pod, err = client.Client.Pods(namespaces.DpdkTest).Get(pod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))

			err = client.Client.Pods(namespaces.DpdkTest).Delete(pod.Name, &metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				_, err := client.Client.Pods(namespaces.DpdkTest).Get(pod.Name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					return err
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(HaveOccurred())

			pod = pods.DefineWithHugePages(namespaces.DpdkTest, dpdkWorkloadPod.Spec.NodeName, "400000000")
			pod, err = client.Client.Pods(namespaces.DpdkTest).Create(pod)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() corev1.PodPhase {
				pod, err = client.Client.Pods(namespaces.DpdkTest).Get(pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				// waiting for the pod to be in failed status because we try to write more then 4Gi
				// of hugepages that was the allocated amount for this pod
				return pod.Status.Phase
			}, 5*time.Minute, 5*time.Second).Should(Equal(corev1.PodFailed))

			out, err := pods.GetLog(pod)
			Expect(err).ToNot(HaveOccurred())

			findErr := false
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "(core dumped) LD_PRELOAD=libhugetlbfs.so HUGETLB_VERBOSE=10 HUGETLB_MORECORE=yes HUGETLB_FORCE_ELFMAP=yes python3 printer.py > /dev/null") {
					findErr = true
				}
			}

			Expect(findErr).To(BeTrue())
		})
	})

	// TODO: find a better why to restore the configuration
	// This will not work if we use a random order running
	Context("restoring configuration", func() {
		It("should restore the cluster to the original status", func() {
			By("restore performance profile")
			RestorePerformanceProfile()

			By("cleaning the sriov test configuration")
			CleanSriov()

			By("restore sriov policies")
			RestoreSriovPolicy()

			By("restore sriov networks")
			RestoreSriovNetwork()
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
	pod, exist, err := findDPDKDeploymentConfigWorkloadPod()
	Expect(err).ToNot(HaveOccurred())

	if exist {
		return pod, true
	}

	pod, exist, err = findDPDKWorkloadPod()
	Expect(err).ToNot(HaveOccurred())

	if exist {
		return pod, true
	}

	return nil, false
}

func findOrOverridePerformanceProfile() {
	performanceProfile, valid, err := ValidatePerformanceProfile()
	Expect(err).ToNot(HaveOccurred())

	if !valid {
		if performanceProfile != nil {
			OriginalPerformanceProfile = performanceProfile.DeepCopy()
		}
		// Clean and create a new performance profile for the dpdk application
		err = CleanPerformanceProfiles()
		Expect(err).ToNot(HaveOccurred())

		err = WaitForClusterToBeStable()
		Expect(err).ToNot(HaveOccurred())

		err := CreatePerformanceProfile()
		Expect(err).ToNot(HaveOccurred())

		err = WaitForClusterToBeStable()
		Expect(err).ToNot(HaveOccurred())
	}
}

func findOrOverrideSriovNetwork() {
	valid, err := ValidateNetworkAttachmentDefinition()
	Expect(err).ToNot(HaveOccurred())

	if !valid {
		BackupSriovPolicy()
		BackupSriovNetwork()

		// Clean and create a new sriov policy and network for the dpdk application
		CleanSriov()
		sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, SRIOV_OPERATOR_NAMESPACE)
		Expect(err).ToNot(HaveOccurred())

		Expect(sriovInfos).ToNot(BeNil())
		Expect(len(sriovInfos.Nodes)).To(BeNumerically(">", 0))

		sriovDevice, err := sriovInfos.FindOneSriovDevice(sriovInfos.Nodes[0])
		Expect(err).ToNot(HaveOccurred())

		CreateSriovPolicy(sriovDevice, sriovInfos.Nodes[0], 5, "dpdknic")
		CreateSriovNetwork(sriovDevice, "dpdk-network", "dpdknic")
	}
}

func createPod() *corev1.Pod {
	pod, err := CreateDPDKWorkload()
	Expect(err).ToNot(HaveOccurred())

	return pod
}

func ValidatePerformanceProfile() (*perfv1alpha1.PerformanceProfile, bool, error) {
	performanceProfile := &perfv1alpha1.PerformanceProfile{}
	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: performanceProfileName}, performanceProfile)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	// Check we have more then two isolated CPU
	cpuSet, err := cpuset.Parse(string(*performanceProfile.Spec.CPU.Isolated))
	if err != nil {
		return performanceProfile, false, err
	}

	cpuSetSlice := cpuSet.ToSlice()
	if len(cpuSetSlice) < 6 {
		return performanceProfile, false, nil
	}

	if performanceProfile.Spec.HugePages == nil {
		return performanceProfile, false, nil
	}

	if *performanceProfile.Spec.HugePages.DefaultHugePagesSize != "1G" {
		return performanceProfile, false, nil
	}

	if len(performanceProfile.Spec.HugePages.Pages) == 0 {
		return performanceProfile, false, nil
	}

	if performanceProfile.Spec.HugePages.Pages[0].Count < 4 {
		return performanceProfile, false, nil
	}

	if performanceProfile.Spec.HugePages.Pages[0].Size != "1G" {
		return performanceProfile, false, nil
	}

	return performanceProfile, true, nil
}

func CleanPerformanceProfiles() error {
	performanceProfileList := &perfv1alpha1.PerformanceProfileList{}
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
	err := machineconfigpool.WaitForCondition(
		client.Client,
		&mcv1.MachineConfigPool{ObjectMeta: metav1.ObjectMeta{Name: machineConfigPoolName}},
		mcv1.MachineConfigPoolUpdating,
		corev1.ConditionTrue,
		2*time.Minute)
	if err != nil {
		return err
	}

	mcp := &mcv1.MachineConfigPool{}
	err = client.Client.Get(context.TODO(), goclient.ObjectKey{Name: machineConfigPoolName}, mcp)
	if err != nil {
		return err
	}

	// We need to wait a long time here for the node to reboot
	err = machineconfigpool.WaitForCondition(
		client.Client,
		&mcv1.MachineConfigPool{ObjectMeta: metav1.ObjectMeta{Name: machineConfigPoolName}},
		mcv1.MachineConfigPoolUpdated,
		corev1.ConditionTrue,
		time.Duration(15*mcp.Status.MachineCount)*time.Minute)

	return err
}

func CreatePerformanceProfile() error {
	isolatedCPUSet := perfv1alpha1.CPUSet("8-15")
	reservedCPUSet := perfv1alpha1.CPUSet("0-7")
	hugepageSize := perfv1alpha1.HugePageSize("1G")
	performanceProfile := &perfv1alpha1.PerformanceProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: performanceProfileName,
		},
		Spec: perfv1alpha1.PerformanceProfileSpec{
			CPU: &perfv1alpha1.CPU{
				Isolated: &isolatedCPUSet,
				Reserved: &reservedCPUSet,
			},
			HugePages: &perfv1alpha1.HugePages{
				DefaultHugePagesSize: &hugepageSize,
				Pages: []perfv1alpha1.HugePage{
					{
						Count: 16,
						Size:  hugepageSize,
						Node:  pointer.Int32Ptr(0),
					},
				},
			},
			NodeSelector: map[string]string{
				"node-role.kubernetes.io/worker-cnf": "",
			},
		},
	}

	return client.Client.Create(context.TODO(), performanceProfile)
}

func ValidateNetworkAttachmentDefinition() (bool, error) {
	netattachdef := &sriovk8sv1.NetworkAttachmentDefinition{}
	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: "dpdk-network", Namespace: namespaces.DpdkTest}, netattachdef)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func CreateSriovPolicy(sriovDevice *sriovv1.InterfaceExt, testNode string, numVfs int, resourceName string) {
	nodePolicy := &sriovv1.SriovNetworkNodePolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-policy-",
			Namespace:    SRIOV_OPERATOR_NAMESPACE,
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
	if sriovDevice.DeviceID == "1015" {
		nodePolicy.Spec.IsRdma = true
	}

	// Intel device
	if sriovDevice.DeviceID == "8086" {
		nodePolicy.Spec.DeviceType = "vfio-pci"
	}

	err := sriovclient.Create(context.Background(), nodePolicy)
	Expect(err).ToNot(HaveOccurred())
	waitForSRIOVStable()

	Eventually(func() int64 {
		testedNode, err := sriovclient.Nodes().Get(testNode, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/"+resourceName)]
		capacity, _ := resNum.AsInt64()
		return capacity
	}, 10*time.Minute, time.Second).Should(Equal(int64(numVfs)))
}

func CreateSriovNetwork(sriovDevice *sriovv1.InterfaceExt, sriovNetworkName string, resourceName string) {
	ipam := `{"type": "host-local","ranges": [[{"subnet": "1.1.1.0/24"}]],"dataDir": "/run/my-orchestrator/container-ipam-state"}`
	err := sriovnetwork.CreateSriovNetwork(sriovclient, sriovDevice, sriovNetworkName, namespaces.DpdkTest, SRIOV_OPERATOR_NAMESPACE, resourceName, ipam)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		netAttDef := &sriovk8sv1.NetworkAttachmentDefinition{}
		return sriovclient.Get(context.Background(), goclient.ObjectKey{Name: sriovNetworkName, Namespace: namespaces.DpdkTest}, netAttDef)
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

}

func CreateDPDKWorkload() (*corev1.Pod, error) {
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
			"cd /root/test-app/ && ./run.sh"},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: pointer.Int64Ptr(0),
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"IPC_LOCK", "SYS_RESOURCE"},
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
				"k8s.v1.cni.cncf.io/networks": "dpdk-testing/dpdk-network",
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
		}}

	dpdkPod, err := client.Client.Pods(namespaces.DpdkTest).Create(dpdkPod)
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

// findDPDKDeploymentConfigWorkloadPod finds a pod running a DPDK application from a deployment config
func findDPDKDeploymentConfigWorkloadPod() (*corev1.Pod, bool, error) {
	return findDPDKWorkloadPodByLabelSelector(labels.SelectorFromSet(labels.Set{"deploymentconfig": "s2i-dpdk-app"}).String(), DEMO_APP_NAMESPACE)
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

	p, err := client.Client.Pods(namespace).List(listOptions)
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

			for _, cpu := range strings.Split(strings.Replace(numaLine[1], "\r\n", "", 1), ",") {
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
	for _, line := range strings.Split(buff.String(), "\r\n") {
		if strings.Contains(line, PCI_ENV_VARIABLE_NAME) {
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
	return -1, fmt.Errorf("failed to find the numa for pci %s", PCI_ENV_VARIABLE_NAME)
}

func BackupSriovPolicy() {
	sriovPolicyList := &sriovv1.SriovNetworkNodePolicyList{}
	err := sriovclient.List(context.TODO(), sriovPolicyList, &goclient.ListOptions{Namespace: SRIOV_OPERATOR_NAMESPACE})
	Expect(err).ToNot(HaveOccurred())

	for _, policy := range sriovPolicyList.Items {
		if policy.Name == "default" {
			continue
		}

		OriginalSriovPolicies = append(OriginalSriovPolicies, &policy)
	}
}

func BackupSriovNetwork() {
	sriovNetworkList := &sriovv1.SriovNetworkList{}
	err := sriovclient.List(context.TODO(), sriovNetworkList, &goclient.ListOptions{Namespace: SRIOV_OPERATOR_NAMESPACE})
	Expect(err).ToNot(HaveOccurred())

	for _, network := range sriovNetworkList.Items {
		OriginalSriovNetworks = append(OriginalSriovNetworks, &network)
	}
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

func RestoreSriovPolicy() {
	for _, policy := range OriginalSriovPolicies {
		name := policy.Name
		policy.ObjectMeta = metav1.ObjectMeta{Name: name, Namespace: SRIOV_OPERATOR_NAMESPACE}
		err := sriovclient.Create(context.TODO(), policy)
		Expect(err).ToNot(HaveOccurred())
	}
	waitForSRIOVStable()
}

func RestoreSriovNetwork() {
	for _, network := range OriginalSriovNetworks {
		name := network.Name
		network.ObjectMeta = metav1.ObjectMeta{Name: name, Namespace: SRIOV_OPERATOR_NAMESPACE}
		err := sriovclient.Create(context.TODO(), network)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			netAttDef := &sriovk8sv1.NetworkAttachmentDefinition{}
			return sriovclient.Get(context.Background(), goclient.ObjectKey{Name: network.Name, Namespace: network.Spec.NetworkNamespace}, netAttDef)
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}
}

func CleanSriov() {
	err := sriovnamespaces.Clean(SRIOV_OPERATOR_NAMESPACE, namespaces.DpdkTest, sriovclient)
	Expect(err).ToNot(HaveOccurred())
	waitForSRIOVStable()
}

func waitForSRIOVStable() {
	// This used to be to check for sriov not to be stable first,
	// then stable. The issue is that if no configuration is applied, then
	// the status won't never go to not stable and the test will fail.
	// TODO: find a better way to handle this scenario
	time.Sleep(5 * time.Second)
	Eventually(func() bool {
		res, err := sriovcluster.SriovStable(SRIOV_OPERATOR_NAMESPACE, sriovclient)
		Expect(err).ToNot(HaveOccurred())
		return res
	}, 10*time.Minute, 1*time.Second).Should(BeTrue())

	Eventually(func() bool {
		isClusterReady, err := sriovcluster.IsClusterStable(sriovclient)
		Expect(err).ToNot(HaveOccurred())
		return isClusterReady
	}, 10*time.Minute, 1*time.Second).Should(BeTrue())
}
