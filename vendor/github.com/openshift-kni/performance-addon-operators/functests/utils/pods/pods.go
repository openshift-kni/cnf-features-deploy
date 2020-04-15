package pods

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetBusybox returns pod with the busybox image
func GetBusybox() *corev1.Pod {
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
					Image:   "busybox",
					Command: []string{"sleep", "10h"},
				},
			},
		},
	}
}

// WaitForDeletion waits until the pod will be removed from the cluster
func WaitForDeletion(c client.Client, pod *corev1.Pod, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pod := &corev1.Pod{}
		if err := c.Get(context.TODO(), key, pod); errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// WaitForCondition waits until the pod will have specified condition type with the expected status
func WaitForCondition(c client.Client, pod *corev1.Pod, conditionType corev1.PodConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatedPod := &corev1.Pod{}
		if err := c.Get(context.TODO(), key, updatedPod); err != nil {
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
func WaitForPhase(c client.Client, pod *corev1.Pod, phase corev1.PodPhase, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatedPod := &corev1.Pod{}
		if err := c.Get(context.TODO(), key, updatedPod); err != nil {
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
	logStream, err := c.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream()
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
func ExecCommandOnPod(c client.Client, pod *corev1.Pod, command []string) ([]byte, error) {
	initialArgs := []string{
		"exec",
		"-i",
		"-n", pod.Namespace,
		pod.Name,
		"--",
	}
	initialArgs = append(initialArgs, command...)
	return exec.Command("oc", initialArgs...).CombinedOutput()
}
