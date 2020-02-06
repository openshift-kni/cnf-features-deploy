package pods

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
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
	req := client.Client.CoreV1Interface.RESTClient().
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
	if err != nil {
		return buf, err
	}

	return buf, nil
}

// DetectDefaultRouteInterface returns pod's interface bounded to the default route which contains
// Destination=00000000 and Metric=00000000
// Typical route table look like present below:
//Iface	  Destination(1) Gateway 	Flags	RefCnt	Use	Metric	Mask(7)		MTU	Window	IRTT
//enp2s0	00000000	016FA8C0	0003	0		0	101		00000000	0	0	0  < This is default route
//tun0		0000800A	00000000	0001	0		0	0		0000FCFF	0	0	0
//enp2s0	006FA8C0	00000000	0001	0		0	101		00FFFFFF	0	0	0
//enp6s0	00C7A8C0	00000000	0001	0		0	102		00FFFFFF	0	0	0
func DetectDefaultRouteInterface(cs *testclient.ClientSet, pod corev1.Pod) (string, error) {
	collectRoutesCommand := []string{"cat", "/proc/net/route"}
	routeTableBuf, err := ExecCommand(cs, pod, collectRoutesCommand)
	if err != nil {
		return "", err
	}
	routeTable := routeTableBuf.String()
	for _, route := range strings.Split(routeTable, "\n")[1 : len(strings.Split(routeTable, "\n"))-1] {
		if strings.Split(strings.Join(strings.Fields(route), " "), " ")[1] == "00000000" &&
			strings.Split(strings.Join(strings.Fields(route), " "), " ")[7] == "00000000" {
			return strings.Split(strings.Join(strings.Fields(route), " "), " ")[0], nil
		}
	}
	return "", fmt.Errorf("Default route not present")
}
