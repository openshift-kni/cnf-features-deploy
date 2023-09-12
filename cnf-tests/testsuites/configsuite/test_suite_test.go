//go:build !unittests
// +build !unittests

package setup_test

import (
	"flag"
	"log"
	"path"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	kniK8sReporter "github.com/openshift-kni/k8sreporter"
	qe_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/0_config" // this is needed otherwise the performance test won't be executed)
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var (
	junitPath  *string
	reportPath *string
	reporter   *kniK8sReporter.KubernetesReporter
	err        error
)

func init() {
	junitPath = flag.String("junit", "", "the path for the junit format report")
	reportPath = flag.String("report", "", "the path of the report file containing details for failed tests")
}

func TestTest(t *testing.T) {
	RegisterFailHandler(
		func(message string, callerSkip ...int) {
			if reporter != nil {
				reporter.Dump(testutils.LogsExtractDuration, CurrentSpecReport().LeafNodeText)
			}

			// Ensure failing line location is not affected by this wrapper
			for i := range callerSkip {
				callerSkip[i]++
			}

			Fail(message, callerSkip...)
		})

	if *reportPath != "" {
		reportFile := path.Join(*reportPath, "setup_failure_report.log")
		reporter, err = testutils.NewReporter(reportFile)
		if err != nil {
			log.Fatalf("Failed to create log reporter %s", err)
		}
	}

	RunSpecs(t, "CNF Features e2e setup")
}

var _ = BeforeSuite(func() {
	Expect(testclient.Client).NotTo(BeNil(), "Verify the KUBECONFIG environment variable")
})

var _ = AfterSuite(func() {

})

var _ = ReportAfterSuite("setup", func(report types.Report) {
	if *junitPath != "" {
		junitFile := path.Join(*junitPath, "junit_setup.xml")
		reporters.GenerateJUnitReportWithConfig(report, junitFile, reporters.JunitReportConfig{
			OmitTimelinesForSpecState: types.SpecStatePassed | types.SpecStateSkipped,
			OmitLeafNodeType:          true,
			OmitSuiteSetupNodes:       true,
		})
	}

	if qe_reporters.Polarion.Run {
		reporters.ReportViaDeprecatedReporter(&qe_reporters.Polarion, report)
	}
})
