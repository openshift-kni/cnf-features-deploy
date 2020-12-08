package vrf

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	client "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	sameNode                     = "Same Node"
	ipStackIPv4                  = "ipv4"
	ipStackIPv6                  = "ipv6"
	hostnameLabel                = "kubernetes.io/hostname"
	TestNamespace                = "vrf-testing"
	podWaitingTime time.Duration = 2 * time.Minute
	VRFBlueName                  = "blue"
	VRFRedName                   = "red"
)

var (
	nodeParameters    = []string{sameNode}
	ipStackParameters = []string{ipStackIPv4, ipStackIPv6}
)

type vrfTestParameters struct {
	Node    string
	IPStack string
}

var _ = Describe("[vrf]", func() {
	describe := func(node string, ipStack string) string {

		VRFParameters, err := newVRFTestParameters(node, ipStack)
		if err != nil {
			return fmt.Sprintf("error in parameters: node=%s, ipStack=%s", node, ipStack)
		}
		params, err := json.Marshal(VRFParameters)
		if err != nil {
			return fmt.Sprintf("error in parameters: node=%s, ipStack=%s", node, ipStack)
		}

		return fmt.Sprintf("%s", string(params))
	}
	apiclient := client.New("")

	var nodesList []k8sv1.Node
	var vrfBlue netattdefv1.NetworkAttachmentDefinition
	var vrfRed netattdefv1.NetworkAttachmentDefinition

	execute.BeforeAll(func() {
		err := namespaces.Create(TestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
		nodesList, err = nodes.GetByRole(apiclient, "worker")
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
		DescribeTable("Integration: NAD, IPAM: static, Interfaces: 2, Scheme: 2 Pods 2 VRFs OCP Primary network overlap",
			func(node string, ipStack string) {
				var podClientVRFBlueIPAddress string
				var podServerVRFBlueIPAddress string

				By("Create overlapping IP pods")
				podClientVRFRedOverlappingIP := getOverlapIP(apiclient, nodesList[0].Name, "overlap-client-ip")
				podServerVRFRedOverlappingIP := getOverlapIP(apiclient, nodesList[0].Name, "overlap-server-ip")

				if ipStack == ipStackIPv4 && net.ParseIP(podClientVRFRedOverlappingIP).To4() == nil {
					Skip("Skipping IPv4 test. Cluster supports IPv6 protocol")
				} else if ipStack == ipStackIPv4 && net.ParseIP(podClientVRFRedOverlappingIP).To4() != nil {
					podClientVRFBlueIPAddress = "10.255.255.1"
					podServerVRFBlueIPAddress = "10.255.255.2"
				} else {
					Skip("Unpsupported protocol parameter")
				}

				By("Create VRFs client/server pods")
				podClientIpamConfig := fmt.Sprintf(`[{"name": "%s", "mac": "%s", "ips": ["%s/24"]}, {"name": "%s", "mac": "%s", "ips": ["%s/24"]}]`,
					vrfBlue.Name, "20:04:0f:f1:88:A1", podClientVRFBlueIPAddress, vrfRed.Name, "20:04:0f:f1:88:B2", podClientVRFRedOverlappingIP)
				podClient := redefineAsPrivilegedWithNamePrefix(
					pods.RedefinePodWithNetwork(pods.DefinePodOnNode(TestNamespace, nodesList[0].Name), podClientIpamConfig), "client-vrf")
				podServerIpamConfig := fmt.Sprintf(`[{"name": "%s", "mac": "%s", "ips": ["%s/24"]}, {"name": "%s", "mac": "%s", "ips": ["%s/24"]}]`,
					vrfBlue.Name, "20:04:0f:f1:88:A3", podServerVRFBlueIPAddress, vrfRed.Name, "20:04:0f:f1:88:B4", podServerVRFRedOverlappingIP)
				podServer := redefineAsPrivilegedWithNamePrefix(
					pods.RedefinePodWithNetwork(pods.DefinePodOnNode(TestNamespace, nodesList[0].Name), podServerIpamConfig), "server-vrf")
				waitUntilPodCreatedAndRunning(apiclient, podClient)
				waitUntilPodCreatedAndRunning(apiclient, podServer)

				By("Validate that client/server VRFs have correct configuration")
				podHasCorrectVrfConfig(apiclient, podClient.Name,
					[]map[string]string{
						{"vrfName": VRFBlueName, "vrfClientIP": podClientVRFBlueIPAddress, "vrfInterface": "net1"},
						{"vrfName": VRFRedName, "vrfClientIP": podClientVRFRedOverlappingIP, "vrfInterface": "net2"}})
				podHasCorrectVrfConfig(apiclient, podServer.Name,
					[]map[string]string{
						{"vrfName": VRFBlueName, "vrfClientIP": podServerVRFBlueIPAddress, "vrfInterface": "net1"},
						{"vrfName": VRFRedName, "vrfClientIP": podServerVRFRedOverlappingIP, "vrfInterface": "net2"}})
				err := pingIPViaVRF(apiclient, podClient.Name, VRFRedName, podServerVRFRedOverlappingIP)
				Expect(err).ToNot(HaveOccurred())

				By("Run client/server ICMP connectivity tests")
				err = pingIPViaVRF(apiclient, podClient.Name, VRFBlueName, podServerVRFBlueIPAddress)
				Expect(err).ToNot(HaveOccurred())
				err = apiclient.Pods(TestNamespace).Delete(context.Background(), podServer.Name, metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() error {
					_, err := apiclient.Pods(TestNamespace).Get(context.Background(), podServer.Name, metav1.GetOptions{})
					return err
				}, podWaitingTime, 5*time.Second).Should(HaveOccurred())

				By("Run client/server ICMP negative connectivity tests")
				err = pingIPViaVRF(apiclient, podClient.Name, VRFBlueName, podServerVRFBlueIPAddress)
				Expect(err).To(HaveOccurred())
				err = pingIPViaVRF(apiclient, podClient.Name, VRFRedName, podServerVRFRedOverlappingIP)
				Expect(err).To(HaveOccurred())
				err = pingIPViaVRF(apiclient, podClient.Name, "eth0", podServerVRFRedOverlappingIP)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry(describe, sameNode, ipStackIPv4),
		)
	})
})

func pingIPViaVRF(cs *client.ClientSet, client string, vrfName string, DestIPAddr string) error {
	pingCommand := []string{"ping", "-I", vrfName, "-c5", DestIPAddr}
	pod, err := cs.Pods(TestNamespace).Get(context.Background(), client, metav1.GetOptions{})
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

func getOverlapIP(cs *client.ClientSet, nodeName string, podNamePrefix string) string {
	tempPodDefinition := redefineAsPrivilegedWithNamePrefix(pods.DefinePodOnNode(TestNamespace, nodeName), podNamePrefix)
	err := cs.Create(context.Background(), tempPodDefinition)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() k8sv1.PodPhase {
		tempPod, _ := cs.Pods(TestNamespace).Get(context.Background(), tempPodDefinition.Name, metav1.GetOptions{})
		return tempPod.Status.Phase
	}, podWaitingTime, time.Second).Should(Equal(k8sv1.PodRunning))

	pod, err := cs.Pods(TestNamespace).Get(context.Background(), tempPodDefinition.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return pod.Status.PodIP
}

func waitUntilPodCreatedAndRunning(cs *client.ClientSet, podStruct *k8sv1.Pod) {
	err := cs.Create(context.Background(), podStruct)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() k8sv1.PodPhase {
		tempPod, _ := cs.Pods(TestNamespace).Get(context.Background(), podStruct.Name, metav1.GetOptions{})
		return tempPod.Status.Phase
	}, podWaitingTime, time.Second).Should(Equal(k8sv1.PodRunning))
}

func podHasCorrectVrfConfig(cs *client.ClientSet, podName string, vrfMapsConfig []map[string]string) {
	pod, err := cs.Pods(TestNamespace).Get(context.Background(), podName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, vrfMapConfig := range vrfMapsConfig {
		validateVrfIPAddrCommand := []string{"ip", "addr", "show", fmt.Sprintf("%s", vrfMapConfig["vrfInterface"])}
		Eventually(func() bool {
			vrfIface, _ := pods.ExecCommand(cs, *pod, validateVrfIPAddrCommand)
			return strings.Contains(vrfIface.String(), vrfMapConfig["vrfClientIP"])
		}, podWaitingTime, 5*time.Second).Should(BeTrue(), fmt.Errorf("VRF interface is not present"))

		validateVRFRouteTableCommand := []string{"ip", "route", "show", "vrf", fmt.Sprintf("%s", vrfMapConfig["vrfName"])}
		Eventually(func() bool {
			vrfRouteTable, _ := pods.ExecCommand(cs, *pod, validateVRFRouteTableCommand)
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

func newVRFTestParameters(Node string, IPStack string) (*vrfTestParameters, error) {
	VRFTestParameters := new(vrfTestParameters)
	err := paramInParamList(Node, nodeParameters)
	if err != nil {
		return nil, err
	}
	VRFTestParameters.Node = Node

	err = paramInParamList(IPStack, ipStackParameters)
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
