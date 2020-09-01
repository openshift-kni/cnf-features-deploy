package __performance

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/node/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/openshift-kni/performance-addon-operators/functests/utils"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/discovery"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"
)

const (
	testTimeout      = 480
	testPollInterval = 2
)

var _ = Describe("[rfe_id:27368][performance]", func() {

	var workerRTNodes []corev1.Node
	var profile *performancev1.PerformanceProfile

	BeforeEach(func() {
		if discovery.Enabled() && utils.ProfileNotFound {
			Skip("Discovery mode enabled, performance profile not found")
		}

		var err error
		workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for node with role %q: %v", testutils.RoleWorkerCNF, err))
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for the optional selector: %v", err))
		Expect(workerRTNodes).ToNot(BeEmpty(), fmt.Sprintf("no nodes with role %q found", testutils.RoleWorkerCNF))
		profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Tuned CRs generated from profile", func() {
		It("[test_id:31748] Should have the expected name for tuned from the profile owner object", func() {
			tunedExpectedName := components.GetComponentName(testutils.PerformanceProfileName, components.ProfileNamePerformance)
			tunedList := &tunedv1.TunedList{}
			key := types.NamespacedName{
				Name:      components.GetComponentName(testutils.PerformanceProfileName, components.ProfileNamePerformance),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			tuned := &tunedv1.Tuned{}
			err := testclient.Client.Get(context.TODO(), key, tuned)
			Expect(err).ToNot(HaveOccurred(), "cannot find the Cluster Node Tuning Operator object "+tuned.Name)

			Eventually(func() bool {
				err := testclient.Client.List(context.TODO(), tunedList)
				Expect(err).NotTo(HaveOccurred())
				for t := range tunedList.Items {
					tunedItem := tunedList.Items[t]
					ownerReferences := tunedItem.ObjectMeta.OwnerReferences
					for o := range ownerReferences {
						if ownerReferences[o].Name == profile.Name && tunedItem.Name != tunedExpectedName {
							return false
						}
					}
				}
				return true
			}, 120*time.Second, testPollInterval*time.Second).Should(BeTrue(),
				"tuned CR name owned by a performance profile CR should only be "+tunedExpectedName)
		})
	})

	Context("Pre boot tuning adjusted by tuned ", func() {

		It("[test_id:31198] Should set CPU affinity kernel argument", func() {
			for _, node := range workerRTNodes {
				cmdline, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/proc/cmdline"})
				Expect(err).ToNot(HaveOccurred())
				// since systemd.cpu_affinity is calculated on node level using tuned we can check only the key in this context.
				Expect(string(cmdline)).To(ContainSubstring("systemd.cpu_affinity="))
			}
		})

		It("Should set CPU isolcpu's kernel argument managed_irq flag", func() {
			for _, node := range workerRTNodes {
				cmdline, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/proc/cmdline"})
				Expect(err).ToNot(HaveOccurred())
				if profile.Spec.CPU.BalanceIsolated != nil && *profile.Spec.CPU.BalanceIsolated == false {
					Expect(string(cmdline)).To(ContainSubstring("isolcpus=domain,managed_irq,"))
				} else {
					Expect(string(cmdline)).To(ContainSubstring("isolcpus=managed_irq,"))
				}
			}
		})

		It("[test_id:27081][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should set workqueue CPU mask", func() {
			for _, node := range workerRTNodes {
				By("Getting tuned.non_isolcpus kernel argument")
				cmdline, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/proc/cmdline"})
				re := regexp.MustCompile(`tuned.non_isolcpus=\S+`)
				nonIsolcpusFullArgument := re.FindString(string(cmdline))
				Expect(nonIsolcpusFullArgument).To(ContainSubstring("tuned.non_isolcpus="))
				nonIsolcpusMask := strings.Split(string(nonIsolcpusFullArgument), "=")[1]
				nonIsolcpusMaskNoDelimiters := strings.Replace(nonIsolcpusMask, ",", "", -1)
				Expect(err).ToNot(HaveOccurred())
				By("executing the command \"cat /sys/devices/virtual/workqueue/cpumask\"")
				workqueueMask, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/sys/devices/virtual/workqueue/cpumask"})
				Expect(err).ToNot(HaveOccurred())
				workqueueMaskTrimmed := strings.TrimSpace(string(workqueueMask))
				workqueueMaskTrimmedNoDelimiters := strings.Replace(workqueueMaskTrimmed, ",", "", -1)
				Expect(strings.TrimLeft(nonIsolcpusMaskNoDelimiters, "0")).Should(Equal(strings.TrimLeft(workqueueMaskTrimmedNoDelimiters, "0")), "workqueueMask is not set to "+workqueueMaskTrimmed)
				By("executing the command \"cat /sys/bus/workqueue/devices/writeback/cpumask\"")
				workqueueWritebackMask, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/sys/bus/workqueue/devices/writeback/cpumask"})
				Expect(err).ToNot(HaveOccurred())
				workqueueWritebackMaskTrimmed := strings.TrimSpace(string(workqueueWritebackMask))
				workqueueWritebackMaskTrimmedNoDelimiters := strings.Replace(workqueueWritebackMaskTrimmed, ",", "", -1)
				Expect(strings.TrimLeft(nonIsolcpusMaskNoDelimiters, "0")).Should(Equal(strings.TrimLeft(workqueueWritebackMaskTrimmedNoDelimiters, "0")), "workqueueMask is not set to "+workqueueWritebackMaskTrimmed)
			}
		})

		It("[test_id:32375][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] initramfs should not have injected configuration", func() {
			Skip("Skipping test until BZ#1858347 is resolved")
			for _, node := range workerRTNodes {
				rhcosId, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"awk", "-F", "/", "{printf $3}", "/rootfs/proc/cmdline"})
				Expect(err).ToNot(HaveOccurred())
				initramfsImagesPath, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"find", filepath.Join("/rootfs/boot/ostree", string(rhcosId)), "-name", "*.img"})
				Expect(err).ToNot(HaveOccurred())
				initrd, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"lsinitrd", strings.TrimSpace(string(initramfsImagesPath))})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(initrd)).ShouldNot(ContainSubstring("'/etc/systemd/system.conf /etc/systemd/system.conf.d/setAffinity.conf'"))
			}
		})
	})

	Context("Additional kernel arguments added from perfomance profile", func() {
		It("[test_id:28611][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should set additional kernel arguments on the machine", func() {
			if profile.Spec.AdditionalKernelArgs != nil {
				for _, node := range workerRTNodes {
					cmdline, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/proc/cmdline"})
					Expect(err).ToNot(HaveOccurred())
					for _, arg := range profile.Spec.AdditionalKernelArgs {
						Expect(string(cmdline)).To(ContainSubstring(arg))
					}
				}
			}
		})
	})

	Context("Tuned kernel parameters", func() {
		It("[test_id:28466][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should contain configuration injected through openshift-node-performance profile", func() {
			sysctlMap := map[string]string{
				"kernel.hung_task_timeout_secs": "600",
				"kernel.nmi_watchdog":           "0",
				"kernel.sched_rt_runtime_us":    "-1",
				"vm.stat_interval":              "10",
				"kernel.timer_migration":        "0",
			}

			key := types.NamespacedName{
				Name:      components.GetComponentName(testutils.PerformanceProfileName, components.ProfileNamePerformance),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			tuned := &tunedv1.Tuned{}
			err := testclient.Client.Get(context.TODO(), key, tuned)
			Expect(err).ToNot(HaveOccurred(), "cannot find the Cluster Node Tuning Operator object "+key.String())
			validatTunedActiveProfile(workerRTNodes)
			execSysctlOnWorkers(workerRTNodes, sysctlMap)
		})
	})

	Context("Network latency parameters adjusted by the Node Tuning Operator", func() {
		It("[test_id:28467][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should contain configuration injected through the openshift-node-performance profile", func() {
			sysctlMap := map[string]string{
				"net.ipv4.tcp_fastopen":           "3",
				"kernel.sched_min_granularity_ns": "10000000",
				"vm.dirty_ratio":                  "10",
				"vm.dirty_background_ratio":       "3",
				"vm.swappiness":                   "10",
				"kernel.sched_migration_cost_ns":  "5000000",
			}
			key := types.NamespacedName{
				Name:      components.GetComponentName(testutils.PerformanceProfileName, components.ProfileNamePerformance),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			tuned := &tunedv1.Tuned{}
			err := testclient.Client.Get(context.TODO(), key, tuned)
			Expect(err).ToNot(HaveOccurred(), "cannot find the Cluster Node Tuning Operator object "+components.ProfileNamePerformance)
			validatTunedActiveProfile(workerRTNodes)
			execSysctlOnWorkers(workerRTNodes, sysctlMap)
		})
	})

	Context("Create second performance profiles on a cluster", func() {
		It("[test_id:32364] Verifies that cluster can have multiple profiles", func() {
			newRole := "worker-new"
			newLabel := fmt.Sprintf("%s/%s", testutils.LabelRole, newRole)

			reserved := performancev1.CPUSet("0")
			isolated := performancev1.CPUSet("1-3")

			secondProfile := &performancev1.PerformanceProfile{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PerformanceProfile",
					APIVersion: performancev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "profile2",
				},
				Spec: performancev1.PerformanceProfileSpec{
					CPU: &performancev1.CPU{
						Reserved: &reserved,
						Isolated: &isolated,
					},
					NodeSelector: map[string]string{newLabel: ""},
					RealTimeKernel: &performancev1.RealTimeKernel{
						Enabled: pointer.BoolPtr(true),
					},
					AdditionalKernelArgs: []string{
						"NEW_ARGUMENT",
					},
					NUMA: &performancev1.NUMA{
						TopologyPolicy: pointer.StringPtr("restricted"),
					},
				},
			}
			Expect(testclient.Client.Create(context.TODO(), secondProfile)).ToNot(HaveOccurred())

			By("Checking that new KubeletConfig, MachineConfig and RuntimeClass created")
			configKey := types.NamespacedName{
				Name:      components.GetComponentName(secondProfile.Name, components.ComponentNamePrefix),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			kubeletConfig := &machineconfigv1.KubeletConfig{}
			err := testclient.GetWithRetry(context.TODO(), configKey, kubeletConfig)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot find KubeletConfig object %s", configKey.Name))
			Expect(kubeletConfig.Spec.MachineConfigPoolSelector.MatchLabels[machineconfigv1.MachineConfigRoleLabelKey]).Should(Equal(newRole))
			Expect(kubeletConfig.Spec.KubeletConfig.Raw).Should(ContainSubstring("restricted"), "Can't find value in KubeletConfig")

			machineConfig := &machineconfigv1.MachineConfig{}
			err = testclient.GetWithRetry(context.TODO(), configKey, machineConfig)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot find MachineConfig object %s", configKey.Name))
			Expect(machineConfig.Labels[machineconfigv1.MachineConfigRoleLabelKey]).Should(Equal(newRole))

			runtimeClass := &v1beta1.RuntimeClass{}
			err = testclient.GetWithRetry(context.TODO(), configKey, runtimeClass)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot find RuntimeClass profile object %s", runtimeClass.Name))
			Expect(runtimeClass.Handler).Should(Equal(machineconfig.HighPerformanceRuntime))

			By("Checking that new Tuned profile created")
			tunedKey := types.NamespacedName{
				Name:      components.GetComponentName(secondProfile.Name, components.ProfileNamePerformance),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			tunedProfile := &tunedv1.Tuned{}
			err = testclient.GetWithRetry(context.TODO(), tunedKey, tunedProfile)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot find Tuned profile object %s", tunedKey.Name))
			Expect(tunedProfile.Spec.Recommend[0].MachineConfigLabels[machineconfigv1.MachineConfigRoleLabelKey]).Should(Equal(newRole))
			Expect(*tunedProfile.Spec.Profile[0].Data).Should(ContainSubstring("NEW_ARGUMENT"), "Can't find value in Tuned profile")

			By("Checking that the initial MCP does not start updating")
			Consistently(func() corev1.ConditionStatus {
				return mcps.GetConditionStatus(testutils.RoleWorkerCNF, machineconfigv1.MachineConfigPoolUpdating)
			}, 30, 5).Should(Equal(corev1.ConditionFalse))

			By("Remove second profile and verify that KubeletConfig and MachineConfig were removed")
			Expect(testclient.Client.Delete(context.TODO(), secondProfile)).ToNot(HaveOccurred())

			Consistently(func() corev1.ConditionStatus {
				return mcps.GetConditionStatus(testutils.RoleWorkerCNF, machineconfigv1.MachineConfigPoolUpdating)
			}, 30, 5).Should(Equal(corev1.ConditionFalse))

			Expect(testclient.Client.Get(context.TODO(), configKey, kubeletConfig)).To(HaveOccurred(), fmt.Sprintf("KubeletConfig %s should be removed", configKey.Name))
			Expect(testclient.Client.Get(context.TODO(), configKey, machineConfig)).To(HaveOccurred(), fmt.Sprintf("MachineConfig %s should be removed", configKey.Name))
			Expect(testclient.Client.Get(context.TODO(), configKey, runtimeClass)).To(HaveOccurred(), fmt.Sprintf("RuntimeClass %s should be removed", configKey.Name))
			Expect(testclient.Client.Get(context.TODO(), tunedKey, tunedProfile)).To(HaveOccurred(), fmt.Sprintf("Tuned profile object %s should be removed", tunedKey.Name))

			By("Checking that initial KubeletConfig and MachineConfig still exist")
			initialKey := types.NamespacedName{
				Name:      components.GetComponentName(profile.Name, components.ComponentNamePrefix),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			err = testclient.GetWithRetry(context.TODO(), initialKey, &machineconfigv1.KubeletConfig{})
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot find KubeletConfig object %s", initialKey.Name))
			err = testclient.GetWithRetry(context.TODO(), initialKey, &machineconfigv1.MachineConfig{})
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot find MachineConfig object %s", initialKey.Name))
		})
	})
})

func execSysctlOnWorkers(workerNodes []corev1.Node, sysctlMap map[string]string) {
	var err error
	var out []byte
	for _, node := range workerNodes {
		for param, expected := range sysctlMap {
			By(fmt.Sprintf("executing the command \"sysctl -n %s\"", param))
			out, err = nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"sysctl", "-n", param})
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.TrimSpace(string(out))).Should(Equal(expected), fmt.Sprintf("parameter %s value is not %s.", param, expected))
		}
	}
}

// execute sysctl command inside container in a tuned pod
func validatTunedActiveProfile(nodes []corev1.Node) {
	var err error
	var out []byte
	activeProfileName := components.GetComponentName(testutils.PerformanceProfileName, components.ProfileNamePerformance)
	for _, node := range nodes {
		tuned := tunedForNode(&node)
		tunedName := tuned.ObjectMeta.Name
		By(fmt.Sprintf("executing the command cat /etc/tuned/active_profile inside the pod %s", tunedName))
		Eventually(func() string {
			out, err = pods.ExecCommandOnPod(tuned, []string{"cat", "/etc/tuned/active_profile"})
			return strings.TrimSpace(string(out))
		}, testTimeout*time.Second, testPollInterval*time.Second).Should(Equal(activeProfileName),
			fmt.Sprintf("active_profile is not set to %s. %v", activeProfileName, err))
	}
}

// find tuned pod for appropriate node
func tunedForNode(node *corev1.Node) *corev1.Pod {
	listOptions := &client.ListOptions{
		Namespace:     components.NamespaceNodeTuningOperator,
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}),
		LabelSelector: labels.SelectorFromSet(labels.Set{"openshift-app": "tuned"}),
	}

	tunedList := &corev1.PodList{}
	Eventually(func() bool {
		if err := testclient.Client.List(context.TODO(), tunedList, listOptions); err != nil {
			return false
		}

		if len(tunedList.Items) == 0 {
			return false
		}
		for _, s := range tunedList.Items[0].Status.ContainerStatuses {
			if s.Ready == false {
				return false
			}
		}
		return true

	}, testTimeout*time.Second, testPollInterval*time.Second).Should(BeTrue(),
		"there should be one tuned daemon per node")

	return &tunedList.Items[0]
}
