package ptp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/discovery"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	ptpv1 "github.com/openshift/ptp-operator/pkg/apis/ptp/v1"
	v1core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/utils/pointer"
)

const (
	ptpLinuxDaemonNamespace  = "openshift-ptp"
	ptpSlaveNodeLabel        = "ptp/test-slave"
	ptpGrandmasterNodeLabel  = "ptp/test-grandmaster"
	ptpGrandMasterPolicyName = "test-grandmaster"
	ptpSlavePolicyName       = "test-slave"
	ptpDaemonsetName         = "linuxptp-daemon"
)

var _ = Describe("ptp", func() {

	execute.BeforeAll(func() {
		if !discovery.Enabled() {
			Clean()
			ptpNodes, err := nodes.GetNodeTopology(client.Client)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(ptpNodes)).To(BeNumerically(">", 1), "need at least two nodes with ptp capable nics")

			By("Labeling the grandmaster node")
			ptpGrandMasterNode := ptpNodes[0]
			ptpGrandMasterNode.NodeObject, err = nodes.LabelNode(ptpGrandMasterNode.NodeName, ptpGrandmasterNodeLabel, "")
			Expect(err).ToNot(HaveOccurred())

			By("Labeling the slave node")
			ptpSlaveNode := ptpNodes[1]
			ptpSlaveNode.NodeObject, err = nodes.LabelNode(ptpSlaveNode.NodeName, ptpSlaveNodeLabel, "")
			Expect(err).ToNot(HaveOccurred())

			By("Creating the policy for the grandmaster node")
			err = createConfig(ptpGrandMasterPolicyName,
				ptpGrandMasterNode.InterfaceList[0],
				"",
				"-a -r -r",
				ptpGrandmasterNodeLabel,
				pointer.Int64Ptr(5))
			Expect(err).ToNot(HaveOccurred())

			By("Creating the policy for the slave node")
			err = createConfig(ptpSlavePolicyName,
				ptpSlaveNode.InterfaceList[0],
				"-s",
				"-a -r",
				ptpSlaveNodeLabel,
				pointer.Int64Ptr(5))
			Expect(err).ToNot(HaveOccurred())

			By("Restart the linuxptp-daemon pods")
			ptpPods, err := client.Client.Pods(ptpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
			Expect(err).ToNot(HaveOccurred())

			for _, pod := range ptpPods.Items {
				err = client.Client.Pods(ptpLinuxDaemonNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
				Expect(err).ToNot(HaveOccurred())
			}
		}

		daemonset, err := client.Client.DaemonSets(ptpLinuxDaemonNamespace).Get(context.Background(), ptpDaemonsetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		expectedNumber := daemonset.Status.DesiredNumberScheduled
		Eventually(func() int32 {
			daemonset, err = client.Client.DaemonSets(ptpLinuxDaemonNamespace).Get(context.Background(), ptpDaemonsetName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return daemonset.Status.NumberReady
		}, 2*time.Minute, 2*time.Second).Should(Equal(expectedNumber))

		Eventually(func() int {
			ptpPods, err := client.Client.Pods(ptpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
			Expect(err).ToNot(HaveOccurred())
			return len(ptpPods.Items)
		}, 2*time.Minute, 2*time.Second).Should(Equal(int(expectedNumber)))
	})

	var _ = Describe("Test Offset", func() {
		slaveLabel := ptpSlaveNodeLabel
		BeforeEach(func() {
			if !discovery.Enabled() {
				nodes, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
					LabelSelector: ptpSlaveNodeLabel,
				})
				Expect(err).ToNot(HaveOccurred())
				if len(nodes.Items) < 1 {
					Skip(fmt.Sprintf("PTP Nodes with label %s are not deployed on cluster", ptpSlaveNodeLabel))
				}
			}
			_, slaveConfigs := getPTPConfigs(ptpLinuxDaemonNamespace)
			if len(slaveConfigs) == 0 {
				Skip("No nodes configured as ptp slaves found on the cluster")
			}
			slaveLabel := retrievePTPProfileLabels(slaveConfigs)
			if slaveLabel == "" {
				Skip("No nodes configured as ptp slaves found on the cluster: no node with PTP slave labels found")
			}
		})

		var _ = Context("PTP configuration verifications", func() {

			// 27324
			It("PTP time diff between Grandmaster and Slave should be in range -100ms and 100ms", func() {
				var timeDiff string
				ptpPods, err := client.Client.Pods(ptpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ptpPods.Items)).To(BeNumerically(">", 0))
				slavePodDetected := false
				for _, pod := range ptpPods.Items {
					if podRole(pod, slaveLabel) {
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

	var _ = Describe("prometheus", func() {
		Context("Metrics reported by PTP pods", func() {
			It("Should all be reported by prometheus", func() {
				ptpPods, err := client.Client.Pods(openshiftPtpNamespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: "app=linuxptp-daemon",
				})
				Expect(err).ToNot(HaveOccurred())
				ptpMonitoredEntriesByPod, uniqueMetricKeys := collectPtpMetrics(ptpPods.Items)
				Eventually(func() error {
					podsPerPrometheusMetricKey := collectPrometheusMetrics(uniqueMetricKeys)
					return containSameMetrics(ptpMonitoredEntriesByPod, podsPerPrometheusMetricKey)
				}, 2*time.Minute, 2*time.Second).Should(Not(HaveOccurred()))

			})
		})
	})
})

// Returns the slave node label to be used in the test, empty string label cound not be found
func getPTPConfigs(namespace string) ([]ptpv1.PtpConfig, []ptpv1.PtpConfig) {
	var masters []ptpv1.PtpConfig
	var slaves []ptpv1.PtpConfig

	configList, err := client.Client.PtpConfigs(namespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, config := range configList.Items {
		for _, profile := range config.Spec.Profile {
			if isPtpMaster(*profile.Ptp4lOpts, *profile.Phc2sysOpts) {
				masters = append(masters, config)
			}
			if isPtpSlave(*profile.Ptp4lOpts, *profile.Phc2sysOpts) {
				slaves = append(masters, config)
			}
		}
	}
	return masters, slaves
}

func retrievePTPProfileLabels(configs []ptpv1.PtpConfig) string {
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

func isPtpSlave(ptp4lOpts string, phc2sysOpts string) bool {
	return strings.Contains(ptp4lOpts, "-s") && strings.Count(phc2sysOpts, "-a") == 1 && strings.Count(phc2sysOpts, "-r") == 1

}

func isPtpMaster(ptp4lOpts string, phc2sysOpts string) bool {
	return !strings.Contains(ptp4lOpts, "-s") && strings.Count(phc2sysOpts, "-a") == 1 && strings.Count(phc2sysOpts, "-r") == 2
}

// Clean removes the current ptp configuration
func Clean() {
	ptpconfigList, err := client.Client.PtpConfigs(ptpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{})
	if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
		for _, ptpConfig := range ptpconfigList.Items {
			if ptpConfig.Name == ptpGrandMasterPolicyName || ptpConfig.Name == ptpSlavePolicyName {
				err = client.Client.PtpConfigs(ptpLinuxDaemonNamespace).Delete(context.Background(), ptpConfig.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		}
	}

	nodeList, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", ptpGrandmasterNodeLabel)})
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		delete(node.Labels, ptpGrandmasterNodeLabel)
		_, err = client.Client.Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	nodeList, err = client.Client.Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=", ptpSlaveNodeLabel)})
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		delete(node.Labels, ptpSlaveNodeLabel)
		_, err = client.Client.Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
}

func podRole(runningPod v1core.Pod, role string) bool {
	nodeList, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{
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

func createConfig(profileName, ifaceName, ptp4lOpts, phc2sysOpts, nodeLabel string, priority *int64) error {
	ptpProfile := ptpv1.PtpProfile{Name: &profileName, Interface: &ifaceName, Phc2sysOpts: &phc2sysOpts, Ptp4lOpts: &ptp4lOpts}
	matchRule := ptpv1.MatchRule{NodeLabel: &nodeLabel}
	ptpRecommend := ptpv1.PtpRecommend{Profile: &profileName, Priority: priority, Match: []ptpv1.MatchRule{matchRule}}
	policy := ptpv1.PtpConfig{ObjectMeta: metav1.ObjectMeta{Name: profileName, Namespace: ptpLinuxDaemonNamespace},
		Spec: ptpv1.PtpConfigSpec{Profile: []ptpv1.PtpProfile{ptpProfile}, Recommend: []ptpv1.PtpRecommend{ptpRecommend}}}

	_, err := client.Client.PtpConfigs(ptpLinuxDaemonNamespace).Create(context.Background(), &policy, metav1.CreateOptions{})
	return err
}
