package networks

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	. "github.com/onsi/gomega"

	sriovk8sv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	sriovnetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/sriov"
)

func CleanSriov(sriovclient *sriovtestclient.ClientSet, namespace string) {
	// This clean only the policy and networks with the prefix of test
	err := sriovnamespaces.CleanPods(namespace, sriovclient)
	Expect(err).ToNot(HaveOccurred())
	err = sriovnamespaces.CleanNetworks(namespaces.SRIOVOperator, sriovclient)
	Expect(err).ToNot(HaveOccurred())

	if !discovery.Enabled() {
		err = sriovnamespaces.CleanPolicies(namespaces.SRIOVOperator, sriovclient)
		Expect(err).ToNot(HaveOccurred())
	}
	sriov.WaitStable(sriovclient)
}

func CreateSriovPolicyAndNetwork(sriovclient *sriovtestclient.ClientSet, namespace, networkName, resourceName, metaPluginsConfig string) {
	numVfs := 4
	sriovInfos, err := sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
	Expect(err).ToNot(HaveOccurred())
	Expect(sriovInfos).ToNot(BeNil())

	nodes, err := nodes.MatchingOptionalSelectorByName(sriovInfos.Nodes)
	Expect(err).ToNot(HaveOccurred())
	Expect(len(nodes)).To(BeNumerically(">", 0))

	node := nodes[0]
	sriovDevice, err := sriovInfos.FindOneSriovDevice(node)
	Expect(err).ToNot(HaveOccurred())

	sriovnetwork.CreateSriovPolicy(sriovclient, "test-policy", namespaces.SRIOVOperator, sriovDevice.Name, node, numVfs, resourceName, "netdevice")
	sriov.WaitStable(sriovclient)

	Eventually(func() int64 {
		testedNode, err := sriovclient.Nodes().Get(context.Background(), node, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		resNum, _ := testedNode.Status.Allocatable[corev1.ResourceName("openshift.io/"+resourceName)]
		capacity, _ := resNum.AsInt64()
		return capacity
	}, 10*time.Minute, time.Second).Should(Equal(int64(numVfs)))

	CreateSriovNetwork(sriovclient, sriovDevice, networkName, namespace, namespaces.SRIOVOperator, resourceName, metaPluginsConfig)
}

func CreateSriovNetwork(sriovclient *sriovtestclient.ClientSet, sriovDevice *sriovv1.InterfaceExt, sriovNetworkName, sriovNetworkNamespace, operatorNamespace, resourceName, metaPluginsConfig string) {
	ipam := `{"type": "host-local","ranges": [[{"subnet": "1.1.1.0/24"}]],"dataDir": "/run/my-orchestrator/container-ipam-state"}`
	err := sriovnetwork.CreateSriovNetwork(sriovclient, sriovDevice, sriovNetworkName, sriovNetworkNamespace, operatorNamespace, resourceName, ipam, func(network *sriovv1.SriovNetwork) {
		if metaPluginsConfig != "" {
			network.Spec.MetaPluginsConfig = metaPluginsConfig
		}
	})
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		netAttDef := &sriovk8sv1.NetworkAttachmentDefinition{}
		return sriovclient.Get(context.Background(), goclient.ObjectKey{Name: sriovNetworkName, Namespace: sriovNetworkNamespace}, netAttDef)
	}, time.Minute, 5*time.Second).ShouldNot(HaveOccurred())
}
