package utils


func CreateAcmPolicy(name string, namespace string) AcmPolicy {
	policy := AcmPolicy{}
	policy.ApiVersion = "policy.open-cluster-management.io/v1"
	policy.Kind = "Policy"
	policy.Metadata.Name = name
	annotations := make([]string, 3)
	annotations[0] = "policy.open-cluster-management.io/standards: NIST SP 800-53"
	annotations[1] = "policy.open-cluster-management.io/categories: CM Configuration Management"
	annotations[2] = "policy.open-cluster-management.io/controls: CM-2 Baseline Configuration"
	policy.Metadata.Annotations = annotations
	policy.Metadata.Namespace = namespace
	policy.Spec.Disabled = false
	policy.Spec.RemediationAction = "enforce"

	return policy
}

func CreateAcmConfigPolicy(name string) AcmConfigurationPolicy {
	configPolicy := AcmConfigurationPolicy{}
	configPolicy.ApiVersion = "policy.open-cluster-management.io/v1"
	configPolicy.Kind = "ConfigurationPolicy"
	configPolicy.Metadata.Name = name + "-policy-config"
	configPolicy.Spec.RemediationAction = "enforce"
	configPolicy.Spec.Severity = "low"
	exclude := make([]string, 1)
	exclude[0] = "kube-*"
	configPolicy.Spec.NamespaceSelector.Exclude = exclude
	include := make([]string, 1)
	include[0] = "*"
	configPolicy.Spec.NamespaceSelector.Include = include

	return configPolicy
}

func CreateObjTemplates(objDef map[string]interface{}) ObjectTemplates {
	objTemp := ObjectTemplates{}
	objTemp.ComplianceType = "musthave"
	objTemp.ObjectDefinition = objDef

	return objTemp
}
