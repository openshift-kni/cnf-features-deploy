package ovs_qos

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	igntypes "github.com/coreos/ignition/config/v2_2/types"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/machineconfigpool"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	utilNodes "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
)

type iperf3Output struct {
	Intervals []struct {
		Sum struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum"`
	} `json:"intervals"`
}

type qosTestParameters struct {
	Connectivity string
}

const (
	hostToHost = "Host Pod to Host Pod"
	hostToSDN  = "Host Pod to SDN Pod"
	sdnToSDN   = "SDN Pod to SDN Pod"
	iperfPort  = 30003
	// EgressMCName contains the name of the egress MC applied by the ovs_qos tests
	EgressMCName = "qos-egress"
	// IngressMCName contains the name of the ingress MC applied by the ovs_qos tests
	IngressMCName = "qos-ingress"
	// LimitMultiplier is the multiplier to get the MC limit rate without discovery
	LimitMultiplier = 0.3 // TODO: see commit message
)

var (
	qosNodeSelector        string
	roleWorkerCNF          string
	iperf3BitrateOverride  string
	egressBitLimit         float64
	ingressBitLimit        float64
	isSingleNode           bool
	connectivityParameters = []string{hostToHost, hostToSDN, sdnToSDN}
)

func init() {
	iperf3BitrateOverride = os.Getenv("IPERF3_BITRATE_OVERRIDE")
	if iperf3BitrateOverride == "" {
		iperf3BitrateOverride = "0"
	}

	roleWorkerCNF = os.Getenv("ROLE_WORKER_CNF")
	if roleWorkerCNF == "" {
		roleWorkerCNF = "worker-cnf"
	}

	qosNodeSelector = fmt.Sprintf("node-role.kubernetes.io/%s=", roleWorkerCNF)
}

var _ = Describe("[ovs_qos]", func() {
	describe := func(connectivity string) string {
		QOSParameters, err := newQoSTestParameters(connectivity)
		if err != nil {
			return fmt.Sprintf("error in parameters: Connectivity=%s", connectivity)
		}
		params, err := json.Marshal(QOSParameters)
		if err != nil {
			return fmt.Sprintf("error in parameters: Connectivity=%s", connectivity)
		}

		return string(params)
	}

	gSkipReason := ""
	execute.BeforeAll(func() {
		err := namespaces.Create(namespaces.OVSQOSTest, client.Client)
		Expect(err).ToNot(HaveOccurred())
		isSingleNode, err = utilNodes.IsSingleNodeCluster()
		Expect(err).ToNot(HaveOccurred())
		if isSingleNode {
			gSkipReason = "At least two nodes are required for ovs_qos tests."
			return
		}
		fmt.Fprintln(GinkgoWriter, "iperf3BitrateOverride:", iperf3BitrateOverride)
	})

	Describe("ovs_qos_egress", func() {
		Context("validate egress QoS limitation", func() {
			var receiverNode string
			var senderNode string
			var err error
			var mcp mcfgv1.MachineConfigPool

			eSkipReason := ""
			mcpUpdated := false
			execute.BeforeAll(func() {
				if gSkipReason != "" {
					Skip(gSkipReason)
				}

				if discovery.Enabled() {
					qosNodeSelector, egressBitLimit, err = findQoSRateWithNodeSelector(isEgressLimiting, qosNodeSelector)
					if err != nil {
						eSkipReason = err.Error()
						Skip(eSkipReason)
					}
				}

				filtered, err := getFilteredNodes()
				if err != nil {
					eSkipReason = err.Error()
					Fail(eSkipReason)
				}

				if len(filtered) <= 1 {
					if discovery.Enabled() {
						eSkipReason = "Did not find enough nodes with ovs_qos limitations"
						Skip(eSkipReason)
					} else {
						eSkipReason = "Not enough nodes for ovs_qos tests"
						Fail(eSkipReason)
					}

				}

				receiverNode = filtered[0].Name
				senderNode = filtered[1].Name

				if !discovery.Enabled() {
					mcp, err = machineconfigpool.FindMCPByMCLabel(roleWorkerCNF)
					if err != nil {
						eSkipReason = err.Error()
						Fail(err.Error())
					}

					_, egressBitLimit, err = findQoSRateWithNodeSelector(isEgressLimiting, qosNodeSelector)
					if err != nil {
						initialRate, err := findInitialRate(receiverNode, senderNode, true)
						if err != nil {
							namespaces.Clean(namespaces.OVSQOSTest, "testpod-", client.Client)
							eSkipReason = err.Error()
							Fail(eSkipReason)
						}

						egressBitLimit = initialRate * LimitMultiplier
						_, err = createEgressMC(int64(egressBitLimit))
						if err != nil {
							eSkipReason = err.Error()
							Fail(eSkipReason)
						}
						mcpUpdated = true
					}

					if mcpUpdated {
						err = machineconfigpool.WaitForMCPStable(mcp)
						if err != nil {
							eSkipReason = err.Error()
							Fail(err.Error())
						}
					}
				}
				fmt.Fprintln(GinkgoWriter, "MC egress bit limit:", egressBitLimit)
			})

			BeforeEach(func() {
				if gSkipReason != "" {
					Skip(gSkipReason)
				}
				if eSkipReason != "" {
					Skip(eSkipReason)
				}

				err := namespaces.Clean(namespaces.OVSQOSTest, "testpod-", client.Client)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Validate MCO applied egress MachineConfig on the relevant nodes", func() {
				err := validateEgressQosOnNodes(int64(egressBitLimit))
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("Test limitations are correctly applied", func(connectivity string) {
				By("Getting connectivity implied networks")
				senderHostNetwork, receiverHostNetwork, err := networksByConnectivity(connectivity)
				Expect(err).ToNot(HaveOccurred())

				By("Creating receiver pod")
				receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
				receiverIP := receiverPod.Status.PodIP

				By("Creating sender pod")
				senderPod := createSenderPod(senderNode, senderHostNetwork, receiverIP)

				By("Parsing the sender's output")
				senderOutput, err := parseIperfOutput(senderPod)
				Expect(err).ToNot(HaveOccurred())
				Expect(rateEnforced(senderOutput, egressBitLimit)).To(BeTrue())
			},
				Entry(describe, sdnToSDN),
				Entry(describe, hostToSDN),
				Entry(describe, hostToHost))

			DescribeTable("Test limitations are not applied within the same node", func(connectivity string) {
				By("Getting connectivity implied networks")
				senderHostNetwork, receiverHostNetwork, err := networksByConnectivity(connectivity)
				Expect(err).ToNot(HaveOccurred())

				By("Creating receiver pod")
				receiverPod := createReceiverPod(senderNode, receiverHostNetwork)
				receiverIP := receiverPod.Status.PodIP

				By("Creating sender pod")
				senderPod := createSenderPod(senderNode, senderHostNetwork, receiverIP)

				By("Parsing the sender's output")
				senderOutput, err := parseIperfOutput(senderPod)
				Expect(err).ToNot(HaveOccurred())
				Expect(rateEnforced(senderOutput, egressBitLimit)).To(BeFalse())
			},
				Entry(describe, sdnToSDN),
				Entry(describe, hostToSDN))

			It("Validate MCO removed egress MachineConfig and disabled QOS limitation on the relevant nodes", func() {
				if !mcpUpdated {
					Skip("No egress MC was applied by the test")
				}
				err = client.Client.MachineConfigs().Delete(context.Background(), EgressMCName, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = machineconfigpool.WaitForMCPStable(mcp)
				Expect(err).ToNot(HaveOccurred())

				err = validateEgressQosOnNodes(0)
				Expect(err).ToNot(HaveOccurred())

				senderHostNetwork, receiverHostNetwork, err := networksByConnectivity(sdnToSDN)
				Expect(err).ToNot(HaveOccurred())

				receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
				receiverIP := receiverPod.Status.PodIP
				senderPod := createSenderPod(senderNode, senderHostNetwork, receiverIP)
				senderOutput, err := parseIperfOutput(senderPod)
				Expect(err).ToNot(HaveOccurred())
				Expect(rateEnforced(senderOutput, egressBitLimit)).To(BeFalse())
			})

		})
	})

	Describe("ovs_qos_ingress", func() {
		Context("validate ingress QoS limitation", func() {
			var receiverNode string
			var senderNode string
			var err error
			var mcp mcfgv1.MachineConfigPool

			iSkipReason := ""
			mcpUpdated := false
			execute.BeforeAll(func() {
				if gSkipReason != "" {
					Skip(gSkipReason)
				}

				if discovery.Enabled() {
					qosNodeSelector, ingressBitLimit, err = findQoSRateWithNodeSelector(isIngressLimiting, qosNodeSelector)
					if err != nil {
						iSkipReason = err.Error()
						Skip(iSkipReason)
					}
				}

				filtered, err := getFilteredNodes()
				if err != nil {
					iSkipReason = err.Error()
					Fail(iSkipReason)
				}

				if len(filtered) <= 1 {
					if discovery.Enabled() {
						iSkipReason = "Did not find enough nodes with ovs_qos limitations"
						Skip(iSkipReason)
					} else {
						iSkipReason = "Not enough nodes for ovs_qos tests"
						Fail(iSkipReason)
					}
				}

				receiverNode = filtered[0].Name
				senderNode = filtered[1].Name

				if !discovery.Enabled() {
					mcp, err = machineconfigpool.FindMCPByMCLabel(roleWorkerCNF)
					if err != nil {
						iSkipReason = err.Error()
						Fail(iSkipReason)
					}

					_, ingressBitLimit, err = findQoSRateWithNodeSelector(isIngressLimiting, qosNodeSelector)
					if err != nil {
						initialRate, err := findInitialRate(receiverNode, senderNode, false)
						if err != nil {
							namespaces.Clean(namespaces.OVSQOSTest, "testpod-", client.Client)
							iSkipReason = err.Error()
							Fail(iSkipReason)
						}

						ingressBitLimit = initialRate * LimitMultiplier
						_, err = createIngressMC(int64(ingressBitLimit / 1000))
						if err != nil {
							iSkipReason = err.Error()
							Fail(iSkipReason)
						}
						Expect(err).ToNot(HaveOccurred())
						mcpUpdated = true
					}

					if mcpUpdated {
						err = machineconfigpool.WaitForMCPStable(mcp)
						if err != nil {
							iSkipReason = err.Error()
							Skip(iSkipReason)
						}
					}
				}
				fmt.Fprintln(GinkgoWriter, "MC ingress bit limit:", ingressBitLimit)
			})

			BeforeEach(func() {
				if gSkipReason != "" {
					Skip(gSkipReason)
				}
				if iSkipReason != "" {
					Skip(iSkipReason)
				}

				err := namespaces.Clean(namespaces.OVSQOSTest, "testpod-", client.Client)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Validate MCO applied ingress MachineConfig on the relevant nodes", func() {
				err := validateIngressQosOnNodes(int64(ingressBitLimit / 1000))
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("Test limitations are correctly applied", func(connectivity string) {
				By("Getting connectivity implied networks")
				senderHostNetwork, receiverHostNetwork, err := networksByConnectivity(connectivity)
				Expect(err).ToNot(HaveOccurred())

				By("Creating receiver pod")
				receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
				receiverIP := receiverPod.Status.PodIP

				By("Creating sender pod")
				_ = createSenderPod(senderNode, senderHostNetwork, receiverIP)

				By("Parsing the receiver's output")
				receiverOutput, err := parseIperfOutput(receiverPod)
				Expect(err).ToNot(HaveOccurred())
				Expect(rateEnforced(receiverOutput, ingressBitLimit)).To(BeTrue())
			},
				Entry(describe, sdnToSDN),
				Entry(describe, hostToSDN),
				Entry(describe, hostToHost))

			DescribeTable("Test limitations are not applied within the same node", func(connectivity string) {
				By("Getting connectivity implied networks")
				senderHostNetwork, receiverHostNetwork, err := networksByConnectivity(connectivity)
				Expect(err).ToNot(HaveOccurred())

				By("Creating receiver pod")
				receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
				receiverIP := receiverPod.Status.PodIP

				By("Creating sender pod")
				_ = createSenderPod(receiverNode, senderHostNetwork, receiverIP)

				By("Parsing the receiver's output")
				receiverOutput, err := parseIperfOutput(receiverPod)
				Expect(err).ToNot(HaveOccurred())
				Expect(rateEnforced(receiverOutput, ingressBitLimit)).To(BeFalse())
			},
				Entry(describe, sdnToSDN),
				Entry(describe, hostToSDN))

			It("Validate MCO removed ingress MachineConfig and disabled QOS limitation on the relevant nodes", func() {
				if !mcpUpdated {
					Skip("No ingress MC was applied by the test")
				}
				err := client.Client.MachineConfigs().Delete(context.Background(), IngressMCName, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = machineconfigpool.WaitForMCPStable(mcp)
				Expect(err).ToNot(HaveOccurred())

				err = validateIngressQosOnNodes(0)
				Expect(err).ToNot(HaveOccurred())

				senderHostNetwork, receiverHostNetwork, err := networksByConnectivity(sdnToSDN)
				Expect(err).ToNot(HaveOccurred())

				receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
				receiverIP := receiverPod.Status.PodIP
				_ = createSenderPod(senderNode, senderHostNetwork, receiverIP)

				receiverOutput, err := parseIperfOutput(receiverPod)
				Expect(err).ToNot(HaveOccurred())
				Expect(rateEnforced(receiverOutput, ingressBitLimit)).To(BeFalse())
			})
		})
	})

})

func parseIperfOutput(pod *corev1.Pod) (*iperf3Output, error) {
	parsedOutput := &iperf3Output{}
	out, err := pods.GetLog(pod)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(out), parsedOutput)
	if err != nil {
		return nil, err
	}

	if len(parsedOutput.Intervals) == 0 {
		return nil, fmt.Errorf("parsed iperf3output returned 0 intervals")
	}

	return parsedOutput, nil
}

func createSenderPod(nodeName string, hostNetwork bool, receiverIP string) *corev1.Pod {
	senderPodDefinition := defineQosPod(nodeName, hostNetwork, fmt.Sprintf("iperf3 -u -b %s -c %s -t 10 -J -p %d --pacing-timer 10;", iperf3BitrateOverride, receiverIP, iperfPort))
	senderPodDefinition.GenerateName = "sender-"
	senderPod, err := client.Client.Pods(namespaces.OVSQOSTest).Create(context.Background(), senderPodDefinition, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	err = pods.WaitForPhase(client.Client, senderPod, corev1.PodSucceeded, 5*time.Minute)
	Expect(err).ToNot(HaveOccurred())

	return senderPod
}

func createReceiverPod(nodeName string, hostNetwork bool) *corev1.Pod {
	receiverPodDefinition := defineQosPod(nodeName, hostNetwork, fmt.Sprintf("iperf3 -s -J -1 -p %d;", iperfPort))
	receiverPodDefinition.GenerateName = "receiver-"
	receiverPod, err := client.Client.Pods(namespaces.OVSQOSTest).Create(context.Background(), receiverPodDefinition, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	err = pods.WaitForCondition(client.Client, receiverPod, corev1.ContainersReady, corev1.ConditionTrue, 5*time.Minute)
	Expect(err).ToNot(HaveOccurred())

	receiverPod, err = client.Client.Pods(receiverPod.Namespace).Get(context.Background(), receiverPod.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return receiverPod
}

func defineQosPod(nodeName string, hostNetwork bool, cmd string) *corev1.Pod {
	var pod *corev1.Pod
	if hostNetwork {
		pod = pods.DefinePodOnHostNetwork(namespaces.OVSQOSTest, nodeName)
	} else {
		pod = pods.DefinePodOnNode(namespaces.OVSQOSTest, nodeName)
	}
	pod = pods.RedefineWithCommand(pod, []string{"/bin/bash", "-c"}, []string{cmd})
	pod = pods.RedefineWithRestartPolicy(pod, corev1.RestartPolicyNever)
	return pod
}

func findQoSRateWithNodeSelector(qosType func(string) float64, qosNodeSelector string) (string, float64, error) {
	mcList, err := client.Client.MachineConfigs().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mc := range mcList.Items {
		enables, err := isMCLimiting(mc, qosType)
		if err != nil {
			return "", -1, err
		}
		if enables == -1 {
			continue
		}

		mcLabel, found := mc.ObjectMeta.Labels["machineconfiguration.openshift.io/role"]
		if !found {
			continue
		}
		nodeSelector, err := machineconfigpool.FindNodeSelectorByMCLabel(mcLabel)
		if err != nil {
			continue
		}
		if qosNodeSelector != "" && qosNodeSelector != nodeSelector {
			continue
		}

		return nodeSelector, enables, nil
	}

	return "", -1, fmt.Errorf("cannot find a machine configuration with specified qosNodeSelector that applies limitation")
}

func isMCLimiting(mc mcfgv1.MachineConfig, qosType func(string) float64) (float64, error) {
	ignitionConfig := igntypes.Config{}
	if mc.Spec.Config.Raw == nil {
		return -1, nil
	}

	err := json.Unmarshal(mc.Spec.Config.Raw, &ignitionConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to unmarshal ignition config %v", err)
	}

	for _, unit := range ignitionConfig.Systemd.Units {
		bitLimit := qosType(unit.Contents)
		if bitLimit != -1 {
			return bitLimit, nil
		}

	}
	return -1, nil
}

func isEgressLimiting(s string) float64 {
	egressRegexp, err := regexp.Compile("create qos .* other-config:max-rate=([0-9]*)")
	if err != nil {
		return -1
	}
	m := egressRegexp.FindStringSubmatch(s)
	if len(m) < 2 {
		return -1
	}
	bitRate, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return -1
	}
	return bitRate
}

func isIngressLimiting(s string) float64 {
	ingressRegexp, err := regexp.Compile("set interface .* ingress_policing_rate=([0-9]*)")
	if err != nil {
		return -1
	}
	m := ingressRegexp.FindStringSubmatch(s)
	if len(m) < 2 {
		return -1
	}
	bitRate, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return -1
	}
	// ingress QoS is set in kbits
	return bitRate * 1000
}

func rateEnforced(o *iperf3Output, limitRate float64) bool {
	for _, sum := range o.Intervals {
		if sum.Sum.BitsPerSecond > 1.10*limitRate {
			return false
		}
	}
	return true
}

func createMCWithQOSContent(mcContent string) (*mcfgv1.MachineConfig, error) {
	mc, err := machineconfigpool.DecodeMCYaml(mcContent)
	if err != nil {
		return nil, err
	}

	err = client.Client.Create(context.TODO(), mc)
	return mc, err
}

func createEgressMC(egressRate int64) (*mcfgv1.MachineConfig, error) {
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
            Description=Configure egress bandwidth limiting on br-ex
            Requires=ovs-configuration.service
            After=ovs-configuration.service
            Before=kubelet.service crio.service
            [Service]
            Type=oneshot
            RemainAfterExit=yes
            ExecStart=/bin/bash -c 'phs=$(/bin/nmcli --get-values GENERAL.DEVICES conn show ovs-if-phys0); \
                        /bin/ovs-vsctl set port $phs qos=[]; \
                        existing_qos=$(ovs-vsctl --columns=_uuid find qos other_config={max-rate="%d"} | head -n1 | awk \'{print $NF}\'); \
                        if [ "$existing_qos" == "" ]; then \
                        /bin/ovs-vsctl set port $phs qos=@newqos -- --id=@newqos create qos type=linux-htb other-config:max-rate=%d; else \
                        /bin/ovs-vsctl set port $phs qos=$existing_qos; fi'
            ExecStop=/bin/bash -c 'phs=$(/bin/nmcli --get-values GENERAL.DEVICES conn show ovs-if-phys0); /bin/ovs-vsctl set port $phs qos=[]'
            [Install]
            WantedBy=multi-user.target
          enabled: true
          name: egress-limit.service 
`, roleWorkerCNF, EgressMCName, egressRate, egressRate)

	mc, err := createMCWithQOSContent(mcContent)
	return mc, err
}

func createIngressMC(ingressRate int64) (*mcfgv1.MachineConfig, error) {
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
            Description=Configure ingress bandwidth limiting on br-ex
            Requires=ovs-configuration.service
            After=ovs-configuration.service
            Before=kubelet.service crio.service
            [Service]
            Type=oneshot
            RemainAfterExit=yes
            ExecStart=/bin/bash -c 'phs=$(/bin/nmcli --get-values GENERAL.DEVICES conn show ovs-if-phys0); /bin/ovs-vsctl set interface $phs ingress_policing_rate=%d; /bin/ovs-vsctl set interface $phs ingress_policing_burst=%d'
            ExecStop=/bin/bash -c 'phs=$(/bin/nmcli --get-values GENERAL.DEVICES conn show ovs-if-phys0); /bin/ovs-vsctl set interface $phs ingress_policing_rate=0; /bin/ovs-vsctl set interface $phs ingress_policing_burst=0'
            [Install]
            WantedBy=multi-user.target
          enabled: true
          name: ingress-limit.service
`, roleWorkerCNF, IngressMCName, ingressRate, ingressRate/10)

	mc, err := createMCWithQOSContent(mcContent)
	return mc, err
}

func findInitialRate(receiverNode, senderNode string, grabSender bool) (float64, error) {
	var o *iperf3Output
	var s float64
	var err error

	senderHostNetwork, receiverHostNetwork, err := networksByConnectivity(sdnToSDN)
	if err != nil {
		return 0, err
	}

	receiverPod := createReceiverPod(receiverNode, receiverHostNetwork)
	receiverIP := receiverPod.Status.PodIP

	senderPod := createSenderPod(senderNode, senderHostNetwork, receiverIP)

	if grabSender {
		o, err = parseIperfOutput(senderPod)
	} else {
		o, err = parseIperfOutput(receiverPod)
	}

	if err != nil {
		return 0, err
	}

	for _, sum := range o.Intervals {
		s += sum.Sum.BitsPerSecond
	}

	s = s / float64(len(o.Intervals))

	if s == 0 {
		return 0, fmt.Errorf("rate sum of iperf3 intervals is 0, can't determine rate")
	}

	return s, nil
}

func qosOnNode(node *corev1.Node, cmdWithNICTemplate string) (string, error) {
	ovsPod, err := utilNodes.GetOvnkubePodByNode(client.Client, node)
	if err != nil {
		return "", err
	}

	nic, err := findOvsPhysicalNic(ovsPod)
	if err != nil {
		return "", err
	}

	buf, err := pods.ExecCommand(client.Client, *ovsPod, []string{"/bin/bash", "-c", fmt.Sprintf(cmdWithNICTemplate, nic)})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

/*
findOvsPhysicalNic finds the physical nic attached to the br-ex interface, output of ovs-vsctl command should be something like:
/usr/bin/ovs-vsctl list-ports br-ex

	eno1
	patch-br-ex_HOSTNAME-to-br-int

we want the nic that is not the patch port
*/
func findOvsPhysicalNic(ovsPod *corev1.Pod) (string, error) {
	buf, err := pods.ExecCommand(client.Client, *ovsPod, []string{"/bin/bash", "-c", "/usr/bin/ovs-vsctl list-ports br-ex | /usr/bin/grep -v patch"})
	if err != nil {
		return "", err
	}

	exp, err := regexp.Compile("\r\n")
	if err != nil {
		return "", err
	}

	return exp.ReplaceAllString(buf.String(), ""), nil
}

func validateIngressQosOnNodes(limit int64) error {
	filtered, err := getFilteredNodes()
	if err != nil {
		return err
	}

	for _, node := range filtered {
		qos, err := qosOnNode(&node, "/usr/bin/ovs-vsctl list interface %s")
		if err != nil {
			return err
		}

		if !strings.Contains(qos, fmt.Sprintf("ingress_policing_rate: %d", limit)) {
			return fmt.Errorf("ingress QoS not applied correctly on node %s for limit: %d", node.Name, limit)
		}
	}

	return nil
}

func validateEgressQosOnNodes(limit int64) error {
	var appliedLimit int64
	var matches []string
	filtered, err := getFilteredNodes()
	if err != nil {
		return err
	}

	findLimit, err := regexp.Compile("max-rate: ([0-9]*)")
	if err != nil {
		return err
	}

	for _, node := range filtered {
		qos, err := qosOnNode(&node, "/usr/bin/ovs-appctl qos/show %s")
		if err != nil {
			return err
		}
		if limit == 0 {
			if !strings.Contains(qos, "QoS not configured") {
				return fmt.Errorf("egress QoS not applied correctly on node %s for limit: %d", node.Name, limit)
			}
		} else {
			matches = findLimit.FindStringSubmatch(qos)
			if len(matches) < 2 {
				return fmt.Errorf("couldn't find rate from ovs-appctl qos/show on node %s", node.Name)
			}

			appliedLimit, err = strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				return err
			}

			if math.Abs(float64(limit-appliedLimit)) > 100 {
				return fmt.Errorf("egress QoS not applied correctly on node %s for limit: %d", node.Name, limit)
			}
		}
	}

	return nil
}

func newQoSTestParameters(connectivity string) (*qosTestParameters, error) {
	QoSTestParameters := &qosTestParameters{}

	err := paramInParamList(connectivity, connectivityParameters)
	if err != nil {
		return nil, err
	}
	QoSTestParameters.Connectivity = connectivity
	return QoSTestParameters, nil
}

func paramInParamList(param string, paramRange []string) error {
	for _, parameter := range paramRange {
		if param == parameter {
			return nil
		}
	}
	return fmt.Errorf("error: wrong parameter %v", param)
}

func networksByConnectivity(connectivity string) (senderHostNetwork, receiverHostNetwork bool, err error) {
	switch connectivity {
	case hostToHost:
		return true, true, nil
	case hostToSDN:
		return true, false, nil
	case sdnToSDN:
		return false, false, nil
	default:
		return false, false, fmt.Errorf("requested connectivity not in supported list")
	}
}

func getFilteredNodes() ([]corev1.Node, error) {
	nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: qosNodeSelector,
	})
	if err != nil {
		return nil, err
	}

	filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
	if err != nil {
		return nil, err
	}

	return filtered, nil
}
