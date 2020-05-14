package ptp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	ptpv1 "github.com/openshift/ptp-operator/pkg/apis/ptp/v1"

	. "github.com/openshift/ptp-operator/test/utils"
	"github.com/openshift/ptp-operator/test/utils/client"
	testclient "github.com/openshift/ptp-operator/test/utils/client"
	"github.com/openshift/ptp-operator/test/utils/execute"
	"github.com/openshift/ptp-operator/test/utils/nodes"
	"github.com/openshift/ptp-operator/test/utils/pods"
)

var _ = Describe("[ptp]", func() {
	BeforeEach(func() {
		Expect(testclient.Client).NotTo(BeNil())
	})

	Context("PTP configuration verifications", func() {
		// Setup verification
		It("Should check whether PTP operator appropriate resource exists", func() {
			By("Getting list of available resources")
			rl, err := client.Client.ServerPreferredResources()
			Expect(err).ToNot(HaveOccurred())

			found := false
			By("Find appropriate resources")
			for _, g := range rl {
				if strings.Contains(g.GroupVersion, PtpResourcesGroupVersionPrefix) {
					for _, r := range g.APIResources {
						By("Search for resource " + PtpResourcesNameOperatorConfigs)
						if r.Name == PtpResourcesNameOperatorConfigs {
							found = true
						}
					}
				}
			}

			Expect(found).To(BeTrue(), fmt.Sprintf("resource %s not found", PtpResourcesNameOperatorConfigs))
		})
		// Setup verification
		It("Should check that all nodes are running at least one replica of linuxptp-daemon", func() {
			By("Getting list of nodes")
			nodes, err := client.Client.Nodes().List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			By("Checking number of nodes")
			Expect(len(nodes.Items)).To(BeNumerically(">", 0), "number of nodes should be more than 0")

			By("Get daemonsets collection for the namespace " + PtpLinuxDaemonNamespace)
			ds, err := client.Client.DaemonSets(PtpLinuxDaemonNamespace).List(metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(ds.Items)).To(BeNumerically(">", 0), "no damonsets found in the namespace "+PtpLinuxDaemonNamespace)
			By("Checking number of scheduled instances")
			Expect(ds.Items[0].Status.CurrentNumberScheduled).To(BeNumerically("==", len(nodes.Items)), "should be one instance per node")
		})
		// Setup verification
		It("Should check that operator is deployed", func() {
			By("Getting deployment " + PtpOperatorDeploymentName)
			dep, err := client.Client.Deployments(PtpLinuxDaemonNamespace).Get(PtpOperatorDeploymentName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Checking availability of the deployment")
			for _, c := range dep.Status.Conditions {
				if c.Type == v1.DeploymentAvailable {
					Expect(string(c.Status)).Should(Equal("True"), PtpOperatorDeploymentName+" deployment is not available")
				}
			}
		})
	})

	Describe("PTP e2e tests", func() {
		var ptpRunningPods []v1core.Pod

		execute.BeforeAll(func() {
			ptpconfigList, err := client.Client.PtpConfigs(PtpLinuxDaemonNamespace).List(metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			for _, ptpConfig := range ptpconfigList.Items {
				if ptpConfig.Name == PtpGrandMasterPolicyName || ptpConfig.Name == PtpSlavePolicyName {
					err = client.Client.PtpConfigs(PtpLinuxDaemonNamespace).Delete(ptpConfig.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			}

			nodeList, err := client.Client.Nodes().List(metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", PtpGrandmasterNodeLabel)})
			Expect(err).ToNot(HaveOccurred())
			for _, node := range nodeList.Items {
				delete(node.Labels, PtpGrandmasterNodeLabel)
				_, err = client.Client.Nodes().Update(&node)
				Expect(err).ToNot(HaveOccurred())
			}

			nodeList, err = client.Client.Nodes().List(metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", PtpSlaveNodeLabel)})
			Expect(err).ToNot(HaveOccurred())
			for _, node := range nodeList.Items {
				delete(node.Labels, PtpSlaveNodeLabel)
				_, err = client.Client.Nodes().Update(&node)
				Expect(err).ToNot(HaveOccurred())
			}

			ptpNodes, err := nodes.GetNodeTopology(client.Client)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(ptpNodes)).To(BeNumerically(">", 1), "need at least two nodes with ptp capable nics")

			By("Labeling the grandmaster node")
			ptpGrandMasterNode := ptpNodes[0]
			ptpGrandMasterNode.NodeObject, err = nodes.LabelNode(ptpGrandMasterNode.NodeName, PtpGrandmasterNodeLabel, "")
			Expect(err).ToNot(HaveOccurred())

			By("Labeling the slave node")
			ptpSlaveNode := ptpNodes[1]
			ptpSlaveNode.NodeObject, err = nodes.LabelNode(ptpSlaveNode.NodeName, PtpSlaveNodeLabel, "")
			Expect(err).ToNot(HaveOccurred())

			By("Creating the policy for the grandmaster node")
			err = createConfig(PtpGrandMasterPolicyName,
				ptpGrandMasterNode.InterfaceList[0],
				"",
				"-a -r -r",
				PtpGrandmasterNodeLabel,
				pointer.Int64Ptr(5))
			Expect(err).ToNot(HaveOccurred())

			By("Creating the policy for the slave node")
			err = createConfig(PtpSlavePolicyName,
				ptpSlaveNode.InterfaceList[0],
				"-s",
				"-a -r",
				PtpSlaveNodeLabel,
				pointer.Int64Ptr(5))
			Expect(err).ToNot(HaveOccurred())

			By("Restart the linuxptp-daemon pods")
			ptpPods, err := client.Client.Pods(PtpLinuxDaemonNamespace).List(metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
			Expect(err).ToNot(HaveOccurred())
			for _, pod := range ptpPods.Items {
				err = client.Client.Pods(PtpLinuxDaemonNamespace).Delete(pod.Name, &metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
				Expect(err).ToNot(HaveOccurred())
			}

			daemonset, err := client.Client.DaemonSets(PtpLinuxDaemonNamespace).Get(PtpDaemonsetName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			expectedNumber := daemonset.Status.DesiredNumberScheduled
			Eventually(func() int32 {
				daemonset, err = client.Client.DaemonSets(PtpLinuxDaemonNamespace).Get(PtpDaemonsetName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return daemonset.Status.NumberReady
			}, 2*time.Minute, 2*time.Second).Should(Equal(expectedNumber))

			Eventually(func() int {
				ptpPods, err := client.Client.Pods(PtpLinuxDaemonNamespace).List(metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).ToNot(HaveOccurred())
				return len(ptpPods.Items)
			}, 2*time.Minute, 2*time.Second).Should(Equal(int(expectedNumber)))
		})

		Context("PTP Interfaces discovery", func() {
			BeforeEach(func() {
				ptpRunningPods = []v1core.Pod{}
				ptpPods, err := client.Client.Pods(PtpLinuxDaemonNamespace).List(metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(ptpPods.Items)).To(BeNumerically(">", 0), fmt.Sprint("linuxptp-daemon is not deployed on cluster"))
				for _, pod := range ptpPods.Items {
					if podRole(pod, PtpSlaveNodeLabel) || podRole(pod, PtpGrandmasterNodeLabel) {
						waitUntilLogIsDetected(pod, 3*time.Minute, "Profile Name:")
						ptpRunningPods = append(ptpRunningPods, pod)
					}
				}
				Expect(len(ptpRunningPods)).To(BeNumerically(">=", 2), fmt.Sprint("Fail to detect PTP slave/master pods on Cluster"))
			})

			// 25729
			It("The interfaces support ptp can be discovered correctly", func() {
				for _, pod := range ptpRunningPods {
					ptpSupportedInt := getPtpMasterSlaveAttachedInterfaces(pod)
					Expect(len(ptpSupportedInt)).To(BeNumerically(">", 0), fmt.Sprint("Fail to detect PTP Supported interfaces on slave/master pods"))
					ptpDiscoveredInterfaces := ptpDiscoveredInterfaceList(NodePtpDeviceAPIPath + pod.Spec.NodeName)
					Expect(len(ptpSupportedInt)).To(Equal(len(ptpDiscoveredInterfaces)), fmt.Sprint("The interfaces discovered incorrectly"))
					for _, intfc := range ptpSupportedInt {
						Expect(ptpDiscoveredInterfaces).To(ContainElement(intfc))
					}
				}
			})

			// 25730
			It("The virtual interfaces should be not discovered by ptp", func() {
				for _, pod := range ptpRunningPods {
					ptpNotSupportedInt := getNonPtpMasterSlaveAttachedInterfaces(pod)
					ptpDiscoveredInterfaces := ptpDiscoveredInterfaceList(NodePtpDeviceAPIPath + pod.Spec.NodeName)
					for _, inter := range ptpNotSupportedInt {
						Expect(ptpDiscoveredInterfaces).ToNot(ContainElement(inter), fmt.Sprint("The interfaces discovered incorrectly. PTP non supported Interfaces in list"))
					}
				}
			})

			// 25733
			It("PTP daemon apply match rule based on nodeLabel", func() {
				profileSlave := "Profile Name: test-slave"
				profileMaster := "Profile Name: test-grandmaster"
				for _, pod := range ptpRunningPods {
					podLogs, err := pods.GetLog(&pod)
					Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)
					if podRole(pod, PtpSlaveNodeLabel) {
						Expect(podLogs).Should(ContainSubstring(profileSlave),
							fmt.Sprintf("Profile \"%s\" not found in pod's log %s", profileSlave, pod.Name))
					}
					if podRole(pod, PtpGrandmasterNodeLabel) {
						Expect(podLogs).Should(ContainSubstring(profileMaster),
							fmt.Sprintf("Profile \"%s\" not found in pod's log %s", profileSlave, pod.Name))
					}
				}
			})

			// 25738
			It("Slave can sync to master", func() {
				var masterID string
				var slaveMasterID string
				grandMaster := "assuming the grand master role"
				for _, pod := range ptpRunningPods {
					if podRole(pod, PtpGrandmasterNodeLabel) {
						podLogs, err := pods.GetLog(&pod)
						Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)
						Expect(podLogs).Should(ContainSubstring(grandMaster),
							fmt.Sprintf("Log message \"%s\" not found in pod's log %s", grandMaster, pod.Name))
						for _, line := range strings.Split(podLogs, "\n") {
							if strings.Contains(line, "selected local clock") && strings.Contains(line, "as best master") {
								masterID = strings.Split(line, " ")[4]
							}
						}
					}
					if podRole(pod, PtpSlaveNodeLabel) {
						podLogs, err := pods.GetLog(&pod)
						Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)
						for _, line := range strings.Split(podLogs, "\n") {
							if strings.Contains(line, "new foreign master") {
								slaveMasterID = strings.Split(line, " ")[6]
							}
						}
					}
				}
				Expect(masterID).NotTo(BeNil())
				Expect(slaveMasterID).NotTo(BeNil())
				Expect(slaveMasterID).Should(HavePrefix(masterID), "Error match MasterID with the SlaveID. Slave connected to another Master")
			})

			//25743
			It("Can provide a profile with higher priority", func() {
				var testPtpPod v1core.Pod

				By("Creating a config with higher priority", func() {
					ptpConfigSlave, err := client.Client.PtpV1Interface.PtpConfigs("openshift-ptp").Get(PtpSlavePolicyName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					nodes, err := client.Client.Nodes().List(metav1.ListOptions{
						LabelSelector: PtpSlaveNodeLabel,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(nodes.Items)).To(BeNumerically(">", 0),
						fmt.Sprintf("PTP Nodes with label %s are not deployed on cluster", PtpSlaveNodeLabel))

					ptpConfigTest := mutateProfile(*ptpConfigSlave, PtpSlavePolicyName, nodes.Items[0].Name)
					_, err = client.Client.PtpV1Interface.PtpConfigs("openshift-ptp").Create(ptpConfigTest)
					Expect(err).NotTo(HaveOccurred())

					testPtpPod, err = getPtpPodOnNode(nodes.Items[0].Name)
					Expect(err).NotTo(HaveOccurred())

					testPtpPod, err = replaceTestPod(testPtpPod, time.Minute)
					Expect(err).NotTo(HaveOccurred())
				})

				By("Checking if Node has Profile", func() {
					waitUntilLogIsDetected(testPtpPod, 3*time.Minute, "Profile Name: test")
				})

				By("Deleting the test profile", func() {
					err := client.Client.PtpV1Interface.PtpConfigs("openshift-ptp").Delete("test", &metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
					Eventually(func() bool {
						_, err := client.Client.PtpV1Interface.PtpConfigs("openshift-ptp").Get("test", metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, 1*time.Minute, 1*time.Second).Should(BeTrue(), "Could not delete the test profile")
				})

				By("Checking the profile is reverted", func() {
					waitUntilLogIsDetected(testPtpPod, 3*time.Minute, "Profile Name: test-slave")
				})
			})
		})
	})

})

func podRole(runningPod v1core.Pod, role string) bool {
	nodeList, err := client.Client.Nodes().List(metav1.ListOptions{
		LabelSelector: role,
	})
	Expect(err).NotTo(HaveOccurred())
	for NodeNumber := range nodeList.Items {
		if runningPod.Spec.NodeName == nodeList.Items[NodeNumber].Name {
			return true
		}
	}
	return false
}

// This function parses ethtool command output and detect interfaces which supports ptp protocol
func isPTPEnabled(ethToolOutput *bytes.Buffer) bool {
	var RxEnabled bool
	var TxEnabled bool
	var RawEnabled bool

	scanner := bufio.NewScanner(ethToolOutput)
	for scanner.Scan() {
		line := strings.TrimPrefix(scanner.Text(), "\t")
		parts := strings.Fields(line)
		if parts[0] == ETHTOOL_HARDWARE_RECEIVE_CAP {
			RxEnabled = parts[1] == ETHTOOL_RX_HARDWARE_FLAG
		}
		if parts[0] == ETHTOOL_HARDWARE_TRANSMIT_CAP {
			TxEnabled = parts[1] == ETHTOOL_TX_HARDWARE_FLAG
		}
		if parts[0] == ETHTOOL_HARDWARE_RAW_CLOCK_CAP {
			RawEnabled = parts[1] == ETHTOOL_RAW_HARDWARE_FLAG
		}
	}
	return RxEnabled && TxEnabled && RawEnabled
}

func ptpDiscoveredInterfaceList(path string) []string {
	var ptpInterfaces []string
	var nodePtpDevice ptpv1.NodePtpDevice
	fg, err := client.Client.CoreV1Interface.RESTClient().Get().AbsPath(path).DoRaw()
	Expect(err).ToNot(HaveOccurred())

	err = json.Unmarshal(fg, &nodePtpDevice)
	Expect(err).ToNot(HaveOccurred())

	for _, intConf := range nodePtpDevice.Status.Devices {
		ptpInterfaces = append(ptpInterfaces, intConf.Name)
	}
	return ptpInterfaces
}

func mutateProfile(profile ptpv1.PtpConfig, profileName string, nodeName string) *ptpv1.PtpConfig {
	mutatedConfig := profile.DeepCopy()
	priority := int64(0)
	mutatedConfig.ObjectMeta.Reset()
	mutatedConfig.ObjectMeta.Name = "test"
	mutatedConfig.ObjectMeta.Namespace = PtpLinuxDaemonNamespace
	mutatedConfig.Spec.Profile[0].Name = &profileName
	mutatedConfig.Spec.Recommend[0].Priority = &priority
	mutatedConfig.Spec.Recommend[0].Match[0].NodeLabel = nil
	mutatedConfig.Spec.Recommend[0].Match[0].NodeName = &nodeName
	mutatedConfig.Spec.Recommend[0].Profile = &profileName
	return mutatedConfig
}

func waitUntilLogIsDetected(pod v1core.Pod, timeout time.Duration, neededLog string) {
	Eventually(func() string {
		logs, _ := pods.GetLog(&pod)
		return logs
	}, timeout, 1*time.Second).Should(ContainSubstring(neededLog), fmt.Sprintf("Timeout to detect log \"%s\" in pod \"%s\"", neededLog, pod.Name))
}

func getPtpPodOnNode(nodeName string) (v1core.Pod, error) {
	runningPod, err := client.Client.Pods(PtpLinuxDaemonNamespace).List(metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	Expect(err).NotTo(HaveOccurred(), fmt.Sprint("Error to get list of pods by label: app=linuxptp-daemon"))
	Expect(len(runningPod.Items)).To(BeNumerically(">", 0), fmt.Sprint("PTP pods are  not deployed on cluster"))
	for _, pod := range runningPod.Items {

		if pod.Spec.NodeName == nodeName {
			return pod, nil
		}
	}
	return v1core.Pod{}, fmt.Errorf("Pod not found")
}

func getMasterSlaveAttachedInterfaces(pod v1core.Pod) []string {
	var IntList []string
	Eventually(func() error {
		stdout, err := pods.ExecCommand(client.Client, pod, []string{"ls", "/sys/class/net/"})
		if err != nil {
			return err
		}

		if stdout.String() == "" {
			return fmt.Errorf("empty response from pod retrying")
		}

		IntList = strings.Split(strings.Join(strings.Fields(stdout.String()), " "), " ")
		if len(IntList) == 0 {
			return fmt.Errorf("No interface detected")
		}

		return nil
	}, 3*time.Minute, 5*time.Second).Should(BeNil())

	return IntList
}

func getPtpMasterSlaveAttachedInterfaces(pod v1core.Pod) []string {
	var ptpSupportedInterfaces []string
	var stdout bytes.Buffer

	intList := getMasterSlaveAttachedInterfaces(pod)
	for _, interf := range intList {
		skipInterface := false
		PCIAddr := ""
		var err error

		// Get readlink status
		Eventually(func() error {
			stdout, err = pods.ExecCommand(client.Client, pod, []string{"readlink", "-f", fmt.Sprintf("/sys/class/net/%s", interf)})
			if err != nil {
				return err
			}

			if stdout.String() == "" {
				return fmt.Errorf("empty response from pod retrying")
			}

			// Skip virtual interface
			if strings.Contains(stdout.String(), "devices/virtual/net") {
				skipInterface = true
				return nil
			}

			// sysfs address looks like: /sys/devices/pci0000:17/0000:17:02.0/0000:19:00.5/net/eno1
			pathSegments := strings.Split(stdout.String(), "/")
			if len(pathSegments) != 8 {
				skipInterface = true
				return nil
			}

			PCIAddr = pathSegments[5] // 0000:19:00.5
			return nil
		}, 3*time.Minute, 5*time.Second).Should(BeNil())

		if skipInterface || PCIAddr == "" {
			continue
		}

		// Check if this is a virtual function
		Eventually(func() error {
			// If the physfn doesn't exist this means the interface is not a virtual function so we ca add it to the list
			stdout, err = pods.ExecCommand(client.Client, pod, []string{"ls", fmt.Sprintf("/sys/bus/pci/devices/%s/physfn", PCIAddr)})
			if err != nil {
				if strings.Contains(stdout.String(), "No such file or directory") {
					return nil
				}
				return err
			}

			if stdout.String() == "" {
				return fmt.Errorf("empty response from pod retrying")
			}

			// Virtual function
			skipInterface = true
			return nil
		}, 2*time.Minute, 1*time.Second).Should(BeNil())

		if skipInterface {
			continue
		}

		Eventually(func() error {
			stdout, err = pods.ExecCommand(client.Client, pod, []string{"ethtool", "-T", interf})
			if stdout.String() == "" {
				return fmt.Errorf("empty response from pod retrying")
			}

			if err != nil {
				if strings.Contains(stdout.String(), "No such device") {
					skipInterface = true
					return nil
				}
				return err
			}
			return nil
		}, 2*time.Minute, 1*time.Second).Should(BeNil())

		if skipInterface {
			continue
		}

		if isPTPEnabled(&stdout) {
			ptpSupportedInterfaces = append(ptpSupportedInterfaces, interf)
		}
	}
	return ptpSupportedInterfaces
}

func replaceTestPod(pod v1core.Pod, timeout time.Duration) (v1core.Pod, error) {
	var newPod v1core.Pod

	err := client.Client.Pods(PtpLinuxDaemonNamespace).Delete(pod.Name, &metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0)})
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() error {
		newPod, err = getPtpPodOnNode(pod.Spec.NodeName)

		if err == nil && newPod.Name != pod.Name && newPod.Status.Phase == "Running" {
			return nil
		}

		return fmt.Errorf("Can not replace PTP pod")
	}, timeout, 1*time.Second).Should(BeNil())

	return newPod, nil
}

func getNonPtpMasterSlaveAttachedInterfaces(pod v1core.Pod) []string {
	var ptpSupportedInterfaces []string
	var err error
	var stdout bytes.Buffer

	intList := getMasterSlaveAttachedInterfaces(pod)
	for _, interf := range intList {
		Eventually(func() error {
			stdout, err = pods.ExecCommand(client.Client, pod, []string{"ethtool", "-T", interf})
			if err != nil && !strings.Contains(stdout.String(), "No such device") {
				return err
			}
			if stdout.String() == "" {
				return fmt.Errorf("empty response from pod retrying")
			}
			return nil
		}, 3*time.Minute, 2*time.Second).Should(BeNil())

		if strings.Contains(stdout.String(), "No such device") {
			continue
		}

		if !isPTPEnabled(&stdout) {
			ptpSupportedInterfaces = append(ptpSupportedInterfaces, interf)
		}
	}
	return ptpSupportedInterfaces
}

func createConfig(profileName, ifaceName, ptp4lOpts, phc2sysOpts, nodeLabel string, priority *int64) error {
	ptpProfile := ptpv1.PtpProfile{Name: &profileName, Interface: &ifaceName, Phc2sysOpts: &phc2sysOpts, Ptp4lOpts: &ptp4lOpts}
	matchRule := ptpv1.MatchRule{NodeLabel: &nodeLabel}
	ptpRecommend := ptpv1.PtpRecommend{Profile: &profileName, Priority: priority, Match: []ptpv1.MatchRule{matchRule}}
	policy := ptpv1.PtpConfig{ObjectMeta: metav1.ObjectMeta{Name: profileName, Namespace: PtpLinuxDaemonNamespace},
		Spec: ptpv1.PtpConfigSpec{Profile: []ptpv1.PtpProfile{ptpProfile}, Recommend: []ptpv1.PtpRecommend{ptpRecommend}}}

	_, err := client.Client.PtpConfigs(PtpLinuxDaemonNamespace).Create(&policy)
	return err
}
