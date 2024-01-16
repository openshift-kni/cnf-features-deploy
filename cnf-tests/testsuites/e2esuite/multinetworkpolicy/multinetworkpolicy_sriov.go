package multinetworkpolicy

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2" //lint:ignore ST1001 used for tests
	. "github.com/onsi/gomega"    //lint:ignore ST1001 used for tests

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	multinetpolicyv1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovNetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	np "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networkpolicy"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	SriovResource             = "sriovnicMultiNetworkpolicyResource"
	TestNetworkName           = "test-multi-networkpolicy-sriov-network"
	TestNetworkNamespace      = namespaces.Default
	TestNetworkNamespacedName = TestNetworkNamespace + "/" + TestNetworkName
)

// The following test model is common to all tests in the [multinetworkpolicy] test suite:
// Three namespaces nsX, nsY, nsZ, each of them with three pods podA, podB, podC
const (
	nsX string = utils.MultiNetworkPolicyNamespaceX
	nsY string = utils.MultiNetworkPolicyNamespaceY
	nsZ string = utils.MultiNetworkPolicyNamespaceZ
)

var (
	nsX_podA, nsX_podB, nsX_podC,
	nsY_podA, nsY_podB, nsY_podC,
	nsZ_podA, nsZ_podB, nsZ_podC *corev1.Pod
)

var (
	port5555  intstr.IntOrString = intstr.FromInt(5555)
	port6666  intstr.IntOrString = intstr.FromInt(6666)
	protoTCP  corev1.Protocol    = corev1.ProtocolTCP
	protoUDP  corev1.Protocol    = corev1.ProtocolUDP
	protoSCTP corev1.Protocol    = corev1.ProtocolSCTP
)

var _ = Describe("[multinetworkpolicy] MultiNetworkPolicy SR-IOV integration", func() {

	sriovclient := sriovtestclient.New("")
	sctpEnabled := false
	sriovEnabled := false

	BeforeEach(func() {
		// Tests can be triggered by setting `FEATURE=sriov` and can fail because the
		// feature is not enabled.
		crdPresent, err := np.IsMultiEnabled()
		Expect(err).ToNot(HaveOccurred())
		if !crdPresent {
			Fail("feature [multinetworkpolicy] not enabled on cluster. run FEATURES=multinetworkpolicy make feature-deploy to enable it.")
		}
	})

	execute.BeforeAll(func() {
		sriovEnabled = networks.IsSriovOperatorInstalled()
		if !sriovEnabled {
			return
		}

		namespaces.CleanPodsIn(client.Client, nsX, nsY, nsZ)

		Expect(namespaces.Create(nsX, client.Client)).ToNot(HaveOccurred())
		Expect(namespaces.Create(nsY, client.Client)).ToNot(HaveOccurred())
		Expect(namespaces.Create(nsZ, client.Client)).ToNot(HaveOccurred())

		// Wait for SR-IOV operator to be stable (each SriovNetworkNodeState.Status == Succeeded), as there can be a leftover of other test cases.
		networks.WaitStable(sriovclient)

		sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
		Expect(err).ToNot(HaveOccurred())
		Expect(sriovInfos).ToNot(BeNil())

		sriovNodes, err := nodes.MatchingOptionalSelectorByName(sriovInfos.Nodes)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(sriovNodes)).To(BeNumerically(">", 0))

		sriovAndSctpNodes, err := nodes.HavingSCTPEnabled(sriovNodes)
		Expect(err).ToNot(HaveOccurred())

		var testNodeNames []string
		if len(sriovAndSctpNodes) == 0 {
			sctpEnabled = false
			testNodeNames = sriovNodes
		} else {
			sctpEnabled = true
			testNodeNames = sriovAndSctpNodes
		}

		sriovDevice, err := sriovInfos.FindOneSriovDevice(testNodeNames[0])
		Expect(err).ToNot(HaveOccurred())

		networks.CleanSriov(sriovclient)

		_, err = sriovNetwork.CreateSriovPolicy(sriovclient, "test-policy-", namespaces.SRIOVOperator,
			sriovDevice.Name, testNodeNames[0], 10, SriovResource, "netdevice")
		Expect(err).ToNot(HaveOccurred())

		networks.WaitStable(sriovclient)

		ipam := `{"type": "host-local","ranges": [ [{"subnet": "3ffe:ffff:0:01ff::/64"}], [{"subnet": "2.2.2.0/24"}] ]}`

		err = sriovNetwork.CreateSriovNetwork(sriovclient, sriovDevice, TestNetworkName, TestNetworkNamespace,
			namespaces.SRIOVOperator, SriovResource, ipam)
		Expect(err).ToNot(HaveOccurred())

		networkAttachDef := netattdefv1.NetworkAttachmentDefinition{}
		client.WaitForObject(
			runtimeclient.ObjectKey{Name: TestNetworkName, Namespace: TestNetworkNamespace},
			&networkAttachDef)

		nsX_podA, nsX_podB, nsX_podC = createPodsInNamespace(nsX, sctpEnabled)
		nsY_podA, nsY_podB, nsY_podC = createPodsInNamespace(nsY, sctpEnabled)
		nsZ_podA, nsZ_podB, nsZ_podC = createPodsInNamespace(nsZ, sctpEnabled)
	})

	BeforeEach(func() {
		if discovery.Enabled() {
			Skip("Discovery is not supported.")
		}

		if !sriovEnabled {
			Skip("SR-IOV not supported by cluster.")
		}

		// Check if BeforeAll initialization has been completed and avoid nil reference errors
		if nsX_podA == nil {
			Fail("[multinetworkpolicy] test suite encountered initialization errors")
		}

		cleanMultiNetworkPoliciesFromNamespace(nsX)
		cleanMultiNetworkPoliciesFromNamespace(nsY)
		cleanMultiNetworkPoliciesFromNamespace(nsZ)

	})

	Context("Ingress", func() {
		It("DENY all traffic to a pod", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithEmptyIngressRules(),
				np.CreateInNamespace(nsX),
			)

			// Pod B and C are not affected by the policy
			eventually30s(nsX_podB).Should(Reach(nsX_podC, ViaTCP))
			eventually30s(nsX_podB).Should(Reach(nsX_podC, ViaUDP))

			// Pod A should not be reacheable by B and C
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, ViaTCP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, ViaUDP))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA, ViaTCP))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA, ViaUDP))
		})

		It("DENY all traffic to and within a namespace", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithEmptyIngressRules(),
				np.CreateInNamespace(nsX),
			)

			// Traffic within nsX is not allowed
			eventually30s(nsX_podA).ShouldNot(Reach(nsX_podB))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podC))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA))

			// Traffic to/from nsX is not allowed
			eventually30s(nsY_podA).ShouldNot(Reach(nsX_podA))
			eventually30s(nsZ_podA).ShouldNot(Reach(nsX_podA))

			// Traffic within other namespaces is allowed
			eventually30s(nsY_podA).Should(Reach(nsY_podB))
			eventually30s(nsZ_podA).Should(Reach(nsZ_podB))

			// Traffic between other namespaces is allowed
			eventually30s(nsY_podA).Should(Reach(nsZ_podA))
			eventually30s(nsZ_podB).Should(Reach(nsY_podC))
		})

		It("ALLOW traffic to a pod from pods selected by labels", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// The subject of test case
			eventually30s(nsX_podB).Should(Reach(nsX_podA))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA))

			// Traffic that should not be affected
			eventually30s(nsX_podB).Should(Reach(nsX_podC))
		})

		It("ALLOW traffic to a pod from all pods in a namespace", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": nsY,
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsY_podA).Should(Reach(nsX_podA))
			eventually30s(nsY_podB).Should(Reach(nsX_podA))
			eventually30s(nsY_podC).Should(Reach(nsX_podA))

			// Not allowed
			eventually30s(nsZ_podA).ShouldNot(Reach(nsX_podA))
			eventually30s(nsZ_podB).ShouldNot(Reach(nsX_podA))
			eventually30s(nsZ_podC).ShouldNot(Reach(nsX_podA))

			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA))
		})

		It("ALLOW traffic to a pod from using an OR combination of namespace and pod labels", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": nsY,
								},
							},
						},
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"pod": "b",
								},
							},
						},
					},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsY_podA).Should(Reach(nsX_podA))
			eventually30s(nsY_podB).Should(Reach(nsX_podA))
			eventually30s(nsY_podC).Should(Reach(nsX_podA))

			eventually30s(nsZ_podB).Should(Reach(nsX_podA))
			eventually30s(nsX_podB).Should(Reach(nsX_podA))

			// Not allowed
			eventually30s(nsZ_podA).ShouldNot(Reach(nsX_podA))
			eventually30s(nsZ_podC).ShouldNot(Reach(nsX_podA))

			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA))
		})

		It("ALLOW traffic to a pod using an AND combination of namespace and pod labels", func() {

			Skip("LabelSelectorwith multiple In values is not yet supported by multi-networkpolicy-iptables - https://github.com/k8snetworkplumbingwg/multi-networkpolicy/issues/4")
			//	E0511 14:37:07.698115       1 policyrules.go:238] pod selector: operator "In" without a single value cannot be converted into the old label selector format

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      "kubernetes.io/metadata.name",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{nsY, nsZ},
							}},
						},
						PodSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      "pod",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{"b", "c"},
							}},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsY_podB).Should(Reach(nsX_podA))
			eventually30s(nsY_podC).Should(Reach(nsX_podA))
			eventually30s(nsZ_podB).Should(Reach(nsX_podA))
			eventually30s(nsZ_podC).Should(Reach(nsX_podA))

			eventually30s(nsX_podA).Should(Reach(nsX_podA))

			// Not allowed
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA))

			eventually30s(nsY_podA).ShouldNot(Reach(nsX_podA))
			eventually30s(nsZ_podA).ShouldNot(Reach(nsX_podA))
		})
	})

	Context("Egress", func() {

		It("DENY all traffic from a pod", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithEmptyEgressRules(),
				np.CreateInNamespace(nsX),
			)

			// Pod B and C are not affected by the policy
			eventually30s(nsX_podB).Should(Reach(nsX_podC))

			// Pod A should not be reacheable by B and C
			eventually30s(nsX_podA).ShouldNot(Reach(nsX_podB))
			eventually30s(nsX_podA).ShouldNot(Reach(nsX_podC))
		})

		It("ALLOW traffic from a specific pod only to a specific pod", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithEgressRule(multinetpolicyv1.MultiNetworkPolicyEgressRule{
					To: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// The subject of test case
			eventually30s(nsX_podA).Should(Reach(nsX_podB))
			eventually30s(nsX_podA).ShouldNot(Reach(nsX_podC))

			// Traffic that should not be affected
			eventually30s(nsX_podB).Should(Reach(nsX_podC))
		})
	})

	Context("Stacked policies", func() {

		It("enforce multiple Ingress stacked policies with overlapping podSelector", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": nsY,
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed all connection from nsY
			eventually30s(nsY_podA).Should(Reach(nsX_podA))
			eventually30s(nsY_podB).Should(Reach(nsX_podA))
			eventually30s(nsY_podC).Should(Reach(nsX_podA))

			// Allowed all connection from podB
			eventually30s(nsZ_podB).Should(Reach(nsX_podA))
			eventually30s(nsX_podB).Should(Reach(nsX_podA))

			// Not allowed
			eventually30s(nsZ_podA).ShouldNot(Reach(nsX_podA))
			eventually30s(nsZ_podC).ShouldNot(Reach(nsX_podA))

			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA))
		})

		It("enforce multiple Ingress stacked policies with overlapping podSelector and different ports", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
					Ports: []multinetpolicyv1.MultiNetworkPolicyPort{{
						Port:     &port5555,
						Protocol: &protoTCP,
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "c",
							},
						},
					}},
					Ports: []multinetpolicyv1.MultiNetworkPolicyPort{{
						Port:     &port6666,
						Protocol: &protoTCP,
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsX_podB).Should(Reach(nsX_podA, OnPort(port5555)))
			eventually30s(nsY_podB).Should(Reach(nsX_podA, OnPort(port5555)))
			eventually30s(nsZ_podB).Should(Reach(nsX_podA, OnPort(port5555)))

			eventually30s(nsX_podC).Should(Reach(nsX_podA, OnPort(port6666)))
			eventually30s(nsY_podC).Should(Reach(nsX_podA, OnPort(port6666)))
			eventually30s(nsZ_podC).Should(Reach(nsX_podA, OnPort(port6666)))

			// Not allowed
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666)))
			eventually30s(nsY_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666)))
			eventually30s(nsZ_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666)))

			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA, OnPort(port5555)))
			eventually30s(nsY_podC).ShouldNot(Reach(nsX_podA, OnPort(port5555)))
			eventually30s(nsZ_podC).ShouldNot(Reach(nsX_podA, OnPort(port5555)))
		})
	})

	Context("Ports/Protocol", func() {
		It("Allow access only to a specific port/protocol TCP", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					Ports: []multinetpolicyv1.MultiNetworkPolicyPort{{
						Port:     &port5555, // Default protocol: TCP
						Protocol: &protoTCP,
					}},
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsX_podB).Should(Reach(nsX_podA, OnPort(port5555), ViaTCP))

			// Not allowed
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaTCP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaUDP))

			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaUDP))

			if sctpEnabled {
				eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaSCTP))
				eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaSCTP))
			}
		})

		It("Allow access only to a specific port/protocol UDP", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					Ports: []multinetpolicyv1.MultiNetworkPolicyPort{{
						Port:     &port6666,
						Protocol: &protoUDP,
					}},
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsX_podB).Should(Reach(nsX_podA, OnPort(port6666), ViaUDP))

			// Not allowed
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaTCP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaUDP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaTCP))

			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaUDP))

			if sctpEnabled {
				eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaSCTP))
				eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaSCTP))
			}
		})

		It("Allow access only to a specific port/protocol SCTP", func() {

			if !sctpEnabled {
				Skip("SCTP not enabled on test nodes")
			}

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					Ports: []multinetpolicyv1.MultiNetworkPolicyPort{{
						Port:     &port6666,
						Protocol: &protoSCTP,
					}},
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsX_podB).Should(Reach(nsX_podA, OnPort(port6666), ViaSCTP))

			// Not allowed
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaUDP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaTCP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaTCP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaUDP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaSCTP))
		})

		It("Allow access only to a specific port/protocol TCP+UDP", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					Ports: []multinetpolicyv1.MultiNetworkPolicyPort{{
						Port:     &port5555, // Default protocol: TCP
						Protocol: &protoTCP,
					}, {
						Port:     &port6666,
						Protocol: &protoUDP,
					}},
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsX_podB).Should(Reach(nsX_podA, OnPort(port5555), ViaTCP))
			eventually30s(nsX_podB).Should(Reach(nsX_podA, OnPort(port6666), ViaUDP))

			// Not allowed
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaTCP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaUDP))
		})

		It("Allow access only to a specific UDP port from any pod", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					Ports: []multinetpolicyv1.MultiNetworkPolicyPort{{
						Port:     &port6666,
						Protocol: &protoUDP,
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// Allowed
			eventually30s(nsX_podB).Should(Reach(nsX_podA, OnPort(port6666), ViaUDP))

			// Not allowed
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaTCP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port5555), ViaUDP))
			eventually30s(nsX_podB).ShouldNot(Reach(nsX_podA, OnPort(port6666), ViaTCP))
		})
	})

	Context("IPv6", func() {
		It("ALLOW traffic to a specific pod only from a specific pod", func() {

			np.MakeMultiNetworkPolicy(TestNetworkNamespacedName,
				np.WithPodSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "a",
					},
				}),
				np.WithIngressRule(multinetpolicyv1.MultiNetworkPolicyIngressRule{
					From: []multinetpolicyv1.MultiNetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "b",
							},
						},
					}},
				}),
				np.CreateInNamespace(nsX),
			)

			// IPv4
			eventually30s(nsX_podB).Should(Reach(nsX_podA, ViaIPv4))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA, ViaIPv4))
			eventually30s(nsX_podB).Should(Reach(nsX_podC, ViaIPv4))

			// IPv6
			eventually30s(nsX_podB).Should(Reach(nsX_podA, ViaIPv6))
			eventually30s(nsX_podC).ShouldNot(Reach(nsX_podA, ViaIPv6))
			eventually30s(nsX_podB).Should(Reach(nsX_podC, ViaIPv6))
		})
	})

})

func cleanMultiNetworkPoliciesFromNamespace(namespace string) {
	err := client.Client.MultiNetworkPolicies(namespace).
		DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() int {
		ret, err := client.Client.MultiNetworkPolicies(namespace).
			List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		return len(ret.Items)
	}, 30*time.Second, 1*time.Second).Should(BeZero())
}

func createPodsInNamespace(namespace string, addSCTPServer bool) (*corev1.Pod, *corev1.Pod, *corev1.Pod) {
	var err error

	podA := pods.DefinePod(namespace)
	pods.RedefineWithLabel(podA, "pod", "a")
	pods.RedefinePodWithNetwork(podA, TestNetworkNamespacedName)
	addNetcatContainers(podA, addSCTPServer)
	AddIPTableDebugContainer(podA)
	podA.ObjectMeta.GenerateName = "testpod-a-"
	podA, err = pods.CreateAndStart(podA)
	Expect(err).ToNot(HaveOccurred())

	podB := pods.DefinePod(namespace)
	pods.RedefineWithLabel(podB, "pod", "b")
	pods.RedefinePodWithNetwork(podB, TestNetworkNamespacedName)
	addNetcatContainers(podB, addSCTPServer)
	AddIPTableDebugContainer(podB)
	podB.ObjectMeta.GenerateName = "testpod-b-"
	podB, err = pods.CreateAndStart(podB)
	Expect(err).ToNot(HaveOccurred())

	podC := pods.DefinePod(namespace)
	pods.RedefineWithLabel(podC, "pod", "c")
	pods.RedefinePodWithNetwork(podC, TestNetworkNamespacedName)
	addNetcatContainers(podC, addSCTPServer)
	AddIPTableDebugContainer(podC)
	podC.ObjectMeta.GenerateName = "testpod-c-"
	podC, err = pods.CreateAndStart(podC)
	Expect(err).ToNot(HaveOccurred())

	return podA, podB, podC
}

func addNetcatContainers(pod *corev1.Pod, addSCTPServer bool) {

	AddTCPNetcatServerToPod(pod, port5555)
	AddUDPNetcatServerToPod(pod, port5555)

	AddTCPNetcatServerToPod(pod, port6666)
	AddUDPNetcatServerToPod(pod, port6666)

	if addSCTPServer {
		AddSCTPNetcatServerToPod(pod, port5555)
		AddSCTPNetcatServerToPod(pod, port6666)
	}
}

func eventually30s(actual interface{}) AsyncAssertion {
	return Eventually(actual, "30s", "1s")
}
