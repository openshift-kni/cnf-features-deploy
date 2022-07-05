package security

import (
	"context"
	"fmt"
	//v1 "k8s.io/client-go/applyconfigurations/apps/v1"
	"strings"
	"time"

	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	kappsv1 "k8s.io/api/apps/v1"
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
		It("pods with sysctls over bond should be able to ping each other",
			func() {
				macvlanNadName := "macvlan-nad"
				ip1 := "10.10.0.22"
				bondNadName1 := "bond-nad1"
				bondNadName2 := "bond-nad2"
				bondInterfaceName := "bond0"

				ds, err := createDS(client.Client, "cp-bond-cni", "default", "quay.io/schseba/bond-cni:latest",
					[]string{"/bin/bash", "-c", "cp /bond/bond /host/opt/cni/bin/; chmod 777 /host/opt/cni/bin/bond; sleep INF"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ds).ToNot(BeNil())
				time.Sleep(15 * time.Second)
				By("DS CRETATED")

				sysctls, err := networks.SysctlConfig(map[string]string{fmt.Sprintf(Sysctl, "IFNAME"): "1"})
				Expect(err).ToNot(HaveOccurred())

				bondWithTuningNad1, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, bondNadName1).WithBond(bondInterfaceName, "net1", "net2").WithStaticIpam(ip1).WithTuning(sysctls).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), bondWithTuningNad1)
				Expect(err).ToNot(HaveOccurred())

				bondWithTuningNad2, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, bondNadName2).WithBond(bondInterfaceName, "net1", "net2").WithStaticIpam("10.10.0.23").WithTuning(sysctls).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), bondWithTuningNad2)
				Expect(err).ToNot(HaveOccurred())

				macVlandNad, err := networks.NewNetworkAttachmentDefinitionBuilder(TestNamespace, macvlanNadName).WithMacVlan().Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), macVlandNad)
				Expect(err).ToNot(HaveOccurred())

				podDefinition1 := pods.DefineWithNetworks(TestNamespace, []string{fmt.Sprintf("%s/%s, %s/%s, %s/%s@%s", TestNamespace, macvlanNadName, TestNamespace, macvlanNadName, TestNamespace, bondNadName1, bondInterfaceName)})
				podDefinition1.Labels = map[string]string{"test": "1"}
				pod1, err := client.Client.Pods(TestNamespace).Create(context.Background(), podDefinition1, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(30 * time.Second)

				poddbug, err := client.Client.Pods(TestNamespace).Get(context.Background(), pod1.Name, metav1.GetOptions{})
				By(fmt.Sprintf("\n\n  ===MMM===   STATUS:  %+v \n", poddbug.Status))

				events, _ := client.Client.Events(TestNamespace).List(context.TODO(), metav1.ListOptions{TypeMeta: metav1.TypeMeta{Kind: "Pod"}, FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod1.Name)})
				By(fmt.Sprintf("\n\n  ===MMM===    events:  %+v \n\n\n", events))
				for _, e := range events.Items {
					By(fmt.Sprintf("\n ===MMM===  --------------  EVENT:  %+v \n", e))
				}

				//////////////
				copyPods, err := client.Client.Pods("default").List(context.Background(), metav1.ListOptions{LabelSelector: "cp-bond-cni=true"})
				By(fmt.Sprintf("\n\n  ===MMM===   COPY PODS len:  %+v \n", len(copyPods.Items)))

				for _, cpPod := range copyPods.Items {
					By(fmt.Sprintf("\n\n  ===MMM===   POD name:  %+v \n", cpPod.Name))
					cmd := []string{"cat", "/host/tmp/bond_log"}
					stats, err := pods.ExecCommand(client.Client, cpPod, cmd)
					By(fmt.Sprintf("--------------------------------------------------------------------------------------\n"))
					By(fmt.Sprintf("\n\n  ===MMM===   POD OUTPUT:  %+v \n", stats.String()))
					By(fmt.Sprintf("--------------------------------------------------------------------------------------\n"))

					By(fmt.Sprintf("\n\n  ===MMM===   POD ERROR:  %+v \n", err))
				}
				time.Sleep(180 * time.Second)

				err = pods.WaitForPhase(client.Client, pod1, corev1.PodRunning, 1*time.Minute)
				if err != nil {
					fmt.Printf("SLEEPING as bug got reproduced")
					time.Sleep(9 * time.Hour)
				}
				Expect(err).ToNot(HaveOccurred())

				podDefinition2 := pods.DefineWithNetworks(TestNamespace, []string{fmt.Sprintf("%s/%s, %s/%s, %s/%s@%s", TestNamespace, macvlanNadName, TestNamespace, macvlanNadName, TestNamespace, bondNadName2, bondInterfaceName)})
				podDefinition2 = pods.RedefineWithCommand(podDefinition2, []string{"/bin/bash", "-c", fmt.Sprintf("ping -c 1 %s", ip1)}, nil)
				podDefinition2 = pods.RedefineWithRestartPolicy(podDefinition2, corev1.RestartPolicyNever)
				podDefinition2.Spec.NodeName = pod1.Spec.NodeName
				pod2, err := client.Client.Pods(TestNamespace).Create(context.Background(), podDefinition2, metav1.CreateOptions{})

				Expect(err).ToNot(HaveOccurred())
				err = pods.WaitForPhase(client.Client, pod2, corev1.PodSucceeded, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())

				err = client.Client.DaemonSets("default").Delete(context.TODO(), "cp-bond-cni", metav1.DeleteOptions{})
				By(fmt.Sprintf("\n\n  ===MMM===   DELETE DS ERROR:  %+v \n", err))

			})
	})
})

func createDS(client *client.ClientSet, name, namespace, image string, command []string) (*kappsv1.DaemonSet, error) {
	privileged := true
	daemonset, err := client.DaemonSets(namespace).Create(context.TODO(), &kappsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kappsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{name: "true"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{name: "true"},
					Annotations: map[string]string{
						fmt.Sprintf("alpha.image.policy.openshift.io/%s", name): "*",
					},
				},
				Spec: corev1.PodSpec{

					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           image,
							Command:         command,
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cnibin",
									MountPath: "/host/opt/cni/bin",
									ReadOnly:  false,
								},
								{
									Name:      "tmp",
									MountPath: "/host/tmp",
									ReadOnly:  false,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cnibin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/cni/bin"},
							},
						},
						{
							Name: "tmp",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{Path: "/tmp"},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	return daemonset, err

}
