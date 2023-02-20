package privilegeddaemonset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pointer "k8s.io/utils/pointer"
)

const (
	tolerationsPeriodSecs = 300
)

type DaemonSetClient struct {
	K8sClient kubernetes.Interface
}

var daemonsetClient = DaemonSetClient{}

func SetDaemonSetClient(k8sClient kubernetes.Interface) {
	daemonsetClient.K8sClient = k8sClient
}

const waitingTime = 5 * time.Second

//nolint:funlen
func createDaemonSetsTemplate(dsName, namespace, containerName, imageWithVersion string, labelsMap map[string]string) *appsv1.DaemonSet {
	dsAnnotations := make(map[string]string)
	dsAnnotations["debug.openshift.io/source-container"] = containerName
	dsAnnotations["openshift.io/scc"] = "node-exporter"

	matchLabels := make(map[string]string)
	matchLabels["name"] = dsName

	if len(labelsMap) != 0 {
		for key, value := range labelsMap {
			matchLabels[key] = value
		}
	}

	rootUser := pointer.Int64(0)

	container := v1core.Container{
		Name:            containerName,
		Image:           imageWithVersion,
		ImagePullPolicy: "IfNotPresent",
		SecurityContext: &v1core.SecurityContext{
			Privileged: pointer.Bool(true),
			RunAsUser:  rootUser,
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

	preemptPolicyLowPrio := v1core.PreemptLowerPriority
	hostPathTypeDir := v1core.HostPathDirectory
	tolerationsSeconds := pointer.Int64(tolerationsPeriodSecs)

	return &appsv1.DaemonSet{

		ObjectMeta: metav1.ObjectMeta{
			Name:        dsName,
			Namespace:   namespace,
			Annotations: dsAnnotations,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: v1core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchLabels,
				},
				Spec: v1core.PodSpec{
					Containers:       []v1core.Container{container},
					PreemptionPolicy: &preemptPolicyLowPrio,
					Priority:         pointer.Int32(0),
					HostNetwork:      true,
					HostIPC:          true,
					HostPID:          true,
					Tolerations: []v1core.Toleration{
						{
							Effect:            "NoExecute",
							Key:               "node.kubernetes.io/not-ready",
							Operator:          "Exists",
							TolerationSeconds: tolerationsSeconds,
						},
						{
							Effect:            "NoExecute",
							Key:               "node.kubernetes.io/unreachable",
							Operator:          "Exists",
							TolerationSeconds: tolerationsSeconds,
						},
						{
							Effect: "NoSchedule",
							Key:    "node-role.kubernetes.io/master",
						},
						{
							Effect: "NoSchedule",
							Key:    "node-role.kubernetes.io/control-plane",
						},
					},
					Volumes: []v1core.Volume{
						{
							Name: "host",
							VolumeSource: v1core.VolumeSource{
								HostPath: &v1core.HostPathVolumeSource{
									Path: "/",
									Type: &hostPathTypeDir,
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
	dsDeleted := false
	start := time.Now()
	for time.Since(start) < Timeout {
		if !doesDaemonSetExist(daemonSetName, namespace) {
			dsDeleted = true
			break
		}
		time.Sleep(waitingTime)
	}

	if !dsDeleted {
		return fmt.Errorf("timeout waiting for daemonset's to be deleted")
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
func CreateDaemonSet(daemonSetName, namespace, containerName, imageWithVersion string, labels map[string]string, timeout time.Duration) (*v1core.PodList, error) {
	daemonSet := createDaemonSetsTemplate(daemonSetName, namespace, containerName, imageWithVersion, labels)

	if doesDaemonSetExist(daemonSetName, namespace) {
		err := DeleteDaemonSet(daemonSetName, namespace)
		if err != nil {
			logrus.Errorf("Failed to delete %s daemonset because: %s", daemonSetName, err)
		}
	}

	logrus.Infof("Creating daemonset %s", daemonSetName)
	_, err := daemonsetClient.K8sClient.AppsV1().DaemonSets(namespace).Create(context.TODO(), daemonSet, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	err = WaitDaemonsetReady(namespace, daemonSetName, timeout)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Daemonset is ready")

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

		if daemonSet.Status.DesiredNumberScheduled == nodesCount {
			logrus.Infof("Waiting for (%d) daemonset pods to be ready: %+v", nodesCount, daemonSet.Status)
			if isDaemonSetReady(&daemonSet.Status) {
				isReady = true
				break
			}
		} else {
			logrus.Warnf("Daemonset %s (ns %s) could not be deployed: DesiredNumberScheduled=%d - NodesCount=%d",
				name, namespace, daemonSet.Status.DesiredNumberScheduled, nodesCount)
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
