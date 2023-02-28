package dpdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovClean "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/clean"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/images"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/performanceprofile"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
)

const (
	LOG_ENTRY                 = "Accumulated forward statistics for all ports"
	SERVER_TESTPMD_COMMAND    = "testpmd -l ${CPU} -a ${PCIDEVICE_OPENSHIFT_IO_%s} --iova-mode=va -- -i --portmask=0x1 --nb-cores=2 --forward-mode=mac --port-topology=loop --no-mlockall"
	CLIENT_TESTPMD_COMMAND    = "testpmd -l ${CPU} -a ${PCIDEVICE_OPENSHIFT_IO_%s} --iova-mode=va -- -i --portmask=0x1 --nb-cores=2 --eth-peer=0,ff:ff:ff:ff:ff:ff --forward-mode=txonly --no-mlockall"
	CREATE_TAP_DEVICE_COMMAND = `
		ip tuntap add tap23 mode tap multi_queue
	`
	DPDK_SERVER_WORKLOAD_MAC = "60:00:00:00:00:01"
	DPDK_CLIENT_WORKLOAD_MAC = "60:00:00:00:00:02"
)

var (
	machineConfigPoolName          string
	performanceProfileName         string
	enforcedPerformanceProfileName string

	dpdkResourceName       = "dpdknic"
	regularPodResourceName = "regularnic"

	sriovclient *sriovtestclient.ClientSet

	sriovNicsTable []TableEntry

	workerCnfLabelSelector string
)

func init() {
	machineConfigPoolName = os.Getenv("ROLE_WORKER_CNF")
	if machineConfigPoolName == "" {
		machineConfigPoolName = "worker-cnf"
	}
	workerCnfLabelSelector = fmt.Sprintf("%s/%s=", utils.LabelRole, machineConfigPoolName)

	performanceProfileName = os.Getenv("PERF_TEST_PROFILE")
	if performanceProfileName == "" {
		performanceProfileName = "performance"
	} else {
		enforcedPerformanceProfileName = performanceProfileName
	}

	// When running in dry run as part of the docgen we want to skip the creation of descriptions for the nics table entries.
	// This way we don't add tests descriptions for the dynamically created table entries that depend on the environment they run on.
	isFillRun := os.Getenv("FILL_RUN") != ""
	if !isFillRun {
		supportedNicsConfigMap, err := networks.GetSupportedSriovNics()
		if err != nil {
			sriovNicsTable = append(sriovNicsTable, Entry("Failed getting supported SR-IOV nics", err.Error()))
		}

		for k, v := range supportedNicsConfigMap {
			ids := strings.Split(v, " ")
			sriovNicsTable = append(sriovNicsTable, Entry(k, ids[0], ids[1]))
		}
	}

	// Reuse the sriov client
	// Use the SRIOV test client
	sriovclient = sriovtestclient.New("")
}

var _ = Describe("[dpdk]", func() {
	var dpdkWorkloadPod *corev1.Pod
	var discoverySuccessful bool
	discoveryFailedReason := "Can not run tests in discovery mode. Failed to discover required resources"
	var nodeSelector map[string]string

	execute.BeforeAll(func() {
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
			profile, sriovDevice := discovered.Profile, discovered.Device
			dpdkResourceName = discovered.Resource

			nodeSelector = nodes.SelectorUnion(nodeSelector, profile.Spec.NodeSelector)
			networks.CreateSriovNetwork(sriovclient, sriovDevice, "test-dpdk-network", namespaces.DpdkTest, namespaces.SRIOVOperator, dpdkResourceName, "")

		} else {
			err = performanceprofile.FindOrOverridePerformanceProfile(performanceProfileName, machineConfigPoolName)
			Expect(err).ToNot(HaveOccurred())
		}

	})

	BeforeEach(func() {
		if discovery.Enabled() && !discoverySuccessful {
			Skip(discoveryFailedReason)
		}
	})

	Context("vhostnet", func() {
		var policyHasVhostnet bool
		execute.BeforeAll(func() {
			if !discovery.Enabled() {
				namespaces.CleanPods(namespaces.DpdkTest, sriovclient)
				networks.CleanSriov(sriovclient)
				networks.CreateSriovPolicyAndNetworkDPDKOnlyWithVhost(dpdkResourceName, workerCnfLabelSelector)
			} else {
				sriovNetworkNodePolicyList := &sriovv1.SriovNetworkNodePolicyList{}
				err := client.Client.List(context.TODO(), sriovNetworkNodePolicyList)
				Expect(err).ToNot(HaveOccurred())
				for _, policy := range sriovNetworkNodePolicyList.Items {
					if policy.Spec.ResourceName == dpdkResourceName && policy.Spec.NeedVhostNet {
						policyHasVhostnet = true
					}
				}
			}
		})

		BeforeEach(func() {
			if discovery.Enabled() && !policyHasVhostnet {
				Skip("Missing SriovNetworkNodePolicy with NeedVhostNet enabled")
			}
		})

		AfterEach(func() {
			namespaces.CleanPods(namespaces.DpdkTest, client.Client)
		})

		Context("Client should be able to forward packets", func() {
			It("Should be able to transmit packets", func() {
				var out string
				var err error

				command := CREATE_TAP_DEVICE_COMMAND + `
				dpdk-testpmd --vdev net_tap0,iface=tap23 --no-pci -- -ia --forward-mode txonly
				sleep INF
				`
				txDpdkWorkloadPod, err := pods.CreateDPDKWorkload(nodeSelector,
					command,
					images.For(images.Dpdk),
					[]corev1.Capability{"NET_ADMIN"},
					DPDK_SERVER_WORKLOAD_MAC,
				)
				Expect(err).ToNot(HaveOccurred())

				By("Parsing output from the client DPDK application")
				Eventually(func() string {
					out, err = pods.GetLog(txDpdkWorkloadPod)
					Expect(err).ToNot(HaveOccurred())
					return out
				}, 2*time.Minute, 1*time.Second).Should(ContainSubstring(LOG_ENTRY),
					"Cannot find accumulated statistics")
				By("Checking the tx output from the client DPDK application")
				checkTxOnly(out)

				bytes, err := getDeviceRXBytes(txDpdkWorkloadPod, "tap23")
				Expect(err).ToNot(HaveOccurred())
				Expect(bytes).To(BeNumerically(">", 0))
			})
		})

		Context("Server should be able to receive packets and forward to tap device", func() {
			It("Should be able to transmit packets", func() {
				var err error
				var out string
				// --stats-period is used to keep the command alive once the pod is started
				serverCommand := fmt.Sprintf(`
%s
dpdk-testpmd --vdev net_tap0,iface=tap23 -a ${PCIDEVICE_OPENSHIFT_IO_%s} -- --stats-period 5
sleep INF
				`, CREATE_TAP_DEVICE_COMMAND, strings.ToUpper(dpdkResourceName))
				dpdkWorkloadPod, err := pods.CreateDPDKWorkload(nodeSelector,
					serverCommand,
					images.For(images.Dpdk),
					[]corev1.Capability{"NET_ADMIN"},
					DPDK_SERVER_WORKLOAD_MAC)
				Expect(err).ToNot(HaveOccurred())

				clientCommand := fmt.Sprintf(`
dpdk-testpmd -a ${PCIDEVICE_OPENSHIFT_IO_%s} -- --forward-mode txonly --eth-peer=0,%s --stats-period 5
sleep INF
				`, strings.ToUpper(dpdkResourceName), DPDK_SERVER_WORKLOAD_MAC)
				_, err = pods.CreateDPDKWorkload(nodeSelector,
					clientCommand,
					images.For(images.Dpdk),
					[]corev1.Capability{"NET_ADMIN"},
					DPDK_CLIENT_WORKLOAD_MAC,
				)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() string {
					out, err = pods.GetLog(dpdkWorkloadPod)
					Expect(err).ToNot(HaveOccurred())
					return out
				}, 2*time.Minute, 1*time.Second).Should(ContainSubstring("Port statistics"),
					"Cannot find port statistics")

				By("Parsing output from the DPDK application")
				Eventually(func() bool {
					out, err = pods.GetLog(dpdkWorkloadPod)
					Expect(err).ToNot(HaveOccurred())
					return checkRxOnly(out)
				}, 8*time.Minute, 1*time.Second).Should(BeTrue(),
					"number of received packets should be greater than 0")

				By("Checking the rx output of tap device from the client DPDK application")
				Eventually(func() int {
					bytes, err := getDeviceRXBytes(dpdkWorkloadPod, "tap23")
					Expect(err).ToNot(HaveOccurred())
					return bytes
				}, 8*time.Minute, 1*time.Second).Should(BeNumerically(">", 0))
			})
		})
	})

	Context("VFS allocated for dpdk", func() {
		execute.BeforeAll(func() {
			if !discovery.Enabled() {
				namespaces.CleanPods(namespaces.DpdkTest, sriovclient)
				networks.CleanSriov(sriovclient)
				networks.CreateSriovPolicyAndNetworkDPDKOnly(dpdkResourceName, workerCnfLabelSelector)
			}
			var err error
			dpdkWorkloadPod, err = pods.CreateDPDKWorkload(nodeSelector,
				dpdkWorkloadCommand(strings.ToUpper(dpdkResourceName), fmt.Sprintf(SERVER_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName)), 60),
				images.For(images.Dpdk),
				nil,
				DPDK_SERVER_WORKLOAD_MAC,
			)
			Expect(err).ToNot(HaveOccurred())

			_, err = pods.CreateDPDKWorkload(nodeSelector,
				dpdkWorkloadCommand(strings.ToUpper(dpdkResourceName), fmt.Sprintf(CLIENT_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName)), 10),
				images.For(images.Dpdk),
				nil,
				DPDK_CLIENT_WORKLOAD_MAC,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("Validate a DPDK workload running inside a pod", func() {
			It("Should forward and receive packets", func() {
				Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
				var out string
				var err error

				By("Parsing output from the DPDK application")
				Eventually(func() string {
					out, err = pods.GetLog(dpdkWorkloadPod)
					Expect(err).ToNot(HaveOccurred())
					return out
				}, 8*time.Minute, 1*time.Second).Should(ContainSubstring(LOG_ENTRY),
					"Cannot find accumulated statistics")
				checkRxTx(out)
			})
		})

		Context("Validate NUMA aliment", func() {
			var cpuList []string
			BeforeEach(func() {
				Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
			})

			execute.BeforeAll(func() {
				buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
				Expect(err).ToNot(HaveOccurred())
				cpuList, err = getCpuSet(buff.String())
				Expect(err).ToNot(HaveOccurred())
			})

			// 28078
			It("should allocate the requested number of cpus", func() {
				numOfCPU := dpdkWorkloadPod.Spec.Containers[0].Resources.Limits.Cpu().Value()
				Expect(len(cpuList)).To(Equal(int(numOfCPU)))
			})

			// 28432
			It("should allocate all the resources on the same NUMA node", func() {
				By("finding the CPUs numa")
				cpuNumaNode, err := findNUMAForCPUs(dpdkWorkloadPod, cpuList)
				Expect(err).ToNot(HaveOccurred())

				By("finding the pci numa")
				pciNumaNode, err := findNUMAForSRIOV(dpdkWorkloadPod)
				Expect(err).ToNot(HaveOccurred())

				By("expecting cpu and pci to be on the same numa")
				Expect(cpuNumaNode).To(Equal(pciNumaNode))
			})
		})

		Context("Validate HugePages", func() {
			var activeNumberOfFreeHugePages int
			var numaNode int

			BeforeEach(func() {
				Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
				buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
				Expect(err).ToNot(HaveOccurred())
				cpuList, err := getCpuSet(buff.String())
				Expect(err).ToNot(HaveOccurred())
				numaNode, err = findNUMAForCPUs(dpdkWorkloadPod, cpuList)
				Expect(err).ToNot(HaveOccurred())

				buff, err = pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat",
					fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-1048576kB/free_hugepages", numaNode)})
				Expect(err).ToNot(HaveOccurred())
				activeNumberOfFreeHugePages, err = strconv.Atoi(strings.Replace(buff.String(), "\r\n", "", 1))
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allocate the amount of hugepages requested", func() {
				Expect(activeNumberOfFreeHugePages).To(BeNumerically(">", 0))

				// In case of nodeselector set, this pod will end up in a compliant node because the selection
				// logic is applied to the workload pod.
				pod := pods.DefineWithHugePages(namespaces.DpdkTest, dpdkWorkloadPod.Spec.NodeName)

				// The pod needs to request the same sriov device as the dpdk workload pod
				// using the topology manager it will ensure the hugepages allocated to this pod
				// are in the same numa.
				pod = pods.RedefinePodWithNetwork(pod, fmt.Sprintf("%s/test-dpdk-network", namespaces.DpdkTest))

				pod, err := client.Client.Pods(namespaces.DpdkTest).Create(context.Background(), pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() int {
					buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"cat",
						fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-1048576kB/free_hugepages", numaNode)})
					Expect(err).ToNot(HaveOccurred())
					numberOfFreeHugePages, err := strconv.Atoi(strings.Replace(buff.String(), "\r\n", "", 1))
					Expect(err).ToNot(HaveOccurred())
					return numberOfFreeHugePages
				}, 5*time.Minute, 5*time.Second).Should(Equal(activeNumberOfFreeHugePages - 1))

				pod, err = client.Client.Pods(namespaces.DpdkTest).Get(context.Background(), pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))

				err = client.Client.Pods(namespaces.DpdkTest).Delete(context.Background(), pod.Name, metav1.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() error {
					_, err := client.Client.Pods(namespaces.DpdkTest).Get(context.Background(), pod.Name, metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						return err
					}
					return nil
				}, 10*time.Second, 1*time.Second).Should(HaveOccurred())
			})
		})

		Context("Validate HugePages SeLinux access", func() {
			BeforeEach(func() {
				Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
			})

			It("should allow to remove the hugepage file inside the pod", func() {
				By("Checking hugepage file exist in the directory")
				buff, err := pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"ls", "/mnt/huge/"})
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute the list command error: %s", buff.String()))
				Expect(strings.Contains(buff.String(), "rtemap")).To(BeTrue())

				hugePagesFile := strings.TrimSuffix(buff.String(), "\r\n")
				buff, err = pods.ExecCommand(client.Client, *dpdkWorkloadPod, []string{"rm", fmt.Sprintf("/mnt/huge/%s", hugePagesFile)})
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute the remove command error: %s", buff.String()))
			})
		})
	})

	Context("VFS split for dpdk and netdevice", func() {
		BeforeEach(func() {
			if discovery.Enabled() {
				Skip("Split VF test disabled for discovery mode")
			}
		})
		execute.BeforeAll(func() {
			namespaces.CleanPods(namespaces.DpdkTest, sriovclient)
			networks.CleanSriov(sriovclient)
			createSriovPolicyAndNetworkShared()
			var err error
			dpdkWorkloadPod, err = pods.CreateDPDKWorkload(nodeSelector,
				dpdkWorkloadCommand(strings.ToUpper(dpdkResourceName), fmt.Sprintf(SERVER_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName)), 60),
				images.For(images.Dpdk),
				nil,
				DPDK_SERVER_WORKLOAD_MAC)
			Expect(err).ToNot(HaveOccurred())

			_, err = pods.CreateDPDKWorkload(nodeSelector,
				dpdkWorkloadCommand(strings.ToUpper(dpdkResourceName), fmt.Sprintf(CLIENT_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName)), 10),
				images.For(images.Dpdk),
				nil,
				DPDK_CLIENT_WORKLOAD_MAC)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should forward and receive packets from a pod running dpdk base", func() {
			Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
			var out string
			var err error

			By("Parsing output from the DPDK application")
			Eventually(func() string {
				out, err = pods.GetLog(dpdkWorkloadPod)
				Expect(err).ToNot(HaveOccurred())
				return out
			}, 8*time.Minute, 1*time.Second).Should(ContainSubstring(LOG_ENTRY),
				"Cannot find accumulated statistics")
			checkRxTx(out)
		})

		It("Run a regular pod using a vf shared with the dpdk's pf", func() {
			podDefinition := pods.DefineWithNetworks(namespaces.DpdkTest, []string{"test-regular-network"})
			pod, err := client.Client.Pods(namespaces.DpdkTest).Create(context.Background(), podDefinition, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForCondition(client.Client, pod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("dpdk application on different vendors", func() {
		var nodeNames []string
		var sriovInfos *sriovcluster.EnabledNodes
		var err error
		var out string

		execute.BeforeAll(func() {
			Expect(len(sriovNicsTable)).To(BeNumerically(">", 1))

			sriovInfos, err = sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
			Expect(err).ToNot(HaveOccurred())

			Expect(sriovInfos).ToNot(BeNil())

			nodeNames, err = nodes.MatchingCustomSelectorByName(sriovInfos.Nodes, workerCnfLabelSelector)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(nodeNames)).To(BeNumerically(">", 0))
		})

		BeforeEach(func() {
			if discovery.Enabled() {
				Skip("Split VF test disabled for discovery mode")
			}

			namespaces.CleanPods(namespaces.DpdkTest, sriovclient)
			networks.CleanSriov(sriovclient)
		})

		DescribeTable("Test connectivity using the requested nic", func(vendorID, deviceID string) {
			By("searching for the requested network card")
			if vendorID == networks.MlxVendorID {
				AllNodesHaveSecureBoot := true
				for _, nodeName := range sriovInfos.Nodes {
					if !sriovInfos.IsSecureBootEnabled[nodeName] {
						AllNodesHaveSecureBoot = false
						break
					}
				}

				if AllNodesHaveSecureBoot {
					Skip("skip nic validate as the nic is a Mellanox and all the nodes have secure boot enabled")
				}
			}
			node, sriovDevice, exist := findSriovDeviceForDPDK(sriovInfos, nodeNames, vendorID, deviceID)
			if !exist {
				Skip(fmt.Sprintf("skip nic validate as wasn't able to find a nic with vendorID %s and deviceID %s", vendorID, deviceID))
			}

			By("creating a network policy")
			networks.CreatePoliciesDPDKOnly(sriovDevice, node, dpdkResourceName, false)

			By("creating a network")
			networks.CreateSriovNetwork(sriovclient, sriovDevice, "test-dpdk-network", namespaces.DpdkTest, namespaces.SRIOVOperator, dpdkResourceName, "")

			By("creating a pod")
			txOnlydpdkWorkloadPod, err := pods.CreateDPDKWorkload(nodeSelector,
				dpdkWorkloadCommand(strings.ToUpper(dpdkResourceName), fmt.Sprintf(CLIENT_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName)), 5),
				images.For(images.Dpdk),

				nil, DPDK_SERVER_WORKLOAD_MAC)
			Expect(err).ToNot(HaveOccurred())

			By("Parsing output from the DPDK application")
			Eventually(func() string {
				out, err = pods.GetLog(txOnlydpdkWorkloadPod)
				Expect(err).ToNot(HaveOccurred())
				return out
			}, 8*time.Minute, 1*time.Second).Should(ContainSubstring(LOG_ENTRY),
				"Cannot find accumulated statistics")
			checkTxOnly(out)

		}, sriovNicsTable)
	})

	Context("Downward API", func() {
		execute.BeforeAll(func() {
			if discovery.Enabled() {
				Skip("Downward API test disabled for discovery mode")
			}
			networks.CleanSriov(sriovclient)
			createSriovPolicyAndNetworkShared()
			var err error
			dpdkWorkloadPod, err = pods.CreateDPDKWorkload(nodeSelector,
				dpdkWorkloadCommand(strings.ToUpper(dpdkResourceName), fmt.Sprintf(SERVER_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName)), 60),
				images.For(images.Dpdk),
				nil,
				DPDK_SERVER_WORKLOAD_MAC)
			Expect(err).ToNot(HaveOccurred())
			_, err = pods.CreateDPDKWorkload(nodeSelector,
				dpdkWorkloadCommand(strings.ToUpper(dpdkResourceName), fmt.Sprintf(CLIENT_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName)), 10),
				images.For(images.Dpdk),
				nil,
				DPDK_CLIENT_WORKLOAD_MAC)
			Expect(dpdkWorkloadPod.Spec.Volumes).ToNot(BeNil(), "Downward API volume not found")
			Expect(err).ToNot(HaveOccurred())

		})

		It("Volume is readable in container", func() {
			By("Label is present in container downward volume")
			containerlabel, _ := checkDownwardApi(dpdkWorkloadPod, "labels", "label")
			podLabel := dpdkWorkloadPod.Labels
			Expect(containerlabel).To(ContainSubstring(podLabel["app"]))

			By("Pod IP, MAC and PCI are present in container downward volume")
			containerIP, err := checkDownwardApi(dpdkWorkloadPod, "annotations", "IP")
			podIP := dpdkWorkloadPod.Status.PodIP
			containerMac, err := checkDownwardApi(dpdkWorkloadPod, "annotations", "MAC")
			podMacAdd := getPodMac(dpdkWorkloadPod)
			containerPci, err := checkDownwardApi(dpdkWorkloadPod, "annotations", "PCI")
			podPci := getPodPci(dpdkWorkloadPod)
			Expect(containerIP).To(ContainSubstring(podIP))
			Expect(containerMac).To(ContainSubstring(podMacAdd))
			Expect(containerPci).To(ContainSubstring(podPci))

			By("Huge pages is present in container downward volume")
			containerHPrequest, err := checkDownwardApi(dpdkWorkloadPod, "hugepages_1G_request_dpdk", "hugepages_request")
			containerHPlimit, err := checkDownwardApi(dpdkWorkloadPod, "hugepages_1G_limit_dpdk", "hugepages_limit")
			podHp := getHugePages(dpdkWorkloadPod)
			Expect(containerHPrequest).To(ContainSubstring(podHp))
			Expect(containerHPlimit).To(ContainSubstring(podHp))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	// TODO: find a better why to restore the configuration
	// This will not work if we use a random order running
	Context("restoring configuration", func() {
		It("should restore the cluster to the original status", func() {
			if !discovery.Enabled() {
				By(" restore performance profile")
				err := performanceprofile.RestorePerformanceProfile(machineConfigPoolName)
				Expect(err).ToNot(HaveOccurred())
			}

			By("cleaning the sriov test configuration")
			namespaces.CleanPods(namespaces.DpdkTest, sriovclient)
			networks.CleanSriov(sriovclient)
		})
	})
})

// checkRxTx parses the output from the DPDK test application
// and verifies that packets have passed the NIC TX and RX queues
func checkRxTx(out string) {
	lines := strings.Split(out, "\n")
	Expect(len(lines)).To(BeNumerically(">=", 3))
	for i, line := range lines {
		if strings.Contains(line, LOG_ENTRY) {
			d := getNumberOfPackets(lines[i+1], "RX")
			Expect(d).To(BeNumerically(">", 0), "number of received packets should be greater than 0")
			d = getNumberOfPackets(lines[i+2], "TX")
			Expect(d).To(BeNumerically(">", 0), "number of transferred packets should be greater than 0")
			break
		}
	}
}

// checkRx parses the output from the DPDK test application
// and verifies that packets have passed the NIC RX queues
func checkRxOnly(out string) bool {
	lines := strings.Split(out, "\n")
	Expect(len(lines)).To(BeNumerically(">=", 3))
	for i, line := range lines {
		if strings.Contains(line, "NIC statistics for port") {
			if len(lines) > i && getNumberOfPackets(lines[i+1], "RX") > 0 {
				return true
			}
		}
	}
	return false
}

// checkTxOnly parses the output from the DPDK test application
// and verifies that packets have passed the NIC TX queues
func checkTxOnly(out string) {
	lines := strings.Split(out, "\n")
	Expect(len(lines)).To(BeNumerically(">=", 3))
	for i, line := range lines {
		if strings.Contains(line, LOG_ENTRY) {
			d := getNumberOfPackets(lines[i+2], "TX")
			Expect(d).To(BeNumerically(">", 0), "number of transferred packets should be greater than 0")
			break
		}
	}
}

// getNumberOfPackets parses the string
// and returns an element representing the number of packets
func getNumberOfPackets(line, firstFieldSubstr string) int {
	r := strings.Fields(line)
	Expect(r[0]).To(ContainSubstring(firstFieldSubstr))
	Expect(len(r)).To(Equal(6), "the slice doesn't contain 6 elements")
	d, err := strconv.Atoi(r[1])
	Expect(err).ToNot(HaveOccurred())
	return d
}

func createSriovPolicyAndNetworkShared() {
	sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
	Expect(err).ToNot(HaveOccurred())

	Expect(sriovInfos).ToNot(BeNil())

	nn, err := nodes.MatchingCustomSelectorByName(sriovInfos.Nodes, workerCnfLabelSelector)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(nn)).To(BeNumerically(">", 0))

	sriovDevice, err := sriovInfos.FindOneSriovDevice(nn[0])
	Expect(err).ToNot(HaveOccurred())

	createPoliciesSharedPF(sriovDevice, nn[0], dpdkResourceName, regularPodResourceName)

	networks.CreateSriovNetwork(sriovclient, sriovDevice, "test-dpdk-network", namespaces.DpdkTest, namespaces.SRIOVOperator, dpdkResourceName, "")
	networks.CreateSriovNetwork(sriovclient, sriovDevice, "test-regular-network", namespaces.DpdkTest, namespaces.SRIOVOperator, regularPodResourceName, "")
}

func findSriovDeviceForDPDK(sriovInfos *sriovcluster.EnabledNodes, nodeNames []string, vendorID, deviceID string) (string, *sriovv1.InterfaceExt, bool) {
	for _, nodeName := range nodeNames {
		nodeState := sriovInfos.States[nodeName]

		for _, iface := range nodeState.Status.Interfaces {
			if iface.DeviceID == deviceID && iface.Vendor == vendorID {

				// If secure boot is enable and the request nic is a mlx one lets skip
				if sriovInfos.IsSecureBootEnabled[nodeName] && iface.Vendor == networks.MlxVendorID {
					continue
				}

				if networks.IsIntelDisabledNic(iface) {
					continue
				}

				return nodeName, &iface, true
			}
		}
	}

	return "", nil, false
}

func createPoliciesSharedPF(sriovDevice *sriovv1.InterfaceExt, testNode string, dpdkResourceName, regularResorceName string) {
	networks.CreateDpdkPolicy(sriovDevice, testNode, dpdkResourceName, "#0-1", 5, false)
	createRegularPolicy(sriovDevice, testNode, regularResorceName, "#2-4", 5)
	networks.WaitStable(sriovclient)

	Eventually(func() int64 {
		testedNode, err := sriovclient.Nodes().Get(context.Background(), testNode, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/"+dpdkResourceName)]
		capacity, _ := resNum.AsInt64()
		return capacity
	}, 10*time.Minute, time.Second).Should(Equal(int64(2)))

	Eventually(func() int64 {
		testedNode, err := sriovclient.Nodes().Get(context.Background(), testNode, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/"+regularResorceName)]
		capacity, _ := resNum.AsInt64()
		return capacity
	}, 10*time.Minute, time.Second).Should(Equal(int64(3)))
}

func createRegularPolicy(sriovDevice *sriovv1.InterfaceExt, testNode, dpdkResourceName, pfPartition string, vfsNum int) {
	regularPolicy := &sriovv1.SriovNetworkNodePolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-policy",
			Namespace:    namespaces.SRIOVOperator,
		},

		Spec: sriovv1.SriovNetworkNodePolicySpec{
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": testNode,
			},
			NumVfs:       vfsNum,
			ResourceName: dpdkResourceName,
			Priority:     99,
			NicSelector: sriovv1.SriovNetworkNicSelector{
				PfNames: []string{sriovDevice.Name + pfPartition},
			},
			DeviceType: "netdevice",
		},
	}

	err := sriovclient.Create(context.Background(), regularPolicy)
	Expect(err).ToNot(HaveOccurred())
}

func dpdkWorkloadCommand(dpdkResourceName, testpmdCommand string, runningTime int) string {
	return fmt.Sprintf(`set -ex
export CPU=$(cat /sys/fs/cgroup/cpuset/cpuset.cpus)
echo ${CPU}
echo ${PCIDEVICE_OPENSHIFT_IO_%s}

cat <<EOF >test.sh
spawn %s
set timeout 10000
expect "testpmd>"
send -- "port stop 0\r"
expect "testpmd>"
send -- "port detach 0\r"
expect "testpmd>"
send -- "port attach ${PCIDEVICE_OPENSHIFT_IO_%s}\r"
expect "testpmd>"
send -- "port start 0\r"
expect "testpmd>"
send -- "start\r"
expect "testpmd>"
sleep %d
send -- "stop\r"
expect "testpmd>"
send -- "quit\r"
expect eof
EOF

expect -f test.sh

sleep INF
`, dpdkResourceName, testpmdCommand, dpdkResourceName, runningTime)
}

// findDPDKWorkloadPod finds a pod running a DPDK application using a know label
// Label: app="dpdk"
func findDPDKWorkloadPod() (*corev1.Pod, bool, error) {
	return findDPDKWorkloadPodByLabelSelector(labels.SelectorFromSet(labels.Set{"app": "dpdk"}).String(), namespaces.DpdkTest)
}

func findDPDKWorkloadPodByLabelSelector(labelSelector, namespace string) (*corev1.Pod, bool, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	p, err := client.Client.Pods(namespace).List(context.Background(), listOptions)
	if err != nil {
		return nil, false, fmt.Errorf("cannot list pods for %s: %w", labelSelector, err)
	}

	if len(p.Items) == 0 {
		return nil, false, nil
	}

	var pod corev1.Pod
	podReady := false
	for _, pod = range p.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podReady = true
			break
		}
	}

	if !podReady {
		return nil, false, nil
	}

	err = pods.WaitForCondition(client.Client, &pod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
	if err != nil {
		return nil, false, fmt.Errorf("error while waiting for pod %s to be ready: %w", pod.Name, err)
	}

	return &pod, true, nil
}

func getCpuSet(cpuListOutput string) ([]string, error) {
	cpuList := make([]string, 0)
	cpuListOutputClean := strings.Replace(cpuListOutput, "\r\n", "", 1)
	cpuListOutputClean = strings.Replace(cpuListOutputClean, " ", "", -1)
	cpuRangeList := strings.Split(cpuListOutputClean, ",")

	for _, cpuRange := range cpuRangeList {
		cpuSplit := strings.Split(cpuRange, "-")
		if len(cpuSplit) == 1 {
			cpuList = append(cpuList, cpuSplit[0])
			continue
		}

		if len(cpuSplit) != 2 {
			return nil, fmt.Errorf("unexpected cpu list: %s", cpuListOutput)
		}

		idx, err := strconv.Atoi(cpuSplit[0])
		if err != nil {
			return nil, fmt.Errorf("bad conversion for string %s: %w", cpuSplit[0], err)
		}
		endIdx, err := strconv.Atoi(cpuSplit[1])
		if err != nil {
			return nil, fmt.Errorf("bad conversion for string %s:: %w", cpuSplit[1], err)
		}

		for ; idx <= endIdx; idx++ {
			cpuList = append(cpuList, strconv.Itoa(idx))
		}

	}

	return cpuList, nil
}

// findNUMAForCPUs finds the NUMA node if all the CPUs in the list are in the same one
func findNUMAForCPUs(pod *corev1.Pod, cpuList []string) (int, error) {
	buff, err := pods.ExecCommand(client.Client, *pod, []string{"lscpu"})
	Expect(err).ToNot(HaveOccurred())
	findCPUOnSameNuma := false
	numaNode := -1
	for _, line := range strings.Split(buff.String(), "\r\n") {
		if strings.Contains(line, "CPU(s)") && strings.Contains(line, "NUMA") {
			numaNode++
			numaLine := strings.Split(line, "CPU(s):   ")
			Expect(len(numaLine)).To(Equal(2))
			cpuMap := make(map[string]bool)

			cpuNumaList, err := getCpuSet(numaLine[1])
			Expect(err).ToNot(HaveOccurred())
			for _, cpu := range cpuNumaList {
				cpuMap[cpu] = true
			}

			findCPUs := true
			for _, cpu := range cpuList {
				if _, ok := cpuMap[cpu]; !ok {
					findCPUs = false
					break
				}
			}

			if findCPUs {
				findCPUOnSameNuma = true
				break
			}
		}
	}

	if !findCPUOnSameNuma {
		return numaNode, fmt.Errorf("not all the cpus are in the same numa node")
	}

	return numaNode, nil
}

// findNUMAForSRIOV finds the NUMA node for a give PCI address
func findNUMAForSRIOV(pod *corev1.Pod) (int, error) {
	buff, err := pods.ExecCommand(client.Client, *pod, []string{"env"})
	Expect(err).ToNot(HaveOccurred())
	pciAddress := ""
	pciEnvVariableName := fmt.Sprintf("PCIDEVICE_OPENSHIFT_IO_%s", strings.ToUpper(dpdkResourceName))
	for _, line := range strings.Split(buff.String(), "\r\n") {
		if strings.Contains(line, pciEnvVariableName) {
			envSplit := strings.Split(line, "=")
			Expect(len(envSplit)).To(Equal(2))
			pciAddress = envSplit[1]
		}
	}
	Expect(pciAddress).ToNot(BeEmpty())

	buff, err = pods.ExecCommand(client.Client, *pod, []string{"lspci", "-v", "-nn", "-mm", "-s", pciAddress})
	Expect(err).ToNot(HaveOccurred())
	for _, line := range strings.Split(buff.String(), "\r\n") {
		if strings.Contains(line, "NUMANode:") {
			numaSplit := strings.Split(line, "NUMANode:\t")
			Expect(len(numaSplit)).To(Equal(2))
			pciNuma, err := strconv.Atoi(numaSplit[1])
			Expect(err).ToNot(HaveOccurred())
			return pciNuma, err
		}
	}
	return -1, fmt.Errorf("failed to find the numa for pci %s", pciEnvVariableName)
}

func checkDownwardApi(pod *corev1.Pod, path string, c string) (string, error) {
	var output string

	switch {
	case c == "label" || c == "IP":
		buff, err := pods.ExecCommand(client.Client, *pod, []string{"cat", "/etc/podnetinfo/" + path})
		Expect(err).ToNot(HaveOccurred())
		output = buff.String()

	case c == "MAC":
		buff, err := pods.ExecCommand(client.Client, *pod, []string{"cat", "/etc/podnetinfo/" + path})
		Expect(err).ToNot(HaveOccurred())
		for _, line := range strings.Split(buff.String(), "\n") {
			if strings.Contains(line, "mac_address") {
				r := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)
				arr := r.FindAllString(line, -1)

				for _, v := range arr {
					trim := v[:len(v)-1]
					hw, _ := net.ParseMAC(trim)
					if hw != nil {
						output = fmt.Sprintln(hw)
						break
					}
				}

			}
		}
	case c == "PCI":
		buff, err := pods.ExecCommand(client.Client, *pod, []string{"cat", "/etc/podnetinfo/" + path})
		Expect(err).ToNot(HaveOccurred())
		for _, line := range strings.Split(buff.String(), "\n") {
			if strings.Contains(line, "pci-address") {
				r := regexp.MustCompile(`pci-address\\": \\".+?\\`)
				arr := r.FindAllString(line, -1)
				output = fmt.Sprintln(arr[0])
				break
			}
		}
	case c == "hugepages_request" || c == "hugepages_limit":
		buff, err := pods.ExecCommand(client.Client, *pod, []string{"cat", "/etc/podnetinfo/" + path})
		Expect(err).ToNot(HaveOccurred())
		output = buff.String()
	}
	return output, nil
}

func getPodMac(pod *corev1.Pod) string {
	var output string
	podMacAdd := pod.ObjectMeta.Annotations
	fmt.Println(podMacAdd["networks-status:"])
	contMacAdd, _ := json.Marshal(podMacAdd)
	strout := string(contMacAdd)
	for _, line := range strings.Split(strout, "\".\"") {
		if strings.Contains(line, "mac_address") {
			r := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)
			arr := r.FindAllString(line, -1)
			for _, v := range arr {
				trim := v[:len(v)-1]
				hw, _ := net.ParseMAC(trim)
				if hw != nil {
					output = fmt.Sprintln(hw)
					break
				}
			}
		}
	}
	return output
}

func getPodPci(pod *corev1.Pod) string {
	var output string
	podPciAdd := pod.ObjectMeta.Annotations
	contMacAdd, _ := json.Marshal(podPciAdd)
	strout := string(contMacAdd)
	for _, line := range strings.Split(strout, "\".\"") {
		if strings.Contains(line, "pci-address") {
			r := regexp.MustCompile(`pci-address\\": \\".+?\\`)
			arr := r.FindAllString(line, -1)
			output = fmt.Sprintln(arr[0])
			break
		}
	}
	return output
}

func getHugePages(pod *corev1.Pod) string {
	var mb int64
	podHp := pod.Spec.Containers
	s := fmt.Sprintln(podHp)
	for _, line := range strings.Split(s, " ") {
		if strings.Contains(line, "hugepages-1Gi:") {
			r := regexp.MustCompile(`\d{2,}|[7-9]`)
			b := r.FindAllString(line, -1)
			num := path.Join(b...)
			number, _ := strconv.ParseInt(num, 10, 0)
			fmt.Sprintln(number)
			mb = number / 1024 / 1024
		}
	}
	return fmt.Sprint(mb)
}

// getDeviceRXBytes queries the specied interface on given pod for RX bytes
// returns the number of RX bytes as int or error if the query fail
func getDeviceRXBytes(pod *corev1.Pod, device string) (int, error) {
	statsCommand := []string{"ip", "-s", "l", "show", "dev", device}
	stats, err := pods.ExecCommand(client.Client, *pod, statsCommand)
	if err != nil {
		return 0, fmt.Errorf("command %v error: %w", statsCommand, err)
	}
	statsLines := strings.Split(stats.String(), "\n")
	for i, line := range statsLines {
		if strings.Contains(strings.Trim(line, " "), "RX:") {
			if len(statsLines) < i+2 {
				return -1, fmt.Errorf("could not find RX in stats %v", statsLines)
			}
			nextLine := strings.Trim(statsLines[i+1], " ")
			return strconv.Atoi(strings.Split(strings.Trim(nextLine, " "), " ")[0])

		}
	}
	return -1, fmt.Errorf("could not find RX stats: %v", statsLines)
}
