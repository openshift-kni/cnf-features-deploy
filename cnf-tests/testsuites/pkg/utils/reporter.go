package utils

import (
	"os"

	gkopv1alpha "github.com/gatekeeper/gatekeeper-operator/api/v1alpha1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovNamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	gkv1alpha "github.com/open-policy-agent/gatekeeper/apis/mutations/v1alpha1"
	srov1beta1 "github.com/openshift-psap/special-resource-operator/api/v1beta1"
	ocpbuildv1 "github.com/openshift/api/build/v1"
	ocpv1 "github.com/openshift/api/config/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	perfUtils "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ptpv1 "github.com/openshift/ptp-operator/api/v1"
	ptpUtils "github.com/openshift/ptp-operator/test/utils"
	"k8s.io/apimachinery/pkg/runtime"

	n3000v1 "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/apis/N3000/api/v1"
	sriovfecv2 "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/apis/sriov-fec/api/v2"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/k8sreporter"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
)

// NewReporter creates a specific reporter for CNF tests
func NewReporter(reportPath string) (*k8sreporter.KubernetesReporter, error) {
	addToScheme := func(s *runtime.Scheme) {
		ptpv1.AddToScheme(s)
		mcfgv1.AddToScheme(s)
		performancev2.SchemeBuilder.AddToScheme(s)
		sriovv1.AddToScheme(s)
		gkv1alpha.AddToScheme(s)
		gkopv1alpha.AddToScheme(s)
		nfdv1.AddToScheme(s)
		srov1beta1.AddToScheme(s)
		ocpv1.Install(s)
		ocpbuildv1.Install(s)
	}

	namespacesToDump := map[string]string{
		namespaces.PTPOperator:                  "ptp",
		namespaces.SRIOVOperator:                "sriov",
		NamespaceTesting:                        "other",
		perfUtils.NamespaceTesting:              "performance",
		namespaces.DpdkTest:                     "dpdk",
		sriovNamespaces.Test:                    "sriov",
		ptpUtils.NamespaceTesting:               "ptp",
		namespaces.SCTPTest:                     "sctp",
		namespaces.Default:                      "sctp",
		namespaces.XTU32Test:                    "xt_u32",
		namespaces.IntelOperator:                "intel",
		namespaces.OVSQOSTest:                   "ovs_qos",
		GatekeeperNamespace:                     "gatekeeper",
		OperatorNamespace:                       "gatekeeper",
		GatekeeperTestingNamespace:              "gatekeeper",
		GatekeeperMutationIncludedNamespace:     "gatekeeper",
		GatekeeperMutationExcludedNamespace:     "gatekeeper",
		GatekeeperMutationEnabledNamespace:      "gatekeeper",
		GatekeeperMutationDisabledNamespace:     "gatekeeper",
		GatekeeperTestObjectNamespace:           "gatekeeper",
		GatekeeperConstraintValidationNamespace: "gatekeeper",
		NfdNamespace:                            "sro",
		namespaces.SpecialResourceOperator:      "sro",
		namespaces.SroTestNamespace:             "sro",
	}

	crds := []k8sreporter.CRData{
		{Cr: &mcfgv1.MachineConfigPoolList{}},
		{Cr: &mcfgv1.MachineConfigList{}},
		{Cr: &ptpv1.PtpConfigList{}},
		{Cr: &ptpv1.NodePtpDeviceList{}},
		{Cr: &ptpv1.PtpOperatorConfigList{}},
		{Cr: &performancev2.PerformanceProfileList{}},
		{Cr: &sriovv1.SriovNetworkNodePolicyList{}},
		{Cr: &sriovv1.SriovNetworkList{}},
		{Cr: &sriovv1.SriovNetworkNodeStateList{}},
		{Cr: &sriovv1.SriovOperatorConfigList{}},
		{Cr: &sriovfecv2.SriovFecNodeConfigList{}},
		{Cr: &sriovfecv2.SriovFecClusterConfigList{}},
		{Cr: &n3000v1.N3000ClusterList{}},
		{Cr: &n3000v1.N3000NodeList{}},
		{Cr: &gkv1alpha.AssignList{}},
		{Cr: &gkv1alpha.AssignMetadataList{}},
		{Cr: &gkopv1alpha.GatekeeperList{}},
		{Cr: &srov1beta1.SpecialResourceList{}},
		{Cr: &nfdv1.NodeFeatureDiscoveryList{}},
		{Cr: &ocpbuildv1.BuildConfigList{}, Namespace: &namespaces.SroTestNamespace},
		{Cr: &ocpbuildv1.BuildList{}, Namespace: &namespaces.SroTestNamespace},
	}

	skipByNamespace := func(ns string) bool {
		_, found := namespacesToDump[ns]
		return !found
	}

	err := os.Mkdir(reportPath, 0755)
	if err != nil {
		return nil, err
	}

	res, err := k8sreporter.New("", addToScheme, skipByNamespace, reportPath, crds...)
	if err != nil {
		return nil, err
	}
	return res, nil
}
