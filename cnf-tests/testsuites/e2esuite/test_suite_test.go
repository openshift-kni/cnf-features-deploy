//go:build !unittests
// +build !unittests

package test_test

import (
	"flag"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/features"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	kniK8sReporter "github.com/openshift-kni/k8sreporter"
	qe_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/bond"               // this is needed otherwise the bond test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/dpdk"               // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/fec"                // this is needed otherwise the fec test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/multinetworkpolicy" // this is needed otherwise the multinetworkpolicy test won't be executed'
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/ovs_qos"            // this is needed otherwise the ovs_qos test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/s2i"                // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sctp"               // this is needed otherwise the sctp test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/security"           // this is needed otherwise the security test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/vrf"                // this is needed otherwise the vrf test won't be executed
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var (
	junitPath  *string
	reportPath *string
	reporter   *kniK8sReporter.KubernetesReporter
	err        error
)

var suiteFixtureMap = map[string]features.SuiteFixture{
	"performance":        &features.PAOFixture{},
	"sriov":              &features.SriovFixture{},
	"dpdk":               &features.DPDKFixture{},
	"bondcni":            &features.BondcniFixture{},
	"tuningcni":          &features.TuningcniFixture{},
	"fec":                &features.FECFixture{},
	"vrf":                &features.VRFFixture{},
	"ovs_qos":            &features.OVSQOSFixture{},
	"sctp":               &features.SCTPFixture{},
	"multinetworkpolicy": &features.MultiNetworkPolicyFixture{},
}

func init() {
	junitPath = flag.String("junit", "", "the path for the junit format report")
	reportPath = flag.String("report", "", "the path of the report file containing details for failed tests")

	featuresVar := os.Getenv("FEATURES")
	if featuresVar != "" {
		for feature := range suiteFixtureMap {
			if !strings.Contains(featuresVar, feature) {
				delete(suiteFixtureMap, feature)
			}
		}
	}
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
		reportFile := path.Join(*reportPath, "cnftests_failure_report.log")
		reporter, err = testutils.NewReporter(reportFile)
		if err != nil {
			log.Fatalf("Failed to create log reporter %s", err)
		}
	}

	RunSpecs(t, "CNF Features e2e integration tests")
}

var _ = BeforeSuite(func() {
	for _, feature := range suiteFixtureMap {
		err := feature.Setup()
		Expect(err).ToNot(HaveOccurred())
	}
})

// We do the cleanup in AfterSuite because the failure reporter is triggered
// after a test fails. If we did it as part of the test body, the reporter would not
// find the items we want to inspect.
var _ = AfterSuite(func() {
	for _, feature := range suiteFixtureMap {
		err := feature.Cleanup()
		Expect(err).ToNot(HaveOccurred())
	}
})

var _ = ReportAfterSuite("cnftests", func(report types.Report) {
	if *junitPath != "" {
		junitFile := path.Join(*junitPath, "junit_cnftests.xml")
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
