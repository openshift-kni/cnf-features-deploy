package pods

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/test-network-function/l2discovery-lib/pkg/l2client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// GetLog connects to a pod and fetches log
func GetLog(p *corev1.Pod, containerName string) (string, error) {
	req := l2client.Client.K8sClient.CoreV1().Pods(p.Namespace).GetLogs(p.Name, &corev1.PodLogOptions{Container: containerName})
	log, err := req.Stream(context.Background())
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
func ExecCommand(pod *corev1.Pod, containerName string, command []string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	req := l2client.Client.K8sClient.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(l2client.Client.Rest, "POST", req.URL())
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

// returns true if the pod passed as paremeter is running on the node selected by the label passed as a parameter.
// the label represent a ptp conformance test role such as: grandmaster, clock under test, slave1, slave2
func PodRole(runningPod *corev1.Pod, label string) (bool, error) {
	nodeList, err := l2client.Client.K8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return false, fmt.Errorf("error getting node list")
	}
	for NodeNumber := range nodeList.Items {
		if runningPod.Spec.NodeName == nodeList.Items[NodeNumber].Name {
			return true, nil
		}
	}
	return false, nil
}

// returns true if a pod has a given label or node name
func HasPodLabelOrNodeName(pod *corev1.Pod, label, nodeName *string) (result bool, err error) {
	if label == nil && nodeName == nil {
		return result, fmt.Errorf("label and nodeName are nil")
	}
	if label != nil && nodeName != nil {
		return result, fmt.Errorf("label or nodeName must be nil")
	}
	if label != nil {
		result, err = PodRole(pod, *label)
		if err != nil {
			return result, fmt.Errorf("could not check %s pod role, err: %s", *label, err)
		}
	}
	if nodeName != nil {
		result = pod.Spec.NodeName == *nodeName
	}
	return result, nil
}
