package ptphelper

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/gomega"
	"github.com/openshift/ptp-operator/test/pkg"
	"github.com/sirupsen/logrus"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	ptpv1 "github.com/openshift/ptp-operator/api/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openshift/ptp-operator/test/pkg/client"
	"github.com/openshift/ptp-operator/test/pkg/nodes"
	"github.com/openshift/ptp-operator/test/pkg/pods"
	l2exports "github.com/test-network-function/l2discovery-exports"
)

func GetProfileLogID(ptpConfigName string, label *string, nodeName *string) (id string, err error) {
	const logIDRegex = `(?m).*?Ptp4lConf: #profile: %s(.|\n)*?message_tag \[(.*)\]`
	const logIDIndex = 2
	ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	if err != nil {
		return id, err
	}
	for _, pod := range ptpPods.Items {
		isPodFound, err := pods.HasPodLabelOrNodeName(&pod, label, nodeName)
		if err != nil {
			return id, fmt.Errorf("could not check %s pod role, err: %s", *label, err)
		}

		if !isPodFound {
			continue
		}

		renderedRegex := fmt.Sprintf(logIDRegex, ptpConfigName)
		matches, err := pods.GetPodLogsRegex(pod.Namespace,
			pod.Name, pkg.PtpContainerName,
			renderedRegex, false, pkg.TimeoutIn3Minutes)
		if err != nil {
			return id, fmt.Errorf("could not get any profile line, err=%s", err)
		}
		return matches[len(matches)-1][logIDIndex], nil

	}
	return id, nil
}

func GetClockIDMaster(ptpConfigName string, label *string, nodeName *string, isGM bool) (id string, err error) {
	const clockIDGMRegex = `(?m)\[%s\] selected local clock (.*) as best master`
	const clockIDBCRegex = `(?m)\[%s\] selected best master clock (.*)`
	const clockIDIndex = 1
	clockIDRegex := ""
	if isGM {
		clockIDRegex = clockIDGMRegex
	} else {
		clockIDRegex = clockIDBCRegex
	}
	logID, err := GetProfileLogID(ptpConfigName, label, nodeName)
	if err != nil {
		return id, err
	}
	ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	if err != nil {
		return id, err
	}
	for _, pod := range ptpPods.Items {
		isPodFound, err := pods.HasPodLabelOrNodeName(&pod, label, nodeName)
		if err != nil {
			return id, fmt.Errorf("could not check %s pod role, err: %s", *label, err)
		}

		if !isPodFound {
			continue
		}
		renderedRegex := fmt.Sprintf(clockIDRegex, logID)
		matches, err := pods.GetPodLogsRegex(pod.Namespace,
			pod.Name, pkg.PtpContainerName,
			renderedRegex, false, pkg.TimeoutIn10Minutes)
		if err != nil {
			return id, fmt.Errorf("could not get any profile line, err=%s", err)
		}
		return matches[len(matches)-1][clockIDIndex], nil
	}
	return id, err
}

func GetClockIDForeign(ptpConfigName string, label *string, nodeName *string) (id string, err error) {
	const clockIDForeignRegex = `(?m)\[%s\].* selected best master clock (.*)`
	const clockIDForeignIndex = 1
	logID, err := GetProfileLogID(ptpConfigName, label, nodeName)
	if err != nil {
		return id, err
	}
	ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	if err != nil {
		return id, err
	}
	for _, pod := range ptpPods.Items {

		isPodFound, err := pods.HasPodLabelOrNodeName(&pod, label, nodeName)
		if err != nil {
			return id, fmt.Errorf("could not check %s pod role, err: %s", *label, err)
		}

		if !isPodFound {
			continue
		}

		renderedRegex := fmt.Sprintf(clockIDForeignRegex, logID)
		matches, err := pods.GetPodLogsRegex(pod.Namespace,
			pod.Name, pkg.PtpContainerName,
			renderedRegex, false, pkg.TimeoutIn10Minutes)
		if err != nil {
			return id, fmt.Errorf("could not get any profile line, err=%s", err)
		}
		return matches[len(matches)-1][clockIDForeignIndex], nil
	}
	return id, err
}

// returns true if the pod is running a grandmaster
func IsGrandMasterPod(aPod *v1core.Pod) (result bool, err error) {
	result, err = pods.PodRole(aPod, pkg.PtpGrandmasterNodeLabel)
	if err != nil {
		return false, fmt.Errorf("could not check Grandmaster pod role, err: %s", err)
	}
	return result, nil
}

// returns true if the pod is running the clock under test
func IsClockUnderTestPod(aPod *v1core.Pod) (result bool, err error) {
	result, err = pods.PodRole(aPod, pkg.PtpClockUnderTestNodeLabel)
	if err != nil {
		return false, fmt.Errorf("could not check Clock under test pod role, err: %s", err)
	}
	return result, nil
}

// Returns the slave node label to be used in the test, empty string label cound not be found
func GetPTPConfigs(namespace string) ([]ptpv1.PtpConfig, []ptpv1.PtpConfig) {
	var masters []ptpv1.PtpConfig
	var slaves []ptpv1.PtpConfig

	configList, err := client.Client.PtpConfigs(namespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, config := range configList.Items {
		for _, profile := range config.Spec.Profile {
			if IsPtpSlave(profile.Ptp4lOpts, profile.Phc2sysOpts) {
				slaves = append(slaves, config)
			}
		}
	}
	return masters, slaves
}
func GetPtpPodOnNode(nodeName string) (v1core.Pod, error) {
	WaitForPtpDaemonToBeReady()
	runningPod, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	Expect(err).NotTo(HaveOccurred(), "Error to get list of pods by label: app=linuxptp-daemon")
	Expect(len(runningPod.Items)).To(BeNumerically(">", 0), "PTP pods are  not deployed on cluster")
	for podIndex := range runningPod.Items {
		if runningPod.Items[podIndex].Spec.NodeName == nodeName {
			return runningPod.Items[podIndex], nil
		}
	}
	return v1core.Pod{}, errors.New("pod not found")
}

func GetMasterSlaveAttachedInterfaces(pod *v1core.Pod) []string {
	var IntList []string
	Eventually(func() error {
		stdout, _, err := pods.ExecCommand(client.Client, pod, pkg.PtpContainerName, []string{"ls", "/sys/class/net/"})
		if err != nil {
			return err
		}

		if stdout.String() == "" {
			return errors.New("empty response from pod retrying")
		}

		IntList = strings.Split(strings.Join(strings.Fields(stdout.String()), " "), " ")
		if len(IntList) == 0 {
			return errors.New("no interface detected")
		}

		return nil
	}, pkg.TimeoutIn3Minutes, 5*time.Second).Should(BeNil())

	return IntList
}

func GetPtpMasterSlaveAttachedInterfaces(pod *v1core.Pod) []string {
	var ptpSupportedInterfaces []string
	var stdout bytes.Buffer

	intList := GetMasterSlaveAttachedInterfaces(pod)
	for _, interf := range intList {
		skipInterface := false
		PCIAddr := ""
		var err error

		// Get readlink status
		Eventually(func() error {
			stdout, _, err = pods.ExecCommand(client.Client, pod, pkg.PtpContainerName, []string{"readlink", "-f", fmt.Sprintf("/sys/class/net/%s", interf)})
			if err != nil {
				return err
			}

			if stdout.String() == "" {
				return errors.New("empty response from pod retrying")
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
		}, pkg.TimeoutIn3Minutes, 5*time.Second).Should(BeNil())

		if skipInterface || PCIAddr == "" {
			continue
		}

		// Check if this is a virtual function
		Eventually(func() error {
			// If the physfn doesn't exist this means the interface is not a virtual function so we ca add it to the list
			stdout, _, err = pods.ExecCommand(client.Client, pod, pkg.PtpContainerName, []string{"ls", fmt.Sprintf("/sys/bus/pci/devices/%s/physfn", PCIAddr)})
			if err != nil {
				if strings.Contains(stdout.String(), "No such file or directory") {
					return nil
				}
				return err
			}

			if stdout.String() == "" {
				return errors.New("empty response from pod retrying")
			}

			// Virtual function
			skipInterface = true
			return nil
		}, 2*time.Minute, 1*time.Second).Should(BeNil())

		if skipInterface {
			continue
		}

		Eventually(func() error {
			stdout, _, err = pods.ExecCommand(client.Client, pod, pkg.PtpContainerName, []string{"ethtool", "-T", interf})
			if stdout.String() == "" {
				return errors.New("empty response from pod retrying")
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

		if IsPTPEnabled(&stdout) {
			ptpSupportedInterfaces = append(ptpSupportedInterfaces, interf)
			logrus.Debugf("Append ptp interface=%s from node=%s", interf, pod.Spec.NodeName)
		}
	}
	return ptpSupportedInterfaces
}

// This function parses ethtool command output and detect interfaces which supports ptp protocol
func IsPTPEnabled(ethToolOutput *bytes.Buffer) bool {
	var RxEnabled bool
	var TxEnabled bool
	var RawEnabled bool

	scanner := bufio.NewScanner(ethToolOutput)
	for scanner.Scan() {
		line := strings.TrimPrefix(scanner.Text(), "\t")
		parts := strings.Fields(line)
		if parts[0] == pkg.ETHTOOL_HARDWARE_RECEIVE_CAP {
			RxEnabled = true
		}
		if parts[0] == pkg.ETHTOOL_HARDWARE_TRANSMIT_CAP {
			TxEnabled = true
		}
		if parts[0] == pkg.ETHTOOL_HARDWARE_RAW_CLOCK_CAP {
			RawEnabled = true
		}
	}
	return RxEnabled && TxEnabled && RawEnabled
}

func PtpDiscoveredInterfaceList(nodeName string) []string {
	var ptpInterfaces []string
	var nodePtpDevice ptpv1.NodePtpDevice
	fg, err := client.Client.PtpV1Interface.NodePtpDevices(nodePtpDevice.Namespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, aNodePtpDevice := range fg.Items {
		if aNodePtpDevice.Name == nodeName {
			for _, aIfName := range aNodePtpDevice.Status.Devices {
				ptpInterfaces = append(ptpInterfaces, aIfName.Name)
			}
		}
	}
	return ptpInterfaces
}

func MutateProfile(profile *ptpv1.PtpConfig, profileName, nodeName string) *ptpv1.PtpConfig {
	mutatedConfig := profile.DeepCopy()
	priority := int64(0)
	mutatedConfig.ObjectMeta.Reset()
	mutatedConfig.ObjectMeta.Name = pkg.PtpTempPolicyName
	mutatedConfig.ObjectMeta.Namespace = pkg.PtpLinuxDaemonNamespace
	mutatedConfig.Spec.Profile[0].Name = &profileName
	mutatedConfig.Spec.Recommend[0].Priority = &priority
	mutatedConfig.Spec.Recommend[0].Match[0].NodeLabel = nil
	mutatedConfig.Spec.Recommend[0].Match[0].NodeName = &nodeName
	mutatedConfig.Spec.Recommend[0].Profile = &profileName
	return mutatedConfig
}

func ReplaceTestPod(pod *v1core.Pod, timeout time.Duration) (v1core.Pod, error) {
	var newPod v1core.Pod

	err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0)})
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() error {
		newPod, err = GetPtpPodOnNode(pod.Spec.NodeName)

		if err == nil && newPod.Name != pod.Name && newPod.Status.Phase == "Running" {
			return nil
		}

		return errors.New("cannot replace PTP pod")
	}, timeout, 1*time.Second).Should(BeNil())

	return newPod, nil
}

func RestartPTPDaemon() {
	ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	Expect(err).ToNot(HaveOccurred())
	for podIndex := range ptpPods.Items {
		err = client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).Delete(context.Background(), ptpPods.Items[podIndex].Name, metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
		Expect(err).ToNot(HaveOccurred())
	}

	WaitForPtpDaemonToBeReady()
}

func WaitForPtpDaemonToBeReady() int {
	daemonset, err := client.Client.DaemonSets(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpDaemonsetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	expectedNumber := daemonset.Status.DesiredNumberScheduled
	Eventually(func() int32 {
		daemonset, err = client.Client.DaemonSets(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpDaemonsetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return daemonset.Status.NumberReady
	}, pkg.TimeoutIn5Minutes, 2*time.Second).Should(Equal(expectedNumber))

	Eventually(func() int {
		ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
		Expect(err).ToNot(HaveOccurred())
		return len(ptpPods.Items)
	}, pkg.TimeoutIn5Minutes, 2*time.Second).Should(Equal(int(expectedNumber)))
	return 0
}

// Returns the slave node label to be used in the test
func DiscoveryPTPConfiguration(namespace string) (masters, slaves []*ptpv1.PtpConfig) {
	configList, err := client.Client.PtpConfigs(namespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for configIndex := range configList.Items {
		for _, profile := range configList.Items[configIndex].Spec.Profile {
			if IsPtpMaster(profile.Ptp4lOpts, profile.Phc2sysOpts) {
				masters = append(masters, &configList.Items[configIndex])
			}
			if IsPtpSlave(profile.Ptp4lOpts, profile.Phc2sysOpts) {
				slaves = append(slaves, &configList.Items[configIndex])
			} else {
				slaves = append(slaves, &configList.Items[configIndex])

			}
		}
	}

	return masters, slaves
}

func EnablePTPEvent() error {
	ptpConfig, err := client.Client.PtpV1Interface.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpConfigOperatorName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	if ptpConfig.Spec.EventConfig == nil {
		ptpConfig.Spec.EventConfig = &ptpv1.PtpEventConfig{
			EnableEventPublisher: true,
			TransportHost:        "http://mock",
		}
	}
	if ptpConfig.Spec.EventConfig.TransportHost == "" {
		ptpConfig.Spec.EventConfig.TransportHost = "http://mock"
	}

	ptpConfig.Spec.EventConfig.EnableEventPublisher = true
	_, err = client.Client.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Update(context.Background(), ptpConfig, metav1.UpdateOptions{})
	return err
}

func PtpEventEnabled() bool {
	ptpConfig, err := client.Client.PtpV1Interface.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpConfigOperatorName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	if ptpConfig.Spec.EventConfig == nil {
		return false
	}
	return ptpConfig.Spec.EventConfig.EnableEventPublisher
}

func EnablePTPReferencePlugin() error {
	ptpOperatorConfig, err := client.Client.PtpV1Interface.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpConfigOperatorName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	var plugindata apiextensions.JSON
	plugindata.Raw = []byte("1")
	if ptpOperatorConfig.Spec.EnabledPlugins != nil {
		(*ptpOperatorConfig.Spec.EnabledPlugins)["reference"] = &plugindata
	}

	_, err = client.Client.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Update(context.Background(), ptpOperatorConfig, metav1.UpdateOptions{})
	return err
}

func DisablePTPReferencePlugin() error {
	ptpOperatorConfig, err := client.Client.PtpV1Interface.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpConfigOperatorName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	(*ptpOperatorConfig.Spec.EnabledPlugins)["reference"] = nil

	_, err = client.Client.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Update(context.Background(), ptpOperatorConfig, metav1.UpdateOptions{})
	return err
}

func GetPtpOperatorVersion() (string, error) {
	const releaseVersionStr = "RELEASE_VERSION"

	var ptpOperatorVersion string

	deploy, err := client.Client.AppsV1Interface.Deployments(pkg.PtpLinuxDaemonNamespace).Get(context.TODO(), pkg.PtpOperatorDeploymentName, metav1.GetOptions{})

	if err != nil {
		logrus.Infof("PTP Operator version is not found: %v", err)
		return "", err
	}

	envs := deploy.Spec.Template.Spec.Containers[0].Env
	for _, env := range envs {
		if env.Name == releaseVersionStr {
			ptpOperatorVersion = env.Value
			ptpOperatorVersion = ptpOperatorVersion[1:]
		}
	}

	logrus.Infof("PTP operator version is %v", ptpOperatorVersion)

	return ptpOperatorVersion, err
}

// Checks for DualNIC BC
func IsSecondaryBc(config *ptpv1.PtpConfig) bool {
	for _, profile := range config.Spec.Profile {
		if profile.Phc2sysOpts != nil {
			return false
		}
	}
	return true
}

// Checks for OC
func IsPtpSlave(ptp4lOpts, phc2sysOpts *string) bool {
	return /*strings.Contains(*ptp4lOpts, "-s") &&*/ ((phc2sysOpts != nil && (strings.Count(*phc2sysOpts, "-a") == 1 && strings.Count(*phc2sysOpts, "-r") == 1)) ||
		phc2sysOpts == nil)
}

// Checks for Grand master
func IsPtpMaster(ptp4lOpts, phc2sysOpts *string) bool {
	return ptp4lOpts != nil && phc2sysOpts != nil && !strings.Contains(*ptp4lOpts, "-s ") && strings.Count(*phc2sysOpts, "-a") == 1 && strings.Count(*phc2sysOpts, "-r") == 2
}

// Checks for DualNIC BC
func GetProfileName(config *ptpv1.PtpConfig) (string, error) {
	for _, profile := range config.Spec.Profile {
		if profile.Name != nil && *profile.Name == pkg.PtpGrandMasterPolicyName ||
			*profile.Name == pkg.PtpBcMaster1PolicyName ||
			*profile.Name == pkg.PtpBcMaster2PolicyName ||
			*profile.Name == pkg.PtpSlave1PolicyName ||
			*profile.Name == pkg.PtpSlave2PolicyName ||
			*profile.Name == pkg.PtpTempPolicyName {
			return *profile.Name, nil
		}
	}
	return "", fmt.Errorf("cannot find valid test profile name")
}
func RetrievePTPProfileLabels(configs []ptpv1.PtpConfig) string {
	for _, config := range configs {
		for _, recommend := range config.Spec.Recommend {
			for _, match := range recommend.Match {
				label := *match.NodeLabel
				nodeCount, err := nodes.LabeledNodesCount(label)
				Expect(err).ToNot(HaveOccurred())
				if nodeCount > 0 {
					return label
				}
			}
		}
	}
	return ""
}

func GetPTPPodWithPTPConfig(ptpConfig *ptpv1.PtpConfig) (aPtpPod *v1core.Pod, err error) {
	ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
	if err != nil {
		return aPtpPod, err
	}

	label, err := GetLabel(ptpConfig)
	if err != nil {
		logrus.Debugf("GetLabel err=%s", err)
	}

	nodeName, err := GetFirstNode(ptpConfig)
	if err != nil {
		logrus.Debugf("GetFirstNode err=%s", err)
	}

	for _, pod := range ptpPods.Items {

		isPodFound, err := pods.HasPodLabelOrNodeName(&pod, label, nodeName)
		if err != nil {
			logrus.Errorf("could not check %s pod role, err: %s", *label, err)
		}

		if isPodFound {
			aPtpPod = &pod
			break
		}
	}
	return aPtpPod, nil
}

// Gets the first label configured in the ptpconfig->spec->recommend
func GetLabel(ptpConfig *ptpv1.PtpConfig) (*string, error) {
	for _, r := range ptpConfig.Spec.Recommend {
		for _, m := range r.Match {
			if m.NodeLabel == nil {
				continue
			}
			aLabel := ""
			switch *m.NodeLabel {
			case pkg.PtpClockUnderTestNodeLabel:
				aLabel = pkg.PtpClockUnderTestNodeLabel
			case pkg.PtpGrandmasterNodeLabel:
				aLabel = pkg.PtpGrandmasterNodeLabel
			case pkg.PtpSlave1NodeLabel:
				aLabel = pkg.PtpSlave1NodeLabel
			case pkg.PtpSlave2NodeLabel:
				aLabel = pkg.PtpSlave2NodeLabel
			}
			return &aLabel, nil
		}
	}
	return nil, fmt.Errorf("label not found")
}

// gets the first nodename configured in the ptpconfig->spec->recommend
func GetFirstNode(ptpConfig *ptpv1.PtpConfig) (*string, error) {
	for _, r := range ptpConfig.Spec.Recommend {
		for _, m := range r.Match {
			if m.NodeName == nil {
				continue
			}
			return m.NodeName, nil
		}
	}
	return nil, fmt.Errorf("nodeName not found")
}

func GetPtpInterfacePerNode(nodeName string, ifList map[string]*l2exports.PtpIf) (out []string) {
	for _, aIf := range ifList {
		if aIf.NodeName == nodeName {
			out = append(out, aIf.IfName)
		}
	}
	return out
}

var mu sync.RWMutex

// saves events to file
func SaveStoreEventsToFile(allEvents, filename string) {
	mu.Lock()
	err := os.WriteFile(filename, []byte(allEvents), 0644)
	if err != nil {
		logrus.Errorf("could not write events to file, err: %s", err)
	}
	mu.Unlock()
}

func IsExternalGM() (out bool) {
	value, isSet := os.LookupEnv("EXTERNAL_GM")
	value = strings.ToLower(value)
	out = isSet && !strings.Contains(value, "false")
	logrus.Infof("EXTERNAL_GM=%t", out)
	return out
}
