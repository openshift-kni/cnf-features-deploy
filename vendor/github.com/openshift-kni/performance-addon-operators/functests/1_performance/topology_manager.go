package __performance

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

var _ = Describe("[rfe_id:27350][performance]Topology Manager", func() {
	var workerRTNodes []corev1.Node
	var profile *performancev1alpha1.PerformanceProfile

	BeforeEach(func() {
		var err error
		workerRTNodes, err = nodes.GetByRole(testutils.RoleWorkerCNF)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
		profile, err = profiles.GetByNodeLabels(
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerCNF): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_id:26932][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] should be enabled with the policy specified in profile", func() {
		kubeletConfig, err := nodes.GetKubeletConfig(&workerRTNodes[0])
		Expect(err).ToNot(HaveOccurred())

		// verify topology manager feature gate
		enabled, ok := kubeletConfig.FeatureGates[testutils.FeatureGateTopologyManager]
		Expect(ok).To(BeTrue())
		Expect(enabled).To(BeTrue())

		// verify topology manager policy
		if profile.Spec.NUMA != nil && profile.Spec.NUMA.TopologyPolicy != nil {
			Expect(kubeletConfig.TopologyManagerPolicy).To(Equal(*profile.Spec.NUMA.TopologyPolicy))
		} else {
			Expect(kubeletConfig.TopologyManagerPolicy).To(Equal(kubeletconfigv1beta1.BestEffortTopologyManagerPolicy))
		}
	})

	Context("with the SR-IOV devices and static CPU's", func() {
		var testpod *corev1.Pod
		var sriovNode *corev1.Node

		BeforeEach(func() {
			sriovNodes := nodes.FilterByResource(workerRTNodes, testutils.ResourceSRIOV)
			// TODO: once we will have different CI job for SR-IOV test cases, this skip should be removed
			// and replaced by ginkgo CLI --focus parameter
			if len(sriovNodes) < 1 {
				Skip(
					fmt.Sprintf(
						"The environment does not have nodes with role %q and available %q resources",
						testutils.RoleWorkerCNF,
						string(testutils.ResourceSRIOV),
					),
				)
			}
			sriovNode = &sriovNodes[0]

			var err error
			if testpod != nil {
				err = testclient.Client.Delete(context.TODO(), testpod)
				Expect(err).ToNot(HaveOccurred())

				err = pods.WaitForDeletion(testpod, 60*time.Second)
				Expect(err).ToNot(HaveOccurred())
			}
			testpod = pods.GetTestPod()
			testpod.Namespace = testutils.NamespaceTesting
			testpod.Spec.Containers[0].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:      resource.MustParse("1"),
				corev1.ResourceMemory:   resource.MustParse("64Mi"),
				testutils.ResourceSRIOV: resource.MustParse("1"),
			}
			testpod.Spec.Containers[0].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:      resource.MustParse("1"),
				corev1.ResourceMemory:   resource.MustParse("64Mi"),
				testutils.ResourceSRIOV: resource.MustParse("1"),
			}
			testpod.Spec.NodeSelector = map[string]string{
				testutils.LabelHostname: sriovNode.Name,
			}
			err = testclient.Client.Create(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForCondition(testpod, corev1.PodReady, corev1.ConditionTrue, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())

			// Get updated testpod
			key := types.NamespacedName{
				Name:      testpod.Name,
				Namespace: testpod.Namespace,
			}
			err = testclient.Client.Get(context.TODO(), key, testpod)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:28527][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] should allocate resources from the same NUMA node", func() {
			sriovPciDevice, err := getSriovPciDeviceFromPod(testpod)
			Expect(err).ToNot(HaveOccurred())

			sriovDeviceNumaNode, err := getSriovPciDeviceNumaNode(sriovNode, sriovPciDevice)
			Expect(err).ToNot(HaveOccurred())

			cpuSet, err := getContainerCPUSet(sriovNode, testpod)
			Expect(err).ToNot(HaveOccurred())

			cpuSetNumaNodes, err := getCPUSetNumaNodes(sriovNode, cpuSet)
			Expect(err).ToNot(HaveOccurred())

			for _, cpuNumaNode := range cpuSetNumaNodes {
				Expect(sriovDeviceNumaNode).To(Equal(cpuNumaNode))
			}
		})
	})
})

func getSriovPciDeviceFromPod(pod *corev1.Pod) (string, error) {
	envBytes, err := pods.ExecCommandOnPod(pod, []string{"env"})
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(fmt.Sprintf("%s=(.*)", testutils.EnvPciSriovDevice))
	results := re.FindSubmatch(envBytes)
	if len(results) < 2 {
		return "", fmt.Errorf("failed to find ENV variable %q under the pod %q", testutils.EnvPciSriovDevice, pod.Name)
	}

	return string(results[1]), nil
}

func getSriovPciDeviceNumaNode(sriovNode *corev1.Node, sriovPciDevice string) (string, error) {
	// we will use machine-config-daemon to get all information from the node, because it has
	// mounted node filesystem under /rootfs
	command := []string{"cat", path.Join("/rootfs", testutils.FilePathSRIOVDevice, sriovPciDevice, "numa_node")}
	numaNode, err := nodes.ExecCommandOnMachineConfigDaemon(sriovNode, command)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(numaNode), "\n"), nil
}

func getContainerCPUSet(sriovNode *corev1.Node, pod *corev1.Pod) ([]int, error) {
	podDir := fmt.Sprintf("kubepods-pod%s.slice", strings.ReplaceAll(string(pod.UID), "-", "_"))

	containerID := strings.Trim(pod.Status.ContainerStatuses[0].ContainerID, "cri-o://")
	containerDir := fmt.Sprintf("crio-%s.scope", containerID)

	// we will use machine-config-daemon to get all information from the node, because it has
	// mounted node filesystem under /rootfs
	command := []string{"cat", path.Join("/rootfs", testutils.FilePathKubePodsSlice, podDir, containerDir, "cpuset.cpus")}
	output, err := nodes.ExecCommandOnMachineConfigDaemon(sriovNode, command)
	if err != nil {
		return nil, err
	}

	cpus, err := cpuset.Parse(strings.Trim(string(output), "\n"))
	if err != nil {
		return nil, err
	}

	return cpus.ToSlice(), nil
}

func getCPUSetNumaNodes(sriovNode *corev1.Node, cpuSet []int) ([]string, error) {
	numaNodes := []string{}
	for _, cpuID := range cpuSet {
		cpuPath := path.Join("/rootfs", testutils.FilePathSysCPU, fmt.Sprintf("cpu%d", cpuID))
		cpuDirContent, err := nodes.ExecCommandOnMachineConfigDaemon(sriovNode, []string{"ls", cpuPath})
		if err != nil {
			return nil, err
		}
		re := regexp.MustCompile(`node(\d+)`)
		match := re.FindStringSubmatch(string(cpuDirContent))
		if len(match) != 2 {
			return nil, fmt.Errorf("incorrect match for 'ls' command: %v", match)
		}
		numaNodes = append(numaNodes, match[1])
	}
	return numaNodes, nil
}
