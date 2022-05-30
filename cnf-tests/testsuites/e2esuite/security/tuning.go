package security

import (
	"context"
	"fmt"
	"strings"
	"time"

	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	TestNamespace = "tuning-testing"
	Sysctl        = "net.ipv4.conf.%s.send_redirects"
)

var _ = Describe("tuningcni", func() {
	apiclient := client.New("")

	execute.BeforeAll(func() {
		err := namespaces.Create(TestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := namespaces.CleanPods(TestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("tuningcni over macvlan", func() {
		It("should be able to create pod with sysctls over macvlan",
			func() {
				nadName := "tuning-nad"
				sysctlValue := "1"
				sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
				Expect(err).ToNot(HaveOccurred())
				nad, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, nadName).WithMacVlan().WithStaticIpam("10.10.0.1").WithTuning(sysctls).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), nad)
				Expect(err).ToNot(HaveOccurred())
				podDefinition := pods.DefineWithNetworks(TestNamespace, []string{fmt.Sprintf("%s/%s", TestNamespace, nadName)})
				pod, err := client.Client.Pods(TestNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = pods.WaitForPhase(client.Client, pod, corev1.PodRunning, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())
				sysctlForInterface := fmt.Sprintf(Sysctl, "net1")
				statsCommand := []string{"sysctl", sysctlForInterface}
				commandOutput, err := pods.ExecCommand(client.Client, *pod, statsCommand)
				Expect(strings.TrimSpace(string(commandOutput.Bytes()))).To(Equal(fmt.Sprintf("%s = %s", sysctlForInterface, sysctlValue)))
			})

		It("pods with sysctl's over macvlan should be able to ping each other", func() {
			nad1Name := "tuning-nad1"
			nad2Name := "tuning-nad2"
			ip1 := "10.10.0.10"
			ip2 := "10.10.0.11"

			sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
			Expect(err).ToNot(HaveOccurred())
			nad1, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, nad1Name).WithMacVlan().WithStaticIpam(ip1).WithTuning(sysctls).Build()
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Create(context.Background(), nad1)
			Expect(err).ToNot(HaveOccurred())

			sysctls, err = networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
			Expect(err).ToNot(HaveOccurred())
			nad2, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, nad2Name).WithMacVlan().WithStaticIpam(ip2).WithTuning(sysctls).Build()
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Create(context.Background(), nad2)
			Expect(err).ToNot(HaveOccurred())

			podDefinition := pods.DefineWithNetworks(TestNamespace, []string{fmt.Sprintf("%s/%s", TestNamespace, nad1Name)})
			pod, err := client.Client.Pods(TestNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForPhase(client.Client, pod, corev1.PodRunning, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			podDefinition2 := pods.DefineWithNetworks(TestNamespace, []string{fmt.Sprintf("%s/%s", TestNamespace, nad2Name)})
			podDefinition2 = pods.RedefineWithCommand(podDefinition2, []string{"/bin/bash", "-c", fmt.Sprintf("ping -c 1 %s", ip1)}, nil)
			podDefinition2 = pods.RedefineWithRestartPolicy(podDefinition2, corev1.RestartPolicyNever)
			podDefinition2.Spec.NodeName = pod.Spec.NodeName
			pod2, err := client.Client.Pods(TestNamespace).Create(context.Background(), podDefinition2, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForPhase(client.Client, pod2, corev1.PodSucceeded, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("tuningcni over bond", func() {
		It("should be able to create pod with sysctls over bond",
			func() {
				macvlanNadName := "macvlan-nad"
				bondNadName := "bond-nad"
				bondInterface := "bond0"
				slaveInterface0 := "slave0"
				slaveInterface1 := "slave1"
				sysctlValue := "1"
				sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): sysctlValue})
				Expect(err).ToNot(HaveOccurred())

				bondWithTuningNad, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, bondNadName).WithBond(bondInterface, slaveInterface0, slaveInterface1, 1300).WithStaticIpam("10.10.0.12").WithTuning(sysctls).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), bondWithTuningNad)
				Expect(err).ToNot(HaveOccurred())

				macVlandNad, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, macvlanNadName).WithMacVlan().Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), macVlandNad)
				Expect(err).ToNot(HaveOccurred())

				podDefinition := pods.DefineWithNetworks(TestNamespace, []string{fmt.Sprintf("%s/%s@%s, %s/%s@%s, %s/%s@%s", TestNamespace, macvlanNadName, slaveInterface0, TestNamespace, macvlanNadName, slaveInterface1, TestNamespace, bondNadName, bondInterface)})
				pod, err := client.Client.Pods(TestNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = pods.WaitForPhase(client.Client, pod, corev1.PodRunning, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())
				sysctlForInterface := fmt.Sprintf(Sysctl, bondInterface)
				statsCommand := []string{"sysctl", sysctlForInterface}
				commandOutput, err := pods.ExecCommand(client.Client, *pod, statsCommand)
				Expect(strings.TrimSpace(string(commandOutput.Bytes()))).To(Equal(fmt.Sprintf("%s = %s", sysctlForInterface, sysctlValue)))
			})
	})
})
