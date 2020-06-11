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

var testDpdkNamespace string

func init() {
	testDpdkNamespace = os.Getenv("DPDK_TEST_NAMESPACE")
	if testDpdkNamespace == "" {
		testDpdkNamespace = "dpdk"
	}
}

var _ = Describe("dpdk", func() {
	var _ = Context("Run a sanity test on a worker", func() {
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
