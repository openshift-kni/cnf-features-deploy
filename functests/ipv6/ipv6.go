package ipv6

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testNamespace = "sriov-testing"
const testerImage = "centos:centos8"
const pingedPodName = "pod-pinged"
const pingingPodName = "pod-pinging"
const sriovNetworkAttachment = "dual-stack-net-attachment-def"
const sriovInterfaceName = "net1"
const workerNodeSelector = "node-role.kubernetes.io/worker-cnf"
const hostnameLabel = "kubernetes.io/hostname"

type Network struct {
	Interface string
	Ips       []string
}

var _ = Describe("sriov", func() {

	var nodeSelector string

	execute.BeforeAll(func() {
		err := namespaces.Create(testNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())
		err = namespaces.Clean(testNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())

		nodes, err := client.Client.Nodes().List(metav1.ListOptions{
			LabelSelector: workerNodeSelector,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(nodes.Items)).NotTo(Equal(0))
		nodeSelector = nodes.Items[0].ObjectMeta.Labels[hostnameLabel]
	})

	var _ = Context("IPv6 configured secondary interfaces on pods", func() {
		It("Should be able to ping each other", func() {
			podDefinition := dualStackTestPod(pingedPodName, []string{"/bin/bash", "-c", "--"}, []string{"while true; do sleep 300000; done;"}, nodeSelector)
			_, err := client.Client.Pods(testNamespace).Create(podDefinition)
			Eventually(func() k8sv1.PodPhase {
				runningPod, err := client.Client.Pods(testNamespace).Get(pingedPodName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return runningPod.Status.Phase
			}, 3*time.Minute, 1*time.Second).Should(Equal(k8sv1.PodRunning))
			pod, err := client.Client.Pods(testNamespace).Get(pingedPodName, metav1.GetOptions{})
			var nets []Network
			err = json.Unmarshal([]byte(pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/networks-status"]), &nets)
			Expect(err).ToNot(HaveOccurred())
			for _, net := range nets {
				if net.Interface != sriovInterfaceName {
					continue
				}
				for i, ip := range net.Ips {
					pingPod(ip, i, nodeSelector)
				}
				return
			}
			Fail("No sriov network inerface found.")
		})
	})
})

func pingPod(ip string, podNumber int, nodeSelector string) {
	podName := fmt.Sprintf("%s-%d", pingingPodName, podNumber)
	podDefinition := dualStackTestPod(podName, []string{"sh", "-c", "ping -c 3 " + ip}, []string{}, nodeSelector)
	_, err := client.Client.Pods(testNamespace).Create(podDefinition)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() k8sv1.PodPhase {
		runningPod, err := client.Client.Pods(testNamespace).Get(podName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return runningPod.Status.Phase
	}, 3*time.Minute, 1*time.Second).Should(Equal(k8sv1.PodSucceeded))
}

func dualStackTestPod(name string, command []string, args []string, node string) *k8sv1.Pod {
	res := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": sriovNetworkAttachment,
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   testerImage,
					Command: command,
					Args:    args,
				},
			},
			NodeSelector: map[string]string{
				hostnameLabel: node,
			},
		},
	}
	return &res
}
