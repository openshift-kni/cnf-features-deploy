package sctp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	igntypes "github.com/coreos/ignition/config/v2_2/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcfgScheme "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/scheme"
	k8sv1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/discovery"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/images"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	utilNodes "github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"

	"k8s.io/utils/pointer"
)

const mcYaml = "../sctp/sctp_module_mc.yaml"
const hostnameLabel = "kubernetes.io/hostname"
const TestNamespace = "sctptest"
const defaultNamespace = "default"

const (
	sctpBlacklistPath = "/etc/modprobe.d/sctp-blacklist.conf"
	sctpLoadPath      = "/etc/modules-load.d/sctp-load.conf"
)

var (
	sctpNodeSelector string
	hasNonCnfWorkers bool
)

type nodesInfo struct {
	clientNode  string
	serverNode  string
	nodeAddress string
}

func init() {
	roleWorkerCNF := os.Getenv("ROLE_WORKER_CNF")
	if roleWorkerCNF != "" {
		sctpNodeSelector = fmt.Sprintf("node-role.kubernetes.io/%s=", roleWorkerCNF)
	}

	hasNonCnfWorkers = true
	if os.Getenv("SCTPTEST_HAS_NON_CNF_WORKERS") == "false" {
		hasNonCnfWorkers = false
	}
}

var _ = Describe("sctp", func() {
	execute.BeforeAll(func() {
		err := namespaces.Create(TestNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())

		// This is because we are seeing intermittent failures for
		// error looking up service account sctptest/default: serviceaccount \"default\" not found"
		// Making sure the account is there, and scream if it's not being created after 5 minutes
		Eventually(func() error {
			_, err := client.Client.ServiceAccounts(TestNamespace).Get(context.Background(), "default", metav1.GetOptions{})
			return err
		}, 5*time.Minute, 5*time.Second).Should(Not(HaveOccurred()))

		err = namespaces.Clean(TestNamespace, "testsctp-", client.Client)
		Expect(err).ToNot(HaveOccurred())

		if sctpNodeSelector == "" {
			sctpNodeSelector, err = findSCTPNodeSelector()
			Expect(err).ToNot(HaveOccurred())
		}

	})

	var _ = Describe("Negative - Sctp disabled", func() {
		var serverNode string
		execute.BeforeAll(func() {
			By("Validate that SCTP present of cluster.")
			checkForSctpReady(client.Client)
		})
		BeforeEach(func() {
			if !hasNonCnfWorkers {
				Skip("Skipping as no non-enabled nodes are available")
			}

			namespaces.Clean(TestNamespace, "testsctp-", client.Client)
			By("Choosing the nodes for the server and the client")
			nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
				LabelSelector: "node-role.kubernetes.io/worker,!" + strings.Replace(sctpNodeSelector, "=", "", -1),
			})
			Expect(err).ToNot(HaveOccurred())

			filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
			Expect(err).ToNot(HaveOccurred())

			if discovery.Enabled() && len(filtered) == 0 {
				Skip("Did not find a node without sctp module enabled")
			} else {
				Expect(len(filtered)).To(BeNumerically(">", 0))
			}
			serverNode = filtered[0].ObjectMeta.Labels[hostnameLabel]

			createSctpService(client.Client, TestNamespace)
		})
		Context("Client Server Connection", func() {
			// OCP-26995
			It("Should NOT start a server pod", func() {
				By("Starting the server")
				serverArgs := []string{"-ip", "0.0.0.0", "-port", "30101", "-server"}
				pod := sctpTestPod("testsctp-server", serverNode, "sctpserver", TestNamespace, serverArgs)
				pod.Spec.Containers[0].Ports = []k8sv1.ContainerPort{
					k8sv1.ContainerPort{
						Name:          "sctpport",
						Protocol:      k8sv1.ProtocolSCTP,
						ContainerPort: 30101,
					},
				}
				serverPod, err := client.Client.Pods(TestNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				By("Checking the server pod fails")
				Eventually(func() k8sv1.PodPhase {
					runningPod, err := client.Client.Pods(TestNamespace).Get(context.Background(), serverPod.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return runningPod.Status.Phase
				}, 1*time.Minute, 1*time.Second).Should(Equal(k8sv1.PodFailed))
			})
		})
	})
	var _ = Describe("Test Connectivity", func() {
		var nodes nodesInfo

		execute.BeforeAll(func() {
			checkForSctpReady(client.Client)
			nodes = selectSctpNodes(sctpNodeSelector)
		})

		Context("Connectivity between client and server", func() {
			BeforeEach(func() {
				namespaces.Clean(defaultNamespace, "testsctp-", client.Client)
				namespaces.Clean(TestNamespace, "testsctp-", client.Client)
			})

			// OCP-26759
			It("Kernel Module is loaded", func() {
				checkForSctpReady(client.Client)
			})

			// OCP-26760
			DescribeTable("Connectivity Test",
				func(namespace string, setup func() error, shouldSucceed bool) {
					By("Starting the server")
					serverPod := startServerPod(nodes.serverNode, namespace)
					By("Setting up")
					err := setup()
					Expect(err).ToNot(HaveOccurred())
					testClientServerConnection(client.Client, namespace, serverPod.Status.PodIP,
						30101, nodes.clientNode, serverPod.Name, shouldSucceed)
				},
				Entry("Custom namespace", TestNamespace, func() error { return nil }, true),
				Entry("Default namespace", defaultNamespace, func() error { return nil }, true),
				Entry("Custom namespace with policy", TestNamespace, func() error {
					return setupIngress(TestNamespace, "sctpclient", "sctpserver", 30101)
				}, true),
				Entry("Custom namespace with policy no port", TestNamespace, func() error {
					// setting an ingress rule with 30103 so 30101 won't work
					return setupIngress(TestNamespace, "sctpclient", "sctpserver", 30103)
				}, false),
				Entry("Default namespace with policy", defaultNamespace, func() error {
					return setupIngress(defaultNamespace, "sctpclient", "sctpserver", 30101)
				}, true),
				Entry("Default namespace with policy no port", defaultNamespace, func() error {
					// setting an ingress rule with 30103 so 30101 won't work
					return setupIngress(defaultNamespace, "sctpclient", "sctpserver", 30103)
				}, false), // TODO Add egress tests too.
			)
			// OCP-26763
			It("Should connect a client pod to a server pod. Feature LatencySensitive Active", func() {
				fg, err := client.Client.FeatureGates().Get(context.Background(), "cluster", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if fg.Spec.FeatureSet == "LatencySensitive" {
					By("Starting the server")
					serverPod := startServerPod(nodes.serverNode, TestNamespace)
					By("Testing the connection")
					testClientServerConnection(client.Client, TestNamespace, serverPod.Status.PodIP,
						30101, nodes.clientNode, serverPod.Name, true)
				} else {
					Skip("Feature LatencySensitive is not ACTIVE")
				}
			})
			// OCP-26761
			DescribeTable("connect a client pod to a server pod via Service ClusterIP",
				func(namespace string, setup func() error, shouldSucceed bool) {
					By("Setting up")
					err := setup()
					Expect(err).ToNot(HaveOccurred())
					By("Starting the server")
					serverPod := startServerPod(nodes.serverNode, namespace)
					service := createSctpService(client.Client, namespace)
					testClientServerConnection(client.Client, namespace, service.Spec.ClusterIP,
						service.Spec.Ports[0].Port, nodes.clientNode, serverPod.Name, shouldSucceed)
				},
				Entry("Custom namespace", TestNamespace, func() error { return nil }, true),
				Entry("Default namespace", defaultNamespace, func() error { return nil }, true),
			)
			// OCP-26762
			DescribeTable("connect a client pod to a server pod via Service Node Port",
				func(namespace string, setup func() error, shouldSucceed bool) {
					By("Setting up")
					err := setup()
					Expect(err).ToNot(HaveOccurred())
					By("Starting the server")
					serverPod := startServerPod(nodes.serverNode, namespace)
					service := createSctpService(client.Client, namespace)
					testClientServerConnection(client.Client, namespace, nodes.nodeAddress,
						service.Spec.Ports[0].Port, nodes.clientNode, serverPod.Name, shouldSucceed)
				},
				Entry("Custom namespace", TestNamespace, func() error { return nil }, true),
				Entry("Default namespace", defaultNamespace, func() error { return nil }, true),
			)
		})
	})

})

func loadMC() *mcfgv1.MachineConfig {
	decode := mcfgScheme.Codecs.UniversalDeserializer().Decode
	mcoyaml, err := ioutil.ReadFile(mcYaml)
	Expect(err).ToNot(HaveOccurred())

	obj, _, err := decode([]byte(mcoyaml), nil, nil)
	Expect(err).ToNot(HaveOccurred())
	mc, ok := obj.(*mcfgv1.MachineConfig)
	Expect(ok).To(Equal(true))
	return mc
}

func getSCTPNodes(selector string) []k8sv1.Node {
	nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: selector,
	})
	Expect(err).ToNot(HaveOccurred())
	filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
	Expect(err).ToNot(HaveOccurred())

	Expect(len(filtered)).To(BeNumerically(">", 0))
	return filtered
}

func nodesToInfo(client, server k8sv1.Node) nodesInfo {
	clientHost := client.ObjectMeta.Labels[hostnameLabel]
	serverHost := server.ObjectMeta.Labels[hostnameLabel]
	clientNodeIP := client.Status.Addresses[0].Address
	return nodesInfo{clientNode: clientHost, serverNode: serverHost, nodeAddress: clientNodeIP}
}

func selectSctpNodes(selector string) nodesInfo {
	By("Choosing the nodes for the server and the client")

	filtered := getSCTPNodes(selector)
	if len(filtered) > 1 {
		return nodesToInfo(filtered[0], filtered[1])
	}
	return nodesToInfo(filtered[0], filtered[0])
}

func startServerPod(node, namespace string, networks ...string) *k8sv1.Pod {
	serverArgs := []string{"-ip", "0.0.0.0", "-port", "30101", "-server"}
	pod := sctpTestPod("testsctp-server", node, "sctpserver", namespace, serverArgs)
	pod.Spec.Containers[0].Ports = []k8sv1.ContainerPort{
		k8sv1.ContainerPort{
			Name:          "sctpport",
			Protocol:      k8sv1.ProtocolSCTP,
			ContainerPort: 30101,
		},
	}

	if networks != nil {
		pod.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": strings.Join(networks, ",")}
	}

	serverPod, err := client.Client.Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	var res *k8sv1.Pod
	By("Fetching the server's ip address")
	Eventually(func() k8sv1.PodPhase {
		res, err = client.Client.Pods(namespace).Get(context.Background(), serverPod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return res.Status.Phase
	}, 3*time.Minute, 1*time.Second).Should(Equal(k8sv1.PodRunning))
	return res
}

func checkForSctpReady(cs *client.ClientSet) {
	nodes, err := cs.Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: sctpNodeSelector,
	})
	Expect(err).ToNot(HaveOccurred())

	filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(filtered)).To(BeNumerically(">", 0))

	args := []string{`set -x; x="$(checksctp 2>&1)"; echo "$x" ; if [ "$x" = "SCTP supported" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`}
	for _, n := range filtered {
		job := jobForNode("testsctp-check", n.ObjectMeta.Labels[hostnameLabel], "checksctp", []string{"/bin/bash", "-c"}, args)
		cs.Pods(TestNamespace).Create(context.Background(), job, metav1.CreateOptions{})
	}

	Eventually(func() bool {
		pods, err := cs.Pods(TestNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=checksctp"})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		for _, p := range pods.Items {
			if p.Status.Phase != k8sv1.PodSucceeded {
				return false
			}
		}
		return true
	}, 10*time.Minute, 10*time.Second).Should(Equal(true))
}

func testClientServerConnection(cs *client.ClientSet, namespace string, destIP string, port int32, clientNode string, serverPodName string, shouldSucceed bool, networks ...string) {
	By("Connecting a client to the server")
	clientArgs := []string{"-ip", destIP, "-port",
		fmt.Sprint(port), "-lport", "30102"}
	clientPod := sctpTestPod("testsctp-client", clientNode, "sctpclient", namespace, clientArgs)
	if networks != nil {
		clientPod.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": strings.Join(networks, ",")}
	}

	_, err := cs.Pods(namespace).Create(context.Background(), clientPod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	if !shouldSucceed {
		Consistently(func() k8sv1.PodPhase {
			pod, err := cs.Pods(namespace).Get(context.Background(), serverPodName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pod.Status.Phase
		}, 30*time.Second, 1*time.Second).Should(Equal(k8sv1.PodRunning))
		return
	}

	Eventually(func() k8sv1.PodPhase {
		pod, err := cs.Pods(namespace).Get(context.Background(), serverPodName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}, 1*time.Minute, 1*time.Second).Should(Equal(k8sv1.PodSucceeded))
}

func createSctpService(cs *client.ClientSet, namespace string) *k8sv1.Service {
	service := k8sv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testsctp-service",
			Namespace: namespace,
		},
		Spec: k8sv1.ServiceSpec{
			Selector: map[string]string{
				"app": "sctpserver",
			},
			Ports: []k8sv1.ServicePort{
				k8sv1.ServicePort{
					Protocol: k8sv1.ProtocolSCTP,
					Port:     30101,
					NodePort: 30101,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 30101,
					},
				},
			},
			Type: "NodePort",
		},
	}
	activeService, err := cs.Services(namespace).Create(context.Background(), &service, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return activeService
}

func sctpTestPod(name, node, app, namespace string, args []string) *k8sv1.Pod {
	res := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Labels: map[string]string{
				"app": app,
			},
			Namespace: namespace,
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   images.For(images.TestUtils),
					Command: []string{"/usr/bin/sctptest"},
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

func jobForNode(name, node, app string, cmd []string, args []string) *k8sv1.Pod {
	job := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Labels: map[string]string{
				"app": app,
			},
			Namespace: TestNamespace,
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   images.For(images.TestUtils),
					Command: cmd,
					Args:    args,
				},
			},
			NodeSelector: map[string]string{
				hostnameLabel: node,
			},
		},
	}

	return &job
}

func setupIngress(namespace, fromPod, toPod string, port int32) error {
	policy := &networkv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testsctp-block-ingress",
			Namespace: namespace,
		},
		Spec: networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": toPod,
				},
			},
			PolicyTypes: []networkv1.PolicyType{"Ingress"},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				networkv1.NetworkPolicyIngressRule{
					From: []networkv1.NetworkPolicyPeer{
						networkv1.NetworkPolicyPeer{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": fromPod,
								},
							},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						networkv1.NetworkPolicyPort{
							Protocol: (*k8sv1.Protocol)(pointer.StringPtr(string(k8sv1.ProtocolSCTP))),
							Port: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: port,
							},
						},
					},
				},
			},
		},
	}
	_, err := client.Client.NetworkPolicies(namespace).Create(context.Background(), policy, metav1.CreateOptions{})
	return err
}

func setupEgress(namespace, fromPod, toPod string, port int32) error {
	policy := &networkv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testsctp-block-egress",
			Namespace: namespace,
		},
		Spec: networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fromPod,
				},
			},
			PolicyTypes: []networkv1.PolicyType{"Egress"},
			Egress: []networkv1.NetworkPolicyEgressRule{
				networkv1.NetworkPolicyEgressRule{
					To: []networkv1.NetworkPolicyPeer{
						networkv1.NetworkPolicyPeer{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": toPod,
								},
							},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						networkv1.NetworkPolicyPort{
							Protocol: (*k8sv1.Protocol)(pointer.StringPtr(string(k8sv1.ProtocolSCTP))),
							Port: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: port,
							},
						},
					},
				},
			},
		},
	}
	_, err := client.Client.NetworkPolicies(namespace).Create(context.Background(), policy, metav1.CreateOptions{})
	return err
}

func findSCTPNodeSelector() (string, error) {
	mcList, err := client.Client.MachineConfigs().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mc := range mcList.Items {
		enables, err := enablesSCTP(mc)
		if err != nil {
			return "", err
		}
		if enables {
			mcLabel, found := mc.ObjectMeta.Labels["machineconfiguration.openshift.io/role"]
			if !found {
				continue
			}

			sctpNodeSelector, err := findSCTPNodeSelectorByMCLabel(mcLabel)
			if err != nil {
				continue
			}

			return sctpNodeSelector, nil
		}
	}

	return "", errors.New("Cannot find a machine configuration that enables SCTP")
}

func enablesSCTP(mc mcfgv1.MachineConfig) (bool, error) {
	blacklistPathFound := false
	loadPathFound := false

	ignitionConfig := igntypes.Config{}
	if mc.Spec.Config.Raw == nil {
		return false, nil
	}

	err := json.Unmarshal(mc.Spec.Config.Raw, &ignitionConfig)
	if err != nil {
		return false, fmt.Errorf("Failed to unmarshal ignition config %v", err)
	}

	for _, file := range ignitionConfig.Storage.Files {
		if !blacklistPathFound && file.Path == sctpBlacklistPath {
			blacklistPathFound = true
		}
		if !loadPathFound && file.Path == sctpLoadPath {
			loadPathFound = true
		}
	}

	return (blacklistPathFound && loadPathFound), nil
}

func findSCTPNodeSelectorByMCLabel(mcLabel string) (string, error) {
	mcpList, err := client.Client.MachineConfigPools().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mcp := range mcpList.Items {
		for _, lsr := range mcp.Spec.MachineConfigSelector.MatchExpressions {
			for _, value := range lsr.Values {
				if value == mcLabel {
					for key, label := range mcp.Spec.NodeSelector.MatchLabels {
						newSCTPNodeSelector := key + "=" + label
						return newSCTPNodeSelector, nil
					}
				}
			}
		}
		for _, v := range mcp.Spec.MachineConfigSelector.MatchLabels {
			if v == mcLabel {
				for key, label := range mcp.Spec.NodeSelector.MatchLabels {
					newSCTPNodeSelector := key + "=" + label
					return newSCTPNodeSelector, nil
				}
			}
		}
	}

	return "", errors.New("Cannot find SCTPNodeSelector")
}
