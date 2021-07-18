package empty_string

import (
	"context"
	//	"encoding/json"
	//	"fmt"
	//	"log"
	//	"os"
	//	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	//	corev1 "k8s.io/api/core/v1"
	//	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	//	"k8s.io/apimachinery/pkg/api/errors"
	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	//	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	clientmachineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	//	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	//	utilNodes "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	//	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/sriov"
	//	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
)

var (
	machineConfigPoolNodeSelector string
)

func init() {
}

var _ = Describe("validation", func() {
	Context("general", func() {
		It("should report all machine config pools are in ready status", func() {
			mcp := &clientmachineconfigv1.MachineConfigPoolList{}
			err := testclient.Client.List(context.TODO(), mcp)
			Expect(err).ToNot(HaveOccurred())

			for _, mcItem := range mcp.Items {
				Expect(mcItem.Status.MachineCount).To(Equal(mcItem.Status.ReadyMachineCount))
			}
		})

	})
})
