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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	Sysctl = "net.ipv4.conf.%s.send_redirects"
)

var _ = Describe("[tuningcni]", func() {
	apiclient := client.New("")

	execute.BeforeAll(func() {
		err := namespaces.Create(namespaces.TuningTest, apiclient)
		Expect(err).ToNot(HaveOccurred())

	})

	BeforeEach(func() {
		namespaces.CleanPods(namespaces.TuningTest, apiclient)
	})

	Context("tuningcni over macvlan", func() {
		It("should be able to create pod with sysctls over macvlan",
			func() {
				nadName := "tuning-nad"
				sysctlValue := "1"
				sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
				Expect(err).ToNot(HaveOccurred())
				nad, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.TuningTest, nadName).WithMacVlan().WithStaticIpam("10.10.0.1").WithTuning(sysctls).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), nad)
				Expect(err).ToNot(HaveOccurred())
				podDefinition := pods.DefineWithNetworks(namespaces.TuningTest, []string{fmt.Sprintf("%s/%s", namespaces.TuningTest, nadName)})
				pod, err := client.Client.Pods(namespaces.TuningTest).Create(context.Background(), podDefinition, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = pods.WaitForPhase(client.Client, pod, corev1.PodRunning, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred(), pods.GetStringEventsForPodFn(client.Client, pod))
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
			nad1, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.TuningTest, nad1Name).WithMacVlan().WithStaticIpam(ip1).WithTuning(sysctls).Build()
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Create(context.Background(), nad1)
			Expect(err).ToNot(HaveOccurred())

			sysctls, err = networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
			Expect(err).ToNot(HaveOccurred())
			nad2, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.TuningTest, nad2Name).WithMacVlan().WithStaticIpam(ip2).WithTuning(sysctls).Build()
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Create(context.Background(), nad2)
			Expect(err).ToNot(HaveOccurred())

			podDefinition := pods.DefineWithNetworks(namespaces.TuningTest, []string{fmt.Sprintf("%s/%s", namespaces.TuningTest, nad1Name)})
			pod, err := client.Client.Pods(namespaces.TuningTest).Create(context.Background(), podDefinition, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForPhase(client.Client, pod, corev1.PodRunning, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred(), pods.GetStringEventsForPodFn(client.Client, pod))

			podDefinition2 := pods.DefineWithNetworks(namespaces.TuningTest, []string{fmt.Sprintf("%s/%s", namespaces.TuningTest, nad2Name)})
			podDefinition2 = pods.RedefineWithCommand(podDefinition2, []string{"/bin/bash", "-c", fmt.Sprintf("ping -c 1 %s", ip1)}, nil)
			podDefinition2 = pods.RedefineWithRestartPolicy(podDefinition2, corev1.RestartPolicyNever)
			podDefinition2.Spec.NodeName = pod.Spec.NodeName
			pod2, err := client.Client.Pods(namespaces.TuningTest).Create(context.Background(), podDefinition2, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForPhase(client.Client, pod2, corev1.PodSucceeded, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred(), pods.GetStringEventsForPodFn(client.Client, pod2))
		})
	})

	Context("tuningcni over bond", func() {
		It("pods with sysctls over bond should be able to ping each other",
			func() {
				macvlanNadName := "macvlan-nad"
				ip1 := "10.10.0.22"
				bondNadName1 := "bond-nad1"
				bondNadName2 := "bond-nad2"

				sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
				Expect(err).ToNot(HaveOccurred())

				bondWithTuningNad1, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.TuningTest, bondNadName1).WithBond("net3", "net1", "net2", 1300).WithStaticIpam(ip1).WithTuning(sysctls).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), bondWithTuningNad1)
				Expect(err).ToNot(HaveOccurred())

				bondWithTuningNad2, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.TuningTest, bondNadName2).WithBond("net3", "net1", "net2", 1300).WithStaticIpam("10.10.0.23").WithTuning(sysctls).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), bondWithTuningNad2)
				Expect(err).ToNot(HaveOccurred())

				macVlandNad, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.TuningTest, macvlanNadName).WithMacVlan().Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), macVlandNad)
				Expect(err).ToNot(HaveOccurred())

				podDefinition1 := pods.DefineWithNetworks(namespaces.TuningTest, []string{fmt.Sprintf("%s/%s, %s/%s, %s/%s", namespaces.TuningTest, macvlanNadName, namespaces.TuningTest, macvlanNadName, namespaces.TuningTest, bondNadName1)})
				pod1, err := client.Client.Pods(namespaces.TuningTest).Create(context.Background(), podDefinition1, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = pods.WaitForPhase(client.Client, pod1, corev1.PodRunning, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred(), pods.GetStringEventsForPodFn(client.Client, pod1))

				podDefinition2 := pods.DefineWithNetworks(namespaces.TuningTest, []string{fmt.Sprintf("%s/%s, %s/%s, %s/%s", namespaces.TuningTest, macvlanNadName, namespaces.TuningTest, macvlanNadName, namespaces.TuningTest, bondNadName2)})
				podDefinition2 = pods.RedefineWithCommand(podDefinition2, []string{"/bin/bash", "-c", fmt.Sprintf("ping -c 1 %s", ip1)}, nil)
				podDefinition2 = pods.RedefineWithRestartPolicy(podDefinition2, corev1.RestartPolicyNever)
				podDefinition2.Spec.NodeName = pod1.Spec.NodeName
				pod2, err := client.Client.Pods(namespaces.TuningTest).Create(context.Background(), podDefinition2, metav1.CreateOptions{})

				Expect(err).ToNot(HaveOccurred())
				err = pods.WaitForPhase(client.Client, pod2, corev1.PodSucceeded, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred(), pods.GetStringEventsForPodFn(client.Client, pod2))
			})
	})

	Context("sysctl allowlist update", func() {
		var originalSysctls = ""

		BeforeEach(func() {
			cm, err := client.Client.ConfigMaps("openshift-multus").Get(context.TODO(), "cni-sysctl-allowlist", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			var ok bool
			originalSysctls, ok = cm.Data["allowlist.conf"]
			Expect(ok).To(BeTrue())
		})

		AfterEach(func() {
			updateAllowlistConfig(originalSysctls)
		})

		It("should start a pod with custom sysctl only after adding sysctl to allowlist", func() {
			macvlanNadName := "macvlan-nad1"
			sysctl := "net.ipv4.conf.IFNAME.accept_local"
			updatedSysctls := originalSysctls + "\n^" + sysctl + "$"

			podSysctls, err := networks.SysctlConfig(map[string]string{sysctl: "1"})
			macVlandNad, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.TuningTest, macvlanNadName).WithMacVlan().WithTuning(podSysctls).Build()
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Create(context.Background(), macVlandNad)
			Expect(err).ToNot(HaveOccurred())

			podDefinition := pods.DefineWithNetworks(namespaces.TuningTest, []string{fmt.Sprintf("%s/%s", namespaces.TuningTest, macvlanNadName)})
			pod, err := client.Client.Pods(namespaces.TuningTest).Create(context.Background(), podDefinition, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				checkPod, err := client.Client.Pods(namespaces.TuningTest).Get(context.Background(), pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if checkPod.Status.Phase != corev1.PodPending {
					return false
				}
				eventExists, err := syscltFailureEventExists(pod.Name, namespaces.TuningTest, sysctl)
				Expect(err).NotTo(HaveOccurred())
				return eventExists
			}, 15*time.Second, 3*time.Second).Should(BeTrue())

			// Until now, the pod keeps on retrying to start, but fails due to CNI errors
			// Once the allowlist is updated, the CNI plugin will exit successfully, and the pod should start
			err = updateAllowlistConfig(updatedSysctls)
			Expect(err).NotTo(HaveOccurred())
			err = pods.WaitForPhase(client.Client, pod, corev1.PodRunning, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred(), pods.GetStringEventsForPodFn(client.Client, pod))
		})
	})
})

func updateAllowlistConfig(sysctls string) error {
	cm, err := client.Client.ConfigMaps("openshift-multus").Get(context.TODO(), "cni-sysctl-allowlist", metav1.GetOptions{})
	if err != nil {
		return err
	}
	cm.Data["allowlist.conf"] = sysctls
	_, err = client.Client.ConfigMaps("openshift-multus").Update(context.TODO(), cm, metav1.UpdateOptions{})
	return err
}

func syscltFailureEventExists(podname string, namespaces string, sysctl string) (bool, error) {
	events, err := client.Client.Events(namespaces).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, event := range events.Items {
		if strings.Index(event.Name, podname) == 0 && event.Reason == "FailedCreatePodSandBox" {
			if strings.Contains(event.Message, fmt.Sprintf("Sysctl %s is not allowed", sysctl)) {
				return true, nil
			}
		}
	}
	return false, nil
}
