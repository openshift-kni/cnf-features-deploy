package ptptesthelper

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"

	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ptpv1 "github.com/openshift/ptp-operator/api/v1"
	"github.com/openshift/ptp-operator/test/pkg"
	"github.com/openshift/ptp-operator/test/pkg/client"
	"github.com/openshift/ptp-operator/test/pkg/metrics"
	nodeshelper "github.com/openshift/ptp-operator/test/pkg/nodes"
	"github.com/openshift/ptp-operator/test/pkg/pods"
	"github.com/openshift/ptp-operator/test/pkg/ptphelper"
	"github.com/openshift/ptp-operator/test/pkg/testconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sPriviledgedDs "github.com/test-network-function/privileged-daemonset"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// helper function for old interface discovery test
func TestPtpRunningPods(ptpPods *corev1.PodList) (ptpRunningPods []*corev1.Pod, err error) {
	ptpSlaveRunningPods := []*corev1.Pod{}
	ptpMasterRunningPods := []*corev1.Pod{}
	for podIndex := range ptpPods.Items {
		isClockUnderTestPod, err := pods.PodRole(&ptpPods.Items[podIndex], pkg.PtpClockUnderTestNodeLabel)
		if err != nil {
			logrus.Errorf("could not check clock under test pod role, err: %s", err)
			return ptpRunningPods, errors.Errorf("could not check clock under test pod role, err: %s", err)
		}

		isGrandmaster, err := pods.PodRole(&ptpPods.Items[podIndex], pkg.PtpGrandmasterNodeLabel)
		if err != nil {
			logrus.Errorf("could not check Grandmaster pod role, err: %s", err)
			return ptpRunningPods, errors.Errorf("could not check Grandmaster pod role, err: %s", err)
		}

		if isClockUnderTestPod {
			pods.WaitUntilLogIsDetected(&ptpPods.Items[podIndex], pkg.TimeoutIn3Minutes, "Profile Name:")
			ptpSlaveRunningPods = append(ptpSlaveRunningPods, &ptpPods.Items[podIndex])
		} else if isGrandmaster {
			pods.WaitUntilLogIsDetected(&ptpPods.Items[podIndex], pkg.TimeoutIn3Minutes, "Profile Name:")
			ptpMasterRunningPods = append(ptpMasterRunningPods, &ptpPods.Items[podIndex])
		}
	}
	if testconfig.GlobalConfig.DiscoveredGrandMasterPtpConfig != nil {
		if len(ptpMasterRunningPods) == 0 {
			return ptpRunningPods, errors.Errorf("Fail to detect PTP master pods on Cluster")
		}
		if len(ptpSlaveRunningPods) == 0 {
			return ptpRunningPods, errors.Errorf("Fail to detect PTP slave pods on Cluster")
		}

	} else {
		if len(ptpSlaveRunningPods) == 0 {
			return ptpRunningPods, errors.Errorf("Fail to detect PTP slave pods on Cluster")
		}
	}
	ptpRunningPods = append(ptpRunningPods, ptpSlaveRunningPods...)
	ptpRunningPods = append(ptpRunningPods, ptpMasterRunningPods...)
	return ptpRunningPods, nil
}

// waits for the foreign master to appear in the logs and checks the clock accuracy
func BasicClockSyncCheck(fullConfig testconfig.TestConfig, ptpConfig *ptpv1.PtpConfig, gmID *string) error {
	if gmID != nil {
		logrus.Infof("expected master=%s", *gmID)
	}
	profileName, errProfile := ptphelper.GetProfileName(ptpConfig)

	if fullConfig.PtpModeDesired == testconfig.Discovery {
		// Only for ptp mode == discovery, if errProfile is not nil just log a info message
		if errProfile != nil {
			logrus.Infof("profile name not detected in log (probably because of log rollover)). Remote clock ID will not be printed")
		}
	} else if errProfile != nil {
		// Otherwise, for other non-discovery modes, report an error
		return errors.Errorf("expects errProfile to be nil, errProfile=%s", errProfile)
	}

	label, err := ptphelper.GetLabel(ptpConfig)
	if err != nil {
		logrus.Debugf("could not get label because of err: %s", err)
	}
	nodeName, err := ptphelper.GetFirstNode(ptpConfig)
	if err != nil {
		logrus.Debugf("could not get nodeName because of err: %s", err)
	}
	slaveMaster, err := ptphelper.GetClockIDForeign(profileName, label, nodeName)
	if errProfile == nil {
		if fullConfig.PtpModeDesired == testconfig.Discovery {
			if err != nil {
				logrus.Infof("slave's Master not detected in log (probably because of log rollover))")
			} else {
				logrus.Infof("slave's Master=%s", slaveMaster)
			}
		} else {
			if err != nil {
				return errors.Errorf("expects err to be nil, err=%s", err)
			}
			if slaveMaster == "" {
				return errors.Errorf("expects slaveMaster to not be empty, slaveMaster=%s", slaveMaster)
			}
			logrus.Infof("slave's Master=%s", slaveMaster)
		}
	}
	if gmID != nil {
		if !strings.HasPrefix(slaveMaster, *gmID) {
			return errors.Errorf("Slave connected to another (incorrect) Master, slaveMaster=%s, gmID=%s", slaveMaster, *gmID)
		}
	}

	Eventually(func() error {
		err = metrics.CheckClockRoleAndOffset(ptpConfig, label, nodeName)
		if err != nil {
			logrus.Infof(fmt.Sprintf("CheckClockRoleAndOffset Failed because of err: %s", err))
		}
		return err
	}, pkg.TimeoutIn10Minutes, pkg.Timeout10Seconds).Should(BeNil(), fmt.Sprintf("Timeout to detect metrics for ptpconfig %s", ptpConfig.Name))
	return nil
}

func VerifyAfterRebootState(rebootedNodes []string, fullConfig testconfig.TestConfig) {
	By("Getting ptp operator config")
	ptpConfig, err := client.Client.PtpV1Interface.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpConfigOperatorName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	listOptions := metav1.ListOptions{}
	if ptpConfig.Spec.DaemonNodeSelector != nil && len(ptpConfig.Spec.DaemonNodeSelector) != 0 {
		listOptions = metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: ptpConfig.Spec.DaemonNodeSelector})}
	}

	By("Getting list of nodes")
	nodes, err := client.Client.CoreV1().Nodes().List(context.Background(), listOptions)
	Expect(err).NotTo(HaveOccurred())
	By("Checking number of nodes")
	Expect(len(nodes.Items)).To(BeNumerically(">", 0), "number of nodes should be more than 0")

	By("Get daemonsets collection for the namespace " + pkg.PtpLinuxDaemonNamespace)
	ds, err := client.Client.DaemonSets(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(len(ds.Items)).To(BeNumerically(">", 0), "no damonsets found in the namespace "+pkg.PtpLinuxDaemonNamespace)

	By("Checking number of scheduled instances")
	Expect(ds.Items[0].Status.CurrentNumberScheduled).To(BeNumerically("==", len(nodes.Items)), "should be one instance per node")

	By("Checking if the ptp offset metric is present")
	for _, slaveNode := range rebootedNodes {

		runningPods := pods.GetRebootDaemonsetPodsAt(slaveNode)

		// Testing for one pod is sufficient as these pods are running on the same node that restarted
		for _, pod := range runningPods.Items {
			Expect(ptphelper.IsClockUnderTestPod(&pod)).To(BeTrue())

			logrus.Printf("Calling metrics endpoint for pod %s with status %s", pod.Name, pod.Status.Phase)

			time.Sleep(pkg.TimeoutIn3Minutes)

			Eventually(func() string {
				commands := []string{
					"curl", "-s", pkg.MetricsEndPoint,
				}
				buf, err := pods.ExecCommand(client.Client, &pod, pkg.RebootDaemonSetContainerName, commands)
				Expect(err).NotTo(HaveOccurred())

				scanner := bufio.NewScanner(strings.NewReader(buf.String()))
				var lines []string = make([]string, 5)
				for scanner.Scan() {
					text := scanner.Text()
					if strings.Contains(text, metrics.OpenshiftPtpOffsetNs+"{from=\"master\"") {
						logrus.Printf("Line obtained is %s", text)
						lines = append(lines, text)
					}
				}
				var offset string
				var offsetVal int
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					tokens := strings.Fields(line)
					len := len(tokens)

					if len > 0 {
						offset = tokens[len-1]
						if offset != "" {
							if val, err := strconv.Atoi(offset); err == nil {
								offsetVal = val
								logrus.Println("Offset value obtained", offsetVal)
								break
							}
						}
					}
				}
				Expect(buf.String()).NotTo(BeEmpty())
				Expect(offsetVal >= pkg.MasterOffsetLowerBound && offsetVal < pkg.MasterOffsetHigherBound).To(BeTrue())
				return buf.String()
			}, pkg.TimeoutIn5Minutes, 5*time.Second).Should(ContainSubstring(metrics.OpenshiftPtpOffsetNs),
				"Time metrics are not detected")
			break
		}
	}
}

func CheckSlaveSyncWithMaster(fullConfig testconfig.TestConfig) {
	By("Checking if slave nodes can sync with the master")

	ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	Expect(err).NotTo(HaveOccurred())
	Expect(len(ptpPods.Items)).To(BeNumerically(">", 0), "linuxptp-daemon is not deployed on cluster")

	ptpSlaveRunningPods := []corev1.Pod{}
	ptpMasterRunningPods := []corev1.Pod{}

	for _, pod := range ptpPods.Items {
		if ptphelper.IsClockUnderTestPod(&pod) {
			pods.WaitUntilLogIsDetected(&pod, pkg.TimeoutIn5Minutes, "Profile Name:")
			ptpSlaveRunningPods = append(ptpSlaveRunningPods, pod)
		} else if ptphelper.IsGrandMasterPod(&pod) {
			pods.WaitUntilLogIsDetected(&pod, pkg.TimeoutIn5Minutes, "Profile Name:")
			ptpMasterRunningPods = append(ptpMasterRunningPods, pod)
		}
	}
	if testconfig.GlobalConfig.DiscoveredGrandMasterPtpConfig != nil {
		Expect(len(ptpMasterRunningPods)).To(BeNumerically(">=", 1), "Fail to detect PTP master pods on Cluster")
		Expect(len(ptpSlaveRunningPods)).To(BeNumerically(">=", 1), "Fail to detect PTP slave pods on Cluster")
	} else {
		Expect(len(ptpSlaveRunningPods)).To(BeNumerically(">=", 1), "Fail to detect PTP slave pods on Cluster")
	}

	var masterID string
	var slaveMasterID string
	grandMaster := "assuming the grand master role"

	for _, pod := range ptpPods.Items {
		if pkg.PtpGrandmasterNodeLabel != "" &&
			ptphelper.IsGrandMasterPod(&pod) {
			podLogs, err := pods.GetLog(&pod, pkg.PtpContainerName)
			Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)
			Expect(podLogs).Should(ContainSubstring(grandMaster),
				fmt.Sprintf("Log message %q not found in pod's log %s", grandMaster, pod.Name))
			for _, line := range strings.Split(podLogs, "\n") {
				if strings.Contains(line, "selected local clock") && strings.Contains(line, "as best master") {
					// Log example: ptp4l[10731.364]: [eno1] selected local clock 3448ed.fffe.f38e00 as best master
					masterID = strings.Split(line, " ")[5]
				}
			}
		}
		if ptphelper.IsClockUnderTestPod(&pod) {
			podLogs, err := pods.GetLog(&pod, pkg.PtpContainerName)
			Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)

			for _, line := range strings.Split(podLogs, "\n") {
				if strings.Contains(line, "new foreign master") {
					// Log example: ptp4l[11292.467]: [eno1] port 1: new foreign master 3448ed.fffe.f38e00-1
					slaveMasterID = strings.Split(line, " ")[7]
				}
			}
		}
	}
	Expect(masterID).NotTo(BeNil())
	Expect(slaveMasterID).NotTo(BeNil())
	Expect(slaveMasterID).Should(HavePrefix(masterID), "Error match MasterID with the SlaveID. Slave connected to another Master")
}

// To delete a ptp test priviledged daemonset
func DeletePtpTestPrivilegedDaemonSet(daemonsetName, daemonsetNamespace string) {
	k8sPriviledgedDs.SetDaemonSetClient(client.Client.Interface)
	err := k8sPriviledgedDs.DeleteDaemonSet(daemonsetName, daemonsetNamespace)
	if err != nil {
		logrus.Errorf("error deleting %s daemonset, err=%s", daemonsetName, err)
	}
}

// To create a ptp test privileged daemonset
func CreatePtpTestPrivilegedDaemonSet(daemonsetName, daemonsetNamespace, daemonsetContainerName string) *corev1.PodList {
	const (
		imageWithVersion = "quay.io/testnetworkfunction/debug-partner:latest"
	)
	// Create the client of Priviledged Daemonset
	k8sPriviledgedDs.SetDaemonSetClient(client.Client.Interface)
	// 1. create a daemon set for the node reboot
	dummyLabels := map[string]string{}
	daemonSetRunningPods, err := k8sPriviledgedDs.CreateDaemonSet(daemonsetName, daemonsetNamespace, daemonsetContainerName, imageWithVersion, dummyLabels, pkg.TimeoutIn5Minutes)

	if err != nil {
		logrus.Errorf("error : +%v\n", err.Error())
	}
	return daemonSetRunningPods
}

func RecoverySlaveNetworkOutage(fullConfig testconfig.TestConfig, skippedInterfaces map[string]bool) {
	logrus.Info("Recovery PTP outage begins ...........")

	// Get a slave pod
	slavePod, err := ptphelper.GetPTPPodWithPTPConfig((*ptpv1.PtpConfig)(fullConfig.DiscoveredClockUnderTestPtpConfig))
	if err != nil {
		logrus.Error("Could not determine ptp daemon pod selected by ptpconfig")
	}
	// Get the slave pod's node name
	slavePodNodeName := slavePod.Spec.NodeName
	logrus.Info("slave node name is ", slavePodNodeName)

	// Get the pod from ptp test daemonset set on the slave node
	outageRecoveryDaemonSetRunningPods := CreatePtpTestPrivilegedDaemonSet(pkg.RecoveryNetworkOutageDaemonSetName, pkg.RecoveryNetworkOutageDaemonSetNamespace, pkg.RecoveryNetworkOutageDaemonSetContainerName)
	Expect(len(outageRecoveryDaemonSetRunningPods.Items)).To(BeNumerically(">", 0), "no damonset pods found in the namespace "+pkg.RecoveryNetworkOutageDaemonSetNamespace)

	var outageRecoveryDaemonsetPod corev1.Pod
	var isOutageRecoveryPodFound bool
	for _, dsPod := range outageRecoveryDaemonSetRunningPods.Items {
		if dsPod.Spec.NodeName == slavePodNodeName {
			outageRecoveryDaemonsetPod = dsPod
			isOutageRecoveryPodFound = true
			break
		}
	}
	Expect(isOutageRecoveryPodFound).To(BeTrue())
	logrus.Infof("outage recovery pod name is %s", outageRecoveryDaemonsetPod.Name)

	// Get the list of network interfaces on the slave node
	slaveIf := ptpv1.GetInterfaces((ptpv1.PtpConfig)(*fullConfig.DiscoveredClockUnderTestPtpConfig), ptpv1.Slave)
	logrus.Infof("Slave interfaces are %+q\n", slaveIf)
	// Toggle the interfaces
	for _, ptpNodeInterface := range slaveIf {
		_, skip := skippedInterfaces[ptpNodeInterface]
		if skip {
			logrus.Infof("Skipping the interface %s", ptpNodeInterface)
		} else {
			logrus.Infof("Simulating PTP outage using interface %s", ptpNodeInterface)
			toggleNetworkInterface(outageRecoveryDaemonsetPod, ptpNodeInterface, slavePodNodeName, fullConfig)
		}
	}
	DeletePtpTestPrivilegedDaemonSet(pkg.RecoveryNetworkOutageDaemonSetName, pkg.RecoveryNetworkOutageDaemonSetNamespace)
	logrus.Info("Recovery PTP outage ends ...........")
}

func toggleNetworkInterface(pod corev1.Pod, interfaceName string, slavePodNodeName string, fullConfig testconfig.TestConfig) {

	const (
		waitingPeriod      = 1 * time.Minute
		offsetRetryCounter = 5
	)

	downInterfaceCommand := fmt.Sprintf("ip link set dev %s down", interfaceName)
	logrus.Infof("Setting the interface %s down", interfaceName)
	pods.ExecutePtpInterfaceCommand(pod, interfaceName, downInterfaceCommand)
	logrus.Infof("Interface %s is set down", interfaceName)

	time.Sleep(waitingPeriod)

	// Check if the port state has changed to faulty
	err := metrics.CheckClockRole(metrics.MetricRoleFaulty, interfaceName, &slavePodNodeName)
	Expect(err).NotTo(HaveOccurred())

	upInterfaceCommand := fmt.Sprintf("ip link set dev %s up", interfaceName)
	pods.ExecutePtpInterfaceCommand(pod, interfaceName, upInterfaceCommand)
	logrus.Infof("Interface %s is up", interfaceName)
	time.Sleep(waitingPeriod)

	// Check if the port has the role of the slave
	err = metrics.CheckClockRole(metrics.MetricRoleSlave, interfaceName, &slavePodNodeName)
	Expect(err).NotTo(HaveOccurred())

	var offsetWithinBound bool
	for i := 0; i < offsetRetryCounter && !offsetWithinBound; i++ {
		offsetVal, err := metrics.GetPtpOffeset(interfaceName, &slavePodNodeName)
		Expect(err).NotTo(HaveOccurred())
		offsetWithinBound = offsetVal >= pkg.MasterOffsetLowerBound && offsetVal < pkg.MasterOffsetHigherBound
	}
	Expect(offsetWithinBound).To(BeTrue())

	logrus.Info("Successfully ended Slave clock sync with master")
}

func RebootSlaveNode(fullConfig testconfig.TestConfig) {
	logrus.Info("Rebooting system starts ..............")

	// 1. Create reboot ptp test priviledged daemonset
	rebootDaemonSetRunningPods := CreatePtpTestPrivilegedDaemonSet(pkg.RebootDaemonSetName, pkg.RebootDaemonSetNamespace, pkg.RebootDaemonSetContainerName)
	Expect(len(rebootDaemonSetRunningPods.Items)).To(BeNumerically(">", 0), "no damonset pods found in the namespace "+pkg.RebootDaemonSetNamespace)

	nodeToPodMapping := make(map[string]corev1.Pod)
	for _, dsPod := range rebootDaemonSetRunningPods.Items {
		nodeToPodMapping[dsPod.Spec.NodeName] = dsPod
	}

	// 2. Get a slave pod
	slavePod, err := ptphelper.GetPTPPodWithPTPConfig((*ptpv1.PtpConfig)(fullConfig.DiscoveredClockUnderTestPtpConfig))
	if err != nil {
		logrus.Error("Could not determine ptp daemon pod selected by ptpconfig")
	}
	slavePodNodeName := slavePod.Spec.NodeName
	logrus.Info("slave node name is ", slavePodNodeName)

	// 3. Restart the slave node
	nodeshelper.RebootNode(nodeToPodMapping[slavePodNodeName], slavePodNodeName)
	restartedNodes := []string{slavePodNodeName}
	logrus.Printf("Restarted node(s) %v", restartedNodes)

	// 3. Verify the setup of PTP
	VerifyAfterRebootState(restartedNodes, fullConfig)

	// 4. Slave nodes can sync to master
	CheckSlaveSyncWithMaster(fullConfig)

	// 5. Delete the reboot ptp test priviledged daemonset
	DeletePtpTestPrivilegedDaemonSet(pkg.RebootDaemonSetName, pkg.RebootDaemonSetNamespace)

	logrus.Info("Rebooting system ends ..............")
}
