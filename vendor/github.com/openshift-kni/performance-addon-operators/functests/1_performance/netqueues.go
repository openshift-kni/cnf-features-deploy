package __performance

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/cluster"
	testlog "github.com/openshift-kni/performance-addon-operators/functests/utils/log"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
)

var _ = Describe("[ref_id: 40307][pao]Resizing Network Queues", func() {
	var workerRTNodes []corev1.Node
	var profile, initialProfile *performancev2.PerformanceProfile
	var performanceProfileName string

	testutils.BeforeAll(func() {
		isSNO, err := cluster.IsSingleNode()
		Expect(err).ToNot(HaveOccurred())
		RunningOnSingleNode = isSNO

		workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())

		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred())

		profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())

		By("Backing up the profile")
		initialProfile = profile.DeepCopy()

		performanceProfileName = profile.Name
	})

	BeforeEach(func() {
		profile, err := profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		if profile.Spec.Net == nil {
			By("Enable UserLevelNetworking in Profile")
			profile.Spec.Net = &performancev2.Net{
				UserLevelNetworking: pointer.BoolPtr(true),
			}
			By("Updating the performance profile")
			profiles.UpdateWithRetry(profile)
		}
	})

	AfterEach(func() {
		By("Reverting the Profile")
		spec, err := json.Marshal(initialProfile.Spec)
		Expect(err).ToNot(HaveOccurred())
		Expect(testclient.Client.Patch(context.TODO(), profile,
			client.RawPatch(
				types.JSONPatchType,
				[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, spec)),
			),
		)).ToNot(HaveOccurred())
	})

	Context("Updating performance profile for netqueues", func() {
		It("[test_id:40308][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Network device queues Should be set to the profile's reserved CPUs count ", func() {
			devices := make(map[string][]int)
			count := 0
			if profile.Spec.Net != nil {
				if profile.Spec.Net.UserLevelNetworking != nil && *profile.Spec.Net.UserLevelNetworking && len(profile.Spec.Net.Devices) == 0 {
					By("To all non virtual network devices when no devices are specified under profile.Spec.Net.Devices")
					err := checkDeviceSupport(workerRTNodes, devices)
					Expect(err).ToNot(HaveOccurred())
					for _, sizes := range devices {
						for _, size := range sizes {
							if size == getReservedCPUSize(profile.Spec.CPU) {
								count++
							}
						}
					}
					if count == 0 {
						Skip("Skipping Test: Unable to set Network queue size to reserved cpu count")
					}
				}
			}
		})

		It("[test_id:40542] Verify the number of network queues of all supported network interfaces are equal to reserved cpus count", func() {
			tunedPaoProfile := fmt.Sprintf("openshift-node-performance-%s", performanceProfileName)
			devices := make(map[string][]int)
			count := 0
			// Populate the device map with queue sizes
			Eventually(func() bool {
				err := checkDeviceSupport(workerRTNodes, devices)
				Expect(err).ToNot(HaveOccurred())
				return true
			}, cluster.ComputeTestTimeout(200*time.Second, RunningOnSingleNode), testPollInterval*time.Second).Should(BeTrue())
			//Verify the tuned profile is created on the worker-cnf nodes:
			tunedCmd := []string{"tuned-adm", "profile_info", tunedPaoProfile}
			for _, node := range workerRTNodes {
				tunedPod := nodes.TunedForNode(&node, RunningOnSingleNode)
				_, err := pods.ExecCommandOnPod(tunedPod, tunedCmd)
				Expect(err).ToNot(HaveOccurred())
			}
			for _, sizes := range devices {
				for _, size := range sizes {
					if size == getReservedCPUSize(profile.Spec.CPU) {
						count++
					}
				}
			}
			if count == 0 {
				Skip("Skipping Test: Unable to set Network queue size to reserved cpu count")
			}
		})

		It("[test_id:40543] Add interfaceName and verify the interface netqueues are equal to reserved cpus count.", func() {
			devices := make(map[string][]int)
			count := 0
			err := checkDeviceSupport(workerRTNodes, devices)
			Expect(err).ToNot(HaveOccurred())
			device := getRandomDevice(devices)
			profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())
			if profile.Spec.Net.UserLevelNetworking != nil && *profile.Spec.Net.UserLevelNetworking && len(profile.Spec.Net.Devices) == 0 {
				By("Enable UserLevelNetworking and add Devices in Profile")
				profile.Spec.Net = &performancev2.Net{
					UserLevelNetworking: pointer.BoolPtr(true),
					Devices: []performancev2.Device{
						{
							InterfaceName: &device,
						},
					},
				}
				By("Updating the performance profile")
				profiles.UpdateWithRetry(profile)

				Eventually(func() bool {
					err := checkDeviceSupport(workerRTNodes, devices)
					Expect(err).ToNot(HaveOccurred())
					return true
				}, cluster.ComputeTestTimeout(200*time.Second, RunningOnSingleNode), testPollInterval*time.Second).Should(BeTrue())
			}
			//Verify the tuned profile is created on the worker-cnf nodes:
			tunedCmd := []string{"bash", "-c",
				fmt.Sprintf("cat /etc/tuned/openshift-node-performance-%s/tuned.conf | grep devices_udev_regex", performanceProfileName)}

			for _, node := range workerRTNodes {
				tunedPod := nodes.TunedForNode(&node, RunningOnSingleNode)
				out, err := pods.ExecCommandOnPod(tunedPod, tunedCmd)
				deviceExists := strings.ContainsAny(string(out), device)
				Expect(deviceExists).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			}
			for _, sizes := range devices {
				for _, size := range sizes {
					if size == getReservedCPUSize(profile.Spec.CPU) {
						count++
					}
				}
			}
			if count == 0 {
				Skip("Skipping Test: Unable to set Network queue size to reserved cpu count")
			}
		})

		It("[test_id:40545] Verify reserved cpus count is applied to specific supported networking devices using wildcard matches", func() {
			devices := make(map[string][]int)
			var device, devicePattern string
			count := 0
			err := checkDeviceSupport(workerRTNodes, devices)
			Expect(err).ToNot(HaveOccurred())
			device = getRandomDevice(devices)
			devicePattern = device[:len(device)-1] + "*"
			profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())
			if profile.Spec.Net.UserLevelNetworking != nil && *profile.Spec.Net.UserLevelNetworking && len(profile.Spec.Net.Devices) == 0 {
				By("Enable UserLevelNetworking and add Devices in Profile")
				profile.Spec.Net = &performancev2.Net{
					UserLevelNetworking: pointer.BoolPtr(true),
					Devices: []performancev2.Device{
						{
							InterfaceName: &devicePattern,
						},
					},
				}
				profiles.UpdateWithRetry(profile)
				Eventually(func() bool {
					err := checkDeviceSupport(workerRTNodes, devices)
					Expect(err).ToNot(HaveOccurred())
					return true
				}, cluster.ComputeTestTimeout(200*time.Second, RunningOnSingleNode), testPollInterval*time.Second).Should(BeTrue())
			}
			//Verify the tuned profile is created on the worker-cnf nodes:
			tunedCmd := []string{"bash", "-c",
				fmt.Sprintf("cat /etc/tuned/openshift-node-performance-%s/tuned.conf | grep devices_udev_regex", performanceProfileName)}

			for _, node := range workerRTNodes {
				tunedPod := nodes.TunedForNode(&node, RunningOnSingleNode)
				out, err := pods.ExecCommandOnPod(tunedPod, tunedCmd)
				deviceExists := strings.ContainsAny(string(out), device)
				Expect(deviceExists).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			}
			for _, sizes := range devices {
				for _, size := range sizes {
					if size == getReservedCPUSize(profile.Spec.CPU) {
						count++
					}
				}
			}
			if count == 0 {
				Skip("Skipping Test: Unable to set Network queue size to reserved cpu count")
			}
		})

		It("[test_id:40668] Verify reserved cpu count is added to networking devices matched with vendor and Device id", func() {
			devices := make(map[string][]int)
			var device, vid, did string
			count := 0
			err := checkDeviceSupport(workerRTNodes, devices)
			Expect(err).ToNot(HaveOccurred())
			device = getRandomDevice(devices)
			for _, node := range workerRTNodes {
				vid = getVendorID(node, device)
				did = getDeviceID(node, device)
			}
			profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())
			if profile.Spec.Net.UserLevelNetworking != nil && *profile.Spec.Net.UserLevelNetworking && len(profile.Spec.Net.Devices) == 0 {
				By("Enable UserLevelNetworking and add DeviceID, VendorID and Interface in Profile")
				profile.Spec.Net = &performancev2.Net{
					UserLevelNetworking: pointer.BoolPtr(true),
					Devices: []performancev2.Device{
						{
							InterfaceName: &device,
						},
						{
							VendorID: &vid,
							DeviceID: &did,
						},
					},
				}
				profiles.UpdateWithRetry(profile)
				Eventually(func() bool {
					err := checkDeviceSupport(workerRTNodes, devices)
					Expect(err).ToNot(HaveOccurred())
					return true
				}, cluster.ComputeTestTimeout(240*time.Second, RunningOnSingleNode), testPollInterval*time.Second).Should(BeTrue())
			}
			//Verify the tuned profile is created on the worker-cnf nodes:
			tunedCmd := []string{"bash", "-c",
				fmt.Sprintf("cat /etc/tuned/openshift-node-performance-%s/tuned.conf | grep devices_udev_regex", performanceProfileName)}
			for _, node := range workerRTNodes {
				tunedPod := nodes.TunedForNode(&node, RunningOnSingleNode)
				out, err := pods.ExecCommandOnPod(tunedPod, tunedCmd)
				deviceExists := strings.ContainsAny(string(out), device)
				Expect(deviceExists).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			}
			for _, sizes := range devices {
				for _, size := range sizes {
					if size == getReservedCPUSize(profile.Spec.CPU) {
						count++
					}
				}
			}
			if count == 0 {
				Skip("Skipping Test: Unable to set Network queue size to reserved cpu count")
			}
		})
	})
})

//Check if the device support multiple queues
func checkDeviceSupport(workernodes []corev1.Node, devices map[string][]int) error {
	cmdGetPhysicalDevices := []string{"find", "/sys/class/net", "-type", "l", "-not", "-lname", "*virtual*", "-printf", "%f "}
	var channelCurrentCombined int
	var err error
	for _, node := range workernodes {
		tunedPod := nodes.TunedForNode(&node, RunningOnSingleNode)
		phyDevs, err := pods.ExecCommandOnPod(tunedPod, cmdGetPhysicalDevices)
		Expect(err).ToNot(HaveOccurred())
		for _, d := range strings.Split(string(phyDevs), " ") {
			if d == "" {
				continue
			}
			_, err := pods.ExecCommandOnPod(tunedPod, []string{"ethtool", "-l", d})
			if err == nil {
				cmdCombinedChannelsCurrent := []string{"bash", "-c",
					fmt.Sprintf("ethtool -l %s | sed -n '/Current hardware settings:/,/Combined:/{s/^Combined:\\s*//p}'", d)}
				out, err := pods.ExecCommandOnPod(tunedPod, cmdCombinedChannelsCurrent)
				if strings.Contains(string(out), "n/a") {
					fmt.Printf("Device %s doesn't support multiple queues\n", d)
				} else {
					channelCurrentCombined, err = strconv.Atoi(strings.TrimSpace(string(out)))
					if err != nil {
						testlog.Warningf(fmt.Sprintf("unable to retrieve current multi-purpose channels hardware settings for device %s on %s",
							d, node.Name))
					}
					if channelCurrentCombined == 1 {
						fmt.Printf("Device %s doesn't support multiple queues\n", d)
					} else {
						fmt.Printf("Device %s supports multiple queues\n", d)
						devices[d] = append(devices[d], channelCurrentCombined)
					}
				}
			}
		}
	}
	// check the length of the map, if the map is empty , then
	// there are no supported devices. Hence we skip
	if len(devices) == 0 {
		Skip("Skipping Test: There are no supported Network Devices")
	}
	return err
}

func getReservedCPUSize(CPU *performancev2.CPU) int {
	reservedCPUs, err := cpuset.Parse(string(*CPU.Reserved))
	Expect(err).ToNot(HaveOccurred())
	return reservedCPUs.Size()
}

func getVendorID(node corev1.Node, device string) string {
	cmd := []string{"bash", "-c",
		fmt.Sprintf("cat /sys/class/net/%s/device/vendor", device)}
	stdout, err := nodes.ExecCommandOnNode(cmd, &node)
	Expect(err).ToNot(HaveOccurred())
	return stdout
}

func getDeviceID(node corev1.Node, device string) string {
	cmd := []string{"bash", "-c",
		fmt.Sprintf("cat /sys/class/net/%s/device/device", device)}
	stdout, err := nodes.ExecCommandOnNode(cmd, &node)
	Expect(err).ToNot(HaveOccurred())
	return stdout
}

func getRandomDevice(devices map[string][]int) string {
	device := ""
	for d := range devices {
		device = d
		break
	}
	return device
}
