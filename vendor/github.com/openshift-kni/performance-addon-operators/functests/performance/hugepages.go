package performance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"
)

const (
	pathHugepages2048kB = "/sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages"
	centosImage         = "centos:latest"
)

var _ = Describe("[performance]Hugepages", func() {
	var workerRTNode *corev1.Node
	var profile *performancev1alpha1.PerformanceProfile

	BeforeEach(func() {
		var err error
		workerRTNodes, err := nodes.GetByRole(testclient.Client, testutils.RoleWorkerRT)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
		workerRTNode = &workerRTNodes[0]

		profile, err = profiles.GetByNodeLabels(
			testclient.Client,
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(profile.Spec.HugePages).ToNot(BeNil())
	})

	Context("[rfe_id:27369]when NUMA node specified", func() {
		It("[test_id:27752][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] should be allocated on the specifed NUMA node ", func() {
			for _, page := range profile.Spec.HugePages.Pages {
				if page.Node == nil {
					continue
				}

				hugepagesSize, err := machineconfig.GetHugepagesSizeKilobytes(page.Size)
				Expect(err).ToNot(HaveOccurred())

				availableHugepagesFile := fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-%skB/nr_hugepages", *page.Node, hugepagesSize)
				nrHugepages := checkHugepagesStatus(availableHugepagesFile, workerRTNode)

				freeHugepagesFile := fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-%skB/free_hugepages", *page.Node, hugepagesSize)
				freeHugepages := checkHugepagesStatus(freeHugepagesFile, workerRTNode)

				Expect(int32(nrHugepages)).To(Equal(page.Count), "The number of available hugepages should be equal to the number in performance profile")
				Expect(nrHugepages).To(Equal(freeHugepages), "On idle system the number of available hugepages should be equal to free hugepages")
			}
		})
	})

	// TODO: enable it once https://github.com/kubernetes/kubernetes/pull/84051
	// is available under the openshift
	// Context("when NUMA node unspecified", func() {
	// 	It("should be allocated equally among NUMA nodes", func() {
	// 		command := []string{"cat", pathHugepages2048kB}
	// 		nrHugepages, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, workerRTNode, command)
	// 		Expect(err).ToNot(HaveOccurred())
	// 		Expect(string(nrHugepages)).To(Equal("128"))
	// 	})
	// })

	Context("[rfe_id:27354]Huge pages support for container workloads", func() {
		var testpod *corev1.Pod

		AfterEach(func() {
			err := testclient.Client.Delete(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForDeletion(testclient.Client, testpod, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:27477][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Huge pages support for container workloads", func() {
			hpSize := profile.Spec.HugePages.Pages[0].Size
			hpSizeKb, err := machineconfig.GetHugepagesSizeKilobytes(hpSize)
			Expect(err).ToNot(HaveOccurred())

			By("checking hugepages usage in bytes - should be 0 on idle system")
			usageHugepagesFile := fmt.Sprintf("/rootfs/sys/fs/cgroup/hugetlb/hugetlb.%sB.usage_in_bytes", hpSize)
			usageHugepages := checkHugepagesStatus(usageHugepagesFile, workerRTNode)
			Expect(usageHugepages).To(Equal(0))

			By("running the POD and waiting while it's installing testing tools")
			testpod = getCentosPod(workerRTNode.Name)
			testpod.Namespace = testutils.NamespaceTesting
			testpod.Spec.Containers[0].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceName(fmt.Sprintf("hugepages-%si", hpSize)): resource.MustParse(fmt.Sprintf("%si", hpSize)),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			}
			err = testclient.Client.Create(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForCondition(testclient.Client, testpod, corev1.PodReady, corev1.ConditionTrue, 180*time.Second)
			Expect(err).ToNot(HaveOccurred())

			cmd1 := []string{"yum", "install", "-y", "libhugetlbfs-utils", "libhugetlbfs", "tmux"}
			_, err = pods.ExecCommandOnPod(testclient.Client, testpod, cmd1)
			Expect(err).ToNot(HaveOccurred())

			cmd2 := []string{"/bin/bash", "-c", "tmux new -d 'LD_PRELOAD=libhugetlbfs.so HUGETLB_MORECORE=yes top -b > /dev/null'"}
			_, err = pods.ExecCommandOnPod(testclient.Client, testpod, cmd2)
			Expect(err).ToNot(HaveOccurred())

			By("checking free hugepages - one should be used by pod")
			availableHugepagesFile := fmt.Sprintf("/sys/devices/system/node/node0/hugepages/hugepages-%skB/nr_hugepages", hpSizeKb)
			availableHugepages := checkHugepagesStatus(availableHugepagesFile, workerRTNode)

			freeHugepagesFile := fmt.Sprintf("/sys/devices/system/node/node0/hugepages/hugepages-%skB/free_hugepages", hpSizeKb)
			freeHugepages := checkHugepagesStatus(freeHugepagesFile, workerRTNode)

			Expect(availableHugepages - freeHugepages).To(Equal(1))

			By("checking hugepages usage in bytes")
			usageHugepages = checkHugepagesStatus(usageHugepagesFile, workerRTNode)
			Expect(strconv.Itoa(usageHugepages/1024)).To(Equal(hpSizeKb), fmt.Sprintf("usage in bytes should be %s", hpSizeKb))
		})
	})
})

func checkHugepagesStatus(path string, workerRTNode *corev1.Node) int {
	command := []string{"cat", path}
	out, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, workerRTNode, command)
	Expect(err).ToNot(HaveOccurred())
	n, err := strconv.Atoi(strings.Trim(string(out), "\n"))
	Expect(err).ToNot(HaveOccurred())
	return n
}

func getCentosPod(nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-hugepages-",
			Labels: map[string]string{
				"test": "",
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "hugepages",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumHugePages},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    "test",
					Image:   centosImage,
					Command: []string{"sleep", "10h"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "hugepages",
							MountPath: "/dev/hugepages",
						},
					},
				},
			},
			NodeSelector: map[string]string{
				testutils.LabelHostname: nodeName,
			},
		},
	}
}
