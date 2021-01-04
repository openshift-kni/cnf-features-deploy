package pods

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/images"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/namespaces"
)

// DefaultDeletionTimeout contains the default pod deletion timeout in seconds
const DefaultDeletionTimeout = 120

// GetTestPod returns pod with the busybox image
func GetTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Labels: map[string]string{
				"test": "",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "test",
					Image:   images.Test(),
					Command: []string{"sleep", "10h"},
				},
			},
		},
	}
}

// WaitForDeletion waits until the pod will be removed from the cluster
func WaitForDeletion(pod *corev1.Pod, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pod := &corev1.Pod{}
		if err := testclient.Client.Get(context.TODO(), key, pod); errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// WaitForCondition waits until the pod will have specified condition type with the expected status
func WaitForCondition(pod *corev1.Pod, conditionType corev1.PodConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatedPod := &corev1.Pod{}
		if err := testclient.Client.Get(context.TODO(), key, updatedPod); err != nil {
			return false, nil
		}

		for _, c := range updatedPod.Status.Conditions {
			if c.Type == conditionType && c.Status == conditionStatus {
				return true, nil
			}
		}
		return false, nil
	})
}

// WaitForPhase waits until the pod will have specified phase
func WaitForPhase(pod *corev1.Pod, phase corev1.PodPhase, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatedPod := &corev1.Pod{}
		if err := testclient.Client.Get(context.TODO(), key, updatedPod); err != nil {
			return false, nil
		}

		if updatedPod.Status.Phase == phase {
			return true, nil
		}

		return false, nil
	})
}

// GetLogs returns logs of the specified pod
func GetLogs(c *kubernetes.Clientset, pod *corev1.Pod) (string, error) {
	logStream, err := c.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer logStream.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, logStream); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ExecCommandOnPod returns the output of the command execution on the pod
func ExecCommandOnPod(pod *corev1.Pod, command []string) ([]byte, error) {
	initialArgs := []string{
		"exec",
		"-i",
		"-n", pod.Namespace,
		pod.Name,
		"--",
	}
	initialArgs = append(initialArgs, command...)
	return testutils.ExecAndLogCommand("oc", initialArgs...)
}

// GetContainerID returns container ID under the pod by the container name
func GetContainerIDByName(pod *corev1.Pod, containerName string) (string, error) {
	updatedPod := &corev1.Pod{}
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	if err := testclient.Client.Get(context.TODO(), key, updatedPod); err != nil {
		return "", err
	}
	for _, containerStatus := range updatedPod.Status.ContainerStatuses {
		if containerStatus.Name == containerName {
			return strings.Trim(containerStatus.ContainerID, "cri-o://"), nil
		}
	}
	return "", fmt.Errorf("failed to find the container ID for the container %q under the pod %q", containerName, pod.Name)
}

// GetPerformanceOperatorPod returns the pod running the Performance Profile Operator
func GetPerformanceOperatorPod() (*corev1.Pod, error) {
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", "name", "performance-operator"))
	if err != nil {
		return nil, err
	}

	pods := &corev1.PodList{}

	opts := &client.ListOptions{LabelSelector: selector, Namespace: namespaces.PerformanceOperator}
	if err := testclient.Client.List(context.TODO(), pods, opts); err != nil {
		return nil, err
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("incorrect performance operator pods count: %d", len(pods.Items))
	}

	return &pods.Items[0], nil
}
