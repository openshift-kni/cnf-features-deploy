package pods

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/images"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
)

func getDefinition(namespace string) *corev1.Pod {
	podObject := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testpod-",
			Namespace:    namespace},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: ptr.To[int64](0),
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

// RedefineWithLabel add a label to the ObjectMeta.Labels field of the pod, instantiating
// the map if necessary. Override the previous label it is already present.
func RedefineWithLabel(pod *corev1.Pod, key, value string) *corev1.Pod {
	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = map[string]string{}
	}
	pod.ObjectMeta.Labels[key] = value
	return pod
}

// RedefineWithPodAfinityOnLabel sets the spec.podAffinity field using the given label key and value.
// It can be use to ensure a pod is scheduled on the same node as another, selecting the reference pod by a label.
func RedefineWithPodAffinityOnLabel(pod *corev1.Pod, key, value string) *corev1.Pod {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}

	if pod.Spec.Affinity.PodAffinity == nil {
		pod.Spec.Affinity.PodAffinity = &corev1.PodAffinity{}
	}

	pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []corev1.PodAffinityTerm{{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{key: value},
		},
		TopologyKey: "kubernetes.io/hostname",
	}}

	return pod
}

// RedefineWithRestrictedPrivileges enforces restricted privileges on the pod.
func RedefineWithRestrictedPrivileges(pod *corev1.Pod) *corev1.Pod {
	pod.Spec.SecurityContext = &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		FSGroup:        ptr.To[int64](1001),
	}
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].SecurityContext.RunAsNonRoot = pointer.BoolPtr(true)
		pod.Spec.Containers[i].SecurityContext.RunAsUser = ptr.To[int64](1001)
		pod.Spec.Containers[i].SecurityContext.RunAsGroup = ptr.To[int64](1001)
		pod.Spec.Containers[i].SecurityContext.Privileged = pointer.BoolPtr(false)
		pod.Spec.Containers[i].SecurityContext.Capabilities.Drop = []corev1.Capability{"ALL"}

		// Capabilities in binaries do not work if below is set to false.
		pod.Spec.Containers[i].SecurityContext.AllowPrivilegeEscalation = pointer.BoolPtr(true)
	}

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

// RedefineWithGuaranteedQoS updates the pod definition by adding resource limits and request
// to the specified values. As requests and limits are equal, the pod will work with a Guarantted
// quality of service (QoS). Resource specification are added to the first container
func RedefineWithGuaranteedQoS(pod *corev1.Pod, cpu, memory string) *corev1.Pod {
	resources := map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceMemory: resource.MustParse(memory),
		corev1.ResourceCPU:    resource.MustParse(cpu),
	}

	pod.Spec.Containers[0].Resources.Requests = resources
	pod.Spec.Containers[0].Resources.Limits = resources

	return pod
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
	pod.Spec.Containers[0].Resources.Requests["cpu"] = *resource.NewQuantity(int64(2), resource.DecimalSI)

	// Resource limit
	pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{}
	pod.Spec.Containers[0].Resources.Limits["memory"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Limits["hugepages-1Gi"] = resource.MustParse("1Gi")
	pod.Spec.Containers[0].Resources.Limits["cpu"] = *resource.NewQuantity(int64(2), resource.DecimalSI)

	// Hugepages volume mount
	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "hugepages", MountPath: "/dev/hugepages"}}

	// Security context capabilities
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsUser: ptr.To[int64](0),
		Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"IPC_LOCK"}}}

	// Hugepages volume
	pod.Spec.Volumes = []corev1.Volume{{Name: "hugepages",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumHugePages}}}}

	return pod
}

func DefineDPDKWorkload(nodeSelector map[string]string, command string, image string, additionalCapabilities []corev1.Capability) *corev1.Pod {
	resources := map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceName("hugepages-1Gi"): resource.MustParse("2Gi"),
		corev1.ResourceMemory:                resource.MustParse("1Gi"),
		corev1.ResourceCPU:                   resource.MustParse("4"),
	}

	// Enable NET_RAW is required by mellanox nics as they are using the netdevice driver
	// NET_RAW was removed from the default capabilities
	// https://access.redhat.com/security/cve/cve-2020-14386
	capabilities := []corev1.Capability{"IPC_LOCK", "SYS_RESOURCE", "NET_RAW"}
	if additionalCapabilities != nil {
		capabilities = append(capabilities, additionalCapabilities...)
	}

	container := corev1.Container{
		Name:  "dpdk",
		Image: image,
		Command: []string{
			"/bin/bash",
			"-c",
			command,
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: ptr.To[int64](0),
			Capabilities: &corev1.Capabilities{
				Add: capabilities,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "RUN_TYPE",
				Value: "testpmd",
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: resources,
			Limits:   resources,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "hugepages",
				MountPath: "/mnt/huge",
			},
		},
	}

	dpdkPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dpdk-",
			Namespace:    namespaces.DpdkTest,
			Labels: map[string]string{
				"app": "dpdk",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{container},
			Volumes: []corev1.Volume{
				{
					Name: "hugepages",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumHugePages},
					},
				},
			},
		},
	}

	if len(nodeSelector) > 0 {
		dpdkPod.Spec.NodeSelector = nodeSelector
	}

	if nodeSelector != nil && len(nodeSelector) > 0 {
		if dpdkPod.Spec.NodeSelector == nil {
			dpdkPod.Spec.NodeSelector = make(map[string]string)
		}
		for k, v := range nodeSelector {
			dpdkPod.Spec.NodeSelector[k] = v
		}
	}

	return dpdkPod
}

func CreateDPDKWorkload(nodeSelector map[string]string, command string, image string, additionalCapabilities []corev1.Capability, mac string) (*corev1.Pod, error) {
	network := fmt.Sprintf(`[{"name": "test-dpdk-network","mac": "%s","namespace": "%s"}]`, mac, namespaces.DpdkTest)
	return CreateAndStart(RedefinePodWithNetwork(DefineDPDKWorkload(nodeSelector, command, image, additionalCapabilities), network))
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

func CreateAndStart(pod *corev1.Pod) (*corev1.Pod, error) {

	pod, err := client.Client.Pods(pod.Namespace).
		Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot create pod [%s]: %w", pod.Name, err)
	}

	err = WaitForCondition(client.Client, pod, corev1.ContainersReady, corev1.ConditionTrue, 3*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("error while waiting pod [%s] to be ready: %w", pod.Name, err)
	}

	err = client.Client.Get(context.Background(),
		goclient.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, pod)
	if err != nil {
		return nil, fmt.Errorf("cannot get just created pod [%s]: %w", pod.Name, err)
	}

	return pod, nil
}

// GetLog connects to a pod and fetches log
func GetLog(p *corev1.Pod) (string, error) {
	return GetLogForContainer(p, p.Spec.Containers[0].Name)
}

// GetLog connects to a pod and fetches log from a specific container
func GetLogForContainer(p *corev1.Pod, containerName string) (string, error) {
	req := testclient.Client.Pods(p.Namespace).GetLogs(p.Name, &corev1.PodLogOptions{Container: containerName})
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

// ExecCommand runs command in the pod's firts container and returns buffer output
func ExecCommand(cs *testclient.ClientSet, pod corev1.Pod, command []string) (bytes.Buffer, error) {
	if len(pod.Spec.Containers) == 0 {
		return *bytes.NewBuffer([]byte{}), fmt.Errorf("pod [%s] has no containers", pod.Name)
	}

	return ExecCommandInContainer(cs, pod, pod.Spec.Containers[0].Name, command)
}

// ExecCommand runs command in the specified container and returns buffer output
func ExecCommandInContainer(cs *testclient.ClientSet, pod corev1.Pod, containerName string, command []string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	req := client.Client.CoreV1Interface.RESTClient().
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
		return buf, fmt.Errorf("cannot create SPDY executor for req %s: %w", req.URL().String(), err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &buf,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		return buf, fmt.Errorf("remote command %v error [%w]. output [%s]", command, err, buf.String())
	}

	return buf, nil
}

// DetectDefaultRouteInterface returns pod's interface bounded to the default route which contains
// Destination=00000000 and Metric=00000000
// Typical route table look like present below:
// Iface	  Destination(1) Gateway 	Flags	RefCnt	Use	Metric	Mask(7)		MTU	Window	IRTT
// enp2s0	00000000	016FA8C0	0003	0		0	101		00000000	0	0	0  < This is default route
// tun0		0000800A	00000000	0001	0		0	0		0000FCFF	0	0	0
// enp2s0	006FA8C0	00000000	0001	0		0	101		00FFFFFF	0	0	0
// enp6s0	00C7A8C0	00000000	0001	0		0	102		00FFFFFF	0	0	0
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

func getStringEventsForPod(cs corev1client.EventsGetter, pod *corev1.Pod) string {
	if pod == nil {
		return "can't retrieve events for nil pod"
	}

	events, err := cs.Events(pod.Namespace).List(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name), TypeMeta: metav1.TypeMeta{Kind: "Pod"}})
	if err != nil {
		return fmt.Sprintf("can't retrieve events for pod %s/%s: %s", pod.Namespace, pod.Name, err.Error())
	}

	var res string
	for _, item := range events.Items {
		eventStr := fmt.Sprintf("%s: %s", item.LastTimestamp, item.Message)
		res = res + fmt.Sprintf("%s\n", eventStr)
	}

	return res
}

func GetStringEventsForPodFn(cs *testclient.ClientSet, pod *corev1.Pod) func() string {
	return func() string {
		return getStringEventsForPod(cs, pod)
	}
}
