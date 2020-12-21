package __performance

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/discovery"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/images"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
)

var workerRTNode *corev1.Node
var profile *performancev2.PerformanceProfile

var _ = Describe("[rfe_id:27363][performance] CPU Management", func() {
	var balanceIsolated bool
	var reservedCPU, isolatedCPU string
	var listReservedCPU, listIsolatedCPU []int
	var reservedCPUSet, isolatedCPUSet cpuset.CPUSet

	BeforeEach(func() {
		if discovery.Enabled() && testutils.ProfileNotFound {
			Skip("Discovery mode enabled, performance profile not found")
		}

		workerRTNodes, err := nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for the optional selector: %v", err))
		Expect(workerRTNodes).ToNot(BeEmpty())
		workerRTNode = &workerRTNodes[0]
		profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("Checking the profile %s with cpus %#v", profile.Name, profile.Spec.CPU))
		balanceIsolated = true
		if profile.Spec.CPU.BalanceIsolated != nil {
			balanceIsolated = *profile.Spec.CPU.BalanceIsolated
		}

		Expect(profile.Spec.CPU.Isolated).NotTo(BeNil())
		isolatedCPU = string(*profile.Spec.CPU.Isolated)
		isolatedCPUSet, err = cpuset.Parse(isolatedCPU)
		Expect(err).ToNot(HaveOccurred())
		listIsolatedCPU = isolatedCPUSet.ToSlice()

		Expect(profile.Spec.CPU.Reserved).NotTo(BeNil())
		reservedCPU = string(*profile.Spec.CPU.Reserved)
		reservedCPUSet, err = cpuset.Parse(reservedCPU)
		Expect(err).ToNot(HaveOccurred())
		listReservedCPU = reservedCPUSet.ToSlice()
	})

	Describe("Verification of configuration on the worker node", func() {
		It("[test_id:28528][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verify CPU reservation on the node", func() {
			By(fmt.Sprintf("Allocatable CPU should be less then capacity by %d", len(listReservedCPU)))
			capacityCPU, _ := workerRTNode.Status.Capacity.Cpu().AsInt64()
			allocatableCPU, _ := workerRTNode.Status.Allocatable.Cpu().AsInt64()
			Expect(capacityCPU - allocatableCPU).To(Equal(int64(len(listReservedCPU))))
		})

		It("[test_id:27081][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verify CPU affinity mask, CPU reservation and CPU isolation on worker node", func() {
			By("checking isolated CPU")
			cmd := []string{"cat", "/sys/devices/system/cpu/isolated"}
			sysIsolatedCpus, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred())
			if balanceIsolated {
				Expect(sysIsolatedCpus).To(BeEmpty())
			} else {
				Expect(sysIsolatedCpus).To(Equal(isolatedCPU))
			}

			By("checking reserved CPU in kubelet config file")
			cmd = []string{"cat", "/rootfs/etc/kubernetes/kubelet.conf"}
			conf, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred(), "failed to cat kubelet.conf")
			// kubelet.conf changed formatting, there is a space after colons atm. Let's deal with both cases with a regex
			Expect(conf).To(MatchRegexp(fmt.Sprintf(`"reservedSystemCPUs": ?"%s"`, reservedCPU)))

			By("checking CPU affinity mask for kernel scheduler")
			cmd = []string{"/bin/bash", "-c", "taskset -pc 1"}
			sched, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred(), "failed to execute taskset")
			mask := strings.SplitAfter(sched, " ")
			maskSet, err := cpuset.Parse(mask[len(mask)-1])
			Expect(err).ToNot(HaveOccurred())

			Expect(reservedCPUSet.IsSubsetOf(maskSet)).To(Equal(true), fmt.Sprintf("The init process (pid 1) should have cpu affinity: %s", reservedCPU))
		})

		It("[test_id:34358] Verify rcu_nocbs kernel argument on the node", func() {
			By("checking that cmdline contains rcu_nocbs with right value")
			cmd := []string{"cat", "/proc/cmdline"}
			cmdline, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred())
			re := regexp.MustCompile(`rcu_nocbs=\S+`)
			rcuNocbsArgument := re.FindString(cmdline)
			Expect(rcuNocbsArgument).To(ContainSubstring("rcu_nocbs="))
			rcuNocbsCpu := strings.Split(rcuNocbsArgument, "=")[1]
			Expect(rcuNocbsCpu).To(Equal(isolatedCPU))

			By("checking that new rcuo processes are running on non_isolated cpu")
			cmd = []string{"pgrep", "rcuo"}
			rcuoList, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred())
			for _, rcuo := range strings.Split(rcuoList, "\n") {
				// check cpu affinity mask
				cmd = []string{"/bin/bash", "-c", fmt.Sprintf("taskset -pc %s", rcuo)}
				taskset, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
				Expect(err).ToNot(HaveOccurred())
				mask := strings.SplitAfter(taskset, " ")
				maskSet, err := cpuset.Parse(mask[len(mask)-1])
				Expect(err).ToNot(HaveOccurred())
				Expect(reservedCPUSet.IsSubsetOf(maskSet)).To(Equal(true), fmt.Sprintf("The process should have cpu affinity: %s", reservedCPU))

				// check which cpu is used
				cmd = []string{"/bin/bash", "-c", fmt.Sprintf("ps -o psr %s | tail -1", rcuo)}
				psr, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
				Expect(err).ToNot(HaveOccurred())
				cpu, err := strconv.Atoi(strings.Trim(psr, " "))
				Expect(err).ToNot(HaveOccurred())
				Expect(cpu).NotTo(BeElementOf(listIsolatedCPU))
			}
		})
	})

	Describe("[test_id:27492][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verification of cpu manager functionality", func() {
		var testpod *corev1.Pod
		var discoveryFailed bool

		testutils.BeforeAll(func() {
			discoveryFailed = false
			if discovery.Enabled() {
				profile, err := profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
				Expect(err).ToNot(HaveOccurred())
				isolatedCPU = string(*profile.Spec.CPU.Isolated)
				isolatedCPUSet, err := cpuset.Parse(isolatedCPU)
				Expect(err).ToNot(HaveOccurred())
				if isolatedCPUSet.Size() <= 1 {
					discoveryFailed = true
				}
			}
		})

		BeforeEach(func() {
			if discoveryFailed {
				Skip("Skipping tests since there are insufficant isolated cores to create a stress pod")
			}
		})

		AfterEach(func() {
			deleteTestPod(testpod)
		})

		table.DescribeTable("Verify CPU usage by stress PODs", func(guaranteed bool) {
			var listCPU []int

			testpod = getStressPod(workerRTNode.Name)
			testpod.Namespace = testutils.NamespaceTesting

			//list worker cpus
			cmd := []string{"/bin/bash", "-c", "lscpu | grep On-line | awk '{print $4}'"}
			lscpu, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred(), "failed to execute lscpu")
			cpus, err := cpuset.Parse(lscpu)
			Expect(err).ToNot(HaveOccurred())
			listCPU = cpus.ToSlice()

			if guaranteed {
				listCPU = cpus.Difference(reservedCPUSet).ToSlice()
				testpod.Spec.Containers[0].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				}
			} else if !balanceIsolated {
				// when balanceIsolated is False - non-guaranteed pod should run on reserved cpu
				listCPU = listReservedCPU
			}

			err = testclient.Client.Create(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForCondition(testpod, corev1.PodReady, corev1.ConditionTrue, 10*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			output, err := nodes.ExecCommandOnNode(
				[]string{"/bin/bash", "-c", "ps -o psr $(pgrep -n stress) | tail -1"},
				workerRTNode,
			)
			Expect(err).ToNot(HaveOccurred(), "failed to get cpu of stress process")
			cpu, err := strconv.Atoi(strings.Trim(output, " "))
			Expect(err).ToNot(HaveOccurred())

			Expect(cpu).To(BeElementOf(listCPU))
		},
			table.Entry("Non-guaranteed POD can work on any CPU", false),
			table.Entry("Guaranteed POD should work on isolated cpu", true),
		)
	})

	When("pod runs with the CPU load balancing runtime class", func() {
		var testpod *corev1.Pod
		var defaultFlags []int

		getCPUSchedulingDomainFlags := func(cpu int) ([]int, error) {
			cmd := []string{"/bin/bash", "-c", fmt.Sprintf("cat /proc/sys/kernel/sched_domain/cpu%d/domain*/flags", cpu)}
			out, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			if err != nil {
				return nil, err
			}

			var domainsFlags []int
			for _, domainFlags := range strings.Split(out, "\n") {
				flags, err := strconv.Atoi(domainFlags)
				if err != nil {
					return nil, err
				}
				domainsFlags = append(domainsFlags, flags)
			}

			sort.Ints(domainsFlags)
			return domainsFlags, nil
		}

		BeforeEach(func() {
			// get the default value for sched_domain flags, we will check the flags only for the CPU 0, because on the
			// clean system it should be the same for all CPUs
			var err error
			defaultFlags, err = getCPUSchedulingDomainFlags(0)
			Expect(err).ToNot(HaveOccurred())

			annotations := map[string]string{
				"cpu-load-balancing.crio.io": "disable",
			}
			testpod = getTestPodWithAnnotations(annotations)
		})

		AfterEach(func() {
			deleteTestPod(testpod)
		})

		It("[test_id:32646] should disable CPU load balancing for CPU's used by the pod", func() {
			By("Starting the pod")
			err := testclient.Client.Create(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForCondition(testpod, corev1.PodReady, corev1.ConditionTrue, 10*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			By("Getting the container cpuset.cpus cgroup")
			containerID, err := pods.GetContainerIDByName(testpod, "test")
			Expect(err).ToNot(HaveOccurred())

			cmd := []string{"/bin/bash", "-c", fmt.Sprintf("find /rootfs/sys/fs/cgroup/cpuset/ -name *%s*", containerID)}
			containerCgroup, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred())

			By("Checking what CPU the pod is using")
			cmd = []string{"/bin/bash", "-c", fmt.Sprintf("cat %s/cpuset.cpus", containerCgroup)}
			output, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred())

			cpu, err := strconv.Atoi(output)
			Expect(err).ToNot(HaveOccurred())

			By("Getting the CPU scheduling flags")
			flags, err := getCPUSchedulingDomainFlags(cpu)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying that the CPU load balancing was disabled")
			Expect(len(flags)).To(Equal(len(defaultFlags)))
			// the CPU flags should be almost the same except the LSB that should be disabled
			for i := range flags {
				Expect(flags[i]).To(Equal(defaultFlags[i] - 1))
			}

			By("Deleting the pod")
			deleteTestPod(testpod)

			By("Getting the CPU scheduling flags")
			flags, err = getCPUSchedulingDomainFlags(cpu)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying that the CPU load balancing was enabled back")
			Expect(len(flags)).To(Equal(len(defaultFlags)))
			// the CPU scheduling flags should be restored to the default values
			for i := range flags {
				Expect(flags[i]).To(Equal(defaultFlags[i]))
			}
		})
	})

	Describe("Verification that IRQ load balance can be disabled per POD", func() {
		var testpod *corev1.Pod

		BeforeEach(func() {
			if profile.Spec.GloballyDisableIrqLoadBalancing != nil && *profile.Spec.GloballyDisableIrqLoadBalancing {
				Skip("IRQ load balance should be enabled (GloballyDisableIrqLoadBalancing=false), skipping test")
			}
		})

		AfterEach(func() {
			deleteTestPod(testpod)
		})

		It("[test_id:36364] should disable IRQ balance for CPU where POD is running", func() {
			By("checking default smp affinity is equal to all active CPUs")
			defaultSmpAffinitySet, err := nodes.GetDefaultSmpAffinitySet(workerRTNode)
			Expect(err).ToNot(HaveOccurred())

			onlineCPUsSet, err := nodes.GetOnlineCPUsSet(workerRTNode)
			Expect(err).ToNot(HaveOccurred())

			Expect(defaultSmpAffinitySet).To(Equal(onlineCPUsSet), fmt.Sprintf("Default SMP Affinity %s should be equal to all active CPUs %s", defaultSmpAffinitySet, onlineCPUsSet))

			By("Running pod with annotations that disable specific CPU from IRQ balancer")
			annotations := map[string]string{
				"irq-load-balancing.crio.io": "disable",
				"cpu-quota.crio.io":          "disable",
			}
			testpod = getTestPodWithAnnotations(annotations)

			err = testclient.Client.Create(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForCondition(testpod, corev1.PodReady, corev1.ConditionTrue, 10*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			By("Checking that the default smp affinity mask was updated and CPU (where POD is running) isolated")
			defaultSmpAffinitySet, err = nodes.GetDefaultSmpAffinitySet(workerRTNode)
			Expect(err).ToNot(HaveOccurred())

			getPsr := []string{"/bin/bash", "-c", "grep Cpus_allowed_list /proc/self/status | awk '{print $2}'"}
			psr, err := pods.ExecCommandOnPod(testpod, getPsr)
			Expect(err).ToNot(HaveOccurred())
			psrSet, err := cpuset.Parse(strings.Trim(string(psr), "\n"))
			Expect(err).ToNot(HaveOccurred())

			Expect(defaultSmpAffinitySet).To(Equal(onlineCPUsSet.Difference(psrSet)), fmt.Sprintf("Default SMP affinity should not contain isolated CPU %s", psr))

			By("Checking that there are no any active IRQ on isolated CPU")
			// It may takes some time for the system to reschedule active IRQs
			Eventually(func() bool {
				getActiveIrq := []string{"/bin/bash", "-c", "for n in $(find /proc/irq/ -name smp_affinity_list); do echo $(cat $n); done"}
				activeIrq, err := nodes.ExecCommandOnNode(getActiveIrq, workerRTNode)
				Expect(err).ToNot(HaveOccurred())
				for _, irq := range strings.Split(activeIrq, "\n") {
					irqAffinity, err := cpuset.Parse(irq)
					Expect(err).ToNot(HaveOccurred())
					if !irqAffinity.Equals(onlineCPUsSet) && psrSet.IsSubsetOf(irqAffinity) {
						return false
					}
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue(),
				fmt.Sprintf("IRQ still active on CPU%s", psr))

			By("Checking that after removing POD default smp affinity is returned back to all active CPUs")
			deleteTestPod(testpod)
			defaultSmpAffinitySet, err = nodes.GetDefaultSmpAffinitySet(workerRTNode)
			Expect(err).ToNot(HaveOccurred())

			Expect(defaultSmpAffinitySet).To(Equal(onlineCPUsSet), fmt.Sprintf("Default SMP Affinity %s should be equal to all active CPUs %s", defaultSmpAffinitySet, onlineCPUsSet))
		})
	})
})

func getStressPod(nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-cpu-",
			Labels: map[string]string{
				"test": "",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "stress-test",
					Image: images.Test(),
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
					Command: []string{"/usr/bin/stresser"},
					Args:    []string{"-cpus", "1"},
				},
			},
			NodeSelector: map[string]string{
				testutils.LabelHostname: nodeName,
			},
		},
	}
}

func getTestPodWithAnnotations(annotations map[string]string) *corev1.Pod {
	testpod := pods.GetTestPod()
	testpod.Annotations = annotations
	testpod.Namespace = testutils.NamespaceTesting

	cpus := resource.MustParse("1")
	memory := resource.MustParse("256Mi")

	// change pod resource requirements, to change the pod QoS class to guaranteed
	testpod.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    cpus,
			corev1.ResourceMemory: memory,
		},
	}

	runtimeClassName := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	testpod.Spec.RuntimeClassName = &runtimeClassName
	testpod.Spec.NodeSelector = map[string]string{testutils.LabelHostname: workerRTNode.Name}

	return testpod
}

func deleteTestPod(testpod *corev1.Pod) {
	// it possible that the pod already was deleted as part of the test, in this case we want to skip teardown
	key := types.NamespacedName{
		Name:      testpod.Name,
		Namespace: testpod.Namespace,
	}
	err := testclient.Client.Get(context.TODO(), key, testpod)
	if errors.IsNotFound(err) {
		return
	}

	err = testclient.Client.Delete(context.TODO(), testpod)
	Expect(err).ToNot(HaveOccurred())

	err = pods.WaitForDeletion(testpod, 60*time.Second)
	Expect(err).ToNot(HaveOccurred())
}
