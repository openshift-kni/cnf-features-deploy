package pods

import (
	"bytes"
	"context"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	testclient "github.com/openshift/ptp-operator/test/utils/client"
)

// GetLog connects to a pod and fetches log
func GetLog(p *corev1.Pod, containerName string) (string, error) {
	req := testclient.Client.Pods(p.Namespace).GetLogs(p.Name, &corev1.PodLogOptions{Container: containerName})
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
func ExecCommand(cs *testclient.ClientSet, pod corev1.Pod, containerName string, command []string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	req := testclient.Client.CoreV1Interface.RESTClient().
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
