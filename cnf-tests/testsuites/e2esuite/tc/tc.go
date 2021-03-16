package tc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	igntypes "github.com/coreos/ignition/config/v2_2/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/machineconfigpool"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	utilNodes "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcfgScheme "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const hostnameLabel = "kubernetes.io/hostname"
const egressMCName = "tc-egress"
const ingressMCName = "tc-ingress"

type iperf3Output struct {
	Intervals []struct {
		Sum struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum"`
	} `json:"intervals"`
}

var (
	tcNodeSelector  string
	roleWorkerCNF   string
	egressOverride  string
	ingressOverride string
	egressBitLimit  float64
	ingressBitLimit float64
	isSingleNode    bool
	testEgress      bool
	testIngress     bool
)

func init() {
	roleWorkerCNF = os.Getenv("ROLE_WORKER_CNF")
	if roleWorkerCNF != "" {
		tcNodeSelector = fmt.Sprintf("node-role.kubernetes.io/%s=", roleWorkerCNF)
	}

	testEgress = true
	if os.Getenv("TC_TEST_EGRESS") == "false" {
		testEgress = false
	}

	testIngress = true
	if os.Getenv("TC_TEST_INGRESS") == "false" {
		testIngress = false
	}

	egressOverride = os.Getenv("TC_TEST_EGRESS_OVERRIDE")
	if egressOverride == "" {
		egressOverride = "500mbit"
	}

	ingressOverride = os.Getenv("TC_TEST_INGRESS_OVERRIDE")
	if ingressOverride == "" {
		ingressOverride = "300mbit"
	}
}

var _ = Describe("tc", func() {
	gSkipReason := ""
	execute.BeforeAll(func() {
		err := namespaces.Create(namespaces.TCTest, client.Client)
		Expect(err).ToNot(HaveOccurred())
		isSingleNode, err = utilNodes.IsSingleNodeCluster()
		Expect(err).ToNot(HaveOccurred())
		if isSingleNode {
			gSkipReason = "At least two nodes are required for tc tests."
			return
		}
		if !discovery.Enabled() {
			if roleWorkerCNF == "" {
				gSkipReason = "ROLE_WORKER_CNF was not specified, can't determine nodes for tc MachineConfig in discovery."
				return
			}
		}
	})

	Context("Test egress", func() {
		var receiverNode string
		var senderNode string
		mcpUpdated := false
		var err error
		eSkipReason := ""
		var mcpName string
		execute.BeforeAll(func() {
			if gSkipReason != "" {
				Skip(gSkipReason)
			}
			if !testEgress {
				eSkipReason = "Egress tests are skipped because of env var"
				Skip(eSkipReason)
			}
			if !discovery.Enabled() {
				mcpName, err = findMCPByMCRole(roleWorkerCNF)
				if err != nil {
					eSkipReason = err.Error()
					Fail(err.Error())
				}

				_, _, err = findTCRateWithNodeSelector(isEgressLimiting, tcNodeSelector)
				if err != nil {
					_, err = createEgressMC()
					if err != nil {
						eSkipReason = err.Error()
						Fail(err.Error())
					}
					mcpUpdated = true
				}

				if mcpUpdated {
					err = waitForMCPStable(mcpName)
					if err != nil {
						eSkipReason = err.Error()
						Fail(err.Error())
					}
				}
			}

			tcNodeSelector, egressBitLimit, err = findTCRateWithNodeSelector(isEgressLimiting, tcNodeSelector)
			if err != nil {
				eSkipReason = err.Error()
				Skip(eSkipReason)
			}

		})

		BeforeEach(func() {
			if gSkipReason != "" {
				Skip(gSkipReason)
			}
			if eSkipReason != "" {
				Skip(eSkipReason)
			}

			namespaces.Clean(namespaces.TCTest, "testpod-", client.Client)
			By("Choosing the test nodes")
			nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
				LabelSelector: tcNodeSelector,
			})
			Expect(err).ToNot(HaveOccurred())

			filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
			Expect(err).ToNot(HaveOccurred())

			if discovery.Enabled() && len(filtered) <= 1 {
				Skip("Did not find enough nodes with tc limitations")
			} else {
				Expect(len(filtered)).To(BeNumerically(">", 1), "Not enough nodes for tc tests")
			}
			receiverNode = filtered[0].ObjectMeta.Labels[hostnameLabel]
			senderNode = filtered[1].ObjectMeta.Labels[hostnameLabel]
		})

		DescribeTable("Test tc limitations are correctly applied", func(senderHostNetwork, receiverHostNetwork bool) {
			By("Creating receiver pod")
			receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
			receiverIP := receiverPod.Status.PodIP

			By("Creating sender pod")
			senderPod := createSenderPod(senderNode, senderHostNetwork, receiverIP)

			By("Waiting for the sender to end")
			time.Sleep(15 * time.Second)

			By("Parsing the sender's output")
			limitWorks := parseIperfOutput(senderPod, egressBitLimit)
			Expect(limitWorks).To(BeTrue())
		},
			Entry("SDN Pod to SDN Pod", false, false),
			Entry("Host Pod to SDN Pod", true, false),
			Entry("Host Pod to Host Pod", true, true))

		It("Cleanup applied egress tc MachineConfigs", func() {
			if !mcpUpdated {
				Skip("No egress MC was applied by the test")
			}
			err = client.Client.MachineConfigs().Delete(context.Background(), egressMCName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = waitForMCPStable(mcpName)
			Expect(err).ToNot(HaveOccurred())
		})

	})
	Context("Test ingress", func() {
		var receiverNode string
		var senderNode string
		var err error
		iSkipReason := ""
		mcpUpdated := false
		var mcpName string
		execute.BeforeAll(func() {
			if gSkipReason != "" {
				Skip(gSkipReason)
			}
			if !testIngress {
				iSkipReason = "Ingress tests are skipped because of env var"
				Skip(iSkipReason)
			}
			if !discovery.Enabled() {
				mcpName, err = findMCPByMCRole(roleWorkerCNF)
				if err != nil {
					iSkipReason = err.Error()
					Fail(iSkipReason)
				}

				_, _, err = findTCRateWithNodeSelector(isIngressLimiting, tcNodeSelector)
				if err != nil {
					_, err = createIngressMC()
					if err != nil {
						iSkipReason = err.Error()
						Fail(iSkipReason)
					}
					Expect(err).ToNot(HaveOccurred())
					mcpUpdated = true
				}
				if mcpUpdated {
					err = waitForMCPStable(mcpName)
					if err != nil {
						iSkipReason = err.Error()
						Skip(iSkipReason)
					}
				}
			}

			tcNodeSelector, ingressBitLimit, err = findTCRateWithNodeSelector(isIngressLimiting, tcNodeSelector)
			if err != nil {
				iSkipReason = err.Error()
			}
		})

		BeforeEach(func() {
			if gSkipReason != "" {
				Skip(gSkipReason)
			}
			if iSkipReason != "" {
				Skip(iSkipReason)
			}
			namespaces.Clean(namespaces.TCTest, "testpod-", client.Client)
			By("Choosing the test nodes")
			nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
				LabelSelector: tcNodeSelector,
			})
			Expect(err).ToNot(HaveOccurred())

			filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
			Expect(err).ToNot(HaveOccurred())

			if discovery.Enabled() && len(filtered) <= 1 {
				Skip("Did not find enough nodes with tc limitations")
			} else {
				Expect(len(filtered)).To(BeNumerically(">", 1), "Not enough nodes for tc tests")
			}
			receiverNode = filtered[0].ObjectMeta.Labels[hostnameLabel]
			senderNode = filtered[1].ObjectMeta.Labels[hostnameLabel]
		})

		DescribeTable("Test tc limitations are correctly applied", func(senderHostNetwork, receiverHostNetwork bool) {
			By("Creating receiver pod")
			receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
			receiverIP := receiverPod.Status.PodIP

			By("Creating sender pod")
			_ = createSenderPod(senderNode, senderHostNetwork, receiverIP)

			By("Waiting for the sender to end")
			time.Sleep(15 * time.Second)

			By("Parsing the receiver's output")
			limitWorks := parseIperfOutput(receiverPod, ingressBitLimit)
			Expect(limitWorks).To(BeTrue())
		},
			Entry("SDN Pod to SDN Pod", false, false),
			Entry("Host Pod to SDN Pod", true, false),
			Entry("Host Pod to Host Pod", true, true))

		It("Cleanup applied ingress tc MachineConfigs", func() {
			if !mcpUpdated {
				Skip("No ingress MC was applied by the test")
			}
			err := client.Client.MachineConfigs().Delete(context.Background(), ingressMCName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = waitForMCPStable(mcpName)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})

func parseIperfOutput(pod *corev1.Pod, limitRate float64) bool {
	parsedOutput := iperf3Output{}
	out, err := pods.GetLog(pod)
	Expect(err).ToNot(HaveOccurred())
	err = json.Unmarshal([]byte(out), &parsedOutput)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(parsedOutput.Intervals)).To(BeNumerically(">", 0))
	return rateEnforced(parsedOutput, limitRate)
}

func createSenderPod(nodeName string, hostNetwork bool, receiverIP string) *corev1.Pod {
	senderPodDefinition := defineSenderPod(nodeName, hostNetwork, receiverIP)
	senderPod, err := client.Client.Pods(namespaces.TCTest).Create(context.Background(), senderPodDefinition, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	err = pods.WaitForCondition(client.Client, senderPod, corev1.ContainersReady, corev1.ConditionTrue, 2*time.Minute)
	Expect(err).ToNot(HaveOccurred())
	return senderPod
}

func createReceiverPod(nodeName string, hostNetwork bool) *corev1.Pod {
	receiverPodDefinition := defineReceiverPod(nodeName, hostNetwork)
	receiverPod, err := client.Client.Pods(namespaces.TCTest).Create(context.Background(), receiverPodDefinition, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	err = pods.WaitForCondition(client.Client, receiverPod, corev1.ContainersReady, corev1.ConditionTrue, 2*time.Minute)
	Expect(err).ToNot(HaveOccurred())

	By("Getting updated receiver spec")
	receiverPod, err = client.Client.Pods(receiverPod.Namespace).Get(context.Background(), receiverPod.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return receiverPod
}

func defineReceiverPod(nodeName string, hostNetwork bool) *corev1.Pod {
	var pod *corev1.Pod
	if hostNetwork {
		pod = pods.DefinePodOnHostNetwork(namespaces.TCTest, nodeName)
	} else {
		pod = pods.DefinePodOnNode(namespaces.TCTest, nodeName)
	}
	pod = pods.RedefineWithCommand(pod, []string{"/bin/bash", "-c"}, []string{"iperf3 -s -J -1; sleep INF"})
	return pod
}

func defineSenderPod(nodeName string, hostNetwork bool, receiverIP string) *corev1.Pod {
	var pod *corev1.Pod
	if hostNetwork {
		pod = pods.DefinePodOnHostNetwork(namespaces.TCTest, nodeName)
	} else {
		pod = pods.DefinePodOnNode(namespaces.TCTest, nodeName)
	}
	pod = pods.RedefineWithCommand(pod, []string{"/bin/bash", "-c"}, []string{fmt.Sprintf("iperf3 -u -b 0 -c %s -t 10 -J; sleep INF", receiverIP)})
	return pod
}

func findTCRateWithNodeSelector(tcType func(string) float64, tcNodeSelector string) (string, float64, error) {
	mcList, err := client.Client.MachineConfigs().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mc := range mcList.Items {
		enables, err := isMCLimiting(mc, tcType)
		if err != nil {
			return "", -1, err
		}
		if enables != -1 {
			mcLabel, found := mc.ObjectMeta.Labels["machineconfiguration.openshift.io/role"]
			if !found {
				continue
			}
			nodeSelector, err := findTCNodeSelectorByMCLabel(mcLabel)
			if err != nil {
				continue
			}
			if tcNodeSelector != "" && tcNodeSelector != nodeSelector {
				continue
			}

			return nodeSelector, enables, nil
		}
	}

	return "", -1, errors.New("Cannot find a machine configuration with specified tcNodeSelector that applies tc limitation")
}

func isMCLimiting(mc mcfgv1.MachineConfig, tcType func(string) float64) (float64, error) {
	ignitionConfig := igntypes.Config{}
	if mc.Spec.Config.Raw == nil {
		return -1, nil
	}

	err := json.Unmarshal(mc.Spec.Config.Raw, &ignitionConfig)
	if err != nil {
		return -1, fmt.Errorf("Failed to unmarshal ignition config %v", err)
	}

	for _, unit := range ignitionConfig.Systemd.Units {
		bitLimit := tcType(unit.Contents)
		if bitLimit != -1 {
			return bitLimit, nil
		}

	}
	return -1, nil
}

func isEgressLimiting(s string) float64 {
	egressRegexp := regexp.MustCompile("tc qdisc add .* rate (([0-9]*)(|k|m|g)bit) burst")
	m := egressRegexp.FindStringSubmatch(s)
	if len(m) < 2 {
		return -1
	}
	return toBits(m[1])
}

func isIngressLimiting(s string) float64 {
	ingressRegexp := regexp.MustCompile("tc filter add .* action police .* drop rate (([0-9]*)(|k|m|g)bit) burst")
	m := ingressRegexp.FindStringSubmatch(s)
	if len(m) < 2 {
		return -1
	}
	return toBits(m[1])
}

func toBits(bitsize string) float64 {
	r := regexp.MustCompile("([0-9]*)(|k|m|g)bit")
	m := r.FindStringSubmatch(bitsize)
	b, _ := strconv.ParseFloat(m[1], 64)
	switch m[2] {
	case "":
		return b
	case "k":
		return b * 1000
	case "m":
		return b * 1000000
	case "g":
		return b * 1000000000
	default:
		return -1
	}
}

func findTCNodeSelectorByMCLabel(mcLabel string) (string, error) {
	mcpList, err := client.Client.MachineConfigPools().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mcp := range mcpList.Items {
		for _, lsr := range mcp.Spec.MachineConfigSelector.MatchExpressions {
			for _, value := range lsr.Values {
				if value == mcLabel {
					for key, label := range mcp.Spec.NodeSelector.MatchLabels {
						newTCNodeSelector := key + "=" + label
						return newTCNodeSelector, nil
					}
				}
			}
		}
		for _, v := range mcp.Spec.MachineConfigSelector.MatchLabels {
			if v == mcLabel {
				for key, label := range mcp.Spec.NodeSelector.MatchLabels {
					newTCNodeSelector := key + "=" + label
					return newTCNodeSelector, nil
				}
			}
		}
	}

	return "", errors.New("Cannot find TCNodeSelector")
}

func findMCPByMCRole(mcLabel string) (string, error) {
	mcpList, err := client.Client.MachineConfigPools().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mcp := range mcpList.Items {
		for _, lsr := range mcp.Spec.MachineConfigSelector.MatchExpressions {
			for _, value := range lsr.Values {
				if value == mcLabel {
					return mcp.Name, nil
				}
			}
		}
		for _, v := range mcp.Spec.MachineConfigSelector.MatchLabels {
			if v == mcLabel {
				return mcp.Name, nil
			}
		}
	}
	return "", errors.New("Cannot find MCP that targets mcLabel")
}

func rateEnforced(o iperf3Output, limitRate float64) bool {
	for _, sum := range o.Intervals {
		if sum.Sum.BitsPerSecond > limitRate {
			return false
		}
	}
	return true
}

func waitForMCPStable(mcpName string) error {
	mcp := &mcfgv1.MachineConfigPool{}
	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: mcpName}, mcp)
	if err != nil {
		return err
	}

	By("Waiting for the mcp to start updating")
	err = machineconfigpool.WaitForCondition(
		client.Client,
		&mcfgv1.MachineConfigPool{ObjectMeta: metav1.ObjectMeta{Name: mcpName}},
		mcfgv1.MachineConfigPoolUpdating,
		corev1.ConditionTrue,
		2*time.Minute)
	if err != nil {
		return err
	}

	By("Waiting for the mcp to be updated")
	// We need to wait a long time here for the nodes to reboot
	err = machineconfigpool.WaitForCondition(
		client.Client,
		&mcfgv1.MachineConfigPool{ObjectMeta: metav1.ObjectMeta{Name: mcpName}},
		mcfgv1.MachineConfigPoolUpdated,
		corev1.ConditionTrue,
		time.Duration(30*mcp.Status.MachineCount)*time.Minute)

	return err
}

func createEgressMC() (*mcfgv1.MachineConfig, error) {
	mcContent := fmt.Sprintf(`
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: %s
  name: %s
spec:
  config:
    ignition:
      version: 2.2.0
    systemd:
      units:
        - contents: |
            [Unit]
            Description=Configure TC egress limiting on br-ex
            Requires=sys-devices-virtual-net-br\x2dex.device 
            After=sys-devices-virtual-net-br\x2dex.device 
            Before=kubelet.service crio.service
            
            [Service]
            Type=oneshot
            RemainAfterExit=yes
            ExecStart=/sbin/tc qdisc add dev br-ex root tbf rate %s burst 256kbit latency 400ms
            
            [Install]
            WantedBy=multi-user.target
          enabled: true
          name: tc-egress-limit.service
`, roleWorkerCNF, egressMCName, egressOverride)
	mc, err := decodeMCYaml(mcContent)
	if err != nil {
		return nil, err
	}

	rate, err := isMCLimiting(*mc, isEgressLimiting)
	if err != nil {
		return nil, err
	}
	if rate == -1 {
		return nil, fmt.Errorf("egress limit requested is not valid")
	}
	err = client.Client.Create(context.TODO(), mc)
	return mc, err
}

func createIngressMC() (*mcfgv1.MachineConfig, error) {
	mcContent := fmt.Sprintf(`
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: %s
  name: %s
spec:
  config:
    ignition:
      version: 2.2.0
    systemd:
      units:
        - contents: |
            [Unit]
            Description=Configure TC ingress policing on br-ex
            Requires=sys-devices-virtual-net-br\x2dex.device
            After=sys-devices-virtual-net-br\x2dex.device
            Before=kubelet.service crio.service
            
            [Service]
            Type=oneshot
            RemainAfterExit=yes
            TimeoutStartSec=10s
            ExecStart=/bin/bash -c 'while ! /sbin/tc qdisc add dev br-ex handle "ffff:" ingress; do /bin/sleep 1; done'
            ExecStart=/sbin/tc filter add dev br-ex parent "ffff:" matchall action police conform-exceed drop rate %s burst 10mbit
            
            [Install]
            WantedBy=multi-user.target
          enabled: true
          name: tc-ingress-limit.service
`, roleWorkerCNF, ingressMCName, ingressOverride)
	mc, err := decodeMCYaml(mcContent)
	if err != nil {
		return nil, err
	}
	rate, err := isMCLimiting(*mc, isIngressLimiting)
	if err != nil {
		return nil, err
	}
	if rate == -1 {
		return nil, fmt.Errorf("ingress limit requested is not valid")
	}
	err = client.Client.Create(context.TODO(), mc)
	return mc, err
}

func decodeMCYaml(mcyaml string) (*mcfgv1.MachineConfig, error) {
	decode := mcfgScheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(mcyaml), nil, nil)
	if err != nil {
		return nil, err
	}
	mc, ok := obj.(*mcfgv1.MachineConfig)
	Expect(ok).To(Equal(true))
	return mc, err
}
