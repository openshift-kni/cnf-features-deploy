package vrf

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	client "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	ipStackIPv4                  = "ipv4"
	ipStackIPv6                  = "ipv6"
	hostnameLabel                = "kubernetes.io/hostname"
	TestNamespace                = "vrf-testing"
	podWaitingTime time.Duration = 2 * time.Minute
	VRFBlueName                  = "blue"
	VRFRedName                   = "red"
)

var (
	ipStackParameters = []string{ipStackIPv4, ipStackIPv6}
)

type vrfTestParameters struct {
	IPStack string
}

var _ = Describe("[vrf]", func() {
	describe := func(ipStack string) string {

		VRFParameters, err := newVRFTestParameters(ipStack)
		if err != nil {
			return fmt.Sprintf("error in parameters: node=%s, ipStack=%s", "Same Node", ipStack)
		}
		params, err := json.Marshal(VRFParameters)
		if err != nil {
			return fmt.Sprintf("error in parameters: node=%s, ipStack=%s", "Same Node", ipStack)
		}

		return string(params)
	}
	apiclient := client.New("")

	var nodesList []k8sv1.Node
	var vrfBlue netattdefv1.NetworkAttachmentDefinition
	var vrfRed netattdefv1.NetworkAttachmentDefinition

	execute.BeforeAll(func() {
		err := namespaces.Create(TestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
		allWorkerNodes, err := nodes.GetByRole(apiclient, "worker")
		Expect(err).ToNot(HaveOccurred())
		nodesList, err = nodes.MatchingOptionalSelector(allWorkerNodes)
		Expect(err).ToNot(HaveOccurred())
		vrfBlue = addVRFNad(apiclient, "test-vrf-blue", VRFBlueName)
		vrfRed = addVRFNad(apiclient, "test-vrf-red", VRFRedName)
	})
	AfterEach(func() {
		err := namespaces.CleanPods(TestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("", func() {
		// OCP-36305
		DescribeTable("Integration: NAD, IPAM: static, Interfaces: 1, Scheme: 2 Pods 2 VRFs OCP Primary network overlap",
			func(ipStack string) {
				testVRFScenario(apiclient, TestNamespace, nodesList[0].Name, vrfBlue.Name, vrfRed.Name, ipStack)
			},
			Entry(describe, ipStackIPv4),
		)
	})
})

func pingIPViaVRF(cs *client.ClientSet, client *k8sv1.Pod, vrfName string, DestIPAddr string) error {
	pingCommand := []string{"ping", "-I", vrfName, "-c5", DestIPAddr}
	pod, err := cs.Pods(client.Namespace).Get(context.Background(), client.Name, metav1.GetOptions{})
	pingStatus, err := pods.ExecCommand(cs, *pod, pingCommand)
	if err != nil {
		return err
	}
	if strings.Contains(pingStatus.String(), " 0% packet loss") {
		return nil
	}
	return fmt.Errorf("Connectivity test error")
}

func addVRFNad(cs *client.ClientSet, NadName string, vrfName string) netattdefv1.NetworkAttachmentDefinition {
	vrfDefinition := netattdefv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: NadName,
			Namespace:    TestNamespace,
		},
		Spec: netattdefv1.NetworkAttachmentDefinitionSpec{
			Config: fmt.Sprintf(`{"cniVersion": "0.4.0", "name": "macvlan-vrf", "plugins": [{"type": "macvlan","ipam": {"type": "static"}},{"type": "vrf","vrfname": "%s"}]}`, vrfName),
		},
	}
	err := cs.Create(context.Background(), &vrfDefinition)
	Expect(err).ToNot(HaveOccurred())
	return vrfDefinition
}

func getOverlapIP(cs *client.ClientSet, namespace string, nodeName string, podNamePrefix string) string {
	tempPodDefinition := redefineAsPrivilegedWithNamePrefix(pods.DefinePodOnNode(namespace, nodeName), podNamePrefix)
	err := cs.Create(context.Background(), tempPodDefinition)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() k8sv1.PodPhase {
		tempPod, _ := cs.Pods(namespace).Get(context.Background(), tempPodDefinition.Name, metav1.GetOptions{})
		return tempPod.Status.Phase
	}, podWaitingTime, time.Second).Should(Equal(k8sv1.PodRunning))

	pod, err := cs.Pods(namespace).Get(context.Background(), tempPodDefinition.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return pod.Status.PodIP
}

func waitUntilPodCreatedAndRunning(cs *client.ClientSet, podStruct *k8sv1.Pod) {
	err := cs.Create(context.Background(), podStruct)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() k8sv1.PodPhase {
		tempPod, _ := cs.Pods(podStruct.Namespace).Get(context.Background(), podStruct.Name, metav1.GetOptions{})
		return tempPod.Status.Phase
	}, podWaitingTime, time.Second).Should(Equal(k8sv1.PodRunning))
}

func podHasCorrectVRFConfig(cs *client.ClientSet, pod *k8sv1.Pod, vrfMapsConfig []map[string]string) {
	runningPod, err := cs.Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, vrfMapConfig := range vrfMapsConfig {
		validateVrfIPAddrCommand := []string{"ip", "addr", "show", fmt.Sprintf("%s", vrfMapConfig["vrfInterface"])}
		Eventually(func() bool {
			vrfIface, _ := pods.ExecCommand(cs, *runningPod, validateVrfIPAddrCommand)
			return strings.Contains(vrfIface.String(), vrfMapConfig["vrfClientIP"])
		}, podWaitingTime, 5*time.Second).Should(BeTrue(), fmt.Errorf("VRF interface is not present"))

		validateVRFRouteTableCommand := []string{"ip", "route", "show", "vrf", fmt.Sprintf("%s", vrfMapConfig["vrfName"])}
		Eventually(func() bool {
			vrfRouteTable, _ := pods.ExecCommand(cs, *runningPod, validateVRFRouteTableCommand)
			return strings.Contains(vrfRouteTable.String(), vrfMapConfig["vrfClientIP"])
		}, podWaitingTime, 5*time.Second).Should(BeTrue(), fmt.Errorf(fmt.Sprintf("VRF %s route table is not present", vrfMapConfig["vrfName"])))
	}
}

func redefineAsPrivilegedWithNamePrefix(pod *k8sv1.Pod, namePrefix string) *k8sv1.Pod {
	pod.ObjectMeta.GenerateName = namePrefix
	pod.Spec.Containers[0].SecurityContext = &k8sv1.SecurityContext{}
	b := true
	pod.Spec.Containers[0].SecurityContext.Privileged = &b
	return pod
}

func newVRFTestParameters(IPStack string) (*vrfTestParameters, error) {
	VRFTestParameters := new(vrfTestParameters)

	err := paramInParamList(IPStack, ipStackParameters)
	if err != nil {
		return nil, err
	}
	VRFTestParameters.IPStack = IPStack
	return VRFTestParameters, nil
}

func paramInParamList(param string, paramRange []string) error {
	for _, parameter := range paramRange {
		if param == parameter {
			return nil
		}
	}
	return fmt.Errorf("error: wrong parameter %v", param)
}

func testVRFScenario(apiclient *client.ClientSet, namespace string, nodeName string, firstVRFNetwork string, secondVRFNetwork string, ipStack string) {
	var podClientFirstVRFNetworkIPAddress string
	var podServerFirstVRFNetworkIPAddress string

	By("Create overlapping IP pods")
	podClientSecondVRFNetworkOverlappingIP := getOverlapIP(apiclient, namespace, nodeName, "overlap-client-ip")
	podServerSecondVRFNetworkOverlappingIP := getOverlapIP(apiclient, namespace, nodeName, "overlap-server-ip")

	if ipStack == ipStackIPv4 && net.ParseIP(podClientSecondVRFNetworkOverlappingIP).To4() == nil {
		Skip("Skipping IPv4 test. Cluster supports IPv6 protocol")
	} else if ipStack == ipStackIPv4 && net.ParseIP(podClientSecondVRFNetworkOverlappingIP).To4() != nil {
		podClientFirstVRFNetworkIPAddress = "10.255.255.1"
		podServerFirstVRFNetworkIPAddress = "10.255.255.2"
	} else {
		Skip("Unsupported protocol parameter")
	}

	By("Create VRFs client/server pods")
	clientNetworkDefinition := fmt.Sprintf(`[{"name": "%s", "mac": "%s", "ips": ["%s/24"]}, {"name": "%s", "mac": "%s", "ips": ["%s/24"]}]`,
		firstVRFNetwork, "20:04:0f:f1:88:A1", podClientFirstVRFNetworkIPAddress, secondVRFNetwork, "20:04:0f:f1:88:B2", podClientSecondVRFNetworkOverlappingIP)
	podClient := redefineAsPrivilegedWithNamePrefix(
		pods.RedefinePodWithNetwork(pods.DefinePodOnNode(namespace, nodeName), clientNetworkDefinition), "client-vrf")
	serverNetworkDefinition := fmt.Sprintf(`[{"name": "%s", "mac": "%s", "ips": ["%s/24"]}, {"name": "%s", "mac": "%s", "ips": ["%s/24"]}]`,
		firstVRFNetwork, "20:04:0f:f1:88:A3", podServerFirstVRFNetworkIPAddress, secondVRFNetwork, "20:04:0f:f1:88:B4", podServerSecondVRFNetworkOverlappingIP)
	podServer := redefineAsPrivilegedWithNamePrefix(
		pods.RedefinePodWithNetwork(pods.DefinePodOnNode(namespace, nodeName), serverNetworkDefinition), "server-vrf")
	waitUntilPodCreatedAndRunning(apiclient, podClient)
	waitUntilPodCreatedAndRunning(apiclient, podServer)

	By("Validate that client/server VRFs have correct configuration")
	podHasCorrectVRFConfig(apiclient, podClient,
		[]map[string]string{
			{"vrfName": VRFBlueName, "vrfClientIP": podClientFirstVRFNetworkIPAddress, "vrfInterface": "net1"},
			{"vrfName": VRFRedName, "vrfClientIP": podClientSecondVRFNetworkOverlappingIP, "vrfInterface": "net2"}})
	podHasCorrectVRFConfig(apiclient, podServer,
		[]map[string]string{
			{"vrfName": VRFBlueName, "vrfClientIP": podServerFirstVRFNetworkIPAddress, "vrfInterface": "net1"},
			{"vrfName": VRFRedName, "vrfClientIP": podServerSecondVRFNetworkOverlappingIP, "vrfInterface": "net2"}})
	err := pingIPViaVRF(apiclient, podClient, VRFRedName, podServerSecondVRFNetworkOverlappingIP)
	Expect(err).ToNot(HaveOccurred())

	By("Run client/server ICMP connectivity tests")
	err = pingIPViaVRF(apiclient, podClient, VRFBlueName, podServerFirstVRFNetworkIPAddress)
	Expect(err).ToNot(HaveOccurred())
	err = apiclient.Pods(namespace).Delete(context.Background(), podServer.Name, metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		_, err := apiclient.Pods(namespace).Get(context.Background(), podServer.Name, metav1.GetOptions{})
		return err
	}, podWaitingTime, 5*time.Second).Should(HaveOccurred())

	By("Run client/server ICMP negative connectivity tests")
	err = pingIPViaVRF(apiclient, podClient, VRFBlueName, podServerFirstVRFNetworkIPAddress)
	Expect(err).To(HaveOccurred())
	err = pingIPViaVRF(apiclient, podClient, VRFRedName, podServerSecondVRFNetworkOverlappingIP)
	Expect(err).To(HaveOccurred())
	err = pingIPViaVRF(apiclient, podClient, "eth0", podServerSecondVRFNetworkOverlappingIP)
	Expect(err).ToNot(HaveOccurred())
}
