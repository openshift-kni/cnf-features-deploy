package xt_u32

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	igntypes "github.com/coreos/ignition/config/v2_2/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/discovery"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/images"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	utilNodes "github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const hostnameLabel = "kubernetes.io/hostname"

const xt_u32LoadPath = "/etc/modules-load.d/xt_u32-load.conf"

var (
	xt_u32NodeSelector string
	hasNonCnfWorkers   bool
)

func init() {
	roleWorkerCNF := os.Getenv("ROLE_WORKER_CNF")
	if roleWorkerCNF != "" {
		xt_u32NodeSelector = fmt.Sprintf("node-role.kubernetes.io/%s=", roleWorkerCNF)
	}

	hasNonCnfWorkers = true
	if os.Getenv("XT_U32TEST_HAS_NON_CNF_WORKERS") == "false" {
		hasNonCnfWorkers = false
	}
}

var _ = Describe("xt_u32", func() {
	execute.BeforeAll(func() {
		err := namespaces.Create(namespaces.XTU32Test, client.Client)
		Expect(err).ToNot(HaveOccurred())

		err = namespaces.Clean(namespaces.XTU32Test, "xt-u32", client.Client)
		Expect(err).ToNot(HaveOccurred())

		if xt_u32NodeSelector == "" {
			xt_u32NodeSelector, err = findXT_U32NodeSelector()
			Expect(err).ToNot(HaveOccurred())
		}
	})

	Context("Negative - xt_u32 disabled", func() {
		var testNode string

		BeforeEach(func() {
			if !hasNonCnfWorkers {
				Skip("Skipping as no non-enabled nodes are available")
			}

			By("Choosing the test node")
			nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
				LabelSelector: "node-role.kubernetes.io/worker,!" + strings.Replace(xt_u32NodeSelector, "=", "", -1),
			})
			Expect(err).ToNot(HaveOccurred())

			filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
			Expect(err).ToNot(HaveOccurred())

			if discovery.Enabled() && len(filtered) == 0 {
				Skip("Did not find a node without xt_u32 module enabled")
			} else {
				Expect(len(filtered)).To(BeNumerically(">", 0))
			}
			testNode = filtered[0].ObjectMeta.Labels[hostnameLabel]
		})
		It("Should NOT create an iptable rule", func() {
			By("Create a pod")
			pod := xt_u32TestPod("xt-u32", testNode)
			xt_u32Pod, err := client.Client.Pods(namespaces.XTU32Test).Create(context.Background(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Check if the pod is running")
			Eventually(func() k8sv1.PodPhase {
				runningPod, err := client.Client.Pods(namespaces.XTU32Test).Get(context.Background(), xt_u32Pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return runningPod.Status.Phase
			}, 1*time.Minute, 1*time.Second).Should(Equal(k8sv1.PodRunning))

			By("Check if iptables command fails")
			cmd := []string{"iptables", "-t", "filter", "-A", "INPUT", "-i", "eth0", "-m",
				"u32", "--u32", "6 & 0xFF = 1 && 4 & 0x3FFF = 0 && 0 >> 22 & 0x3C @ 0 >> 24 = 8",
				"-j", "DROP"}
			out, err := pods.ExecCommand(client.Client, *xt_u32Pod, cmd)
			Expect(out.String()).Should(ContainSubstring("Couldn't load match `u32':No such file or directory"))
		})
	})

	Context("Validate the module is enabled and works", func() {
		var testNode string
		BeforeEach(func() {
			namespaces.Clean(namespaces.XTU32Test, "xt-u32", client.Client)
			By("Choosing the test node")
			nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
				LabelSelector: xt_u32NodeSelector,
			})
			Expect(err).ToNot(HaveOccurred())

			filtered, err := utilNodes.MatchingOptionalSelector(nodes.Items)
			Expect(err).ToNot(HaveOccurred())

			if discovery.Enabled() && len(filtered) == 0 {
				Skip("Did not find a node without xt_u32 module enabled")
			} else {
				Expect(len(filtered)).To(BeNumerically(">", 0))
			}
			testNode = filtered[0].ObjectMeta.Labels[hostnameLabel]

		})
		It("Should create an iptables rule inside a pod that has the module enabled", func() {
			By("Create a pod")
			pod := xt_u32TestPod("xt-u32", testNode)
			xt_u32Pod, err := client.Client.Pods(namespaces.XTU32Test).Create(context.Background(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Check if the pod is running")
			Eventually(func() k8sv1.PodPhase {
				res, err := client.Client.Pods(namespaces.XTU32Test).Get(context.Background(), xt_u32Pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return res.Status.Phase
			}, 3*time.Minute, 1*time.Second).Should(Equal(k8sv1.PodRunning))

			By("Check that xt_u32 module is loaded")
			cmd := []string{"lsmod"}
			out, err := pods.ExecCommand(client.Client, *xt_u32Pod, cmd)
			Expect(err).ToNot(HaveOccurred(), out.String())
			Expect(out.String()).Should(ContainSubstring("xt_u32"))

			By("Create an iptables rule within the pod that drops icmp ping msgs from any source - ip protocol = 1 (ICMP) && not fragmented && icmp type = 8 (echo request)")
			cmd = []string{"iptables", "-t", "filter", "-A", "INPUT", "-i", "eth0", "-m",
				"u32", "--u32", "6 & 0xFF = 1 && 4 & 0x3FFF = 0 && 0 >> 22 & 0x3C @ 0 >> 24 = 8",
				"-j", "DROP"}
			out, err = pods.ExecCommand(client.Client, *xt_u32Pod, cmd)
			Expect(err).ToNot(HaveOccurred(), out.String())
		})
	})
})

func findXT_U32NodeSelector() (string, error) {
	mcList, err := client.Client.MachineConfigs().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mc := range mcList.Items {
		enables, err := isXT_U32Enabled(mc)
		if err != nil {
			return "", err
		}
		if enables {
			mcLabel, found := mc.ObjectMeta.Labels["machineconfiguration.openshift.io/role"]
			if !found {
				continue
			}

			xt_u32NodeSelector, err := findXT_U32NodeSelectorByMCLabel(mcLabel)
			if err != nil {
				continue
			}

			return xt_u32NodeSelector, nil
		}
	}

	return "", errors.New("Cannot find a machine configuration that enables XT_U32")
}

func isXT_U32Enabled(mc mcfgv1.MachineConfig) (bool, error) {
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
		if !loadPathFound && file.Path == xt_u32LoadPath {
			loadPathFound = true
		}
	}
	return loadPathFound, nil
}

func findXT_U32NodeSelectorByMCLabel(mcLabel string) (string, error) {
	mcpList, err := client.Client.MachineConfigPools().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mcp := range mcpList.Items {
		for _, lsr := range mcp.Spec.MachineConfigSelector.MatchExpressions {
			for _, value := range lsr.Values {
				if value == mcLabel {
					for key, label := range mcp.Spec.NodeSelector.MatchLabels {
						newXT_U32NodeSelector := key + "=" + label
						return newXT_U32NodeSelector, nil
					}
				}
			}
		}
		for _, v := range mcp.Spec.MachineConfigSelector.MatchLabels {
			if v == mcLabel {
				for key, label := range mcp.Spec.NodeSelector.MatchLabels {
					newXT_U32NodeSelector := key + "=" + label
					return newXT_U32NodeSelector, nil
				}
			}
		}
	}

	return "", errors.New("Cannot find XT_U32NodeSelector")
}

func xt_u32TestPod(name, node string) *k8sv1.Pod {
	res := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"name": name,
			},
			Namespace: namespaces.XTU32Test,
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   images.For(images.TestUtils),
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{"sleep inf"},
					SecurityContext: &k8sv1.SecurityContext{
						Capabilities: &k8sv1.Capabilities{
							Add: []k8sv1.Capability{"NET_ADMIN"},
						},
					},
				},
			},
			NodeSelector: map[string]string{
				hostnameLabel: node,
			},
		},
	}

	return &res
}
