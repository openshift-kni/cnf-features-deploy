// +build !unittests

package test_test

import (
	"flag"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/cnf-features-deploy/functests/ptp"  // this is needed otherwise the ptp test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/functests/sctp" // this is needed otherwise the sctp test won't be executed
	testutils "github.com/openshift-kni/cnf-features-deploy/functests/utils"
	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var junitPath *string

func init() {
	junitPath = flag.String("junit", "junit.xml", "the path for the junit format report")
}

func TestTest(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if junitPath != nil {
		rr = append(rr, reporters.NewJUnitReporter(*junitPath))
	}
	RunSpecsWithDefaultAndCustomReporters(t, "CNF Features e2e integratoin tests", rr)
}

var _ = BeforeSuite(func() {
	// create test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testutils.NamespaceTesting,
		},
	}
	_, err := testclient.Client.Namespaces().Create(ns)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := testclient.Client.Namespaces().Delete(testutils.NamespaceTesting, &metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
	err = namespaces.WaitForDeletion(testclient.Client, testutils.NamespaceTesting, 5*time.Minute)
})
