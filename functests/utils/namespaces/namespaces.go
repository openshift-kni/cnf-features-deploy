package namespaces

import (
	"os"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
)

// DpdkTest is the namespace of dpdk test suite
var DpdkTest string
const IcmpTest = "icmp-testing"

func init() {
	DpdkTest = os.Getenv("DPDK_TEST_NAMESPACE")
	if DpdkTest == "" {
		DpdkTest = "dpdk-testing"
	}
}

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
func Clean(namespace string, prefix string, cs *testclient.ClientSet) error {
	_, err := cs.Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	policies, err := cs.NetworkPolicies(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, p := range policies.Items {
		if strings.HasPrefix(p.Name, prefix) {
			err = cs.NetworkPolicies(namespace).Delete(p.Name, &metav1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64Ptr(0),
			})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	pods, err := cs.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, prefix) {
			err = cs.Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64Ptr(0),
			})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	allServices, err := cs.Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, s := range allServices.Items {
		if strings.HasPrefix(s.Name, prefix) {

			err = cs.Services(namespace).Delete(s.Name, &metav1.DeleteOptions{
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
