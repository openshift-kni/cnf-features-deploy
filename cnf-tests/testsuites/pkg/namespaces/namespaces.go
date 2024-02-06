package namespaces

import (
	"context"
	"os"
	"strings"
	"time"

	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"

	gomega "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
)

// Default is the default namespace for resources
var Default = "default"

// DpdkTest is the namespace of dpdk test suite
var DpdkTest string

// SRIOVOperator is the namespace where the SR-IOV Operator is installed
var SRIOVOperator = "openshift-sriov-network-operator"

// PTPOperator is the namespace where the PTP Operator is installed
var PTPOperator = "openshift-ptp"

// IntelOperator is the namespace where the intel Operators are installed
var IntelOperator = "vran-acceleration-operators"

// SpecialResourceOperator is the namespace where the SRO is installed
var SpecialResourceOperator = "openshift-special-resource-operator"

// MetalLBOperator is the namespace where the MetalLB Operator is installed
var MetalLBOperator = "openshift-metallb-system"

// SroTestNamespace is the namespace where we run the oot driver builds as part of the sro testing
var SroTestNamespace = "oot-driver"

var BondTestNamespace = "bond-testing"

// TuningTest is the namespace used for testing tuningcni features
var TuningTest = "tuning-testing"

// SriovTuingTest is the namespace used for testing feature related to both tuningcni and sriov
var SriovTuningTest = "tuningsriov-testing"

// SCTPTest is the namespace of the sctp test suite
var SCTPTest string

// Multus is the namespace where multus and multi-networkpolicy are installed
var Multus = "openshift-multus"

var OVSQOSTest string

var namespaceLabels = map[string]string{
	"pod-security.kubernetes.io/audit":               "privileged",
	"pod-security.kubernetes.io/enforce":             "privileged",
	"pod-security.kubernetes.io/warn":                "privileged",
	"security.openshift.io/scc.podSecurityLabelSync": "false",
}

func init() {
	DpdkTest = os.Getenv("DPDK_TEST_NAMESPACE")
	if DpdkTest == "" {
		DpdkTest = "dpdk-testing"
	}

	SCTPTest = os.Getenv("SCTP_TEST_NAMESPACE")
	if SCTPTest == "" {
		SCTPTest = "sctptest"
	}

	OVSQOSTest = os.Getenv("OVS_QOS_TEST_NAMESPACE")
	if OVSQOSTest == "" {
		OVSQOSTest = "ovs-qos-testing"
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

func GetPSALabels() map[string]string {
	psaMap := make(map[string]string)
	for k, v := range namespaceLabels {
		psaMap[k] = v
	}

	return psaMap
}

// WaitForDeletion waits until the namespace will be removed from the cluster
func WaitForDeletion(cs corev1client.NamespacesGetter, nsName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := cs.Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// Create creates a new namespace with the given name.
// If the namespace exists, it returns.
func Create(namespace string, cs corev1client.NamespacesGetter) error {
	_, err := cs.Namespaces().Create(
		context.Background(),
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   namespace,
				Labels: namespaceLabels,
			}},
		metav1.CreateOptions{})

	if k8serrors.IsAlreadyExists(err) {
		return AddPSALabelsToNamespace(namespace, cs)
	}
	return err
}

func AddPSALabelsToNamespace(namespace string, cs corev1client.NamespacesGetter) error {
	ns, err := cs.Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	for k, v := range GetPSALabels() {
		ns.Labels[k] = v
	}

	_, err = cs.Namespaces().Update(context.Background(), ns, metav1.UpdateOptions{})
	return err
}

// Delete deletes a namespace with the given name, and waits for it's deletion.
// If the namespace not found, it returns.
func Delete(namespace string, cs *testclient.ClientSet) error {
	err := cs.Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	err = WaitForDeletion(testclient.Client, namespace, 5*time.Minute)
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

// Exists tells whether the given namespace exists
func Exists(namespace string, cs corev1client.NamespacesGetter) bool {
	_, err := cs.Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	return err == nil || !k8serrors.IsNotFound(err)
}

type NamespacesAndPods interface {
	Namespaces() corev1client.NamespaceInterface
	Pods(namespace string) corev1client.PodInterface
}

// CleanPods deletes all pods in namespace
func CleanPods(namespace string, cs NamespacesAndPods) {
	if !Exists(namespace, cs) {
		return
	}
	err := cs.Pods(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0),
	}, metav1.ListOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	gomega.Eventually(func(g gomega.Gomega) int {
		podsList, err := cs.Pods(namespace).List(context.Background(), metav1.ListOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		return len(podsList.Items)
	}, 3*time.Minute, 10*time.Second).Should(gomega.BeZero())
}

// CleanPodsIn deletes all pods in the given namespace list
func CleanPodsIn(cs NamespacesAndPods, namespaces ...string) {
	for _, ns := range namespaces {
		CleanPods(ns, cs)
	}
}
