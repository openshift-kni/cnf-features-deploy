package security

import (
	"context"
	"fmt"
	"time"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var sriovclient *sriovtestclient.ClientSet

func init() {
	sriovclient = sriovtestclient.New("")
}

var _ = Describe("[sriov] Tuning CNI integration", func() {
	apiclient := client.New("")

	execute.BeforeAll(func() {
		err := namespaces.Create(namespaces.SriovTuningTest, apiclient)
		Expect(err).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		namespaces.CleanPods(namespaces.SriovTuningTest, apiclient)
	})

	Context("tuning cni over sriov", func() {
		BeforeEach(func() {
			if discovery.Enabled() {
				Skip("Tuned sriov tests disabled for discovery mode")
			}

		})

		execute.BeforeAll(func() {
			networks.CleanSriov(sriovclient)
			sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
			Expect(err).ToNot(HaveOccurred())
			networks.CreateSriovPolicyAndNetwork(
				sriovclient, namespaces.SRIOVOperator, "test-network", "testresource", fmt.Sprintf("{%s}", sysctls))

			By("Checking the network-attachment-defintion is ready")
			Eventually(func() error {
				nad := netattdefv1.NetworkAttachmentDefinition{}
				objKey := apitypes.NamespacedName{
					Namespace: namespaces.SRIOVOperator,
					Name:      "test-network",
				}
				err := client.Client.Get(context.Background(), objKey, &nad)
				return err
			}, 2*time.Minute, 1*time.Second).Should(BeNil())
		})

		It("pods with sysctl's over sriov interface should start", func() {
			podDefinition := pods.DefineWithNetworks(namespaces.SriovTuningTest,
				[]string{fmt.Sprintf("%s/%s", namespaces.SRIOVOperator, "test-network")})
			pod, err := client.Client.Pods(namespaces.SriovTuningTest).
				Create(context.Background(), podDefinition, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForCondition(client.Client, pod, corev1.ContainersReady, corev1.ConditionTrue, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			namespaces.CleanPods(namespaces.SriovTuningTest, apiclient)
		})

		It("pods with sysctl's on bond over sriov interfaces should start", func() {
			// NOTE: due to a bond cni bug we need to specify the name of the bond interface in both the cni config and
			// the multus annotation for the network.
			bondLinkName := "bond0"
			sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
			Expect(err).ToNot(HaveOccurred())
			bondNetworkAttachmentDefinition, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.SriovTuningTest, "bond").
				WithBond(bondLinkName, "net1", "net2", 1300).WithHostLocalIpam("1.1.1.0").WithTuning(sysctls).Build()
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Create(context.Background(), bondNetworkAttachmentDefinition)
			Expect(err).ToNot(HaveOccurred())

			podDefinition := pods.DefineWithNetworks(namespaces.SriovTuningTest, []string{
				fmt.Sprintf("%s/%s", namespaces.SRIOVOperator, "test-network"),
				fmt.Sprintf("%s/%s", namespaces.SRIOVOperator, "test-network"),
				fmt.Sprintf("%s/%s@%s", namespaces.SriovTuningTest, "bond", bondLinkName),
			})
			pod, err := client.Client.Pods(namespaces.SriovTuningTest).
				Create(context.Background(), podDefinition, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForCondition(client.Client, pod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			err = client.Client.Delete(context.Background(), bondNetworkAttachmentDefinition)
			Expect(err).ToNot(HaveOccurred())
			namespaces.CleanPods(namespaces.SriovTuningTest, apiclient)
		})
	})
})
