package privilegeddaemonset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DaemonSetClient struct {
	K8sClient kubernetes.Interface
}

var daemonsetClient = DaemonSetClient{}

func SetDaemonSetClient(k8sClient kubernetes.Interface) {
	daemonsetClient.K8sClient = k8sClient
}

const waitingTime = 5 * time.Second

func createDaemonSetsTemplate(dsName, namespace, containerName, imageWithVersion string) *v1.DaemonSet {

	dsAnnotations := make(map[string]string)
	dsAnnotations["debug.openshift.io/source-container"] = containerName
	dsAnnotations["openshift.io/scc"] = "node-exporter"
	matchLabels := make(map[string]string)
	matchLabels["name"] = dsName

	var trueBool bool = true
	var zeroInt int64 = 0
	var zeroInt32 int32 = 0
	var preempt = v1core.PreemptLowerPriority
	var tolerationsSeconds int64 = 300
	var hostPathType = v1core.HostPathDirectory

	container := v1core.Container{
		Name:            containerName,
		Image:           imageWithVersion,
		ImagePullPolicy: "IfNotPresent",
		SecurityContext: &v1core.SecurityContext{
			Privileged: &trueBool,
			RunAsUser:  &zeroInt,
		},
		Stdin:                  true,
		StdinOnce:              true,
		TerminationMessagePath: "/dev/termination-log",
		TTY:                    true,
		VolumeMounts: []v1core.VolumeMount{
			{
				MountPath: "/host",
				Name:      "host",
			},
		},
	}

	return &v1.DaemonSet{

		ObjectMeta: metav1.ObjectMeta{
			Name:        dsName,
			Namespace:   namespace,
			Annotations: dsAnnotations,
		},
		Spec: v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: v1core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchLabels,
				},
				Spec: v1core.PodSpec{
					Containers:       []v1core.Container{container},
					PreemptionPolicy: &preempt,
					Priority:         &zeroInt32,
					HostNetwork:      true,
					Tolerations: []v1core.Toleration{
						{
							Effect:            "NoExecute",
							Key:               "node.kubernetes.io/not-ready",
							Operator:          "Exists",
							TolerationSeconds: &tolerationsSeconds,
						},
						{
							Effect:            "NoExecute",
							Key:               "node.kubernetes.io/unreachable",
							Operator:          "Exists",
							TolerationSeconds: &tolerationsSeconds,
						},
						{
							Effect: "NoSchedule",
							Key:    "node-role.kubernetes.io/master",
						},
					},
					Volumes: []v1core.Volume{
						{
							Name: "host",
							VolumeSource: v1core.VolumeSource{
								HostPath: &v1core.HostPathVolumeSource{
									Path: "/",
									Type: &hostPathType,
								},
							},
						},
					},
				},
			},
		},
	}
}

// This method is used to delete a daemonset specified by the name at a specified namespace
func DeleteDaemonSet(daemonSetName, namespace string) error {
	const (
		Timeout = 5 * time.Minute
	)

	logrus.Infof("Deleting daemonset %s", daemonSetName)
	deletePolicy := metav1.DeletePropagationForeground

	if err := daemonsetClient.K8sClient.AppsV1().DaemonSets(namespace).Delete(context.TODO(), daemonSetName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		logrus.Infof("The daemonset (%s) deletion is unsuccessful due to %+v", daemonSetName, err.Error())
	}

	for start := time.Now(); time.Since(start) < Timeout; {

		pods, err := daemonsetClient.K8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "name=" + daemonSetName})
		if err != nil {
			return fmt.Errorf("failed to get pods, err: %s", err)
		}

		if len(pods.Items) == 0 {
			break
		}
		time.Sleep(waitingTime)
	}

	logrus.Infof("Successfully cleaned up daemonset %s", daemonSetName)
	return nil
}

// Check if the daemonset exists
func doesDaemonSetExist(daemonSetName, namespace string) bool {
	logrus.Infof("Checking if the daemonset exists")
	_, err := daemonsetClient.K8sClient.AppsV1().DaemonSets(namespace).Get(context.TODO(), daemonSetName, metav1.GetOptions{})
	if err != nil {
		logrus.Infof("daemonset %s does not exist, err=%s", daemonSetName, err.Error())
	}
	// If the error is not found, that means the daemonset exists
	return err == nil
}

// This function is used to create a daemonset with the specified name, namespace, container name and image with the timeout to check
// if the deployment is ready and all daemonset pods are running fine
func CreateDaemonSet(daemonSetName, namespace, containerName, imageWithVersion string, timeout time.Duration) (*v1core.PodList, error) {

	rebootDaemonSet := createDaemonSetsTemplate(daemonSetName, namespace, containerName, imageWithVersion)

	if doesDaemonSetExist(daemonSetName, namespace) {
		err := DeleteDaemonSet(daemonSetName, namespace)
		if err != nil {
			logrus.Errorf("Failed to delete %s daemonset because: %s", daemonSetName, err)
		}
	}

	logrus.Infof("Creating daemonset %s", daemonSetName)
	_, err := daemonsetClient.K8sClient.AppsV1().DaemonSets(namespace).Create(context.TODO(), rebootDaemonSet, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	err = WaitDaemonsetReady(namespace, daemonSetName, timeout)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Deamonset is ready")

	var ptpPods *v1core.PodList
	ptpPods, err = daemonsetClient.K8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "name=" + daemonSetName})
	if err != nil {
		return ptpPods, err
	}
	logrus.Infof("Successfully created daemonset %s", daemonSetName)
	return ptpPods, nil
}

// This function is used to wait until daemonset is ready
func WaitDaemonsetReady(namespace, name string, timeout time.Duration) error {

	nodes, err := daemonsetClient.K8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node list, err:%s", err)
	}

	nodesCount := int32(len(nodes.Items))
	isReady := false
	for start := time.Now(); !isReady && time.Since(start) < timeout; {
		daemonSet, err := daemonsetClient.K8sClient.AppsV1().DaemonSets(namespace).Get(context.Background(), name, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("failed to get daemonset, err: %s", err)
		}

		if daemonSet.Status.DesiredNumberScheduled != nodesCount {
			return fmt.Errorf("daemonset DesiredNumberScheduled not equal to number of nodes:%d, please instantiate debug pods on all nodes", nodesCount)
		}

		logrus.Infof("Waiting for (%d) debug pods to be ready: %+v", nodesCount, daemonSet.Status)
		if isDaemonSetReady(&daemonSet.Status) {
			isReady = true
			break
		}
		time.Sleep(waitingTime)
	}

	if !isReady {
		return errors.New("daemonset debug pods not ready")
	}

	logrus.Infof("All the debug pods are ready.")
	return nil
}

func isDaemonSetReady(status *appsv1.DaemonSetStatus) bool {
	//nolint:gocritic
	return status.DesiredNumberScheduled == status.CurrentNumberScheduled &&
		status.DesiredNumberScheduled == status.NumberAvailable &&
		status.DesiredNumberScheduled == status.NumberReady &&
		status.NumberMisscheduled == 0
}
