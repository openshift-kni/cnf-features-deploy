package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
	sriovnamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	sriovnetwork "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/network"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/images"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"

	"github.com/prometheus/common/model"
)

const testNamespace string = "test-sriov-metrics"

var sriovclient *sriovtestclient.ClientSet

func init() {
	sriovclient = sriovtestclient.New("")
}

var _ = Describe("[sriov] SR-IOV Network Metrics Exporter", Ordered, func() {

	var sriovCapableNodes *sriovcluster.EnabledNodes

	BeforeAll(func() {
		if discovery.Enabled() {
			Skip("Discovery mode not supported")
		}

		restoreFeatureGates := enableMetricsExporterFeatureGate()
		DeferCleanup(restoreFeatureGates)

		By("Adding monitoring label to " + namespaces.SRIOVOperator)
		err := sriovnamespaces.AddLabel(sriovclient, context.Background(), namespaces.SRIOVOperator, "openshift.io/cluster-monitoring", "true")
		Expect(err).ToNot(HaveOccurred())

		By("Clean SRIOV policies and networks")
		networks.CleanSriov(sriovclient)

		By("Discover SRIOV devices")
		sriovCapableNodes, err = sriovcluster.DiscoverSriov(sriovclient, namespaces.SRIOVOperator)
		Expect(err).ToNot(HaveOccurred())

		err = namespaces.Create(testNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())
		namespaces.CleanPods(testNamespace, client.Client)
	})

	It("should provide the same metrics as network-metrics-daemon", func() {
		testNode, testDevice, err := sriovCapableNodes.FindOneSriovNodeAndDevice()
		Expect(err).ToNot(HaveOccurred())
		By("Using device " + testDevice.Name + " on node " + testNode)

		sriovNetworkNodePolicy, err := sriovnetwork.CreateSriovPolicy(
			sriovclient, "test-metrics-", namespaces.SRIOVOperator,
			testDevice.Name, testNode, 5,
			"testsriovmetricsresource", "netdevice",
		)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(sriovclient.Delete, context.Background(), sriovNetworkNodePolicy)

		ipam := `{ "type": "host-local", "subnet": "192.0.2.0/24" }`
		err = sriovnetwork.CreateSriovNetwork(sriovclient, testDevice, "test-metrics-network",
			testNamespace, namespaces.SRIOVOperator, "testsriovmetricsresource", ipam)
		Expect(err).ToNot(HaveOccurred())

		serverPod, clientPod := makeClientAndServerNetcatPod()

		// Do not verify pairs
		// "container_network_receive_packets_total":  "sriov_vf_rx_packets",
		// "container_network_receive_bytes_total":    "sriov_vf_rx_bytes",
		// because there might be traffic on the wire that disturbs the counters.
		// An example is a DHCP traffic that other nodes are producing, e.g. (tcpdump):
		//
		// 13:28:00.442893 04:3f:72:fe:d1:d1 > ff:ff:ff:ff:ff:ff, ethertype IPv4 (0x0800), length 327: 0.0.0.0.68 > 255.255.255.255.67: BOOTP/DHCP, Request from 04:3f:72:fe:d1:d1, length 285
		metricsToMatch := map[string]string{
			"container_network_transmit_packets_total": "sriov_vf_tx_packets",
			"container_network_transmit_bytes_total":   "sriov_vf_tx_bytes",
		}
		containerQuery := `%s + on(namespace,pod,interface) group_left(network_name) (pod_network_name_info{interface="net1",pod="%s"})`
		sriovQuery := `%s * on (pciAddr) group_left(pod,namespace,dev_type) sriov_kubepoddevice{pod="%s"}`

		for containerMetricName, sriovMetricName := range metricsToMatch {
			By(fmt.Sprintf("verifying metrics %s == %s", containerMetricName, sriovMetricName))
			assertPromQLHasTheSameResult(
				fmt.Sprintf(containerQuery, containerMetricName, serverPod.Name),
				fmt.Sprintf(sriovQuery, sriovMetricName, serverPod.Name),
			)

			assertPromQLHasTheSameResult(
				fmt.Sprintf(containerQuery, containerMetricName, clientPod.Name),
				fmt.Sprintf(sriovQuery, sriovMetricName, clientPod.Name),
			)
		}
	})
})

func makeClientAndServerNetcatPod() (*corev1.Pod, *corev1.Pod) {
	serverPod := pods.DefinePod(testNamespace)
	serverPod.GenerateName = "testpod-nc-server-"
	serverPod = pods.RedefinePodWithNetwork(serverPod, `[{"name": "test-metrics-network","ips":["192.0.2.101/24"]}]`)
	serverPod.Spec.Containers = append(serverPod.Spec.Containers, corev1.Container{
		Name:            "netcat-tcp-server",
		Image:           images.For(images.TestUtils),
		Command:         []string{"nc", "-vv", "--keep-open", "--listen", "5000"},
		SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
	})
	serverPod, err := pods.CreateAndStart(serverPod)
	Expect(err).ToNot(HaveOccurred())

	clientPod := pods.DefinePod(testNamespace)
	clientPod.GenerateName = "testpod-nc-client-"
	clientPod = pods.RedefinePodWithNetwork(clientPod, `[{"name": "test-metrics-network","ips":["192.0.2.102/24"]}]`)
	clientPod.Spec.Containers = append(clientPod.Spec.Containers, corev1.Container{
		Name:            "netcat-tcp-client",
		Image:           images.For(images.TestUtils),
		Command:         makeNetcatClientCommand("192.0.2.101 5000"),
		SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
	})
	clientPod, err = pods.CreateAndStart(clientPod)
	Expect(err).ToNot(HaveOccurred())

	return clientPod, serverPod
}

func makeNetcatClientCommand(targetIpAddress string) []string {
	// This command send 1001 bytes via netcat
	script := fmt.Sprintf(
		`
	sleep 10; 
	printf %%01000d 1 | nc -w 1 %s;
	sleep inf
`, targetIpAddress)
	return []string{"bash", "-xec", script}
}

func runPromQLQuery(query string) (model.Vector, error) {
	prometheusPods, err := client.Client.Pods("").List(context.Background(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/component=prometheus",
	})
	if err != nil {
		return nil, fmt.Errorf("can't find a Prometheus pod: %w", err)
	}

	if len(prometheusPods.Items) == 0 {
		return nil, fmt.Errorf("no instance of Prometheus found")
	}

	prometheusPod := prometheusPods.Items[0]

	url := fmt.Sprintf("localhost:9090/api/v1/query?%s", (url.Values{"query": []string{query}}).Encode())
	command := []string{"curl", url}
	outputBuffer, err := pods.ExecCommand(client.Client, prometheusPod, command)
	if err != nil {
		return nil, fmt.Errorf("promQL query : [%s/%s] command: [%v]\nout: %s\n%w",
			prometheusPod.Namespace, prometheusPod.Name, command, outputBuffer.String(), err)
	}

	result := struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string       `json:"resultType"`
			Result     model.Vector `json:"result"`
		} `json:"data"`
	}{}

	err = json.Unmarshal(outputBuffer.Bytes(), &result)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal PromQL result: query[%s] response[%s] error: %w", query, outputBuffer.String(), err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("PromQL statement failed: query[%s] result[%v]", query, result)
	}

	return result.Data.Result, nil
}

func enableMetricsExporterFeatureGate() func() {
	operatorConfig, err := sriovclient.SriovOperatorConfigs(namespaces.SRIOVOperator).Get(context.Background(), "default", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	// Save the current feature gates map to allowing restore
	oldFeatureGates := make(map[string]bool)
	for k, v := range operatorConfig.Spec.FeatureGates {
		oldFeatureGates[k] = v
	}

	if operatorConfig.Spec.FeatureGates == nil {
		operatorConfig.Spec.FeatureGates = make(map[string]bool)
	}

	if operatorConfig.Spec.FeatureGates["metricsExporter"] {
		// The feature is already enabled: nothing to do
		return func() {}
	}

	By("Enabling metricsExporter feature gate")
	operatorConfig.Spec.FeatureGates["metricsExporter"] = true

	_, err = sriovclient.SriovOperatorConfigs(namespaces.SRIOVOperator).Update(context.Background(), operatorConfig, metav1.UpdateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return func() {
		By("Resetting feature gate to its previous value")
		operatorConfig, err := sriovclient.SriovOperatorConfigs(namespaces.SRIOVOperator).Get(context.Background(), "default", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		operatorConfig.Spec.FeatureGates = oldFeatureGates
		_, err = sriovclient.SriovOperatorConfigs(namespaces.SRIOVOperator).Update(context.Background(), operatorConfig, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
}

// assertPromQLHasTheSameResult evaluates both PromQL queries and checks if both return the same value.
func assertPromQLHasTheSameResult(queryA, queryB string) {
	failedValues := "time A - B\n	"

	Eventually(func(g Gomega) {
		samplesA, errA := runPromQLQuery(queryA)
		samplesB, errB := runPromQLQuery(queryB)

		failedValues += fmt.Sprintf("%s %v - %v\n", time.Now().Format(time.StampMilli), samplesA, samplesB)

		g.Expect(errA).ToNot(HaveOccurred())
		g.Expect(samplesA).To(HaveLen(1), "queryA[%s]", queryA)
		valueA := float64(samplesA[0].Value)

		g.Expect(errB).ToNot(HaveOccurred())
		g.Expect(samplesB).To(HaveLen(1), "queryB[%s]", queryB)
		valueB := float64(samplesB[0].Value)

		g.Expect(valueA).To(
			Equal(valueB),
			"queries returned different values:\nqueryA[%s]=%f\nqueryB[%s]=%f",
			queryA, valueA, queryB, valueB,
		)
	}).
		WithPolling(1*time.Second).
		WithTimeout(2*time.Minute).
		WithOffset(1).
		Should(Succeed(), func() string {
			return fmt.Sprintf(`queries didn't return congruent values
			queryA = [%s]
			queryB = [%s],
			recent values
			%s`, queryA, queryB, failedValues)
		})
}
