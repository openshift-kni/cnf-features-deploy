// +build !unittests

package test_test

import (
	"flag"
	"log"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/cnf-features-deploy/functests/dpdk" // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/functests/ptp"  // this is needed otherwise the ptp test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/functests/sctp" // this is needed otherwise the sctp test won't be executed

	_ "github.com/openshift-kni/performance-addon-operators/functests/performance" // this is needed otherwise the performance test won't be executed
	_ "github.com/openshift/sriov-network-operator/test/conformance/tests"

	perfUtils "github.com/openshift-kni/performance-addon-operators/functests/utils"

	testutils "github.com/openshift-kni/cnf-features-deploy/functests/utils"
	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/k8sreporter"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"

	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ptpv1 "github.com/openshift/ptp-operator/pkg/apis/ptp/v1"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var junitPath *string
var reportPath *string

func init() {
	junitPath = flag.String("junit", "junit.xml", "the path for the junit format report")
	reportPath = flag.String("report", "", "the path of the report file containing details for failed tests")
}

func TestTest(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}
	if junitPath != nil {
		rr = append(rr, reporters.NewJUnitReporter(*junitPath))
	}
	if reportPath != nil && *reportPath != "" {
		reporter, output, err := newTestsReporter(*reportPath)
		if err != nil {
			log.Fatalf("Failed to create log reporter %s", err)
		}
		defer output.Close()
		rr = append(rr, reporter)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "CNF Features e2e integration tests", rr)
}

var _ = BeforeSuite(func() {
	// create test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testutils.NamespaceTesting,
		},
	}
	_, err := testclient.Client.Namespaces().Create(ns)
	Expect(err).ToNot(HaveOccurred())

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: perfUtils.NamespaceTesting,
		},
	}
	_, err = testclient.Client.Namespaces().Create(ns)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := testclient.Client.Namespaces().Delete(testutils.NamespaceTesting, &metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
	err = namespaces.WaitForDeletion(testclient.Client, testutils.NamespaceTesting, 5*time.Minute)
	Expect(err).ToNot(HaveOccurred())

	err = testclient.Client.Namespaces().Delete(perfUtils.NamespaceTesting, &metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
	err = namespaces.WaitForDeletion(testclient.Client, perfUtils.NamespaceTesting, 5*time.Minute)
	Expect(err).ToNot(HaveOccurred())
})

func newTestsReporter(reportPath string) (*k8sreporter.KubernetesReporter, *os.File, error) {
	addToScheme := func(s *runtime.Scheme) {
		ptpv1.AddToScheme(s)
		mcfgv1.AddToScheme(s)
	}

	filterPods := func(pod *v1.Pod) bool {
		if pod.Namespace == "sctptest" {
			return false
		}
		if pod.Namespace == "openshift-ptp" {
			return false
		}
		return true
	}

	f, err := os.OpenFile(reportPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	crs := []k8sreporter.CRData{
		k8sreporter.CRData{
			Cr: &mcfgv1.MachineConfigPoolList{},
		},
		k8sreporter.CRData{
			Cr: &ptpv1.PtpConfigList{},
		},
	}

	return k8sreporter.New("", addToScheme, filterPods, f, crs...), f, nil
}
