package ptp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	ptpv1 "github.com/openshift/ptp-operator/pkg/apis/ptp/v1"
	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	_ETHTOOL_HARDWARE_RECEIVE_CAP   = "hardware-receive"
	_ETHTOOL_HARDWARE_TRANSMIT_CAP  = "hardware-transmit"
	_ETHTOOL_HARDWARE_RAW_CLOCK_CAP = "hardware-raw-clock"
	_ETHTOOL_RX_HARDWARE_FLAG       = "(SOF_TIMESTAMPING_RX_HARDWARE)"
	_ETHTOOL_TX_HARDWARE_FLAG       = "(SOF_TIMESTAMPING_TX_HARDWARE)"
	_ETHTOOL_RAW_HARDWARE_FLAG      = "(SOF_TIMESTAMPING_RAW_HARDWARE)"
	ptpLinuxDaemonNamespace         = "openshift-ptp"
	ptpOperatorDeploymentName       = "ptp-operator"
	ptpSlaveNodeLabel               = "ptp/slave"
	ptpGrandmasterNodeLabel         = "ptp/grandmaster"
	ptpResourcesGroupVersionPrefix  = "ptp.openshift.io/v"
	ptpResourcesNameOperatorConfigs = "ptpoperatorconfigs"
	nodePtpDeviceAPIPath            = "/apis/ptp.openshift.io/v1/namespaces/openshift-ptp/nodeptpdevices/"
	configPtpAPIPath                = "/apis/ptp.openshift.io/v1/namespaces/openshift-ptp/ptpconfigs"
)

var _ = Describe("ptp", func() {

	var _ = Context("PTP configuration verifications", func() {
		// Setup verification
		It("Should check whether PTP operator appropriate resource exists", func() {
			By("Getting list of available resources")
			rl, err := client.Client.ServerPreferredResources()
			Expect(err).ToNot(HaveOccurred())

			found := false
			By("Find appropriate resources")
			for _, g := range rl {
				if strings.Contains(g.GroupVersion, ptpResourcesGroupVersionPrefix) {
					for _, r := range g.APIResources {
						By("Search for resource " + ptpResourcesNameOperatorConfigs)
						if r.Name == ptpResourcesNameOperatorConfigs {
							found = true
						}
					}
				}
			}

			Expect(found).To(BeTrue(), fmt.Sprintf("resource %s not found", ptpResourcesNameOperatorConfigs))
		})
		// Setup verification
		It("Should check that all nodes are running at least one replica of linuxptp-daemon", func() {
			By("Getting list of nodes")
			nodes, err := client.Client.Nodes().List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			By("Checking number of nodes")
			Expect(len(nodes.Items)).To(BeNumerically(">", 0), "number of nodes should be more than 0")

			By("Get daemonsets collection for the namespace " + ptpLinuxDaemonNamespace)
			ds, err := client.Client.DaemonSets(ptpLinuxDaemonNamespace).List(metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(ds.Items)).To(BeNumerically(">", 0), "no damonsets found in the namespace "+ptpLinuxDaemonNamespace)
			By("Checking number of scheduled instances")
			Expect(ds.Items[0].Status.CurrentNumberScheduled).To(BeNumerically("==", len(nodes.Items)), "should be one instance per node")
		})
		// Setup verification
		It("Should check that operator is deployed", func() {
			By("Getting deployment " + ptpOperatorDeploymentName)
			dep, err := client.Client.Deployments(ptpLinuxDaemonNamespace).Get(ptpOperatorDeploymentName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Checking availability of the deployment")
			for _, c := range dep.Status.Conditions {
				if c.Type == v1.DeploymentAvailable {
					Expect(string(c.Status)).Should(Equal("True"), ptpOperatorDeploymentName+" deployment is not available")
				}
			}
		})
	})
	var _ = Describe("PTP e2e tests", func() {
		var ptpRunningPods []v1core.Pod
		var _ = Context("PTP Interfaces discovery", func() {

			BeforeEach(func() {
				ptpRunningPods = []v1core.Pod{}
				ptpPods, err := client.Client.Pods(ptpLinuxDaemonNamespace).List(metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).NotTo(HaveOccurred())
				if len(ptpPods.Items) < 2 {
					Skip("Skipping there is no enough ptp Pods deployed on Nodes. Check number of available nodes or LabelSelector")
				}
				for _, pod := range ptpPods.Items {
					if podRole(pod, ptpSlaveNodeLabel) || podRole(pod, ptpGrandmasterNodeLabel) {
						waitUntilLogIsDetected(pod, 3*time.Minute, "PTP capable NICs")
						ptpRunningPods = append(ptpRunningPods, pod)
					}
				}
				Expect(len(ptpRunningPods)).To(BeNumerically(">", 0), fmt.Sprint("Fail to detect PTP slave/master pods on Cluster"))
			})

			// 25729
			It("The interfaces support ptp can be discovered correctly", func() {

				for _, pod := range ptpRunningPods {
					ptpSupportedInt := getPtpMasterSlaveAttachedInterfaces(pod)
					Expect(len(ptpSupportedInt)).To(BeNumerically(">", 0), fmt.Sprint("Fail to detect PTP Supported interfaces on slave/master pods"))
					ptpDiscoveredInterfaces := ptpDiscoveredInterfaceList(nodePtpDeviceAPIPath + pod.Spec.NodeName)
					Expect(len(ptpSupportedInt)).To(BeNumerically("==", len(ptpDiscoveredInterfaces)), fmt.Sprint("The interfaces discovered incorrectly"))
					for _, intfc := range ptpSupportedInt {
						Expect(ptpDiscoveredInterfaces).To(ContainElement(intfc))
					}
				}
			})

			// 25730
			It("The virtual interfaces should be not discovered by ptp", func() {

				for _, pod := range ptpRunningPods {
					ptpNotSupportedInt := getNonPtpMasterSlaveAttachedInterfaces(pod)
					ptpDiscoveredInterfaces := ptpDiscoveredInterfaceList(nodePtpDeviceAPIPath + pod.Spec.NodeName)
					for _, inter := range ptpNotSupportedInt {
						Expect(ptpDiscoveredInterfaces).ToNot(ContainElement(inter), fmt.Sprint("The interfaces discovered incorrectly. PTP non supported Interfaces in list"))
					}
				}
			})

			// 25859
			It("PTP discovery should exclude default network interface of nodes", func() {
				for _, pod := range ptpRunningPods {
					var defaultRouteInterface string

					Eventually(func() error {
						var err error
						defaultRouteInterface, err = pods.DetectDefaultRouteInterface(client.Client, pod)
						if Expect(defaultRouteInterface).ShouldNot(BeEmpty()) {
							return nil
						}
						return err
					}, 1*time.Minute, 1*time.Second).Should(BeNil(), fmt.Sprint("Default route interface is not detected"))

					discoveredInterfaces := ptpDiscoveredInterfaceList(nodePtpDeviceAPIPath + pod.Spec.NodeName)
					Expect(len(discoveredInterfaces)).To(BeNumerically(">", 0), fmt.Sprint("Fail to detect PTP Supported interfaces on slave/master pods"))
					Expect(discoveredInterfaces).ToNot(ContainElement(defaultRouteInterface), fmt.Sprint("The interfaces discovered incorrectly. Default Host Interfaces in list"))
					podLogs, err := pods.GetLog(&pod)
					Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)
					testFailed := false
					for _, line := range strings.Split(podLogs, "\n") {
						if strings.Contains(line, "PTP capable NICs") && strings.Contains(line, defaultRouteInterface) {
							testFailed = true
						}
					}
					Expect(testFailed).To(BeFalse(), fmt.Sprint("The interfaces discovered incorrectly. Default Host Interfaces in Pod's logs"))
				}
			})

			// 25733
			It("PTP daemon apply match rule based on nodeLabel", func() {
				profileSlave := "Profile Name: slave"
				profileMaster := "Profile Name: grandmaster"
				for _, pod := range ptpRunningPods {
					podLogs, err := pods.GetLog(&pod)
					Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)
					if podRole(pod, ptpSlaveNodeLabel) {
						Expect(podLogs).Should(ContainSubstring(profileSlave),
							fmt.Sprintf("Profile \"%s\" not found in pod's log %s", profileSlave, pod.Name))
					}
					if podRole(pod, ptpGrandmasterNodeLabel) {
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
					if podRole(pod, ptpGrandmasterNodeLabel) {
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
					if podRole(pod, ptpSlaveNodeLabel) {
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
		})
	})
	var _ = Describe("Test Offset", func() {
		BeforeEach(func() {
			nodes, err := client.Client.Nodes().List(metav1.ListOptions{
				LabelSelector: ptpSlaveNodeLabel,
			})
			Expect(err).ToNot(HaveOccurred())
			if len(nodes.Items) < 2 {
				Skip(fmt.Sprintf("PTP Nodes with label %s are not deployed on cluster", ptpSlaveNodeLabel))
			}
		})

		var _ = Context("PTP configuration verifications", func() {

			// 27324
			It("PTP time diff between Grandmaster and Slave should be in range -100ms and 100ms", func() {
				var timeDiff string
				ptpPods, err := client.Client.Pods(ptpLinuxDaemonNamespace).List(metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ptpPods.Items)).To(BeNumerically(">", 0))
				slavePodDetected := false
				for _, pod := range ptpPods.Items {
					if podRole(pod, ptpSlaveNodeLabel) {
						Eventually(func() string {
							buf, _ := pods.ExecCommand(client.Client, pod, []string{"curl", "127.0.0.1:9091/metrics"})
							timeDiff = buf.String()
							return timeDiff
						}, 3*time.Minute, 2*time.Second).Should(ContainSubstring("openshift_ptp_max_offset_from_master"),
							fmt.Sprint("Time metrics are not detected"))
						Expect(compareOffsetTime(timeDiff)).ToNot(BeFalse(), "Offset is not in acceptable range")
						slavePodDetected = true
					}
				}
				Expect(slavePodDetected).ToNot(BeFalse(), "No slave pods detected")
			})
		})
	})
	var _ = Describe("Test NodeName Selector", func() {
		var testPtpPod v1core.Pod
		var ptpConfigSlave ptpv1.PtpConfig

		BeforeEach(func() {
			ptpConfigName := "test"

			response, err := client.Client.ConfigV1Interface.RESTClient().Get().AbsPath(configPtpAPIPath + "/slave").DoRaw()
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal(response, &ptpConfigSlave)
			Expect(err).NotTo(HaveOccurred())
			nodes, err := client.Client.Nodes().List(metav1.ListOptions{
				LabelSelector: ptpSlaveNodeLabel,
			})
			Expect(err).NotTo(HaveOccurred())

			if len(nodes.Items) < 2 {
				Skip(fmt.Sprintf("PTP Nodes with label %s are not deployed on cluster", ptpSlaveNodeLabel))
			}

			ptpConfigTest := mutateProfile(ptpConfigSlave, ptpConfigName, nodes.Items[0].Name)

			status := client.Client.ConfigV1Interface.RESTClient().Post().AbsPath(configPtpAPIPath).
				Resource("ptpconfigs").Body(ptpConfigTest).Context(context.TODO()).Do()
			Expect(status.Error()).NotTo(HaveOccurred(), fmt.Sprint("PTP config creation Error"))

			testPtpPod, err = getPtpPodOnNode(nodes.Items[0].Name)
			Expect(err).NotTo(HaveOccurred())

			testPtpPod, err = getPtpPodOnNode(nodes.Items[0].Name)
			Expect(err).NotTo(HaveOccurred())

			testPtpPod, err = replaceTestPod(testPtpPod, time.Minute)
			Expect(err).NotTo(HaveOccurred())
		})

		var _ = Context("Check if Node has Profile", func() {
			//25743
			It("PTP daemon can apply match rule based on nodeLabel", func() {
				waitUntilLogIsDetected(testPtpPod, 3*time.Minute, "Profile Name: test")
			})
		})
		AfterEach(func() {
			status := client.Client.ConfigV1Interface.RESTClient().Delete().AbsPath(configPtpAPIPath + "/test").Do()
			Expect(status.Error()).NotTo(HaveOccurred(), fmt.Sprint("Can not delete PTP config"))

			Expect(status.Error()).NotTo(HaveOccurred(), fmt.Sprint("PTP slave config recovery Error"))
			waitUntilLogIsDetected(testPtpPod, 3*time.Minute, "Profile Name: slave")
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

func compareOffsetTime(timeDiff string) bool {
	var timeStampList []int
	for _, line := range strings.Split(timeDiff, "\n") {
		if strings.Contains(line, "openshift_ptp_max_offset_from_master") && !strings.Contains(line, "# ") {
			lineValues := strings.Split(line, " ")
			lastValue := strings.Trim(lineValues[len(lineValues)-1], "\r")
			offsetFromMaster, err := strconv.Atoi(lastValue)
			Expect(err).ToNot(HaveOccurred())
			Expect(offsetFromMaster).To(BeNumerically("<", 100))
			Expect(offsetFromMaster).To(BeNumerically(">", -100))
			timeStampList = append(timeStampList, offsetFromMaster)
		}
	}
	Expect(len(timeStampList)).To(BeNumerically("==", 2))
	return true
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
		if parts[0] == _ETHTOOL_HARDWARE_RECEIVE_CAP {
			RxEnabled = parts[1] == _ETHTOOL_RX_HARDWARE_FLAG
		}
		if parts[0] == _ETHTOOL_HARDWARE_TRANSMIT_CAP {
			TxEnabled = parts[1] == _ETHTOOL_TX_HARDWARE_FLAG
		}
		if parts[0] == _ETHTOOL_HARDWARE_RAW_CLOCK_CAP {
			RawEnabled = parts[1] == _ETHTOOL_RAW_HARDWARE_FLAG
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
	mutatedConfig.ObjectMeta.Namespace = ptpLinuxDaemonNamespace
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
	runningPod, err := client.Client.Pods(ptpLinuxDaemonNamespace).List(metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
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
		buf, err := pods.ExecCommand(client.Client, pod, []string{"ls", "/sys/class/net/"})
		if err != nil {
			return err
		}

		IntList = strings.Split(strings.Join(strings.Fields(buf.String()), " "), " ")
		if len(IntList) == 0 {
			return fmt.Errorf("No interface detected")
		}

		return nil
	}, 3*time.Minute, 2*time.Second).Should(BeNil())

	return IntList
}

func getPtpMasterSlaveAttachedInterfaces(pod v1core.Pod) []string {
	var ptpSupportedInterfaces []string
	var buf bytes.Buffer

	intList := getMasterSlaveAttachedInterfaces(pod)
	for _, interf := range intList {

		Eventually(func() error {
			var err error
			buf, err = pods.ExecCommand(client.Client, pod, []string{"ethtool", "-T", interf})
			if err != nil {
				return err
			}
			return nil
		}, 2*time.Minute, 1*time.Second).Should(BeNil())

		if isPTPEnabled(&buf) {
			ptpSupportedInterfaces = append(ptpSupportedInterfaces, interf)
		}
	}
	return ptpSupportedInterfaces
}

func replaceTestPod(pod v1core.Pod, timeout time.Duration) (v1core.Pod, error) {
	var newPod v1core.Pod

	err := client.Client.Pods(ptpLinuxDaemonNamespace).Delete(pod.Name, &metav1.DeleteOptions{
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
	var buf bytes.Buffer

	intList := getMasterSlaveAttachedInterfaces(pod)
	for _, interf := range intList {

		Eventually(func() error {
			var err error
			buf, err = pods.ExecCommand(client.Client, pod, []string{"ethtool", "-T", interf})
			if err != nil {
				return err
			}
			return nil
		}, 2*time.Minute, 1*time.Second).Should(BeNil())
		if isPTPEnabled(&buf) == false {
			ptpSupportedInterfaces = append(ptpSupportedInterfaces, interf)
		}
		time.Sleep(time.Second)
	}
	return ptpSupportedInterfaces
}
