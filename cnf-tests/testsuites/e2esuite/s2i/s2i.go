package s2i

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/performanceprofile"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

const (
	LOG_ENTRY                = "Aggregate statistics"
	DEMO_APP_NAMESPACE       = "dpdk"
	DPDK_SERVER_WORKLOAD_MAC = "60:00:00:00:00:01"
	DPDK_CLIENT_WORKLOAD_MAC = "60:00:00:00:00:02"
	CLIENT_TESTPMD_COMMAND   = `set -ex
export CPU=$(cat /sys/fs/cgroup/cpuset/cpuset.cpus)
echo ${CPU}
echo ${PCIDEVICE_OPENSHIFT_IO_%s}
testpmd -l ${CPU} -a ${PCIDEVICE_OPENSHIFT_IO_%s} --iova-mode=va --  --portmask=0x1 --nb-cores=2 --eth-peer=0,ff:ff:ff:ff:ff:ff --forward-mode=txonly --no-mlockall --stats-period 5
`
)

var (
	machineConfigPoolName          string
	performanceProfileName         string
	enforcedPerformanceProfileName string

	dpdkResourceName = "dpdknic"

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

var _ = Describe("[s2i]", func() {
	var dpdkWorkloadPod *corev1.Pod
	var nodeSelector map[string]string

	execute.BeforeAll(func() {
		testInfo := CurrentGinkgoTestDescription()
		fmt.Printf("%v", testInfo)
		imageStream, err := client.Client.ImageStreams(DEMO_APP_NAMESPACE).Get(context.TODO(), "s2i-dpdk-app", metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
		}

		if discovery.Enabled() {
			Skip("s2i test is not supported in discovery mode")
		}
		Expect(len(imageStream.Status.Tags)).To(BeNumerically(">", 0))

		// Allow access from the test namespace to the imagestream located namespace in dpdk
		roleBind, err := client.Client.RoleBindings(DEMO_APP_NAMESPACE).Get(context.TODO(), "system:image-puller", metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}

			//We need to create a rolebinding to allow the dpdk-testing project to pull image from the dpdk project
			roleBind := rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "system:image-puller", Namespace: DEMO_APP_NAMESPACE},
				RoleRef: rbacv1.RoleRef{Name: "system:image-puller", Kind: "ClusterRole", APIGroup: "rbac.authorization.k8s.io"},
				Subjects: []rbacv1.Subject{
					{Kind: "ServiceAccount", Name: "default", Namespace: namespaces.DpdkTest},
				}}

			_, err = client.Client.RoleBindings(DEMO_APP_NAMESPACE).Create(context.TODO(), &roleBind, metav1.CreateOptions{})
			if err != nil {
				Expect(err).ToNot(HaveOccurred())
			}
		} else {
			exist := false
			for _, subject := range roleBind.Subjects {
				if subject.Namespace == namespaces.DpdkTest {
					exist = true
					break
				}
			}

			if !exist {
				roleBind.Subjects = append(roleBind.Subjects, rbacv1.Subject{Kind: "ServiceAccount", Name: "default", Namespace: namespaces.DpdkTest})
				_, err = client.Client.RoleBindings(DEMO_APP_NAMESPACE).Update(context.TODO(), roleBind, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		}

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

		err = performanceprofile.FindOrOverridePerformanceProfile(performanceProfileName, machineConfigPoolName)
		Expect(err).ToNot(HaveOccurred())

		namespaces.CleanPods(namespaces.DpdkTest, sriovclient)
		networks.CleanSriov(sriovclient)
		networks.CreateSriovPolicyAndNetworkDPDKOnly(dpdkResourceName, workerCnfLabelSelector)

		dpdkWorkloadPod, err = pods.CreateDPDKWorkload(nodeSelector,
			"/usr/libexec/s2i/run",
			"image-registry.openshift-image-registry.svc:5000/dpdk/s2i-dpdk-app:latest",
			nil,
			DPDK_SERVER_WORKLOAD_MAC,
		)
		Expect(err).ToNot(HaveOccurred())

		_, err = pods.CreateDPDKWorkload(nodeSelector,
			fmt.Sprintf(CLIENT_TESTPMD_COMMAND, strings.ToUpper(dpdkResourceName), strings.ToUpper(dpdkResourceName)),
			images.For(images.Dpdk),
			nil,
			DPDK_CLIENT_WORKLOAD_MAC,
		)
		Expect(err).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		if discovery.Enabled() {
			Skip("s2i test is not supported in discovery mode")
		}
	})

	Context("VFS allocated for l2fw application", func() {
		Context("Validate the build", func() {
			It("Should forward and receive packets from a pod running l2fw application base on a image created by building config", func() {
				Expect(dpdkWorkloadPod).ToNot(BeNil(), "No dpdk workload pod found")
				var out string
				var err error

				if dpdkWorkloadPod.Spec.Containers[0].Image == images.For(images.Dpdk) {
					Skip("skip test as we can't find a dpdk workload running with a s2i build")
				}

				By("Parsing output from the DPDK application")
				Eventually(func() bool {
					out, err = pods.GetLog(dpdkWorkloadPod)
					Expect(err).ToNot(HaveOccurred())
					return checkRxTx(out)
				}, 8*time.Minute, 1*time.Second).Should(BeTrue(),
					"Cannot find accumulated statistics")

			})
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

// checkRxTx parses the output from the L2FWD DPDK test application
// and verifies that packets have passed the NIC TX and RX queues
func checkRxTx(out string) bool {
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if strings.Contains(line, LOG_ENTRY) {
			if len(lines) < i+3 {
				break
			}
			d := getNumberOfPackets(lines[i+1], "Total")
			// put 1000 to be sure the traffic is by the testpmd generator and not random broadcast
			if d <= 1000 {
				continue
			}

			d = getNumberOfPackets(lines[i+2], "Total")
			if d <= 1000 {
				continue
			}
			return true
		}
	}
	return false
}

// getNumberOfPackets parses the string
// and returns an element representing the number of packets
func getNumberOfPackets(line, firstFieldSubstr string) int {
	r := strings.Fields(line)
	Expect(r[0]).To(ContainSubstring(firstFieldSubstr))
	Expect(len(r)).To(Equal(4), "the slice doesn't contain 6 elements")
	d, err := strconv.Atoi(r[3])
	Expect(err).ToNot(HaveOccurred())
	return d
}
