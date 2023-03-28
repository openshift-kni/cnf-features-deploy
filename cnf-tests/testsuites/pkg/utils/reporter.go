package utils

import (
	"os"

	gkopv1alpha "github.com/gatekeeper/gatekeeper-operator/api/v1alpha1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	sriovNamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	metallbv1beta1 "github.com/metallb/metallb-operator/api/v1beta1"
	gkv1alpha "github.com/open-policy-agent/gatekeeper/apis/mutations/v1alpha1"
	srov1beta1 "github.com/openshift-psap/special-resource-operator/api/v1beta1"
	ocpbuildv1 "github.com/openshift/api/build/v1"
	ocpv1 "github.com/openshift/api/config/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	perfUtils "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ptpv1 "github.com/openshift/ptp-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"

	multinetpolicyv1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	n3000v1 "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/apis/N3000/api/v1"
	sriovfecv2 "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/apis/sriov-fec/api/v2"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/k8sreporter"
)

// NewReporter creates a specific reporter for CNF tests
func NewReporter(reportPath string) (*k8sreporter.KubernetesReporter, error) {
	addToScheme := func(s *runtime.Scheme) error {
		err := ptpv1.AddToScheme(s)
		if err != nil {
			return err
		}
		err = mcfgv1.AddToScheme(s)
		if err != nil {
			return err
		}
		err = performancev2.SchemeBuilder.AddToScheme(s)
		if err != nil {
			return err
		}
		err = netattdefv1.SchemeBuilder.AddToScheme(s)
		if err != nil {
			return err
		}
		err = sriovv1.AddToScheme(s)
		if err != nil {
			return err
		}
		err = gkv1alpha.AddToScheme(s)
		if err != nil {
			return err
		}
		err = gkopv1alpha.AddToScheme(s)
		if err != nil {
			return err
		}
		err = nfdv1.AddToScheme(s)
		if err != nil {
			return err
		}
		err = srov1beta1.AddToScheme(s)
		if err != nil {
			return err
		}
		err = metallbv1beta1.AddToScheme(s)
		if err != nil {
			return err
		}
		err = ocpv1.Install(s)
		if err != nil {
			return err
		}
		err = ocpbuildv1.Install(s)
		if err != nil {
			return err
		}
		err = multinetpolicyv1.AddToScheme(s)
		if err != nil {
			return err
		}
		err = netattdefv1.AddToScheme(s)
		if err != nil {
			return err
		}
		return nil
	}

	namespacesToDump := map[string]string{
		namespaces.PTPOperator:                  "ptp",
		namespaces.SRIOVOperator:                "sriov",
		perfUtils.NamespaceTesting:              "performance",
		namespaces.DpdkTest:                     "dpdk",
		sriovNamespaces.Test:                    "sriov",
		namespaces.SriovTuningTest:              "sriov",
		MultiNetworkPolicyNamespaceX:            "multinetworkpolicy",
		MultiNetworkPolicyNamespaceY:            "multinetworkpolicy",
		MultiNetworkPolicyNamespaceZ:            "multinetworkpolicy",
		namespaces.SCTPTest:                     "sctp",
		namespaces.Default:                      "sctp",
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
		namespaces.SroTestNamespace:             "sro",
		namespaces.BondTestNamespace:            "bondcni",
		namespaces.MetalLBOperator:              "metallb",
		namespaces.TuningTest:                   "tuningcni",
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
		{Cr: &multinetpolicyv1.MultiNetworkPolicyList{}},
		{Cr: &netattdefv1.NetworkAttachmentDefinitionList{}},
		{Cr: &metallbv1beta1.MetalLBList{}},
	}

	namespaceToLog := func(ns string) bool {
		_, found := namespacesToDump[ns]
		return found
	}

	err := os.Mkdir(reportPath, 0755)
	if err != nil {
		return nil, err
	}

	res, err := k8sreporter.New("", addToScheme, namespaceToLog, reportPath, crds...)
	if err != nil {
		return nil, err
	}
	return res, nil
}
