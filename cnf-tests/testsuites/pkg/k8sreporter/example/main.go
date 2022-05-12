package main

import (
	"flag"
	"log"
	"time"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/k8sreporter"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"

	sriovNamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	perfUtils "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ptpUtils "github.com/openshift/ptp-operator/test/utils"
	"k8s.io/apimachinery/pkg/runtime"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	ptpv1 "github.com/openshift/ptp-operator/api/v1"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "the kubeconfig path")
	report := flag.String("report", "report.log", "the file name used for the report")

	flag.Parse()

	addToScheme := func(s *runtime.Scheme) {
		ptpv1.AddToScheme(s)
		mcfgv1.AddToScheme(s)
		performancev2.SchemeBuilder.AddToScheme(s)
		sriovv1.AddToScheme(s)

	}

	namespacesToDump := map[string]bool{
		"openshift-ptp":                    true,
		"openshift-sriov-network-operator": true,
		"cnf-features-testing":             true,
		perfUtils.NamespaceTesting:         true,
		namespaces.DpdkTest:                true,
		sriovNamespaces.Test:               true,
		ptpUtils.NamespaceTesting:          true,
	}

	crds := []k8sreporter.CRData{
		{Cr: &mcfgv1.MachineConfigPoolList{}},
		{Cr: &ptpv1.PtpConfigList{}},
		{Cr: &ptpv1.NodePtpDeviceList{}},
		{Cr: &ptpv1.PtpOperatorConfigList{}},
		{Cr: &performancev2.PerformanceProfileList{}},
		{Cr: &sriovv1.SriovNetworkNodePolicyList{}},
		{Cr: &sriovv1.SriovNetworkList{}},
		{Cr: &sriovv1.SriovNetworkNodePolicyList{}},
		{Cr: &sriovv1.SriovOperatorConfigList{}},
	}

	skipByNamespace := func(ns string) bool {
		_, found := namespacesToDump[ns]
		return !found
	}

	reporter, err := k8sreporter.New(*kubeconfig, addToScheme, skipByNamespace, "/tmp", crds...)
	if err != nil {
		log.Fatalf("Failed to initialize the reporter %s", err)
	}
	reporter.Dump(10*time.Minute, *report)
}
