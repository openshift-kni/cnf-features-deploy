package pods

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/images"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/pointer"
)

func getDefinition(namespace string) *corev1.Pod {
	podObject := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testpod-",
			Namespace:    namespace},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: pointer.Int64Ptr(0),
			Containers: []corev1.Container{{Name: "test",
				Image:   images.For(images.TestUtils),
				Command: []string{"/bin/bash", "-c", "sleep INF"}}}}}

	return podObject
}

func containerByName(pod *corev1.Pod, containerName string) *corev1.Container {
	if containerName == "" {
		return &pod.Spec.Containers[0]
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == containerName {
			return &c
		}
	}

	return nil
}

// DefineWithNetworks defines a pod with the given secondary networks.
func DefineWithNetworks(namespace string, networks []string) *corev1.Pod {
	podObject := getDefinition(namespace)
	podObject.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": strings.Join(networks, ",")}
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

// DefinePodOnNode creates the pod defintion with a node selector
func DefinePodOnNode(namespace string, nodeName string) *corev1.Pod {
	pod := getDefinition(namespace)
	pod.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
	return pod
}

// DefinePod creates a pod definition
func DefinePod(namespace string) *corev1.Pod {
	return getDefinition(namespace)
}

// RedefinePodWithNetwork updates the pod defintion with a network annotation
func RedefinePodWithNetwork(pod *corev1.Pod, networksSpec string) *corev1.Pod {
	pod.ObjectMeta.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": networksSpec}
	return pod
}

// RedefineAsPrivileged updates the pod definition to be privileged
func RedefineAsPrivileged(pod *corev1.Pod, containerName string) (*corev1.Pod, error) {
	c := containerByName(pod, containerName)
	if c == nil {
		return pod, fmt.Errorf("container with name: %s not found in pod", containerName)
	}
	if c.SecurityContext == nil {
		c.SecurityContext = &corev1.SecurityContext{}
	}
	c.SecurityContext.Privileged = pointer.BoolPtr(true)

	return pod, nil
}

// DefinePodOnHostNetwork updates the pod defintion with a host network flag
func DefinePodOnHostNetwork(namespace string, nodeName string) *corev1.Pod {
	pod := DefinePodOnNode(namespace, nodeName)
	pod.Spec.HostNetwork = true
	return pod
}

// DefineWithHugePages creates a pod with a 4Gi of hugepages and run command to write data to that memory
func DefineWithHugePages(namespace, nodeName string) *corev1.Pod {
	pod := RedefineWithRestartPolicy(
		RedefineWithCommand(
			getDefinition(namespace),
			[]string{"/bin/bash", "-c",
				`tmux new -d 'LD_PRELOAD=libhugetlbfs.so HUGETLB_MORECORE=yes top -b > /dev/null'
sleep INF`}, []string{},
		),
		corev1.RestartPolicyNever,
	)

	pod.Spec.NodeSelector = map[string]string{
		"kubernetes.io/hostname": nodeName,
	}

	// Resource request
	pod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{}
	pod.Spec.Containers[0].Resources.Requests["memory"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Requests["hugepages-1Gi"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Requests["cpu"] = *resource.NewQuantity(int64(1), resource.DecimalSI)

	// Resource limit
	pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{}
	pod.Spec.Containers[0].Resources.Limits["memory"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Limits["hugepages-1Gi"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Limits["cpu"] = *resource.NewQuantity(int64(1), resource.DecimalSI)

	// Hugepages volume mount
	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "hugepages", MountPath: "/dev/hugepages"}}

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
		_, err := cs.Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// WaitForCondition waits until the pod will have specified condition type with the expected status
func WaitForCondition(cs *testclient.ClientSet, pod *corev1.Pod, conditionType corev1.PodConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatePod, err := cs.Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
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

// WaitForPhase waits until the pod will be in specified phase
func WaitForPhase(cs *testclient.ClientSet, pod *corev1.Pod, phaseType corev1.PodPhase, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatePod, err := cs.Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return updatePod.Status.Phase == phaseType, nil
	})
}

// GetLog connects to a pod and fetches log
func GetLog(p *corev1.Pod) (string, error) {
	req := testclient.Client.Pods(p.Namespace).GetLogs(p.Name, &corev1.PodLogOptions{})
	log, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("cannot get logs for pod %s: %w", p.Name, err)
	}
	defer log.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, log)

	if err != nil {
		return "", fmt.Errorf("cannot copy logs to buffer for pod %s: %w", p.Name, err)
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
		return buf, fmt.Errorf("cannot create SPDY executor for req %s: %w", req.URL().String(), err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &buf,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		return buf, fmt.Errorf("remove command %v error %w", command, err)
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
		return "", fmt.Errorf("command %v error: %w", collectRoutesCommand, err)
	}
	routeTable := routeTableBuf.String()
	for _, route := range strings.Split(routeTable, "\n")[1 : len(strings.Split(routeTable, "\n"))-1] {
		if strings.Split(strings.Join(strings.Fields(route), " "), " ")[1] == "00000000" &&
			strings.Split(strings.Join(strings.Fields(route), " "), " ")[7] == "00000000" {
			return strings.Split(strings.Join(strings.Fields(route), " "), " ")[0], nil
		}
	}
	return "", fmt.Errorf("default route not present")
}
