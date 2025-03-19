package pao

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"k8s.io/utils/cpuset"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovClean "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/clean"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/performanceprofile"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	"github.com/openshift/cluster-node-tuning-operator/pkg/performanceprofile/controller/performanceprofile/components"
	profileutil "github.com/openshift/cluster-node-tuning-operator/pkg/performanceprofile/controller/performanceprofile/components/profile"
	ntonodes "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils/nodes"
)

var (
	sriovclient                    *sriovtestclient.ClientSet
	profile                        *performancev2.PerformanceProfile
	machineConfigPoolName          string
	performanceProfileName         string
	enforcedPerformanceProfileName string
	dpdkResourceName               = "dpdknic"
)

const sriovNetworkName = "test-sriov-with-rps-configuration"

func init() {
	sriovclient = sriovtestclient.New("")
	performanceProfileName = os.Getenv("PERF_TEST_PROFILE")
	if performanceProfileName == "" {
		performanceProfileName = "performance"
	} else {
		enforcedPerformanceProfileName = performanceProfileName
	}
}

var _ = Describe("[rps][sriov] RPS configuration for SR-IOV devices", Ordered, func() {
	var discoverySuccessful bool
	var nodeSelector map[string]string
	var discoveryFailedReason string
	BeforeAll(func() {
		isSNO, err := nodes.IsSingleNodeCluster()
		Expect(err).ToNot(HaveOccurred())
		if isSNO {
			disableDrainState, err := sriovcluster.GetNodeDrainState(sriovclient, namespaces.SRIOVOperator)
			Expect(err).ToNot(HaveOccurred())
			if !disableDrainState {
				err = sriovcluster.SetDisableNodeDrainState(sriovclient, namespaces.SRIOVOperator, true)
				Expect(err).ToNot(HaveOccurred())
				sriovClean.RestoreNodeDrainState = true
			}
		}
		nodeSelector, _ = nodes.PodLabelSelector()

		// This namespace is required for the DiscoverSriov function as it start a pod
		// to check if secure boot is enable on that node
		err = namespaces.Create(sriovnamespaces.Test, client.Client)
		Expect(err).ToNot(HaveOccurred())

		if discovery.Enabled() {
			var performanceProfiles []*performancev2.PerformanceProfile
			discoverySuccessful, discoveryFailedReason, performanceProfiles = performanceprofile.DiscoverPerformanceProfiles(enforcedPerformanceProfileName)

			if !discoverySuccessful {
				discoveryFailedReason = "Could not find a valid performance profile"
				return
			}

			discovered, err := discovery.DiscoverPerformanceProfileAndPolicyWithAvailableNodes(client.Client, sriovclient, namespaces.SRIOVOperator, dpdkResourceName, performanceProfiles, nodeSelector)
			if err != nil {
				discoverySuccessful, discoveryFailedReason = false, "Can not run tests in discovery mode. Failed to discover required resources."
				return
			}
			profile = discovered.Profile
			nodeSelector = nodes.SelectorUnion(nodeSelector, profile.Spec.NodeSelector)
		} else {
			err = performanceprofile.FindOrOverridePerformanceProfile(performanceProfileName, machineConfigPoolName)
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Get(context.TODO(), apitypes.NamespacedName{Name: performanceProfileName}, profile)
			Expect(err).ToNot(HaveOccurred())
		}
		By("CleanSriov...")
		networks.CleanSriov(sriovclient)

		By("CreateSriovPolicyAndNetwork...")
		networks.CreateSriovPolicyAndNetwork(sriovclient, namespaces.SRIOVOperator, sriovNetworkName, "testresource", "")

		By("Checking the SRIOV network-attachment-definition is ready")
		Eventually(func() error {
			nad := netattdefv1.NetworkAttachmentDefinition{}
			objKey := apitypes.NamespacedName{
				Namespace: namespaces.SRIOVOperator,
				Name:      sriovNetworkName,
			}
			err := client.Client.Get(context.Background(), objKey, &nad)
			return err
		}).WithTimeout(2 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
	})
	BeforeEach(func() {
		if discovery.Enabled() && !discoverySuccessful {
			Skip(discoveryFailedReason)
		}
	})
	Context("rps configuration with performance profile applied", func() {
		workerRTNodes, err := ntonodes.GetByLabels(profile.Spec.NodeSelector)
		Expect(err).ToNot(HaveOccurred())
		BeforeEach(func() {
			if profile.Spec.CPU == nil || profile.Spec.CPU.Reserved == nil {
				Skip("Test Skipped due nil Reserved cpus")
			}
		})
		It("check RPS Mask is applied to at least one single rx queue on all veth interface", func() {
			if profile.Spec.WorkloadHints != nil && profile.Spec.WorkloadHints.RealTime != nil &&
				!*profile.Spec.WorkloadHints.RealTime && !profileutil.IsRpsEnabled(profile) {
				Skip("realTime Workload Hints is not enabled")
			}
			count := 0
			expectedRPSCPUs, err := cpuset.Parse(string(*profile.Spec.CPU.Reserved))
			Expect(err).ToNot(HaveOccurred())

			for _, node := range workerRTNodes {
				var vethInterfaces = []string{}
				allInterfaces, err := ntonodes.GetNodeInterfaces(node)
				Expect(err).ToNot(HaveOccurred())
				Expect(allInterfaces).ToNot(BeEmpty())
				// collect all veth interfaces in a list
				for _, iface := range allInterfaces {
					if iface.Bridge == true && iface.Physical == false {
						vethInterfaces = append(vethInterfaces, iface.Name)
					}
				}
				//iterate over all the veth interfaces and
				//check if at least on single rx-queue has rps mask
				klog.Infof("%v", vethInterfaces)
				for _, vethinterface := range vethInterfaces {
					devicePath := path.Join("/rootfs/sys/devices/virtual/net", vethinterface)
					getRPSMaskCmd := []string{"find", devicePath, "-type", "f", "-name", "rps_cpus", "-exec", "cat", "{}", ";"}
					devsRPS, err := ntonodes.ExecCommandOnNode(getRPSMaskCmd, &node)
					Expect(err).ToNot(HaveOccurred())
					for _, devRPS := range strings.Split(devsRPS, "\n") {
						rpsCPUs, err := components.CPUMaskToCPUSet(devRPS)
						Expect(err).ToNot(HaveOccurred())
						if rpsCPUs.String() == string(*profile.Spec.CPU.Reserved) {
							count += 1
						}
					}
					Expect(count > 0).To(BeTrue(), "Not a single receive queues have cpu mask %v", expectedRPSCPUs)
				}
			}
		})
		It("should have the correct RPS configuration", func() {
			if profile.Spec.WorkloadHints != nil && profile.Spec.WorkloadHints.RealTime != nil &&
				!*profile.Spec.WorkloadHints.RealTime && !profileutil.IsRpsEnabled(profile) {
				Skip("realTime Workload Hints is not enabled")
			}

			expectedRPSCPUs, err := cpuset.Parse(string(*profile.Spec.CPU.Reserved))
			Expect(err).ToNot(HaveOccurred())
			expectedRPSCPUsMask, err := components.CPUListToMaskList(expectedRPSCPUs.String())
			Expect(err).ToNot(HaveOccurred())
			klog.Infof("expected RPS CPU mask for virtual network devices=%q", expectedRPSCPUsMask)

			expectedPhysRPSCPUs := expectedRPSCPUs.Clone()
			expectedPhyRPSCPUsMask := expectedRPSCPUsMask
			if !profileutil.IsPhysicalRpsEnabled(profile) {
				// empty cpuset
				expectedPhysRPSCPUs = cpuset.New()
				expectedPhyRPSCPUsMask, err = components.CPUListToMaskList(expectedPhysRPSCPUs.String())
				Expect(err).ToNot(HaveOccurred())
				klog.Infof("physical RPS disabled, expected RPS CPU mask for physical network devices is=%q", expectedPhyRPSCPUsMask)
			} else {
				klog.Infof("physical RPS enabled, expected RPS CPU mask for physical network devices is=%q", expectedRPSCPUsMask)
			}

			for _, node := range workerRTNodes {
				By("verify the systemd RPS service uses the correct RPS mask")
				cmd := []string{"sysctl", "-n", "net.core.rps_default_mask"}
				rpsMaskContent, err := ntonodes.ExecCommandOnNode(cmd, &node)
				Expect(err).ToNot(HaveOccurred(), "failed to exec command %q on node %q", cmd, node)
				rpsMaskContent = strings.TrimSuffix(rpsMaskContent, "\n")
				rpsCPUs, err := components.CPUMaskToCPUSet(rpsMaskContent)
				Expect(err).ToNot(HaveOccurred(), "failed to parse RPS mask %q", rpsMaskContent)
				Expect(rpsCPUs.Equals(expectedRPSCPUs)).To(BeTrue(), "the default rps mask is different from the reserved CPUs mask; have %q want %q", rpsMaskContent, expectedRPSCPUsMask)

				By("verify RPS mask on virtual network devices")
				cmd = []string{
					"find", "/rootfs/sys/devices/virtual/net",
					"-path", "/rootfs/sys/devices/virtual/net/lo",
					"-prune", "-o",
					"-type", "f",
					"-name", "rps_cpus",
					"-printf", "%p ",
					"-exec", "cat", "{}", ";",
				}
				devsRPSContent, err := ntonodes.ExecCommandOnNode(cmd, &node)
				Expect(err).ToNot(HaveOccurred(), "failed to exec command %q on node %q", cmd, node.Name)
				devsRPSMap := makeDevRPSMap(devsRPSContent)
				for path, mask := range devsRPSMap {
					rpsCPUs, err = components.CPUMaskToCPUSet(mask)
					Expect(err).ToNot(HaveOccurred())
					Expect(rpsCPUs.Equals(expectedRPSCPUs)).To(BeTrue(),
						"a host virtual device: %q rps mask is different from the reserved CPUs; have %q want %q", path, mask, expectedRPSCPUsMask)
				}

				By("verify RPS mask on physical network devices")
				cmd = []string{
					"find", "/rootfs/sys/devices",
					"-regex", "/rootfs/sys/devices/pci.*",
					"-type", "f",
					"-name", "rps_cpus",
					"-printf", "%p ",
					"-exec", "cat", "{}", ";",
				}
				devsRPSContent, err = ntonodes.ExecCommandOnNode(cmd, &node)
				Expect(err).ToNot(HaveOccurred(), "failed to exec command %q on node %q", cmd, node.Name)

				devsRPSMap = makeDevRPSMap(devsRPSContent)
				for path, mask := range devsRPSMap {
					rpsCPUs, err = components.CPUMaskToCPUSet(mask)
					Expect(err).ToNot(HaveOccurred())
					Expect(rpsCPUs.Equals(expectedPhysRPSCPUs)).To(BeTrue(), "a host physical device: %q rps mask is different than expected; have %q want %q", path, mask, expectedPhyRPSCPUsMask)
				}
			}
		})
		It("[test_id:54190] Should not have RPS configuration set when realtime workload hint is explicitly set", func() {
			if profile.Spec.WorkloadHints != nil && profile.Spec.WorkloadHints.RealTime != nil &&
				!*profile.Spec.WorkloadHints.RealTime && !profileutil.IsRpsEnabled(profile) {
				for _, node := range workerRTNodes {
					// Verify the systemd RPS services were not created
					cmd := []string{"ls", "/rootfs/etc/systemd/system/update-rps@.service"}
					_, err := ntonodes.ExecCommandOnNode(cmd, &node)
					Expect(err).To(HaveOccurred(), fmt.Sprintf("failed to exec command %q on node %q", cmd, node.Name))
				}
			}
		})
	})
})

// makeDevRPSMap converts the find command output where each line has the following pattern:
// '/rootfs/sys/devices/virtual/net/<dev-id>/queues/rx-<queue-number>/rps_cpus <rps-mask>'
// into a map of devices with their corresponding rps mask
func makeDevRPSMap(content string) map[string]string {
	devRPSMap := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		s := strings.Split(line, " ")
		path, mask := s[0], s[1]
		devRPSMap[path] = mask
	}
	return devRPSMap
}
