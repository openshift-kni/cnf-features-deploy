package profile

import (
	"fmt"

	"github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
)

const (
	hugepagesSize2M = "2M"
	hugepagesSize1G = "1G"
)

func validationError(err string) error {
	return fmt.Errorf("validation error: %s", err)
}

// ValidateParameters validates parameters of the given profile
func ValidateParameters(profile *v1alpha1.PerformanceProfile) error {

	if profile.Spec.CPU == nil {
		return validationError("you should provide CPU section")
	}

	if profile.Spec.MachineConfigLabel != nil && len(profile.Spec.MachineConfigLabel) > 1 {
		return validationError("you should provide only 1 MachineConfigLabel")
	}

	if profile.Spec.MachineConfigPoolSelector != nil && len(profile.Spec.MachineConfigPoolSelector) > 1 {
		return validationError("you should provide only 1 MachineConfigPoolSelector")
	}

	if profile.Spec.NodeSelector == nil {
		return validationError("you should provide NodeSelector")
	}
	if len(profile.Spec.NodeSelector) > 1 {
		return validationError("you should provide ony 1 NodeSelector")
	}

	// in case MachineConfigLabels or MachineConfigPoolSelector are not set, we expect a certain format (domain/role)
	// on the NodeSelector in order to be able to calculate the default values for the former metioned fields.
	if profile.Spec.MachineConfigLabel == nil || profile.Spec.MachineConfigPoolSelector == nil {
		k, _ := components.GetFirstKeyAndValue(profile.Spec.NodeSelector)
		if _, _, err := components.SplitLabelKey(k); err != nil {
			return validationError("invalid NodeSelector label key, can't be split into domain/role")
		}
	}

	if profile.Spec.HugePages != nil {
		if err := validateHugepages(profile.Spec.HugePages); err != nil {
			return err
		}
	}

	if profile.Spec.NUMA != nil {
		if err := validateNUMA(profile.Spec.NUMA); err != nil {
			return err
		}
	}

	// TODO add validation for MachineConfigLabels and MachineConfigPoolSelector if they are not set
	// by checking if a MCP with our default values exists

	return nil
}

// GetMachineConfigPoolSelector returns the MachineConfigPoolSelector from the CR or a default value calculated based on NodeSelector
func GetMachineConfigPoolSelector(profile *v1alpha1.PerformanceProfile) map[string]string {
	if profile.Spec.MachineConfigPoolSelector != nil {
		return profile.Spec.MachineConfigPoolSelector
	}

	return getDefaultLabel(profile)
}

// GetMachineConfigLabel returns the MachineConfigLabels from the CR or a default value calculated based on NodeSelector
func GetMachineConfigLabel(profile *v1alpha1.PerformanceProfile) map[string]string {
	if profile.Spec.MachineConfigLabel != nil {
		return profile.Spec.MachineConfigLabel
	}

	return getDefaultLabel(profile)
}

func getDefaultLabel(profile *v1alpha1.PerformanceProfile) map[string]string {
	nodeSelectorKey, _ := components.GetFirstKeyAndValue(profile.Spec.NodeSelector)
	// no error handling needed, it's validated already
	_, nodeRole, _ := components.SplitLabelKey(nodeSelectorKey)

	labels := make(map[string]string)
	labels[components.MachineConfigRoleLabelKey] = nodeRole

	return labels
}

// IsPaused returns whether or not a performance profile's reconcile loop is paused
func IsPaused(profile *v1alpha1.PerformanceProfile) bool {

	if profile.Annotations == nil {
		return false
	}

	isPaused, ok := profile.Annotations[v1alpha1.PerformanceProfilePauseAnnotation]
	if ok && isPaused == "true" {
		return true
	}

	return false
}

func validateHugepages(hugepages *v1alpha1.HugePages) error {
	// validate that default hugepages size has correct value, currently we support only 2M and 1G(x86_64 architecture)
	if hugepages.DefaultHugePagesSize != nil {
		defaultSize := *hugepages.DefaultHugePagesSize
		if defaultSize != hugepagesSize1G && defaultSize != hugepagesSize2M {
			return validationError(fmt.Sprintf("hugepages default size should be equal to %q or %q", hugepagesSize1G, hugepagesSize2M))
		}
	}
	hugepagesSizes := map[v1alpha1.HugePageSize]string{}
	for _, page := range hugepages.Pages {
		hugepagesSizes[page.Size] = ""
	}

	// TODO: this validation should be removed, once https://github.com/kubernetes/kubernetes/pull/84051
	// is available under the openshift
	// validate that we do not have allocations of hugepages of different sizes
	if len(hugepagesSizes) > 1 {
		return validationError("allocation of hugepages with different sizes not supported")
	}
	return nil
}

func validateNUMA(numa *v1alpha1.NUMA) error {
	// validate NUMA topology policy matches allowed values
	if numa.TopologyPolicy != nil {
		policy := *numa.TopologyPolicy
		if policy != kubeletconfigv1beta1.NoneTopologyManagerPolicy &&
			policy != kubeletconfigv1beta1.BestEffortTopologyManagerPolicy &&
			policy != kubeletconfigv1beta1.RestrictedTopologyManagerPolicy &&
			policy != kubeletconfigv1beta1.SingleNumaNodeTopologyManager {
			return validationError("unrecognized value for topologyPolicy")
		}
	}
	return nil
}
