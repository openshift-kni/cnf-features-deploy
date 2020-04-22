package pods

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	testclient "github.com/openshift/ptp-operator/test/utils/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/pointer"
)

const hugePageCommand = "yum install -y libhugetlbfs python3 && echo -e \"import time\nwhile True:\n\tprint('ABCD'*%s)\n\ttime.sleep(0.5)\" > printer.py && cat printer.py && LD_PRELOAD=libhugetlbfs.so HUGETLB_VERBOSE=10 HUGETLB_MORECORE=yes HUGETLB_FORCE_ELFMAP=yes python3 printer.py > /dev/null"

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

func getDefinition(namespace string) *corev1.Pod {
	podObject := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testpod-",
			Namespace:    namespace},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: pointer.Int64Ptr(0),
			Containers: []corev1.Container{{Name: "test",
				Image:   "quay.io/schseba/utility-container:latest",
				Command: []string{"/bin/bash", "-c", "sleep INF"}}}}}

	return podObject
}

// RedefineWithCommand updates the pod defintion with a different command
func RedefineWithCommand(pod *corev1.Pod, command []string, args []string) *corev1.Pod {
	pod.Spec.Containers[0].Command = command
	pod.Spec.Containers[0].Args = args
	return pod
}

// RedefineWithRestartPolicy updates the pod defintion with a restart policy
func RedefineWithRestartPolicy(pod *corev1.Pod, restartPolicy corev1.RestartPolicy) *corev1.Pod {
	pod.Spec.RestartPolicy = restartPolicy
	return pod
}

// DefineWithHugePages creates a pod with a 4Gi of hugepages and run command to write data to that memory
func DefineWithHugePages(namespace, nodeName, writeSize string) *corev1.Pod {
	pod := RedefineWithRestartPolicy(
		RedefineWithCommand(
			getDefinition(namespace),
			[]string{"/bin/bash", "-c", fmt.Sprintf(hugePageCommand, writeSize)}, []string{},
		),
		corev1.RestartPolicyNever,
	)

	pod.Spec.NodeSelector = map[string]string{
		"kubernetes.io/hostname": nodeName,
	}

	// Resource request
	pod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{}
	pod.Spec.Containers[0].Resources.Requests["memory"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Requests["hugepages-1Gi"] = resource.MustParse("4Gi")
	pod.Spec.Containers[0].Resources.Requests["cpu"] = *resource.NewQuantity(int64(4), resource.DecimalSI)

	// Resource limit
	pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{}
	pod.Spec.Containers[0].Resources.Limits["memory"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Limits["hugepages-1Gi"] = resource.MustParse("4Gi")
	pod.Spec.Containers[0].Resources.Limits["cpu"] = *resource.NewQuantity(int64(4), resource.DecimalSI)

	// Hugepages volume mount
	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "hugepages", MountPath: "/dev/hugepages "}}

	// Security context capabilities
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsUser: pointer.Int64Ptr(0),
		Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"IPC_LOCK"}}}

	// Hugepages volume
	pod.Spec.Volumes = []corev1.Volume{{Name: "hugepages",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumHugePages}}}}

	return pod
}

// WaitForDeletion waits until the pod will be removed from the cluster
func WaitForDeletion(cs *testclient.ClientSet, pod *corev1.Pod, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := cs.Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// WaitForCondition waits until the pod will have specified condition type with the expected status
func WaitForCondition(cs *testclient.ClientSet, pod *corev1.Pod, conditionType corev1.PodConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatePod, err := cs.Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		for _, c := range updatePod.Status.Conditions {
			if c.Type == conditionType && c.Status == conditionStatus {
				return true, nil
			}
		}
		return false, nil
	})
}

// GetLog connects to a pod and fetches log
func GetLog(p *corev1.Pod) (string, error) {
	req := testclient.Client.Pods(p.Namespace).GetLogs(p.Name, &corev1.PodLogOptions{})
	log, err := req.Stream()
	if err != nil {
		return "", err
	}
	defer log.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, log)

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ExecCommand runs command in the pod and returns buffer output
func ExecCommand(cs *testclient.ClientSet, pod corev1.Pod, command []string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	req := testclient.Client.CoreV1Interface.RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cs.Config, "POST", req.URL())
	if err != nil {
		return buf, err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &buf,
		Stderr: os.Stderr,
		Tty:    true,
	})

	return buf, err
}
