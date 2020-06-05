package clean

import (
	sriovv1 "github.com/openshift/sriov-network-operator/pkg/apis/sriovnetwork/v1"
	sriovclient "github.com/openshift/sriov-network-operator/test/util/client"
	sriovNamespaces "github.com/openshift/sriov-network-operator/test/util/namespaces"
	"k8s.io/apimachinery/pkg/runtime"
)

// SriovResources cleans any dangling sriov resources created by the sriov tests
// TODO in 4.6: this would be better to be a public function exporter by the SR-IOV test suite.
func SriovResources() error {
	clients := sriovclient.New("", func(scheme *runtime.Scheme) {
		sriovv1.AddToScheme(scheme)
	})
	err := sriovNamespaces.Clean("openshift-sriov-network-operator", sriovNamespaces.Test, clients)
	return err
}
