package test

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"github.com/openshift/ptp-operator/test/pkg/client"
	"github.com/openshift/ptp-operator/test/pkg/metrics"
	"github.com/openshift/ptp-operator/test/pkg/pods"

	k8sv1 "k8s.io/api/core/v1"
)

const openshiftPtpNamespace = "openshift-ptp"
const openshiftPtpMetricPrefix = "openshift_ptp_"

// Needed to deserialize prometheus query output.
// Sample output (omiting irrelevant fields):
//
//	{"data" : {
//	   "result" : [{
//	     "metric" : {
//	       "pod" : "mm-pod1",
//	}}]}}
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
	prometheusPod, err := metrics.GetPrometheusPod()
	Expect(err).ToNot(HaveOccurred(), "failed to get prometheus pod")

	podsPerPrometheusMetricKey := map[string][]string{}
	for _, metricsKey := range uniqueMetricKeys {
		promResult := []result{}
		promResponse := metrics.PrometheusQueryResponse{}
		promResponse.Data.Result = &promResult

		err := metrics.RunPrometheusQuery(prometheusPod, metricsKey, &promResponse)
		Expect(err).ToNot(HaveOccurred(), "failed to run prometheus query")

		podsPerKey := []string{}
		for _, result := range promResult {
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
			stdout, _, err = pods.ExecCommand(client.Client, &pod, pod.Spec.Containers[0].Name, []string{"curl", "localhost:9091/metrics"})
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
			return fmt.Errorf("metric %s on pod %s was not reported", key, podName)
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
