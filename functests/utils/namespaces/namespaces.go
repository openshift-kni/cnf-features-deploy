package namespaces

import (
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
)

// WaitForDeletion waits until the namespace will be removed from the cluster
func WaitForDeletion(cs *testclient.ClientSet, nsName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := cs.Namespaces().Get(nsName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// Create creates a new namespace with the given name.
// If the namespace exists, it returns.
func Create(namespace string, cs *testclient.ClientSet) error {
	_, err := cs.Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}})

	if k8serrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// Clean cleans all dangling objects from the given namespace.
func Clean(namespace string, cs *testclient.ClientSet) error {
	_, err := cs.Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	err = cs.NetworkPolicies(namespace).DeleteCollection(&metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0),
	}, metav1.ListOptions{})
	if err != nil {
		return err
	}

	err = cs.Pods(namespace).DeleteCollection(&metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0),
	}, metav1.ListOptions{})

	allServices, err := cs.Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, s := range allServices.Items {
		if isPlatformService(namespace, s.Name) {
			continue
		}
		err = cs.Services(namespace).Delete(s.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: pointer.Int64Ptr(0)})
		if err != nil && k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
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
