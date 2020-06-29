// +build !unittests

package test_test

import (
	"flag"
	"log"
	"path"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/cnf-features-deploy/functests/dpdk" // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/functests/icmp"
	"github.com/openshift-kni/cnf-features-deploy/functests/ptp"
	_ "github.com/openshift-kni/cnf-features-deploy/functests/ptp" // this is needed otherwise the ptp test won't be executed
	"github.com/openshift-kni/cnf-features-deploy/functests/sctp"
	_ "github.com/openshift-kni/cnf-features-deploy/functests/sctp" // this is needed otherwise the sctp test won't be executed

	_ "github.com/openshift-kni/performance-addon-operators/functests/1_performance" // this is needed otherwise the performance test won't be executed

	_ "github.com/openshift/ptp-operator/test/ptp"
	_ "github.com/openshift/sriov-network-operator/test/conformance/tests"
	sriovNamespaces "github.com/openshift/sriov-network-operator/test/util/namespaces"

	perfUtils "github.com/openshift-kni/performance-addon-operators/functests/utils"

	testutils "github.com/openshift-kni/cnf-features-deploy/functests/utils"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/clean"
	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var (
	junitPath  *string
	reportPath *string
)

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
	if *junitPath != "" {
		junitFile := path.Join(*junitPath, "cnftests-junit.xml")
		rr = append(rr, reporters.NewJUnitReporter(junitFile))
	}
	if *reportPath != "" {
		reportFile := path.Join(*reportPath, "cnftests_failure_report.log")
		reporter, output, err := testutils.NewReporter(reportFile)
		if err != nil {
			log.Fatalf("Failed to create log reporter %s", err)
		}
		defer output.Close()
		rr = append(rr, reporter)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "CNF Features e2e integration tests", rr)
}

var _ = BeforeSuite(func() {
	Expect(testclient.Client).NotTo(BeNil())
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

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaces.DpdkTest,
		},
	}
	_, err = testclient.Client.Namespaces().Create(ns)

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaces.IcmpTest,
		},
	}

	Expect(err).ToNot(HaveOccurred())
})

// We do the cleanup in AfterSuite because the failure reporter is triggered
// after a test fails. If we did it as part of the test body, the reporter would not
// find the items we want to inspect.
var _ = AfterSuite(func() {
	// Needed because sctp default namespace tests create resources
	// that needs to be selectively deleted
	err := namespaces.Clean("default", "testsctp-", testclient.Client)
	Expect(err).ToNot(HaveOccurred())
	err = clean.SriovResources()
	Expect(err).ToNot(HaveOccurred())
	ptp.Clean()

	nn := []string{testutils.NamespaceTesting,
		perfUtils.NamespaceTesting,
		namespaces.DpdkTest,
		sctp.TestNamespace,
		sriovNamespaces.Test,
		namespaces.IcmpTest,
	}

	for _, n := range nn {
		err := testclient.Client.Namespaces().Delete(n, &metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			continue
		}
		Expect(err).ToNot(HaveOccurred())
		err = namespaces.WaitForDeletion(testclient.Client, n, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred())
	}
})
