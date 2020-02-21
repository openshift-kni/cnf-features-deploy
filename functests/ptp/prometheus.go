package ptp

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	client "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const openshiftPtpNamespace = "openshift-ptp"
const openshiftPtpMetricPrefix = "openshift_ptp_"
const openshiftMonitoringNamespace = "openshift-monitoring"

// Needed to deserialize prometheus query output.
// Sample output (omiting irrelevant fields):
// {"data" : {
//    "result" : [{
//      "metric" : {
//        "pod" : "mm-pod1",
// }}]}}
type queryOutput struct {
	Data data
}

type data struct {
	Result []result
}

type result struct {
	Metric metric
}

type metric struct {
	Pod string
}

var _ = Describe("prometheus", func() {
	Context("Metrics reported by PTP pods", func() {
		It("Should all be reported by prometheus", func() {
			ptpPods, err := client.Client.Pods(openshiftPtpNamespace).List(metav1.ListOptions{
				LabelSelector: "app=linuxptp-daemon",
			})
			Expect(err).ToNot(HaveOccurred())
			ptpMonitoredEntriesByPod, uniqueMetricKeys := collectPtpMetrics(ptpPods.Items)
			podsPerPrometheusMetricKey := collectPrometheusMetrics(uniqueMetricKeys)
			containSameMetrics(ptpMonitoredEntriesByPod, podsPerPrometheusMetricKey)
		})
	})
})

func collectPrometheusMetrics(uniqueMetricKeys []string) map[string][]string {
	prometheusPods, err := client.Client.Pods(openshiftMonitoringNamespace).List(metav1.ListOptions{
		LabelSelector: "app=prometheus",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(len(prometheusPods.Items)).NotTo(BeZero())

	podsPerPrometheusMetricKey := map[string][]string{}
	for _, metricsKey := range uniqueMetricKeys {
		podsPerKey := []string{}
		command := []string{
			"curl",
			"localhost:9090/api/v1/query?query=" + metricsKey,
		}
		stdout, err := pods.ExecCommand(client.Client, prometheusPods.Items[0], command)
		Expect(err).ToNot(HaveOccurred())
		var queryOutput queryOutput
		err = json.Unmarshal([]byte(stdout.String()), &queryOutput)
		Expect(err).ToNot(HaveOccurred())
		for _, result := range queryOutput.Data.Result {
			podsPerKey = append(podsPerKey, result.Metric.Pod)
		}
		podsPerPrometheusMetricKey[metricsKey] = podsPerKey
	}
	return podsPerPrometheusMetricKey
}

func collectPtpMetrics(ptpPods []k8sv1.Pod) (map[string][]string, []string) {
	uniqueMetricKeys := []string{}
	ptpMonitoredEntriesByPod := map[string][]string{}
	for _, pod := range ptpPods {
		podEntries := []string{}
		stdout, err := pods.ExecCommand(client.Client, pod, []string{"curl", "localhost:9091/metrics"})
		Expect(err).ToNot(HaveOccurred())
		for _, line := range strings.Split(stdout.String(), "\n") {
			if strings.HasPrefix(line, openshiftPtpMetricPrefix) {
				metricsKey := line[0:strings.Index(line, "{")]
				podEntries = append(podEntries, metricsKey)
				uniqueMetricKeys = appendIfMissing(uniqueMetricKeys, metricsKey)
			}
		}
		ptpMonitoredEntriesByPod[pod.Name] = podEntries
	}
	return ptpMonitoredEntriesByPod, uniqueMetricKeys
}

func containSameMetrics(ptpMetricsByPod map[string][]string, prometheusMetrics map[string][]string) {
	for podName, monitoringKeys := range ptpMetricsByPod {
		for _, key := range monitoringKeys {
			if podsWithMetric, ok := prometheusMetrics[key]; ok {
				// We only check if the element is present, but do not compare the values
				// New values are reported periodically, and there is a risk of discrepancies
				// in the values read from ptp pods and the ones read from prometheus
				if hasElement(podsWithMetric, podName) {
					continue
				}
			}
			Fail("Metric " + podName + " on pod " + podName + "was not reported.")
		}
	}
}

func hasElement(slice []string, item string) bool {
	for _, sliceItem := range slice {
		if item == sliceItem {
			return true
		}
	}
	return false
}

func appendIfMissing(slice []string, newItem string) []string {
	if hasElement(slice, newItem) {
		return slice
	}
	return append(slice, newItem)
}
