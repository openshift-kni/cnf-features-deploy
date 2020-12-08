package utils

import (
	"os"

	perfUtils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	ptpUtils "github.com/openshift/ptp-operator/test/utils"
	sriovNamespaces "github.com/openshift/sriov-network-operator/test/util/namespaces"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/k8sreporter"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ptpv1 "github.com/openshift/ptp-operator/pkg/apis/ptp/v1"
	sriovv1 "github.com/openshift/sriov-network-operator/api/v1"
)

// NewReporter creates a specific reporter for CNF tests
func NewReporter(reportPath string) (*k8sreporter.KubernetesReporter, *os.File, error) {
	addToScheme := func(s *runtime.Scheme) {
		ptpv1.AddToScheme(s)
		mcfgv1.AddToScheme(s)
		performancev2.SchemeBuilder.AddToScheme(s)
		sriovv1.AddToScheme(s)

	}

	namespacesToDump := map[string]bool{
		namespaces.PerformanceOperator: true,
		namespaces.PTPOperator:         true,
		namespaces.SRIOVOperator:       true,
		NamespaceTesting:               true,
		perfUtils.NamespaceTesting:     true,
		namespaces.DpdkTest:            true,
		sriovNamespaces.Test:           true,
		ptpUtils.NamespaceTesting:      true,
		namespaces.XTU32Test:           true,
	}

	crds := []k8sreporter.CRData{
		{Cr: &mcfgv1.MachineConfigPoolList{}},
		{Cr: &ptpv1.PtpConfigList{}},
		{Cr: &ptpv1.NodePtpDeviceList{}},
		{Cr: &ptpv1.PtpOperatorConfigList{}},
		{Cr: &performancev2.PerformanceProfileList{}},
		{Cr: &sriovv1.SriovNetworkNodePolicyList{}},
		{Cr: &sriovv1.SriovNetworkList{}},
		{Cr: &sriovv1.SriovNetworkNodeStateList{}},
		{Cr: &sriovv1.SriovOperatorConfigList{}},
	}

	skipPods := func(pod *v1.Pod) bool {
		found := namespacesToDump[pod.Namespace]
		return !found
	}

	f, err := os.OpenFile(reportPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	res, err := k8sreporter.New("", addToScheme, skipPods, f, crds...)
	if err != nil {
		return nil, nil, err
	}
	return res, f, nil
}
