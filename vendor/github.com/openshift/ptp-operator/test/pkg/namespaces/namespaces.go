package namespaces

import (
	"context"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	testclient "github.com/openshift/ptp-operator/test/pkg/client"
	"github.com/sirupsen/logrus"
)

// WaitForDeletion waits until the namespace will be removed from the cluster
func WaitForDeletion(cs *testclient.ClientSet, nsName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := cs.CoreV1().Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// Create creates a new namespace with the given name.
// If the namespace exists, it returns.
func Create(namespace string, cs *testclient.ClientSet) error {
	_, err := cs.CoreV1().Namespaces().Create(context.Background(), &k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}},
		metav1.CreateOptions{},
	)

	if k8serrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// Clean cleans all dangling objects from the given namespace.
func Clean(namespace string, prefix string, cs *testclient.ClientSet) error {
	_, err := cs.Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	policies, err := cs.NetworkPolicies(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, p := range policies.Items {
		if strings.HasPrefix(p.Name, prefix) {
			err = cs.NetworkPolicies(namespace).Delete(context.Background(), p.Name, metav1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64Ptr(0),
			})
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}

	pods, err := cs.Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, prefix) {
			err = cs.Pods(namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64Ptr(0),
			})
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}

	allServices, err := cs.Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, s := range allServices.Items {
		if strings.HasPrefix(s.Name, prefix) {

			err = cs.Services(namespace).Delete(context.Background(), s.Name, metav1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64Ptr(0)})
			if err != nil && k8serrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
		}
	}
	return err
}

func isPlatformService(namespace, serviceName string) bool {
	switch {
	case namespace != "default":
		return false
	case serviceName == "kubernetes":
		return true
	case serviceName == "openshift":
		return true
	default:
		return false
	}
}

func Delete(aNamespace string, cs *testclient.ClientSet) error {
	return cs.Namespaces().Delete(context.Background(), aNamespace, metav1.DeleteOptions{})
}

// WaitForCondition waits until the pod will have specified condition type with the expected status
func IsPresent(namespace string, cs *testclient.ClientSet) bool {
	_, err := cs.Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		logrus.Debugf("Is Present err=%s", err)
		return false
	}
	return true
}
