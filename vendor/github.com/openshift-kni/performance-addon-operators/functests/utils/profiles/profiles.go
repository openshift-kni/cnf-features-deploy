package profiles

import (
	"context"
	"fmt"
	"reflect"

	. "github.com/onsi/gomega"

	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	v1 "github.com/openshift/custom-resource-status/conditions/v1"
)

// GetByNodeLabels gets the performance profile that must have node selector equals to passed node labels
func GetByNodeLabels(nodeLabels map[string]string) (*performancev1alpha1.PerformanceProfile, error) {
	profiles := &performancev1alpha1.PerformanceProfileList{}
	if err := testclient.Client.List(context.TODO(), profiles); err != nil {
		return nil, err
	}

	var result *performancev1alpha1.PerformanceProfile
	for _, profile := range profiles.Items {
		if reflect.DeepEqual(profile.Spec.NodeSelector, nodeLabels) {
			if result != nil {
				return nil, fmt.Errorf("found more than one performance profile with specified node selector %v", nodeLabels)
			}
			result = &profile
		}
	}

	if result == nil {
		return nil, fmt.Errorf("failed to find performance profile with specified node selector %v", nodeLabels)
	}

	return result, nil
}

// GetConditionMessage gets the performance profile message for the given type
func GetConditionMessage(nodeLabels map[string]string, conditionType v1.ConditionType) string {
	profile, err := GetByNodeLabels(nodeLabels)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed getting profile by nodelabel")
	for _, condition := range profile.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Message
		}
	}
	return ""
}
