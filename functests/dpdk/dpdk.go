package dpdk

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
)

// Worker node role
const dpdkHostRole = "worker-cnf"

var testDpdkNamespace string

func init() {
	testDpdkNamespace = os.Getenv("DPDK_TEST_NAMESPACE")
	if testDpdkNamespace == "" {
		testDpdkNamespace = "dpdk-testing"
	}
}

var _ = Describe("dpdk", func() {
	var nList []corev1.Node

	execute.BeforeAll(func() {
		// get nodes
		var err error
		nList, err = nodes.GetByRole(client.Client, dpdkHostRole)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(nList)).ShouldNot(Equal(0))
	})

	var _ = Context("Run a sanity test on each worker", func() {
		It("Should forward and receive packets", func() {
			const stopMessage = "Accumulated forward statistics for all ports"
			var out string
			for _, n := range nList {
				p := findPod(n.Name)
				pods.WaitForCondition(client.Client, p, corev1.ContainersReady, corev1.ConditionTrue, 2*time.Minute)
				By("Parsing output from the DPDK application")
				Eventually(func() string {
					out = getPodLog(p)
					return out
				}, 8*time.Minute, 1*time.Second).Should(ContainSubstring(stopMessage),
					"Cannot find accumulated statistics")
				checkRxTx(out)
			}
		})
	})
})

// checkRxTx parses the output from the DPDK test application
// and verifies that packets have passed the NIC TX and RX queues
func checkRxTx(out string) {
	str := strings.Split(out, "\n")
	for i := 0; i < len(str); i++ {
		if strings.Contains(str[i], "all ports") {
			i++
			d := getNumberOfPackets(str, i)
			Expect(d).Should(BeNumerically(">", 0), "number of received packets should be greater than 0")

			i++
			d = getNumberOfPackets(str, i)
			Expect(d).Should(BeNumerically(">", 0), "number of transferred packets should be greater than 0")
		}
	}
}

// getNumber of packets parses the string (represented as a slice)
// and returns an element representing the number of packets
func getNumberOfPackets(s []string, index int) int {
	r := strings.Fields(s[index])
	Expect(len(r)).To(Equal(6), "the slice doesn't contain 6 elements")
	d, err := strconv.Atoi(r[1])
	Expect(err).ToNot(HaveOccurred())
	return d
}

// findPod finds a pod running on appropriate node
func findPod(nodeName string) *corev1.Pod {
	listOptions := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
	}

	listOptions.LabelSelector = labels.SelectorFromSet(labels.Set{"deploymentconfig": "s2i-dpdk-app"}).String()
	p, err := client.Client.Pods(testDpdkNamespace).List(listOptions)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(p.Items)).ShouldNot(Equal(0))

	var pod corev1.Pod
	var i int
	for i, pod = range p.Items {
		if pod.Status.Phase == corev1.PodRunning {
			break
		}
		i++
	}

	Expect(i).ShouldNot(Equal(len(p.Items)))

	pods.WaitForCondition(client.Client, &pod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
	return &pod
}

// getPodLog connects to a pod and fetches log
func getPodLog(p *corev1.Pod) string {
	req := client.Client.Pods(p.Namespace).GetLogs(p.Name, &corev1.PodLogOptions{})
	log, err := req.Stream()
	Expect(err).ToNot(HaveOccurred())

	defer log.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, log)
	Expect(err).ToNot(HaveOccurred())
	str := buf.String()

	return str
}
