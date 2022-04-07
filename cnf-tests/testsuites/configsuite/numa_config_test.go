//go:build !unittests
// +build !unittests

package setup_test

import (
	. "github.com/onsi/ginkgo"

	numaserialconf "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
)

var _ = Describe("[config] numaresources cluster configuration", func() {

	It("[numaresources] Should successfully deploy the infra", func() {
		numaserialconf.Setup()
		// we must leave the cluster in configured state, so we must NOT call Teardown
	})
})
