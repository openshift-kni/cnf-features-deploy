package __performance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/images"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
)

var _ = Describe("[rfe_id:27363][performance] CPU Management", func() {
	var workerRTNode *corev1.Node
	var profile *performancev1alpha1.PerformanceProfile
	var balanceIsolated bool
	var reservedCPU, isolatedCPU string
	var listReservedCPU []int
	var reservedCPUSet cpuset.CPUSet

	BeforeEach(func() {
		workerRTNodes, err := nodes.GetByRole(testutils.RoleWorkerCNF)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
		workerRTNode = &workerRTNodes[0]

		profile, err = profiles.GetByNodeLabels(
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerCNF): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(profile.Spec.HugePages).ToNot(BeNil())

		balanceIsolated = true
		if profile.Spec.CPU.BalanceIsolated != nil {
			balanceIsolated = *profile.Spec.CPU.BalanceIsolated
		}

		Expect(profile.Spec.CPU.Reserved).NotTo(BeNil())
		reservedCPU = string(*profile.Spec.CPU.Reserved)
		reservedCPUSet, err := cpuset.Parse(reservedCPU)
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

		It("[test_id:28026][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verify CPU affinity mask, CPU reservation and CPU isolation on worker node", func() {
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
			Expect(conf).To(ContainSubstring(fmt.Sprintf(`"reservedSystemCPUs":"%s"`, reservedCPU)))

			By("checking CPU affinity mask for kernel scheduler")
			cmd = []string{"/bin/bash", "-c", "taskset -pc $(pgrep rcu_sched)"}
			sched, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred(), "failed to execute taskset")
			mask := strings.SplitAfter(sched, " ")
			maskSet, err := cpuset.Parse(mask[len(mask)-1])
			Expect(err).ToNot(HaveOccurred())

			Expect(reservedCPUSet.IsSubsetOf(maskSet)).To(Equal(true))
		})

	})

	Describe("[test_id:27492][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verification of cpu manager functionality", func() {
		var testpod *corev1.Pod

		AfterEach(func() {
			err := testclient.Client.Delete(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForDeletion(testpod, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())
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
