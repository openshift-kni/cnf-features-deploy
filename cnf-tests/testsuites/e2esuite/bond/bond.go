package bond

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nadtypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[bondcni]", func() {
	apiclient := client.New("")

	execute.BeforeAll(func() {
		err := namespaces.Create(namespaces.BondTestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		namespaces.CleanPods(namespaces.BondTestNamespace, apiclient)
	})

	Context("bond over macvlan", func() {
		It("should be able to create pod with bond interface over macvlan interfaces",
			func() {
				macvlanNadName := "macvlan-nad"
				bondNadName := "bond-nad"
				bondIfcName := "bondifc"
				bondIP := "1.1.1.17"

				macVlanNad, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.BondTestNamespace, macvlanNadName).WithMacVlan().Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), macVlanNad)
				Expect(err).ToNot(HaveOccurred())

				bondNad, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.BondTestNamespace, bondNadName).WithBond(bondIfcName, "net2", "net1", 1300).WithStaticIpam(bondIP).Build()
				Expect(err).ToNot(HaveOccurred())
				err = client.Client.Create(context.Background(), bondNad)
				Expect(err).ToNot(HaveOccurred())

				podDefinition := pods.DefineWithNetworks(namespaces.BondTestNamespace, []string{fmt.Sprintf("%s/%s, %s/%s, %s/%s@%s", namespaces.BondTestNamespace, macvlanNadName, namespaces.BondTestNamespace, macvlanNadName, namespaces.BondTestNamespace, bondNadName, bondIfcName)})
				pod, err := client.Client.Pods(namespaces.BondTestNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = pods.WaitForPhase(client.Client, pod, corev1.PodRunning, 1*time.Minute)
				Expect(err).ToNot(HaveOccurred())

				pod, err = client.Client.Pods(namespaces.BondTestNamespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				networkStatusString, ok := pod.Annotations["k8s.v1.cni.cncf.io/network-status"]
				Expect(ok).To(BeTrue())
				Expect(networkStatusString).ToNot(BeNil())

				networkStatuses := []nadtypes.NetworkStatus{}
				err = json.Unmarshal([]byte(networkStatusString), &networkStatuses)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(networkStatuses)).To(Equal(4))
				Expect(networkStatuses[3].Interface).To(Equal(bondIfcName))
				Expect(networkStatuses[3].Name).To(Equal(fmt.Sprintf("%s/%s", namespaces.BondTestNamespace, bondNadName)))

				// TODO: This will not work due to BZ 2082360. Uncomment once fixed and changes are propagated.
				// Expect(len(networkStatuses[3].Ips)).To(Equal(1))
				// Expect(len(networkStatuses[3].Ips[0])).To(Equal(bondIP))

				stdout, err := pods.ExecCommand(client.Client, *pod, []string{"ip", "addr", "show", bondIfcName})
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Index(stdout.String(), "inet "+bondIP))
			})
	})
})
