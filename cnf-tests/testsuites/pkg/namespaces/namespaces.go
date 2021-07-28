package namespaces

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	gomega "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
)

// DpdkTest is the namespace of dpdk test suite
var DpdkTest string

// PerformanceOperator is the namespace where PAO is installed
var PerformanceOperator = "openshift-performance-addon-operator"

// SRIOVOperator is the namespace where the SR-IOV Operator is installed
var SRIOVOperator = "openshift-sriov-network-operator"

// PTPOperator is the namespace where the PTP Operator is installed
var PTPOperator = "openshift-ptp"

// IntelOperator is the namespace where the intel Operators are installed
var IntelOperator = "vran-acceleration-operators"

// SpecialResourceOperator is the namespace where the SRO is installed
var SpecialResourceOperator = "openshift-special-resource-operator"

// SroTestNamespace is the namespace where we run the oot driver builds as part of the sro testing
var SroTestNamespace = "oot-driver"

// XTU32Test is the namespace of xt_u32 test suite
var XTU32Test string

// SCTPTest is the namespace of the sctp test suite
var SCTPTest string

var OVSQOSTest string

func init() {
	DpdkTest = os.Getenv("DPDK_TEST_NAMESPACE")
	if DpdkTest == "" {
		DpdkTest = "dpdk-testing"
	}

	SCTPTest = os.Getenv("SCTP_TEST_NAMESPACE")
	if SCTPTest == "" {
		SCTPTest = "sctptest"
	}

	XTU32Test = os.Getenv("XT_U32_TEST_NAMESPACE")
	if XTU32Test == "" {
		XTU32Test = "xt-u32-testing"
	}

	OVSQOSTest = os.Getenv("OVS_QOS_TEST_NAMESPACE")
	if OVSQOSTest == "" {
		OVSQOSTest = "ovs-qos-testing"
	}

	if performanceOverride, ok := os.LookupEnv("PERFORMANCE_OPERATOR_NAMESPACE"); ok {
		PerformanceOperator = performanceOverride
	}
	// Legacy value: in the SRIOV operator tests this variable is already
	// used to override the namespace.
	if sriovOverride, ok := os.LookupEnv("OPERATOR_NAMESPACE"); ok {
		SRIOVOperator = sriovOverride
	}
	if sriovOverride, ok := os.LookupEnv("SRIOV_OPERATOR_NAMESPACE"); ok {
		SRIOVOperator = sriovOverride
	}
	if ptpOverride, ok := os.LookupEnv("PTP_OPERATOR_NAMESPACE"); ok {
		PTPOperator = ptpOverride
	}
}

// WaitForDeletion waits until the namespace will be removed from the cluster
func WaitForDeletion(cs *testclient.ClientSet, nsName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := cs.Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// Create creates a new namespace with the given name.
// If the namespace exists, it returns.
func Create(namespace string, cs *testclient.ClientSet) error {
	_, err := cs.Namespaces().Create(
		context.Background(),
		&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			}},
		metav1.CreateOptions{})

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
			if err != nil && !errors.IsNotFound(err) {
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
			if err != nil && !errors.IsNotFound(err) {
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

// Exists tells whether the given namespace exists
func Exists(namespace string, cs *testclient.ClientSet) bool {
	_, err := cs.Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	return err == nil || !k8serrors.IsNotFound(err)
}

// CleanPods deletes all pods in namespace
func CleanPods(namespace string, cs *testclient.ClientSet) error {
	if !Exists(namespace, cs) {
		return nil
	}
	err := cs.Pods(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0),
	}, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Failed to delete pods %v", err)
	}
	gomega.Eventually(func() int {
		podsList, err := cs.Pods(namespace).List(context.Background(), metav1.ListOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		return len(podsList.Items)
	}, 3*time.Minute, 10*time.Second).Should(gomega.BeZero())
	return nil
}
