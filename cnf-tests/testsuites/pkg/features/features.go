package features

import (
	"fmt"

	sriovClean "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/clean"
	sriovNamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	perfUtils "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils"
	perfClean "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils/clean"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/fec"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/security"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sro"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/vrf"
	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	ptpclean "github.com/openshift/ptp-operator/test/pkg/clean"
	ptpclient "github.com/openshift/ptp-operator/test/pkg/client"
)

type SuiteFixture interface {
	Setup() error
	Cleanup() error
}

type PAOFixture struct {
}

func (p *PAOFixture) Setup() error {
	return namespaces.Create(perfUtils.NamespaceTesting, testclient.Client)
}

func (p *PAOFixture) Cleanup() error {
	if !discovery.Enabled() {
		perfClean.All()
	}

	return namespaces.Delete(perfUtils.NamespaceTesting, testclient.Client)
}

type DPDKFixture struct {
}

func (p *DPDKFixture) Setup() error {
	return namespaces.Create(namespaces.DpdkTest, testclient.Client)
}

func (p *DPDKFixture) Cleanup() error {
	return namespaces.Delete(namespaces.DpdkTest, testclient.Client)
}

type GatekeeperFixture struct {
}

func (p *GatekeeperFixture) Setup() error {
	return namespaces.Create(testutils.GatekeeperTestingNamespace, testclient.Client)
}

func (p *GatekeeperFixture) Cleanup() error {
	return namespaces.Delete(testutils.GatekeeperTestingNamespace, testclient.Client)
}

type SROFixture struct {
}

func (p *SROFixture) Setup() error {
	return namespaces.Create(namespaces.SroTestNamespace, testclient.Client)
}

func (p *SROFixture) Cleanup() error {
	sro.Clean()
	return namespaces.Delete(namespaces.SroTestNamespace, testclient.Client)
}

type PTPFixture struct {
}

func (p *PTPFixture) Setup() error {
	ptpclient.Setup()
	return nil
}

func (p *PTPFixture) Cleanup() error {
	return ptpclean.All()
}

type SriovFixture struct {
}

func (p *SriovFixture) Setup() error {
	return nil
}

func (p *SriovFixture) Cleanup() error {
	sriovClean.All()

	err := namespaces.Delete(security.SriovTestNamespace, testclient.Client)
	if err != nil {
		return err
	}

	return namespaces.Delete(sriovNamespaces.Test, testclient.Client)
}

type SCTPFixture struct {
}

func (p *SCTPFixture) Setup() error {
	err := namespaces.Create(namespaces.SCTPTest, testclient.Client)
	if err != nil {
		return err
	}

	err = namespaces.AddPSALabelsToNamespace(namespaces.Default, testclient.Client)
	return err
}

func (p *SCTPFixture) Cleanup() error {
	err := namespaces.Clean("default", "testsctp-", testclient.Client)
	if err != nil {
		return fmt.Errorf("failed to clean 'default' namespace 'testsctp' prefix")
	}

	return namespaces.Delete(namespaces.SCTPTest, testclient.Client)
}

type VRFFixture struct {
}

func (p *VRFFixture) Setup() error {
	return nil
}

func (p *VRFFixture) Cleanup() error {
	return namespaces.Delete(vrf.TestNamespace, testclient.Client)
}

type OVSQOSFixture struct {
}

func (p *OVSQOSFixture) Setup() error {
	return nil
}

func (p *OVSQOSFixture) Cleanup() error {
	return namespaces.Delete(namespaces.OVSQOSTest, testclient.Client)
}

type FECFixture struct {
}

func (p *FECFixture) Setup() error {
	return nil
}

func (p *FECFixture) Cleanup() error {
	fec.Clean()
	return nil
}

type TuningcniFixture struct {
}

func (p *TuningcniFixture) Setup() error {
	return nil
}

func (p *TuningcniFixture) Cleanup() error {
	return namespaces.Delete(security.TestNamespace, testclient.Client)
}

type BondcniFixture struct {
}

func (p *BondcniFixture) Setup() error {
	return nil
}

func (p *BondcniFixture) Cleanup() error {
	return namespaces.Delete(namespaces.BondTestNamespace, testclient.Client)
}

type MultiNetworkPolicyFixture struct {
}

func (p *MultiNetworkPolicyFixture) Setup() error {
	return nil
}

func (p *MultiNetworkPolicyFixture) Cleanup() error {
	sriovClean.All()

	err := namespaces.Delete(testutils.MultiNetworkPolicyNamespaceX, testclient.Client)
	if err != nil {
		return err
	}

	err = namespaces.Delete(testutils.MultiNetworkPolicyNamespaceY, testclient.Client)
	if err != nil {
		return err
	}

	err = namespaces.Delete(testutils.MultiNetworkPolicyNamespaceZ, testclient.Client)
	if err != nil {
		return err
	}

	return nil
}
