package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
)

const (
	mcpUpdateTimeout = 20
)

var _ = Describe("[rfe_id:28761] Updating parameters in performance profile", func() {
	var workerRTNodes []corev1.Node
	var profile, initialProfile *performancev1alpha1.PerformanceProfile
	var timeout time.Duration
	var err error

	chkKernel := []string{"uname", "-a"}
	chkCmdLine := []string{"cat", "/proc/cmdline"}
	chkKubeletConfig := []string{"cat", "/rootfs/etc/kubernetes/kubelet.conf"}

	BeforeEach(func() {
		workerRTNodes, err = nodes.GetByRole(testclient.Client, testutils.RoleWorkerRT)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
		// timeout should be based on the number of worker-rt nodes
		timeout = time.Duration(len(workerRTNodes) * mcpUpdateTimeout)

		profile, err = profiles.GetByNodeLabels(
			testclient.Client,
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Verify that all performance profile parameters can be updated", func() {
		var removedKernelArgs string

		hpSize := performancev1alpha1.HugePageSize("2M")
		isolated := performancev1alpha1.CPUSet("1-2")
		reserved := performancev1alpha1.CPUSet("0,3")
		policy := "best-effort"
		f := false

		// Modify profile and verify that MCO successfully updated the node
		testutils.BeforeAll(func() {
			By("Modifying profile")
			initialProfile = profile.DeepCopy()

			profile.Spec.HugePages = &performancev1alpha1.HugePages{
				DefaultHugePagesSize: &hpSize,
				Pages: []performancev1alpha1.HugePage{
					{
						Count: 5,
						Size:  hpSize,
					},
				},
			}
			profile.Spec.CPU = &performancev1alpha1.CPU{
				BalanceIsolated: &f,
				Reserved:        &reserved,
				Isolated:        &isolated,
			}
			profile.Spec.NUMA = &performancev1alpha1.NUMA{
				TopologyPolicy: &policy,
			}
			profile.Spec.RealTimeKernel = &performancev1alpha1.RealTimeKernel{
				Enabled: &f,
			}

			if profile.Spec.AdditionalKernelArgs == nil {
				By("AdditionalKernelArgs is empty. Checking only adding new arguments")
				profile.Spec.AdditionalKernelArgs = append(profile.Spec.AdditionalKernelArgs, "new-argument=test")
			} else {
				removedKernelArgs = profile.Spec.AdditionalKernelArgs[0]
				profile.Spec.AdditionalKernelArgs = append(profile.Spec.AdditionalKernelArgs[1:], "new-argument=test")
			}

			By("Verifying that mcp is ready for update")
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)

			By("Applying changes in performance profile and waiting until mcp will start updating")
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, timeout)

			By("Waiting when mcp finishes updates")
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)
		})

		table.DescribeTable("Verify that profile parameters were updated", func(cmd, parameter []string, shouldContain bool) {
			for _, node := range workerRTNodes {
				for _, param := range parameter {
					if shouldContain {
						Expect(execCommandOnWorker(cmd, &node)).To(ContainSubstring(param))
					} else {
						Expect(execCommandOnWorker(cmd, &node)).NotTo(ContainSubstring(param))
					}
				}
			}
		},
			table.Entry("[test_id:28024] verify that hugepages size and count updated", chkCmdLine, []string{"default_hugepagesz=2M", "hugepagesz=2M", "hugepages=5"}, true),
			table.Entry("[test_id:28070] verify that hugepages updated (NUMA node unspecified)", chkCmdLine, []string{"hugepagesz=2M"}, true),
			table.Entry("[test_id:28025] verify that cpu affinity mask was updated", chkCmdLine, []string{"tuned.non_isolcpus=00000009"}, true),
			table.Entry("[test_id:28071] verify that cpu balancer disabled", chkCmdLine, []string{"isolcpus=1-2"}, true),
			table.Entry("[test_id:28935] verify that reservedSystemCPUs was updated", chkKubeletConfig, []string{`"reservedSystemCPUs":"0,3"`}, true),
			table.Entry("[test_id:28760] verify that topologyManager was updated", chkKubeletConfig, []string{`"topologyManagerPolicy":"best-effort"`}, true),
			table.Entry("[test_id:27738] verify that realTimeKernerl was updated", chkKernel, []string{"PREEMPT RT"}, false),
		)

		It("[test_id:28612]Verify that Kernel arguments can me updated (added, removed) thru performance profile", func() {
			for _, node := range workerRTNodes {
				// Verifying that new argument was added
				Expect(execCommandOnWorker(chkCmdLine, &node)).To(ContainSubstring("new-argument=test"))

				// Verifying that one of old arguments was removed
				if removedKernelArgs != "" {
					Expect(execCommandOnWorker(chkCmdLine, &node)).NotTo(ContainSubstring(removedKernelArgs), fmt.Sprintf("%s should be removed from /proc/cmdline", removedKernelArgs))
				}
			}
		})

		It("Reverts back all profile configuration", func() {
			// BUG: CNF-385. Workaround - we have to remove hugepages first
			profile.Spec.HugePages = nil
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, timeout)
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)

			// return initial configuration
			spec, err := json.Marshal(initialProfile.Spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(testclient.Client.Patch(context.TODO(), profile,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, spec)),
				),
			)).ToNot(HaveOccurred())
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, timeout)
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)
		})
	})

	Context("Verifies that nodeSelector can be updated", func() {
		AfterEach(func() {
			// need to revert back nodeSelector, otherwise all other tests will be failed
			nodeSelector := fmt.Sprintf(`"%s/%s": ""`, testutils.LabelRole, testutils.RoleWorkerRT)
			Expect(testclient.Client.Patch(context.TODO(), profile,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/nodeSelector", "value": {%s} }]`, nodeSelector)),
				),
			)).ToNot(HaveOccurred())
		})

		It("[test_id:28440]Verifies that nodeSelector can be updated in performance profile", func() {
			var newWorkerNode *corev1.Node
			newRole := "worker-test"
			newLabel := fmt.Sprintf("%s/%s", testutils.LabelRole, newRole)
			newNodeSelector := map[string]string{newLabel: ""}

			By("Skipping test if cluster does not have another available worker node")

			nonRTWorkerNodes, err := nodes.GetNonRTWorkers()
			Expect(err).ToNot(HaveOccurred())

			if len(nonRTWorkerNodes) == 0 {
				Skip("Skipping test - performance worker nodes do not exist in the cluster")
			}

			newWorkerNode = &nonRTWorkerNodes[0]
			newWorkerNode.Labels[newLabel] = ""
			Expect(testclient.Client.Update(context.TODO(), newWorkerNode)).ToNot(HaveOccurred())

			By("Creating new MachineConfigPool")
			mcp := newMCP(newRole, newNodeSelector)
			err = testclient.Client.Create(context.TODO(), mcp)
			Expect(err).ToNot(HaveOccurred())

			By("Updating Node Selector performance profile")
			profile.Spec.NodeSelector = newNodeSelector
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
			waitForMcpCondition(newRole, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, mcpUpdateTimeout)

			By("Waiting when MCP finishes updates and verifying new node has updated configuration")
			waitForMcpCondition(newRole, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, mcpUpdateTimeout)

			_, err = nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, newWorkerNode, []string{"ls", "/rootfs/" + testutils.PerfRtKernelPrebootTuningScript})
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot find the file %s", testutils.PerfRtKernelPrebootTuningScript))
			Expect(execCommandOnWorker(chkKubeletConfig, newWorkerNode)).To(ContainSubstring("topologyManagerPolicy"))
			Expect(execCommandOnWorker(chkCmdLine, newWorkerNode)).To(ContainSubstring("tuned.non_isolcpus"))
		})
	})

	It("[test_id:27484]Verifies that node is reverted to plain worker when the extra labels are removed", func() {
		node := workerRTNodes[0]

		By("Deleting cnf labels from the node")
		for l := range profile.Spec.NodeSelector {
			delete(node.Labels, l)
		}
		Expect(testclient.Client.Update(context.TODO(), &node)).ToNot(HaveOccurred())
		waitForMcpCondition(testutils.RoleWorker, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, mcpUpdateTimeout)

		By("Waiting when MCP Worker complete updates and verifying that node reverted back configuration")
		waitForMcpCondition(testutils.RoleWorker, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, mcpUpdateTimeout)

		// Check if node is Ready
		for i := range node.Status.Conditions {
			if node.Status.Conditions[i].Type == corev1.NodeReady {
				Expect(node.Status.Conditions[i].Status).To(Equal(corev1.ConditionTrue))
			}
		}

		// check that the pre-boot-tuning script and service removed
		out, _ := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, &node, []string{"ls", "/rootfs/" + testutils.PerfRtKernelPrebootTuningScript})
		Expect(out).To(ContainSubstring("No such file or directory"))
		out, _ = nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, &node, []string{"ls", "/rootfs/" + "/etc/systemd/system/pre-boot-tuning.service"})
		Expect(out).To(ContainSubstring("No such file or directory"))

		// check that the configs reverted
		Expect(execCommandOnWorker(chkKernel, &node)).NotTo(ContainSubstring("PREEMPT RT"))
		Expect(execCommandOnWorker(chkCmdLine, &node)).NotTo(ContainSubstring("tuned.non_isolcpus"))
		Expect(execCommandOnWorker(chkKubeletConfig, &node)).NotTo(ContainSubstring("reservedSystemCPUs"))
	})
})

func waitForMcpCondition(mcpName string, conditionType machineconfigv1.MachineConfigPoolConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) {
	mcp := &machineconfigv1.MachineConfigPool{}
	key := types.NamespacedName{
		Name:      mcpName,
		Namespace: "",
	}
	Eventually(func() corev1.ConditionStatus {
		err := testclient.Client.Get(context.TODO(), key, mcp)
		Expect(err).ToNot(HaveOccurred())
		for i := range mcp.Status.Conditions {
			if mcp.Status.Conditions[i].Type == conditionType {
				return mcp.Status.Conditions[i].Status
			}
		}
		return corev1.ConditionUnknown
	}, timeout*time.Minute, 30*time.Second).Should(Equal(conditionStatus))
}

func newMCP(mcpName string, nodeSelector map[string]string) *machineconfigv1.MachineConfigPool {
	return &machineconfigv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpName,
			Namespace: "",
			Labels:    map[string]string{"machineconfiguration.openshift.io/role": mcpName},
		},
		Spec: machineconfigv1.MachineConfigPoolSpec{
			MachineConfigSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "machineconfiguration.openshift.io/role",
						Operator: "In",
						Values:   []string{"worker", mcpName},
					},
				},
			},
			NodeSelector: &metav1.LabelSelector{
				MatchLabels: nodeSelector,
			},
		},
	}
}
