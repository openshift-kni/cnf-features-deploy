//go:build !unittests
// +build !unittests

package test_test

import (
	"context"
	"flag"
	"log"
	"path"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/dpdk"       // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/fec"        // this is needed otherwise the fec test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/gatekeeper" // this is needed otherwise the gatekeeper test won't be executed'
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/ovs_qos"    // this is needed otherwise the ovs_qos test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/ptp"        // this is needed otherwise the ptp test won't be executed
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sctp"
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sctp" // this is needed otherwise the sctp test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sro"  // this is needed otherwise the sro test won't be executed
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/vrf"
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/xt_u32"                     // this is needed otherwise the xt_u32 test won't be executed
	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/1_performance" // this is needed otherwise the performance test won't be executed
	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/4_latency"     // this is needed otherwise the performance test won't be executed

	_ "github.com/k8snetworkplumbingwg/sriov-network-operator/test/conformance/tests"
	sriovNamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	_ "github.com/metallb/metallb-operator/test/e2e/functional/tests"
	_ "github.com/openshift/ptp-operator/test/conformance/ptp"

	perfUtils "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils"

	sriovClean "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/clean"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/clean"
	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	perfClean "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils/clean"
	ptpClean "github.com/openshift/ptp-operator/test/utils/clean"

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
		reporter, err := testutils.NewReporter(reportFile)
		if err != nil {
			log.Fatalf("Failed to create log reporter %s", err)
		}
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
	_, err := testclient.Client.Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: perfUtils.NamespaceTesting,
		},
	}
	_, err = testclient.Client.Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaces.DpdkTest,
		},
	}
	_, err = testclient.Client.Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testutils.GatekeeperTestingNamespace,
		},
	}
	_, err = testclient.Client.Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaces.SroTestNamespace,
		},
	}
	_, err = testclient.Client.Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
})

// We do the cleanup in AfterSuite because the failure reporter is triggered
// after a test fails. If we did it as part of the test body, the reporter would not
// find the items we want to inspect.
var _ = AfterSuite(func() {
	clean.All()
	ptpClean.All()
	sriovClean.All()
	if !discovery.Enabled() {
		perfClean.All()
	}

	nn := []string{testutils.NamespaceTesting,
		perfUtils.NamespaceTesting,
		namespaces.DpdkTest,
		sctp.TestNamespace,
		vrf.TestNamespace,
		sriovNamespaces.Test,
		namespaces.XTU32Test,
		testutils.GatekeeperTestingNamespace,
		namespaces.OVSQOSTest,
		namespaces.SroTestNamespace,
	}

	for _, n := range nn {
		err := testclient.Client.Namespaces().Delete(context.Background(), n, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			continue
		}
		Expect(err).ToNot(HaveOccurred())
		err = namespaces.WaitForDeletion(testclient.Client, n, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred())
	}
})
