package dpdk

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
)

// Entry to find in the pod logfile
const logEntry = "Accumulated forward statistics for all ports"
const pciEnvVariableName = "PCIDEVICE_OPENSHIFT_IO_DPDKNIC"

var testDpdkNamespace string

func init() {
	testDpdkNamespace = os.Getenv("DPDK_TEST_NAMESPACE")
	if testDpdkNamespace == "" {
		testDpdkNamespace = "dpdk-testing"
	}
}

var _ = Describe("dpdk", func() {
	Context("Run a sanity test on a worker", func() {
		It("Should forward and receive packets", func() {
			var out string
			var err error
			p := findDPDKWorkloadPod()
			By("Parsing output from the DPDK application")
			Eventually(func() string {
				out, err = pods.GetLog(p)
				Expect(err).ToNot(HaveOccurred())
				return out
			}, 8*time.Minute, 1*time.Second).Should(ContainSubstring(logEntry),
				"Cannot find accumulated statistics")
			checkRxTx(out)
		})
	})

	Context("Validate NUMA aliment", func() {
		var p *corev1.Pod
		var cpuList []string

		BeforeEach(func() {
			p = findDPDKWorkloadPod()
			buff, err := pods.ExecCommand(client.Client, *p, []string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
			Expect(err).ToNot(HaveOccurred())
			cpuList = strings.Split(strings.Replace(buff.String(), "\r\n", "", 1), ",")
		})

		// 28078
		It("should allocate the requested number of cpus", func() {
			numOfCPU := p.Spec.Containers[0].Resources.Limits.Cpu().Value()
			Expect(len(cpuList)).To(Equal(int(numOfCPU)))
		})

		// 28432
		It("should allocate all the resources on the same NUMA node", func() {
			By("finding the CPUs numa")
			cpuNumaNode, err := findNUMAForCPUs(p, cpuList)
			Expect(err).ToNot(HaveOccurred())

			By("finding the pci numa")
			pciNumaNode, err := findNUMAForSRIOV(p)
			Expect(err).ToNot(HaveOccurred())

			By("expecting cpu and pci to be on the same numa")
			Expect(cpuNumaNode).To(Equal(pciNumaNode))
		})
	})
})

// checkRxTx parses the output from the DPDK test application
// and verifies that packets have passed the NIC TX and RX queues
func checkRxTx(out string) {
	lines := strings.Split(out, "\n")
	Expect(len(lines)).To(BeNumerically(">=", 3))
	for i, line := range lines {
		if strings.Contains(line, logEntry) {
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

// findDPDKWorkloadPod finds a pod running a DPDK application
func findDPDKWorkloadPod() *corev1.Pod {
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"deploymentconfig": "s2i-dpdk-app"}).String(),
	}

	p, err := client.Client.Pods(testDpdkNamespace).List(listOptions)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(p.Items)).ToNot(Equal(0), "no pods found")

	var pod corev1.Pod
	podReady := false
	for _, pod = range p.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podReady = true
			break
		}
	}

	Expect(podReady).To(BeTrue(), fmt.Sprintf("the pod %s is not ready", pod.Name))
	pods.WaitForCondition(client.Client, &pod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
	return &pod
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
