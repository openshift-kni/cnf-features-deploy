package test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/ptp-operator/test/pkg"
	"github.com/openshift/ptp-operator/test/pkg/event"
	"github.com/openshift/ptp-operator/test/pkg/metrics"
	"github.com/openshift/ptp-operator/test/pkg/namespaces"
	"github.com/openshift/ptp-operator/test/pkg/ptphelper"
	"github.com/openshift/ptp-operator/test/pkg/ptptesthelper"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ptpv1 "github.com/openshift/ptp-operator/api/v1"

	"github.com/openshift/ptp-operator/test/pkg/nodes"
	"github.com/openshift/ptp-operator/test/pkg/pods"

	"github.com/openshift/ptp-operator/test/pkg/client"
	"github.com/openshift/ptp-operator/test/pkg/execute"

	"github.com/openshift/ptp-operator/test/pkg/testconfig"
)

type TestCase string

const (
	Reboot TestCase = "reboot"
)

var _ = Describe("[ptp]", Serial, func() {
	BeforeEach(func() {
		Expect(client.Client).NotTo(BeNil())
	})

	Context("PTP configuration verifications", func() {
		// Setup verification
		// if requested enabled  ptp events
		It("Should check whether PTP operator needs to enable PTP events", func() {
			By("Find if variable set to enable ptp events")
			if event.Enable() {
				Expect(ptphelper.EnablePTPEvent()).NotTo(HaveOccurred())
				ptpConfig, err := client.Client.PtpV1Interface.PtpOperatorConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpConfigOperatorName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(ptpConfig.Spec.EventConfig.EnableEventPublisher).Should(BeTrue(), "failed to enable ptp event")
			}
		})
		It("Should check whether PTP operator appropriate resource exists", func() {
			By("Getting list of available resources")
			rl, err := client.Client.ServerPreferredResources()
			Expect(err).ToNot(HaveOccurred())

			found := false
			By("Find appropriate resources")
			for _, g := range rl {
				if strings.Contains(g.GroupVersion, pkg.PtpResourcesGroupVersionPrefix) {
					for _, r := range g.APIResources {
						By("Search for resource " + pkg.PtpResourcesNameOperatorConfigs)
						if r.Name == pkg.PtpResourcesNameOperatorConfigs {
							found = true
						}
					}
				}
			}

			Expect(found).To(BeTrue(), fmt.Sprintf("resource %s not found", pkg.PtpResourcesNameOperatorConfigs))
		})
		// Setup verification
		It("Should check that all nodes are running at least one replica of linuxptp-daemon", func() {
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
		})
		// Setup verification
		It("Should check that operator is deployed", func() {
			By("Getting deployment " + pkg.PtpOperatorDeploymentName)
			dep, err := client.Client.Deployments(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpOperatorDeploymentName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Checking availability of the deployment")
			for _, c := range dep.Status.Conditions {
				if c.Type == v1.DeploymentAvailable {
					Expect(string(c.Status)).Should(Equal("True"), pkg.PtpOperatorDeploymentName+" deployment is not available")
				}
			}
		})

	})

	Describe("PTP e2e tests", func() {
		var ptpRunningPods []*v1core.Pod
		var fifoPriorities map[string]int64
		var fullConfig testconfig.TestConfig
		execute.BeforeAll(func() {
			err := testconfig.CreatePtpConfigurations()
			if err != nil {
				fullConfig.Status = testconfig.DiscoveryFailureStatus
				Fail(fmt.Sprintf("Could not create a ptp config, err=%s", err))
			}
			fullConfig = testconfig.GetFullDiscoveredConfig(pkg.PtpLinuxDaemonNamespace, false)
			if fullConfig.Status != testconfig.DiscoverySuccessStatus {
				logrus.Printf(`ptpconfigs were not properly discovered, Check:
- the ptpconfig has a %s label only in the recommend section (no node section)
- the node running the clock under test is label with: %s`, pkg.PtpClockUnderTestNodeLabel, pkg.PtpClockUnderTestNodeLabel)

				Fail("Failed to find a valid ptp slave configuration")

			}
			if fullConfig.PtpModeDesired != testconfig.Discovery {
				ptphelper.RestartPTPDaemon()
			}

		})

		Context("PTP Outage recovery", func() {
			BeforeEach(func() {
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}
			})

			It("The slave node network interface is taken down and up", func() {
				By("toggling network interfaces and syncing", func() {

					skippedInterfacesStr, isSet := os.LookupEnv("SKIP_INTERFACES")

					if !isSet {
						Skip("Mandatory to provide skipped interface to avoid making a node disconnected from the cluster")
					} else {
						skipInterfaces := make(map[string]bool)
						separated := strings.Split(skippedInterfacesStr, ",")
						for _, val := range separated {
							skipInterfaces[val] = true
						}
						logrus.Info("skipINterfaces", skipInterfaces)
						ptptesthelper.RecoverySlaveNetworkOutage(fullConfig, skipInterfaces)
					}
				})
			})
		})

		Context("PTP Reboot discovery", func() {

			BeforeEach(func() {
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}
			})

			It("The slave node is rebooted and discovered and in sync", func() {
				if testCaseEnabled(Reboot) {
					By("Slave node is rebooted", func() {
						ptptesthelper.RebootSlaveNode(fullConfig)
					})
				} else {
					Skip("Skipping the reboot test")
				}
			})
		})

		Context("PTP Interfaces discovery", func() {

			BeforeEach(func() {
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}
				ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(ptpPods.Items)).To(BeNumerically(">", 0), "linuxptp-daemon is not deployed on cluster")

				ptpRunningPods, err = ptptesthelper.TestPtpRunningPods(ptpPods)
				if err != nil {
					Fail(fmt.Sprintf("cannot get ptp pods, err=%s", err))
				}
			})

			It("The interfaces supporting ptp can be discovered correctly", func() {
				for podIndex := range ptpRunningPods {
					ptpNodeIfacesDiscoveredByL2 := ptphelper.GetPtpInterfacePerNode(ptpRunningPods[podIndex].Spec.NodeName, fullConfig.L2Config.GetPtpIfListUnfiltered())
					Expect(len(ptpNodeIfacesDiscoveredByL2)).To(BeNumerically(">", 0), "Fail to detect PTP Supported interfaces on slave/master pods")
					ptpNodeIfacesFromPtpApi := ptphelper.PtpDiscoveredInterfaceList(ptpRunningPods[podIndex].Spec.NodeName)
					sort.Strings(ptpNodeIfacesDiscoveredByL2)
					sort.Strings(ptpNodeIfacesFromPtpApi)
					logrus.Infof("Interfaces supporting ptp for node        %s: %v", ptpRunningPods[podIndex].Spec.NodeName, ptpNodeIfacesDiscoveredByL2)
					logrus.Infof("Interfaces discovered by ptp API for node %s: %v", ptpRunningPods[podIndex].Spec.NodeName, ptpNodeIfacesFromPtpApi)

					// The discovered PTP interfaces should match exactly the list of interfaces calculated by test
					Expect(len(ptpNodeIfacesDiscoveredByL2)).To(Equal(len(ptpNodeIfacesFromPtpApi)))
					for index := range ptpNodeIfacesDiscoveredByL2 {
						Expect(ptpNodeIfacesDiscoveredByL2[index]).To(Equal(ptpNodeIfacesFromPtpApi[index]))
					}
				}
			})

			It("Should retrieve the details of hardwares for the Ptp", func() {
				By("Getting the version of the OCP cluster")

				ocpVersion, err := getOCPVersion()
				Expect(err).ToNot(HaveOccurred())
				Expect(ocpVersion).ShouldNot(BeEmpty())

				By("Getting the version of the PTP operator")

				ptpOperatorVersion, err := ptphelper.GetPtpOperatorVersion()
				Expect(err).ToNot(HaveOccurred())
				Expect(ptpOperatorVersion).ShouldNot(BeEmpty())

				By("Getting the NIC details of all the PTP enabled interfaces")

				ptpInterfacesList := fullConfig.L2Config.GetPtpIfList()

				for _, ptpInterface := range ptpInterfacesList {
					ifaceHwDetails := fmt.Sprintf("Device: %s, Function: %s, Description: %s",
						ptpInterface.IfPci.Device, ptpInterface.IfPci.Function, ptpInterface.IfPci.Description)

					logrus.Infof("Node: %s, Interface Name: %s, %s", ptpInterface.NodeName, ptpInterface.IfName, ifaceHwDetails)

					AddReportEntry(fmt.Sprintf("Node %s, Interface: %s", ptpInterface.NodeName, ptpInterface.IfName), ifaceHwDetails)
				}

				By("Getting ptp config details")
				ptpConfig := testconfig.GlobalConfig

				masterPtpConfigStr := ptpConfig.DiscoveredGrandMasterPtpConfig.String()
				slavePtpConfigStr := ptpConfig.DiscoveredClockUnderTestPtpConfig.String()

				logrus.Infof("Discovered master ptp config %s", masterPtpConfigStr)
				logrus.Infof("Discovered slave ptp config %s", slavePtpConfigStr)

				AddReportEntry("master-ptp-config", masterPtpConfigStr)
				AddReportEntry("slave-ptp-config", slavePtpConfigStr)
			})
		})

		Context("PTP ClockSync", func() {
			err := metrics.InitEnvIntParamConfig("MAX_OFFSET_IN_NS", metrics.MaxOffsetDefaultNs, &metrics.MaxOffsetNs)
			Expect(err).NotTo(HaveOccurred(), "error getting max offset in nanoseconds %s", err)
			err = metrics.InitEnvIntParamConfig("MIN_OFFSET_IN_NS", metrics.MinOffsetDefaultNs, &metrics.MinOffsetNs)
			Expect(err).NotTo(HaveOccurred(), "error getting min offset in nanoseconds %s", err)

			BeforeEach(func() {
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}
			})
			// 25733
			It("PTP daemon apply match rule based on nodeLabel", func() {

				if fullConfig.PtpModeDesired == testconfig.Discovery {
					Skip("This test needs the ptp-daemon to be rebooted but it is not possible in discovery mode, skipping")
				}
				profileSlave := fmt.Sprintf("Profile Name: %s", fullConfig.DiscoveredClockUnderTestPtpConfig.Name)
				profileMaster := ""
				if fullConfig.DiscoveredGrandMasterPtpConfig != nil {
					profileMaster = fmt.Sprintf("Profile Name: %s", fullConfig.DiscoveredGrandMasterPtpConfig.Name)
				}

				for podIndex := range ptpRunningPods {
					_, err := pods.GetLog(ptpRunningPods[podIndex], pkg.PtpContainerName)
					Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)

					if ptphelper.IsClockUnderTestPod(ptpRunningPods[podIndex]) {
						pods.WaitUntilLogIsDetected(ptpRunningPods[podIndex], pkg.TimeoutIn3Minutes, profileSlave)
					} else if ptphelper.IsGrandMasterPod(ptpRunningPods[podIndex]) && fullConfig.DiscoveredGrandMasterPtpConfig != nil {
						pods.WaitUntilLogIsDetected(ptpRunningPods[podIndex], pkg.TimeoutIn3Minutes, profileMaster)
					}
				}
			})

			// Multinode clock sync test:
			// - waits for the foreign master to appear
			// - verifies that the foreign master has the expected grandmaster ID
			// - use metrics to verify that the offset is below threshold
			//
			// Single node clock sync test:
			// - waits for the foreign master to appear
			// - use metrics to verify that the offset is below threshold
			It("Slave can sync to master", func() {
				isSingleNode, err := nodes.IsSingleNodeCluster()
				if err != nil {
					Skip("cannot determine if cluster is single node")
				}
				var grandmasterID *string
				if fullConfig.L2Config != nil && !isSingleNode {
					aLabel := pkg.PtpGrandmasterNodeLabel
					aString, err := ptphelper.GetClockIDMaster(pkg.PtpGrandMasterPolicyName, &aLabel, nil)
					grandmasterID = &aString
					Expect(err).To(BeNil())
				}
				ptptesthelper.BasicClockSyncCheck(fullConfig, (*ptpv1.PtpConfig)(fullConfig.DiscoveredClockUnderTestPtpConfig), grandmasterID)

				if fullConfig.PtpModeDiscovered == testconfig.DualNICBoundaryClock {
					ptptesthelper.BasicClockSyncCheck(fullConfig, (*ptpv1.PtpConfig)(fullConfig.DiscoveredClockUnderTestSecondaryPtpConfig), grandmasterID)
				}
			})

			// Multinode BCSlave clock sync
			// - waits for the BCSlave foreign master to appear (the boundary clock)
			// - verifies that the BCSlave foreign master has the expected boundary clock ID
			// - use metrics to verify that the offset with boundary clock is below threshold
			It("Downstream slave can sync to BC master", func() {
				isSingleNode, err := nodes.IsSingleNodeCluster()
				if err != nil {
					Skip("cannot determine if cluster is single node")
				}
				if fullConfig.L2Config == nil || isSingleNode {
					Skip("Boundary clock slave sync test is not performed in discovery or SNO mode")
				}
				if fullConfig.PtpModeDiscovered != testconfig.BoundaryClock &&
					fullConfig.PtpModeDiscovered != testconfig.DualNICBoundaryClock {
					Skip("test only valid for Boundary clock in multi-node clusters")
				}
				if (fullConfig.PtpModeDiscovered == testconfig.BoundaryClock &&
					!fullConfig.FoundSolutions[testconfig.AlgoBCWithSlavesString]) ||
					(fullConfig.PtpModeDiscovered == testconfig.DualNICBoundaryClock &&
						!fullConfig.FoundSolutions[testconfig.AlgoDualNicBCWithSlavesString]) {
					Skip("test only valid for Boundary clock in multi-node clusters with slaves")
				}
				aLabel := pkg.PtpClockUnderTestNodeLabel
				masterIDBc1, err := ptphelper.GetClockIDMaster(pkg.PtpBcMaster1PolicyName, &aLabel, nil)
				Expect(err).To(BeNil())
				ptptesthelper.BasicClockSyncCheck(fullConfig, (*ptpv1.PtpConfig)(fullConfig.DiscoveredSlave1PtpConfig), &masterIDBc1)

				if fullConfig.PtpModeDiscovered == testconfig.DualNICBoundaryClock &&
					fullConfig.FoundSolutions[testconfig.AlgoDualNicBCWithSlavesString] {

					aLabel := pkg.PtpClockUnderTestNodeLabel
					masterIDBc2, err := ptphelper.GetClockIDMaster(pkg.PtpBcMaster2PolicyName, &aLabel, nil)
					Expect(err).To(BeNil())
					ptptesthelper.BasicClockSyncCheck(fullConfig, (*ptpv1.PtpConfig)(fullConfig.DiscoveredSlave2PtpConfig), &masterIDBc2)
				}

			})

			// 25743
			It("Can provide a profile with higher priority", func() {
				var testPtpPod v1core.Pod
				isSingleNode, err := nodes.IsSingleNodeCluster()
				if err != nil {
					Skip("cannot determine if cluster is single node")
				}
				if fullConfig.PtpModeDesired == testconfig.Discovery {
					Skip("Skipping because adding a different profile and no modifications are allowed in discovery mode")
				}
				var policyName string
				var modifiedPtpConfig *ptpv1.PtpConfig
				By("Creating a config with higher priority", func() {

					switch fullConfig.PtpModeDiscovered {
					case testconfig.Discovery, testconfig.None:
						Skip("Skipping because Discovery or None is not supported yet for this test")
					case testconfig.OrdinaryClock:
						policyName = pkg.PtpSlave1PolicyName
					case testconfig.BoundaryClock:
						policyName = pkg.PtpBcMaster1PolicyName
					case testconfig.DualNICBoundaryClock:
						policyName = pkg.PtpBcMaster1PolicyName
					}
					ptpConfigToModify, err := client.Client.PtpV1Interface.PtpConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), policyName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					nodes, err := client.Client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
						LabelSelector: pkg.PtpClockUnderTestNodeLabel,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(nodes.Items)).To(BeNumerically(">", 0),
						fmt.Sprintf("PTP Nodes with label %s are not deployed on cluster", pkg.PtpClockUnderTestNodeLabel))

					ptpConfigTest := ptphelper.MutateProfile(ptpConfigToModify, pkg.PtpTempPolicyName, nodes.Items[0].Name)
					modifiedPtpConfig, err = client.Client.PtpV1Interface.PtpConfigs(pkg.PtpLinuxDaemonNamespace).Create(context.Background(), ptpConfigTest, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					testPtpPod, err = ptphelper.GetPtpPodOnNode(nodes.Items[0].Name)
					Expect(err).NotTo(HaveOccurred())

					testPtpPod, err = ptphelper.ReplaceTestPod(&testPtpPod, time.Minute)
					Expect(err).NotTo(HaveOccurred())
				})

				By("Checking if Node has Profile and check sync", func() {
					var grandmasterID *string
					if fullConfig.L2Config != nil && !isSingleNode {
						aLabel := pkg.PtpGrandmasterNodeLabel
						aString, err := ptphelper.GetClockIDMaster(pkg.PtpGrandMasterPolicyName, &aLabel, nil)
						grandmasterID = &aString
						Expect(err).To(BeNil())
					}
					ptptesthelper.BasicClockSyncCheck(fullConfig, modifiedPtpConfig, grandmasterID)
				})

				By("Deleting the test profile", func() {
					err := client.Client.PtpV1Interface.PtpConfigs(pkg.PtpLinuxDaemonNamespace).Delete(context.Background(), pkg.PtpTempPolicyName, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
					Eventually(func() bool {
						_, err := client.Client.PtpV1Interface.PtpConfigs(pkg.PtpLinuxDaemonNamespace).Get(context.Background(), pkg.PtpTempPolicyName, metav1.GetOptions{})
						return kerrors.IsNotFound(err)
					}, 1*time.Minute, 1*time.Second).Should(BeTrue(), "Could not delete the test profile")
				})

				By("Checking the profile is reverted", func() {
					pods.WaitUntilLogIsDetected(&testPtpPod, pkg.TimeoutIn3Minutes, "Profile Name: "+policyName)
				})
			})
		})

		Context("PTP metric is present", func() {
			BeforeEach(func() {
				ptphelper.WaitForPtpDaemonToBeReady()
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}
				ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(ptpPods.Items)).To(BeNumerically(">", 0), "linuxptp-daemon is not deployed on cluster")

				ptpRunningPods, err = ptptesthelper.TestPtpRunningPods(ptpPods)
				if err != nil {
					Fail(fmt.Sprintf("cannot get ptp pods, err=%s", err))
				}
			})

			// 27324
			It("on slave", func() {
				slavePodDetected := false
				for podIndex := range ptpRunningPods {
					if ptphelper.IsClockUnderTestPod(ptpRunningPods[podIndex]) {
						Eventually(func() string {
							buf, _ := pods.ExecCommand(client.Client, ptpRunningPods[podIndex], pkg.PtpContainerName, []string{"curl", pkg.MetricsEndPoint})
							return buf.String()
						}, pkg.TimeoutIn5Minutes, 5*time.Second).Should(ContainSubstring(metrics.OpenshiftPtpOffsetNs),
							"Time metrics are not detected")
						slavePodDetected = true
						break
					}
				}
				Expect(slavePodDetected).ToNot(BeFalse(), "No slave pods detected")
			})
		})

		Context("Running with event enabled", func() {
			ptpSlaveRunningPods := []v1core.Pod{}
			BeforeEach(func() {
				ptphelper.WaitForPtpDaemonToBeReady()
				if !ptphelper.PtpEventEnabled() {
					Skip("Skipping, PTP events not enabled")
				}
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}
				ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(ptpPods.Items)).To(BeNumerically(">", 0), "linuxptp-daemon is not deployed on cluster")

				for podIndex := range ptpPods.Items {
					if role, _ := pods.PodRole(&ptpPods.Items[podIndex], pkg.PtpClockUnderTestNodeLabel); role {
						pods.WaitUntilLogIsDetected(&ptpPods.Items[podIndex], pkg.TimeoutIn3Minutes, "Profile Name:")
						ptpSlaveRunningPods = append(ptpSlaveRunningPods, ptpPods.Items[podIndex])
					}
				}
				if fullConfig.PtpModeDesired == testconfig.Discovery {
					Expect(len(ptpSlaveRunningPods)).To(BeNumerically(">=", 1), "Fail to detect PTP slave pods on Cluster")
				}
			})

			It("Should check for ptp events ", func() {
				By("Checking event side car is present")

				for podIndex := range ptpSlaveRunningPods {
					cloudProxyFound := false
					Expect(len(ptpSlaveRunningPods[podIndex].Spec.Containers)).To(BeNumerically("==", 3), "linuxptp-daemon is not deployed on cluster with cloud event proxy")
					for _, c := range ptpSlaveRunningPods[podIndex].Spec.Containers {
						if c.Name == pkg.EventProxyContainerName {
							cloudProxyFound = true
						}
					}
					Expect(cloudProxyFound).ToNot(BeFalse(), "No event pods detected")
				}

				By("Checking event metrics are present")
				for podIndex := range ptpSlaveRunningPods {
					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, &ptpSlaveRunningPods[podIndex], pkg.EventProxyContainerName, []string{"curl", pkg.MetricsEndPoint})
						return buf.String()
					}, pkg.TimeoutIn5Minutes, 5*time.Second).Should(ContainSubstring(metrics.OpenshiftPtpInterfaceRole),
						"Interface role metrics are not detected")

					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, &ptpSlaveRunningPods[podIndex], pkg.EventProxyContainerName, []string{"curl", pkg.MetricsEndPoint})
						return buf.String()
					}, pkg.TimeoutIn5Minutes, 5*time.Second).Should(ContainSubstring(metrics.OpenshiftPtpThreshold),
						"Threshold metrics are not detected")
				}

				By("Checking event api is healthy")
				for podIndex := range ptpSlaveRunningPods {
					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, &ptpSlaveRunningPods[podIndex], pkg.EventProxyContainerName, []string{"curl", "127.0.0.1:9085/api/cloudNotifications/v1/health"})
						return buf.String()
					}, pkg.TimeoutIn5Minutes, 5*time.Second).Should(ContainSubstring("OK"),
						"Event API is not in healthy state")
				}

				By("Checking ptp publisher is created")
				for podIndex := range ptpSlaveRunningPods {
					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, &ptpSlaveRunningPods[podIndex], pkg.EventProxyContainerName, []string{"curl", "127.0.0.1:9085/api/cloudNotifications/v1/publishers"})
						return buf.String()
					}, pkg.TimeoutIn5Minutes, 5*time.Second).Should(ContainSubstring("endpointUri"),
						"Event API  did not return publishers")
				}

				By("Checking events are generated")
				for podIndex := range ptpSlaveRunningPods {
					podLogs, err := pods.GetLog(&ptpSlaveRunningPods[podIndex], pkg.EventProxyContainerName)
					Expect(err).NotTo(HaveOccurred(), "Error to find needed log due to %s", err)
					Expect(podLogs).Should(ContainSubstring("Created publisher"),
						fmt.Sprintf("PTP event publisher was not created in pod %s", ptpSlaveRunningPods[podIndex].Name))
					Expect(podLogs).Should(ContainSubstring("event sent"),
						fmt.Sprintf("PTP event was not generated in the pod %s", ptpSlaveRunningPods[podIndex].Name))
				}
			})
		})
		Context("Running with fifo scheduling", func() {
			BeforeEach(func() {
				ptphelper.WaitForPtpDaemonToBeReady()
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}

				masterConfigs, slaveConfigs := ptphelper.DiscoveryPTPConfiguration(pkg.PtpLinuxDaemonNamespace)
				ptpConfigs := append(masterConfigs, slaveConfigs...)

				fifoPriorities = make(map[string]int64)
				for _, config := range ptpConfigs {
					for _, profile := range config.Spec.Profile {
						if profile.PtpSchedulingPolicy != nil && *profile.PtpSchedulingPolicy == "SCHED_FIFO" {
							if profile.PtpSchedulingPriority != nil {
								fifoPriorities[*profile.Name] = *profile.PtpSchedulingPriority
							}
						}
					}
				}
				if len(fifoPriorities) == 0 {
					Skip("No SCHED_FIFO policies configured")
				}
			})
			It("Should check whether using fifo scheduling", func() {
				By("checking for chrt logs")
				ptpPods, err := client.Client.CoreV1().Pods(pkg.PtpLinuxDaemonNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=linuxptp-daemon"})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(ptpPods.Items)).To(BeNumerically(">", 0), "linuxptp-daemon is not deployed on cluster")
				for name, priority := range fifoPriorities {
					ptp4lLog := fmt.Sprintf("/bin/chrt -f %d /usr/sbin/ptp4l", priority)
					for podIndex := range ptpPods.Items {
						logs, err := pods.GetLog(&ptpPods.Items[podIndex], pkg.PtpContainerName)
						Expect(err).NotTo(HaveOccurred())
						profileName := fmt.Sprintf("Profile Name: %s", name)
						if strings.Contains(logs, profileName) {
							Expect(logs).Should(ContainSubstring(ptp4lLog))
							delete(fifoPriorities, name)
						}
					}
				}
				Expect(fifoPriorities).To(HaveLen(0))
			})
		})

		// old cnf-feature-deploy tests
		var _ = Describe("PTP socket sharing between pods", func() {
			BeforeEach(func() {
				if fullConfig.Status == testconfig.DiscoveryFailureStatus {
					Skip("Failed to find a valid ptp slave configuration")
				}
				if fullConfig.PtpModeDesired == testconfig.Discovery {
					Skip("PTP socket test not supported in discovery mode")
				}
			})
			AfterEach(func() {
				err := namespaces.Clean(openshiftPtpNamespace, "testpod-", client.Client)
				Expect(err).ToNot(HaveOccurred())
			})
			var _ = Context("Negative - run pmc in a new unprivileged pod on the slave node", func() {
				It("Should not be able to use the uds", func() {
					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, fullConfig.DiscoveredClockUnderTestPod, fullConfig.DiscoveredClockUnderTestPod.Spec.Containers[0].Name, []string{"pmc", "-u", "-f", "/var/run/ptp4l.0.config", "GET CURRENT_DATA_SET"})
						return buf.String()
					}, 1*time.Minute, 2*time.Second).ShouldNot(ContainSubstring("failed to open configuration file"), "ptp config file was not created")
					podDefinition := pods.DefinePodOnNode(pkg.PtpLinuxDaemonNamespace, fullConfig.DiscoveredClockUnderTestPod.Spec.NodeName)
					hostPathDirectoryOrCreate := v1core.HostPathDirectoryOrCreate
					podDefinition.Spec.Volumes = []v1core.Volume{
						{
							Name: "socket-dir",
							VolumeSource: v1core.VolumeSource{
								HostPath: &v1core.HostPathVolumeSource{
									Path: "/var/run/ptp",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
					}
					podDefinition.Spec.Containers[0].VolumeMounts = []v1core.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/var/run",
						},
						{
							Name:      "socket-dir",
							MountPath: "/host",
						},
					}
					pod, err := client.Client.Pods(pkg.PtpLinuxDaemonNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					err = pods.WaitForCondition(client.Client, pod, v1core.ContainersReady, v1core.ConditionTrue, 3*time.Minute)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, pod, pod.Spec.Containers[0].Name, []string{"pmc", "-u", "-f", "/var/run/ptp4l.0.config", "GET CURRENT_DATA_SET"})
						return buf.String()
					}, 1*time.Minute, 2*time.Second).Should(ContainSubstring("Permission denied"), "unprivileged pod can access the uds socket")
				})
			})

			var _ = Context("Run pmc in a new pod on the slave node", func() {
				It("Should be able to sync using a uds", func() {

					Expect(fullConfig.DiscoveredClockUnderTestPod).ToNot(BeNil())
					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, fullConfig.DiscoveredClockUnderTestPod, fullConfig.DiscoveredClockUnderTestPod.Spec.Containers[0].Name, []string{"pmc", "-u", "-f", "/var/run/ptp4l.0.config", "GET CURRENT_DATA_SET"})
						return buf.String()
					}, 1*time.Minute, 2*time.Second).ShouldNot(ContainSubstring("failed to open configuration file"), "ptp config file was not created")
					podDefinition, _ := pods.RedefineAsPrivileged(
						pods.DefinePodOnNode(pkg.PtpLinuxDaemonNamespace, fullConfig.DiscoveredClockUnderTestPod.Spec.NodeName), "")
					hostPathDirectoryOrCreate := v1core.HostPathDirectoryOrCreate
					podDefinition.Spec.Volumes = []v1core.Volume{
						{
							Name: "socket-dir",
							VolumeSource: v1core.VolumeSource{
								HostPath: &v1core.HostPathVolumeSource{
									Path: "/var/run/ptp",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
					}
					podDefinition.Spec.Containers[0].VolumeMounts = []v1core.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/var/run",
						},
						{
							Name:      "socket-dir",
							MountPath: "/host",
						},
					}
					pod, err := client.Client.Pods(pkg.PtpLinuxDaemonNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					err = pods.WaitForCondition(client.Client, pod, v1core.ContainersReady, v1core.ConditionTrue, 3*time.Minute)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() string {
						buf, _ := pods.ExecCommand(client.Client, pod, pod.Spec.Containers[0].Name, []string{"pmc", "-u", "-f", "/var/run/ptp4l.0.config", "GET CURRENT_DATA_SET"})
						return buf.String()
					}, 1*time.Minute, 2*time.Second).ShouldNot(ContainSubstring("failed to open configuration file"), "ptp config file is not shared between pods")

					Eventually(func() int {
						buf, _ := pods.ExecCommand(client.Client, pod, pod.Spec.Containers[0].Name, []string{"pmc", "-u", "-f", "/var/run/ptp4l.0.config", "GET CURRENT_DATA_SET"})
						return strings.Count(buf.String(), "offsetFromMaster")
					}, 3*time.Minute, 2*time.Second).Should(BeNumerically(">=", 1))
					buf, _ := pods.ExecCommand(client.Client, pod, pod.Spec.Containers[0].Name, []string{"ls", "-1", "-q", "-A", "/host/secrets/"})
					Expect(buf.String()).To(Equal(""))
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
})

func getOCPVersion() (string, error) {

	const OpenShiftAPIServer = "openshift-apiserver"

	ocpClient := client.Client.OcpClient
	clusterOperator, err := ocpClient.ClusterOperators().Get(context.TODO(), OpenShiftAPIServer, metav1.GetOptions{})

	var ocpVersion string
	if err != nil {
		switch {
		case kerrors.IsForbidden(err), kerrors.IsNotFound(err):
			logrus.Errorf("OpenShift Version not found (must be logged in to cluster as admin): %v", err)
			err = nil
		}
	}
	if clusterOperator != nil {
		for _, ver := range clusterOperator.Status.Versions {
			if ver.Name == OpenShiftAPIServer {
				ocpVersion = ver.Version
				break
			}
		}
	}
	logrus.Infof("OCP Version is %v", ocpVersion)

	return ocpVersion, err
}

func testCaseEnabled(testCase TestCase) bool {

	enabledTests, isSet := os.LookupEnv("ENABLE_TEST_CASE")

	if isSet {
		tokens := strings.Split(enabledTests, ",")
		for _, token := range tokens {
			token = strings.TrimSpace(token)
			if strings.Contains(token, string(testCase)) {
				return true
			}
		}
	}
	return false
}
