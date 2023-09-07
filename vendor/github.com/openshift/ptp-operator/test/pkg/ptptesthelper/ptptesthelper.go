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
				buf, _, err := pods.ExecCommand(client.Client, &pod, pkg.RebootDaemonSetContainerName, commands)
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

	isExternalMaster := ptphelper.IsExternalGM()
	var grandmasterID *string
	if fullConfig.L2Config != nil && !isExternalMaster {
		aLabel := pkg.PtpGrandmasterNodeLabel
		aString, err := ptphelper.GetClockIDMaster(pkg.PtpGrandMasterPolicyName, &aLabel, nil, true)
		grandmasterID = &aString
		if err != nil {
			logrus.Warnf("could not determine the Grandmaster ID (probably because the log no longer exists), err=%s", err)
		}
	}
	err := BasicClockSyncCheck(fullConfig, (*ptpv1.PtpConfig)(fullConfig.DiscoveredClockUnderTestPtpConfig), grandmasterID)
	Expect(err).NotTo(HaveOccurred())
	if fullConfig.PtpModeDiscovered == testconfig.DualNICBoundaryClock {
		err = BasicClockSyncCheck(fullConfig, (*ptpv1.PtpConfig)(fullConfig.DiscoveredClockUnderTestSecondaryPtpConfig), grandmasterID)
		Expect(err).NotTo(HaveOccurred())
	}
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
	k8sPriviledgedDs.DeleteNamespaceIfPresent(pkg.RecoveryNetworkOutageDaemonSetNamespace)
	logrus.Info("Recovery PTP outage ends ...........")
}

func toggleNetworkInterface(pod corev1.Pod, interfaceName string, slavePodNodeName string, fullConfig testconfig.TestConfig) {

	const (
		waitingPeriod      = 4 * time.Minute
		offsetRetryCounter = 5
	)
	By("Setting interface down then wait")
	downInterfaceCommand := fmt.Sprintf("ip link set dev %s down", interfaceName)
	logrus.Infof("Setting the interface %s down", interfaceName)
	pods.ExecutePtpInterfaceCommand(pod, interfaceName, downInterfaceCommand)
	logrus.Infof("Interface %s is set down", interfaceName)

	time.Sleep(waitingPeriod)

	By("Checking that the port role is FAULTY after wait. Set the interface UP again and wait")

	// Check if the port state has changed to faulty
	err := metrics.CheckClockRole(metrics.MetricRoleFaulty, interfaceName, &slavePodNodeName)
	Expect(err).NotTo(HaveOccurred())

	upInterfaceCommand := fmt.Sprintf("ip link set dev %s up", interfaceName)
	pods.ExecutePtpInterfaceCommand(pod, interfaceName, upInterfaceCommand)
	logrus.Infof("Interface %s is up", interfaceName)
	time.Sleep(waitingPeriod)

	By("Checking that the port role is SLAVE after wait and clock is in sync")
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
	k8sPriviledgedDs.DeleteNamespaceIfPresent(pkg.RebootDaemonSetNamespace)

	logrus.Info("Rebooting system ends ..............")
}

// GetPtpPodsPerNode is a helper method to get a map of ptp-related pods (daemonset + operator)
// that are deployed on each node.
func GetPtpPodsPerNode() (map[string][]*corev1.Pod, error) {
	ptpDaemonsetPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: pkg.PtpLinuxDaemonPodsLabel})
	if err != nil {
		return nil, fmt.Errorf("failed to get linux-ptp daemonset's pods: %w", err)
	}

	ptpOperatorPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: pkg.PtPOperatorPodsLabel})
	if err != nil {
		return nil, fmt.Errorf("failed to get operator pods: %w", err)
	}

	// helper list with all ptp pods
	allPtpPods := []*corev1.Pod{}
	for i := range ptpDaemonsetPods.Items {
		allPtpPods = append(allPtpPods, &ptpDaemonsetPods.Items[i])
	}
	for i := range ptpOperatorPods.Items {
		allPtpPods = append(allPtpPods, &ptpOperatorPods.Items[i])
	}

	podsPerNode := map[string][]*corev1.Pod{}
	// Fill in the map for the ptp daemon pods.
	for _, pod := range allPtpPods {
		if pods, nodeExist := podsPerNode[pod.Spec.NodeName]; nodeExist {
			pods = append(pods, pod)
			podsPerNode[pod.Spec.NodeName] = pods
		} else {
			podsPerNode[pod.Spec.NodeName] = []*corev1.Pod{pod}
		}
	}

	return podsPerNode, nil
}

// GetPodsTotalCpuUsage uses prometheus metric "container_cpu_usage_seconds_total"
// to return the total cpu usage by all the given pods.
// As each query needs to be done inside one of the prometheus pods, an optional
// param prometheusPod can be set for that purpose. If it's nil, the function
// will try to get it on every call.
func GetPodsTotalCpuUsage(pods []*corev1.Pod, prometheusPod *corev1.Pod) (float64, error) {
	const (
		// To make sure that prometheus can use rate() with at least two samples,
		// we should use at least two sampling periods (2 * 30s = 60). We'll add 10
		// extra seconds as a safeguard.
		timeWindow = 70 * time.Second
		// queryFormat params: pod name & time window.
		queryFormat = `rate(container_cpu_usage_seconds_total{container="", pod="%s"}[%s])`
	)

	if prometheusPod == nil {
		logrus.Debugf("Getting prometheus pod...")
		var err error
		prometheusPod, err = metrics.GetPrometheusPod()
		if err != nil {
			return 0, fmt.Errorf("failed to get prometheus pod: %w", err)
		}
	}

	queryTimeWindow := time.Duration(timeWindow).String()
	totalCpu := float64(0)
	for _, pod := range pods {
		query := fmt.Sprintf(queryFormat, pod.Name, queryTimeWindow)

		// Preparing the result part so the unmarshaller can set it accordingly.
		resultVector := metrics.PrometheusVectorResult{}
		promResponse := metrics.PrometheusQueryResponse{}
		promResponse.Data.Result = &resultVector

		err := metrics.RunPrometheusQueryWithRetries(prometheusPod, query, metrics.PrometheusQueryRetries, metrics.PrometheusQueryRetryInterval, &promResponse, func(*metrics.PrometheusQueryResponse) bool {
			// Make sure the result's value is not empty
			logrus.Infof("Checking result vector len: %v", len(resultVector))
			if len(resultVector) != 1 {
				logrus.Infof("Invalid result vector length in prometheus response: %+v", promResponse)
				return false
			}
			return true
		})

		if err != nil {
			return 0, fmt.Errorf("prometheus query failure: %w", err)
		}

		// The rate query should return only one metric, so it's safe to access the first result.
		podCpuUsage, tsMillis, err := metrics.GetPrometheusResultFloatValue(resultVector[0].Value)
		if err != nil {
			return 0, fmt.Errorf("failed to get value from prometheus response from pod %s (ns %s): %w", pod.Name, pod.Namespace, err)
		}

		logrus.Debugf("Pod %s (ns %s) cpu usage: %v (ts: %s)", pod.Name, pod.Namespace, podCpuUsage, time.UnixMilli(tsMillis).String())

		totalCpu += podCpuUsage
	}

	return totalCpu, nil
}

// GetPodTotalCpuUsage uses prometheus metric "container_cpu_usage_seconds_total"
// to return the total cpu usage by all the given pods.
// As each query needs to be done inside one of the prometheus pods, an optional
// param prometheusPod can be set for that purpose. If it's nil, the function
// will try to get it on every call.
func GetPodTotalCpuUsage(podName, podNamespace string, rateTimeWindow time.Duration, prometheusPod *corev1.Pod) (float64, error) {
	// Call function to return the cpu usage of a container, but use empty string for it.
	// That will bring us the total cp usage of all the containers in the pod.
	return GetContainerCpuUsage(podName, "", podNamespace, rateTimeWindow, prometheusPod)
}

// GetContainerCpuUsage uses prometheus metric "container_cpu_usage_seconds_total"
// to return the cpu usage for a container in a pod.
// As each query needs to be done inside one of the prometheus pods, an optional
// param prometheusPod can be set for that purpose. If it's nil, the function
// will try to get it on every call.
func GetContainerCpuUsage(podName, containerName, podNamespace string, rateTimeWindow time.Duration, prometheusPod *corev1.Pod) (float64, error) {
	const (
		// queryFormat params: pod name & time window.
		queryFormat = `rate(container_cpu_usage_seconds_total{namespace="%s", pod="%s", container="%s"}[%s])`
	)

	if prometheusPod == nil {
		logrus.Debugf("Getting prometheus pod...")
		var err error
		prometheusPod, err = metrics.GetPrometheusPod()
		if err != nil {
			return 0, fmt.Errorf("failed to get prometheus pod: %w", err)
		}
	}

	queryTimeWindow := rateTimeWindow.String()

	query := fmt.Sprintf(queryFormat, podNamespace, podName, containerName, queryTimeWindow)

	// Preparing the result part so the unmarshaller can set it accordingly.
	resultVector := metrics.PrometheusVectorResult{}
	promResponse := metrics.PrometheusQueryResponse{}
	promResponse.Data.Result = &resultVector

	err := metrics.RunPrometheusQueryWithRetries(prometheusPod, query, metrics.PrometheusQueryRetries, metrics.PrometheusQueryRetryInterval, &promResponse, func(response *metrics.PrometheusQueryResponse) bool {
		if len(resultVector) != 1 {
			logrus.Warnf("Invalid result vector length in prometheus response: %+v", promResponse)
			return false
		}
		return true
	})

	if err != nil {
		return 0, fmt.Errorf("prometheus query failure: %w", err)
	}

	// The rate query should return only one metric, so it's safe to access the first result.
	cpuUsage, tsMillis, err := metrics.GetPrometheusResultFloatValue(resultVector[0].Value)
	if err != nil {
		return 0, fmt.Errorf("failed to get value from prometheus response from pod %s, container %q (ns %s): %w", podName, containerName, podNamespace, err)
	}

	logrus.Debugf("Pod: %s, container: %s (ns %s) cpu usage: %v (ts: %s)",
		podName, containerName, podNamespace, cpuUsage, time.UnixMilli(tsMillis).String())

	return cpuUsage, nil
}
