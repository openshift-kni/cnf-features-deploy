package clean

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/openshift-kni/cnf-features-deploy/functests/ptp"
	"github.com/openshift-kni/cnf-features-deploy/functests/sctp"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils"
	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// All cleans all the resources created by the local test suite.
// This includes sctp pods / namespaces / policies, ptp configurations
func All() error {
	err := namespaces.Clean("default", "testsctp-", testclient.Client)
	if err != nil {
		return fmt.Errorf("Failed to clean default namespace")
	}

	nn := []string{utils.NamespaceTesting,
		namespaces.DpdkTest,
		sctp.TestNamespace,
	}

	for _, n := range nn {
		err := testclient.Client.Namespaces().Delete(context.Background(), n, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("Failed to delete namespace %s", n)
		}
		err = namespaces.WaitForDeletion(testclient.Client, n, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("Failed to wait namespace %s deletion", n)
		}
	}

	ptp.Clean()
	return nil
}
