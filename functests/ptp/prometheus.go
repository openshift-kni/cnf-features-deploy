package ptp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
		var stdout bytes.Buffer
		var err error
		Eventually(func() error {
			stdout, err = pods.ExecCommand(client.Client, pod, []string{"curl", "localhost:9091/metrics"})
			if len(strings.Split(stdout.String(), "\n")) == 0 {
				return fmt.Errorf("empty response")
			}

			return err
		}, 2*time.Minute, 2*time.Second).Should(Not(HaveOccurred()))

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

func containSameMetrics(ptpMetricsByPod map[string][]string, prometheusMetrics map[string][]string) error {
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
			return fmt.Errorf("Metric %s on pod %s was not reported", podName, podName)
		}
	}
	return nil
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
