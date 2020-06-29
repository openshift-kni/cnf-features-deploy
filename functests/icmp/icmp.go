<<<<<<< HEAD
/*
Package icmp is not implemented, and purely a demo stub!  This stub test suite shows how one might run a custom test in
the "cnf-features-deploy" code base.

To Build:
docker build --no-cache -f cnf-tests/Dockerfile -t quay.io/rgoulding/cnf-tests .

To Push the image to quay.io:
docker push quay.io/rgoulding/cnf-tests

To Run:
docker pull quay.io/rgoulding/cnf-tests
docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/rgoulding/cnf-tests /usr/bin/cnftests -ginkgo.v -ginkgo.focus="icmp"
*/

package icmp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
)

var _ = Describe("icmp", func() {

	execute.BeforeAll(func() {

	})

	Context("Validate ICMP to Google", func() {
		It("Should forward and receive packets", func() {
			// should always fail.
			Expect("nil").ToNot(BeNil())
		})
	})

	var _ = Describe("Test ICMP", func() {

	})
})
=======
package icmp

>>>>>>> 67c9d2b... Test
