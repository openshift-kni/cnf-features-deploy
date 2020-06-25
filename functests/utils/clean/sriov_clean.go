package clean

import (
	"context"

	sriovv1 "github.com/openshift/sriov-network-operator/pkg/apis/sriovnetwork/v1"
	sriovclient "github.com/openshift/sriov-network-operator/test/util/client"
	sriovNamespaces "github.com/openshift/sriov-network-operator/test/util/namespaces"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SriovResources cleans any dangling sriov resources created by the sriov tests
// TODO in 4.6: this would be better to be a public function exporter by the SR-IOV test suite.
func SriovResources() error {
	clients := sriovclient.New("", func(scheme *runtime.Scheme) {
		sriovv1.AddToScheme(scheme)
	})

	// TODO This is a temporary workaround to check if the sriov tests were actually deployed.
	// This is going to be removed with the new cleaning logic
	_, err := clients.Namespaces().Get(context.Background(), sriovNamespaces.Test, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}

	err = sriovNamespaces.Clean("openshift-sriov-network-operator", sriovNamespaces.Test, clients)
	return err
}
