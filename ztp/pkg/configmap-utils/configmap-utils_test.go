package cmutils_test

import (
	"testing"

	cu "github.com/openshift-kni/cnf-features-deploy/ztp/pkg/configmap-utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestWrapObjects(t *testing.T) {
	ob := []unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"kind": "test",
			},
		},
	}
	cm, err := cu.WrapObjects(ob, "test-cm", "test-ns")
	if err != nil {
		t.Errorf("failed to wrap test object in configmap")
	}
	if len(cm.Data) != 1 {
		t.Errorf("wrong number of data elemonts")
	}
	if cm.Name != "test-cm" {
		t.Errorf("wrong configmap name")
	}
	if cm.Namespace != "test-ns" {
		t.Errorf("wrong configmap namespace")
	}

}
