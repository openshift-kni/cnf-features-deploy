package __latency

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/discovery"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/utils/pointer"
)

var (
	latencyTestDelay   = "0"
	latencyTestRun     = false
	latencyTestRuntime = "300"
	latencyTestImage   = "quay.io/openshift-kni/oslat:latest"
	maximumLatency     = -1
)

// LATENCY_TEST_RUN: indicates if the latency test should run
// LATENCY_TEST_RUNTIME: the amount of time in seconds that the latency test should run
// LATENCY_TEST_IMAGE: the image use under the latency test
// OSLAT_MAXIMUM_LATENCY: the expected maximum latency for all buckets in us
func init() {
	latencyTestRunEnv := os.Getenv("LATENCY_TEST_RUN")
	if latencyTestRunEnv != "" {
		if latencyTestRunEnv == "true" {
			latencyTestRun = true
		}
	}

	latencyTestRuntimeEnv := os.Getenv("LATENCY_TEST_RUNTIME")
	if latencyTestRuntimeEnv != "" {
		latencyTestRuntime = latencyTestRuntimeEnv
	}

	latencyTestImageEnv := os.Getenv("LATENCY_TEST_IMAGE")
	if latencyTestImageEnv != "" {
		latencyTestImage = latencyTestImageEnv
	}

	latencyTestDelayEnv := os.Getenv("LATENCY_TEST_DELAY")
	if latencyTestDelayEnv != "" {
		latencyTestDelay = latencyTestDelayEnv
	}

	maximumLatencyEnv := os.Getenv("OSLAT_MAXIMUM_LATENCY")
	if maximumLatencyEnv != "" {
		var err error
		if maximumLatency, err = strconv.Atoi(maximumLatencyEnv); err != nil {
			klog.Errorf("the environment variable OSLAT_MAXIMUM_LATENCY has incorrect value %q", maximumLatencyEnv)
			panic(err)
		}
	}
}

var _ = Describe("[performance] Latency Test", func() {
	var workerRTNode *corev1.Node
	var profile *performancev2.PerformanceProfile
	var oslatPod *corev1.Pod

	BeforeEach(func() {
		if !latencyTestRun {
			Skip("Skip the oslat test, the LATENCY_TEST_RUN set to false")
		}
		if discovery.Enabled() && testutils.ProfileNotFound {
			Skip("Discovery mode enabled, performance profile not found")
		}

		var err error
		profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())

		if profile.Spec.CPU.Isolated == nil {
			Skip(fmt.Sprintf("Skip the oslat test, the profile %q does not have isolated CPUs", profile.Name))
		}

		isolatedCpus := cpuset.MustParse(string(*profile.Spec.CPU.Isolated))
		// we require at least two CPUs to run oslat test, because one CPU should be used to run the main oslat thread
		// we can not use all isolated CPUs, because if reserved and isolated include all node CPUs, and reserved CPUs
		// do not calculated into the Allocated, at least part of time of one of isolated CPUs will be used to run
		// other node containers
		// at least two isolated CPUs to run oslat + one isolated CPU used by other containers on the node = at least 3 isolated CPUs
		if isolatedCpus.Size() < 3 {
			Skip(fmt.Sprintf("Skip the oslat test, the profile %q has less than two isolated CPUs", profile.Name))
		}

		workerRTNodes, err := nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for the optional selector: %v", err))
		Expect(workerRTNodes).ToNot(BeEmpty())
		workerRTNode = &workerRTNodes[0]
	})

	AfterEach(func() {
		var err error
		err = testclient.Client.Delete(context.TODO(), oslatPod)
		if err != nil {
			klog.Error(err)
		}
		err = pods.WaitForDeletion(oslatPod, 60*time.Second)
		if err != nil {
			klog.Error(err)
		}
	})

	Context("with the oslat image", func() {
		It("should succeed", func() {
			oslatPod = getOslatPod(profile, workerRTNode)
			err := testclient.Client.Create(context.TODO(), oslatPod)
			Expect(err).ToNot(HaveOccurred())

			timeout, err := strconv.Atoi(latencyTestRuntime)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting two minutes to download the oslat image")
			err = pods.WaitForPhase(oslatPod, corev1.PodRunning, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting another two minutes to give enough time for the cluster to move the pod to Succeeded phase")
			podTimeout := time.Duration(timeout + 120)
			err = pods.WaitForPhase(oslatPod, corev1.PodSucceeded, podTimeout*time.Second)
			Expect(err).ToNot(HaveOccurred())

			cmd := []string{"cat", "/rootfs/var/log/oslat.log"}
			out, err := nodes.ExecCommandOnNode(cmd, workerRTNode)
			Expect(err).ToNot(HaveOccurred())

			// verify the maximum latency only when it requested, because this value can be very different
			// on different systems
			if maximumLatency == -1 {
				return
			}

			maximumRegex, err := regexp.Compile(`Maximum:\t*\s*(.*)\s*\(us\)`)
			Expect(err).ToNot(HaveOccurred())

			latencies := maximumRegex.FindSubmatch([]byte(out))
			Expect(maximumLatency).ToNot(BeNil())

			// under the output of the oslat very often we have one anomaly high value, for example
			// Maximum:    16543 15 15 14 13 12 12 13 12 12 12 12 12 12 12 12 12 (us)
			// it still unclear if it oslat bug or the kernel one, but we definitely do not want to
			// fail our test on it
			var anomaly bool
			for _, lat := range strings.Split(string(latencies[1]), " ") {
				if lat == "" {
					continue
				}

				curr, err := strconv.Atoi(lat)
				Expect(err).ToNot(HaveOccurred())

				// skip the anomaly value
				if curr > maximumLatency && !anomaly {
					anomaly = true
					continue
				}

				Expect(curr < maximumLatency).To(BeTrue(), fmt.Sprintf("The current latency %d is bigger than the expected one %d", curr, maximumLatency))
			}
		})
	})
})

func getOslatPod(profile *performancev2.PerformanceProfile, node *corev1.Node) *corev1.Pod {
	cpus := cpuset.MustParse(string(*profile.Spec.CPU.Isolated))
	runtimeClass := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	volumeTypeDirectory := corev1.HostPathDirectory
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "oslat-",
			Annotations: map[string]string{
				"cpu-load-balancing.crio.io": "true",
			},
			Namespace: testutils.NamespaceTesting,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:    corev1.RestartPolicyNever,
			RuntimeClassName: &runtimeClass,
			Containers: []corev1.Container{
				{
					Name:  "oslat",
					Image: latencyTestImage,
					Env: []corev1.EnvVar{
						{
							// we mount the host directory under the pod and write all logs to oslat.log under it
							Name:  "LOG_DIR",
							Value: "/host",
						},
						{
							Name:  "RUNTIME_SECONDS",
							Value: latencyTestRuntime,
						},
						{
							Name:  "INITIAL_DELAY",
							Value: latencyTestDelay,
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							// we can not use all isolated CPUs, because if reserved and isolated include all node CPUs, and reserved CPUs
							// do not calculated into the Allocated, at least part of time of one of isolated CPUs will be used to run
							// other node containers
							corev1.ResourceCPU:    resource.MustParse(strconv.Itoa(cpus.Size() - 1)),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: pointer.BoolPtr(true),
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "logs",
							MountPath: "/host",
						},
					},
				},
			},
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": node.Labels["kubernetes.io/hostname"],
			},
			Volumes: []corev1.Volume{
				{
					Name: "logs",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/log",
							Type: &volumeTypeDirectory,
						},
					},
				},
			},
		},
	}
}
