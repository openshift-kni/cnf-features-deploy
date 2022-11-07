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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/bond"                       // this is needed otherwise the bond test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/dpdk"                       // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/fec"                        // this is needed otherwise the fec test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/gatekeeper"                 // this is needed otherwise the gatekeeper test won't be executed'
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/multinetworkpolicy"         // this is needed otherwise the multinetworkpolicy test won't be executed'
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/ovs_qos"                    // this is needed otherwise the ovs_qos test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/s2i"                        // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sctp"                       // this is needed otherwise the sctp test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/security"                   // this is needed otherwise the security test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sro"                        // this is needed otherwise the sro test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/vrf"                        // this is needed otherwise the vrf test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/xt_u32"                     // this is needed otherwise the xt_u32 test won't be executed
	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/1_performance" // this is needed otherwise the performance test won't be executed
	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/4_latency"     // this is needed otherwise the performance test won't be executed

	_ "github.com/k8snetworkplumbingwg/sriov-network-operator/test/conformance/tests"
	_ "github.com/metallb/metallb-operator/test/e2e/functional/tests"
	_ "github.com/openshift/ptp-operator/test/conformance/ptp"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/features"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"

	_ "github.com/openshift-kni/numaresources-operator/test/e2e/serial/tests"

	ptpReporter "github.com/openshift/ptp-operator/test/pkg/ginkgo_reporter"
	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var (
	junitPath  *string
	reportPath *string
)

var suiteFixtureMap = map[string]features.SuiteFixture{
	"performance":        &features.PAOFixture{},
	"sriov":              &features.SriovFixture{},
	"dpdk":               &features.DPDKFixture{},
	"gatekeeper":         &features.GatekeeperFixture{},
	"sro":                &features.SROFixture{},
	"numaresources":      &features.NumaresourcesFixture{},
	"xt_u32":             &features.XTU32Fixture{},
	"ptp":                &features.PTPFixture{},
	"bondcni":            &features.BondcniFixture{},
	"tuningcni":          &features.TuningcniFixture{},
	"fec":                &features.FECFixture{},
	"vrf":                &features.VRFFixture{},
	"ovs_qos":            &features.OVSQOSFixture{},
	"sctp":               &features.SCTPFixture{},
	"multinetworkpolicy": &features.MultiNetworkPolicyFixture{},
}

func init() {
	junitPath = flag.String("junit", "junit.xml", "the path for the junit format report")
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
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}
	if *junitPath != "" {
		junitFile := path.Join(*junitPath, "cnftests-junit.xml")
		// TODO: This custom reporter won't be needed when this project has been
		//       migrated to ginkgo v2, as we will use v2's AddReportEntry() instead.
		rr = append(rr, ptpReporter.NewPTPJUnitReporter(junitFile))
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
