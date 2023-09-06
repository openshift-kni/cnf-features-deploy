package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/openshift/ptp-operator/test/pkg/client"
	"github.com/openshift/ptp-operator/test/pkg/pods"
	prometheusModel "github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	openshiftMonitoringNamespace = "openshift-monitoring"
	prometheusResponseSuccess    = "success"
	kubeletServiceMonitor        = "kubelet"

	PrometheusQueryRetries       = 10
	PrometheusQueryRetryInterval = 1 * time.Second
)

type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   struct {
		ResultType string      `json:"resultType"`
		Result     interface{} `json:"result"`
	} `json:"data"`
}

type PrometheusVectorResult []struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

// GetPrometheusResultFloatValue returns the float value and timestamp from a prometheus value vector.
// The value is a vector with [ float, string ], where the timestamp is the float datum (in seconds) and
// the strings hold the actual, like in: [ 1674137739.625, "0.0015089738179757707" ]
func GetPrometheusResultFloatValue(promValue []interface{}) (value float64, tsMillis int64, err error) {
	const (
		resultTimeStampPos = 0
		resultValuePos     = 1
	)

	if len(promValue) != 2 {
		return 0, 0, fmt.Errorf("invalid value slice lenght - value %+v", promValue)
	}

	floatValue, err := strconv.ParseFloat(reflect.ValueOf(promValue[resultValuePos]).String(), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert prometheus value (%+v) to float: %w", promValue[resultValuePos], err)
	}

	tsMillis = int64(reflect.ValueOf(promValue[resultTimeStampPos]).Float() * 1000)

	return floatValue, tsMillis, nil
}

// GetPrometheusPod is a helper function that returns the first pod of the prometheus statefulset.
func GetPrometheusPod() (*corev1.Pod, error) {
	prometheusPods, err := client.Client.Pods(openshiftMonitoringNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=prometheus",
	})
	if err != nil {
		return nil, err
	}

	if len(prometheusPods.Items) == 0 {
		return nil, errors.New("no prometheus pod found")
	}

	// Return the first pod from the statefulset.
	return &prometheusPods.Items[0], nil
}

// RunPrometheusQuery runs a prometheus query in a prometheus pod and unmarshals the response in a given struct.
// Fails if the curl command failed, the prometheus response is not a "success" or it cannot be unmarshaled.
func RunPrometheusQuery(prometheusPod *corev1.Pod, query string, response *PrometheusQueryResponse) error {
	logrus.Tracef("Prom Query: %s", query)

	command := []string{
		"bash",
		"-c",
		"curl -s http://localhost:9090/api/v1/query --data-urlencode " + fmt.Sprintf("'query=%s'", query),
	}

	stdout, _, err := pods.ExecCommand(client.Client, prometheusPod, prometheusPod.Spec.Containers[0].Name, command)
	if err != nil {
		return fmt.Errorf("failed to exec command (%v) on pod %s (ns %s): %w",
			command, prometheusPod.Name, prometheusPod.Namespace, err)
	}

	outStr := stdout.String()
	logrus.Tracef("Prom Response: %v", outStr)
	err = json.Unmarshal([]byte(outStr), &response)
	if err != nil {
		return fmt.Errorf("failed to unmarshall prometheus response:\n%s\n%v", outStr, err)
	}

	if response.Status != prometheusResponseSuccess {
		return fmt.Errorf("response failed (status: %s, error: %s)", response.Status, response.Error)
	}

	return nil
}

// RunPrometheusQueryWithRetries runs RunPrometheusQuery but retries in case of failure, waiting
// retryInterval perdiod before the next attempt.
func RunPrometheusQueryWithRetries(prometheusPod *corev1.Pod, query string, retries int, retryInterval time.Duration, response *PrometheusQueryResponse,
	checkerFunc func(response *PrometheusQueryResponse) bool) error {
	for i := 0; i <= retries; i++ {
		logrus.Debugf("Querying prometheus, query %s, attempt %d", query, i)
		// In case it's not the first try, sleep before trying again.
		if i != 0 {
			time.Sleep(retryInterval)
		}

		err := RunPrometheusQuery(prometheusPod, query, response)
		if err == nil {
			// If we don't need to use the callback the check the response, or
			// the callback approves the response, we can return without error.
			if checkerFunc == nil || checkerFunc(response) {
				// Valid response, return here.
				return nil
			}
		}

		logrus.Warnf("Failed to get a prometheus response for query %s: %v", query, err)
	}

	return fmt.Errorf("failed to get a (valid) response from prometheus")
}

// GetCadvisorScrapeInterval returns the current scrape internval (in secs) of cadvisor target
// configured in kubelet's prometheus ServiceMonitor.
func GetCadvisorScrapeInterval() (int, error) {
	const (
		cadvisorEndpointPath = "/metrics/cadvisor"
	)

	// Get ServiceMonitor for the kubelet.
	monitor, err := client.Client.ServiceMonitors(openshiftMonitoringNamespace).Get(context.TODO(), kubeletServiceMonitor, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get kubelet client monitor %v", err)
	}

	// Search for the cadvisor endpoint config.
	for i := range monitor.Spec.Endpoints {
		endPoint := &monitor.Spec.Endpoints[i]
		if endPoint.Path != cadvisorEndpointPath {
			continue
		}

		// The interval is a prometheus' string-based type, with its own parsing func.
		// Fortunately, prometheus.Duration is just a time.Duration.
		duration, err := prometheusModel.ParseDuration(string(endPoint.Interval))
		if err != nil {
			return 0, fmt.Errorf("failed to parse interval %v: %v", endPoint.Interval, err)
		}

		// Cast it to standard time.Duration and return in secs.
		return int(time.Duration(duration).Seconds()), nil
	}

	return 0, errors.New("cadvisor endpoint not found")
}
