//go:build !unittests
// +build !unittests

package setup_test

import (
	"flag"
	"log"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/0_config" // this is needed otherwise the performance test won't be executed
	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var junitPath *string
var reportPath *string

func init() {
	junitPath = flag.String("junit", "", "the path for the junit format report")
	reportPath = flag.String("report", "", "the path of the report file containing details for failed tests")
}

func TestTest(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}

	if *junitPath != "" {
		junitFile := path.Join(*junitPath, "setup_junit.xml")
		rr = append(rr, reporters.NewJUnitReporter(junitFile))
	}
	if *reportPath != "" {
		reportFile := path.Join(*reportPath, "setup_failure_report.log")
		reporter, err := testutils.NewReporter(reportFile)
		if err != nil {
			log.Fatalf("Failed to create log reporter %s", err)
		}
		rr = append(rr, reporter)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "CNF Features e2e setup", rr)
}

var _ = BeforeSuite(func() {
	Expect(testclient.Client).NotTo(BeNil(), "Verify the KUBECONFIG environment variable")
})

var _ = AfterSuite(func() {

})
