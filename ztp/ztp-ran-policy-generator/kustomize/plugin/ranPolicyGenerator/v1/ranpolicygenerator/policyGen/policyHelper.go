package policyGen

import (
	utils "github.com/serngawy/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/utils"
)

func CreateAcmPolicy(name string, namespace string) utils.AcmPolicy {
	policy := utils.AcmPolicy{}
	policy.ApiVersion = "policy.open-cluster-management.io/v1"
	policy.Kind = "Policy"
	policy.Metadata.Name = name
	annotations := make(map[string]string, 3)
	annotations["policy.open-cluster-management.io/standards"] = "NIST SP 800-53"
	annotations["policy.open-cluster-management.io/categories"] = "CM Configuration Management"
	annotations["policy.open-cluster-management.io/controls"] = "CM-2 Baseline Configuration"
	policy.Metadata.Annotations = annotations
	policy.Metadata.Namespace = namespace
	policy.Spec.Disabled = false
	policy.Spec.RemediationAction = "enforce"

	return policy
}

func CreateAcmConfigPolicy(name string) utils.AcmConfigurationPolicy {
	configPolicy := utils.AcmConfigurationPolicy{}
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

func CreateObjTemplates(objDef map[string]interface{}) utils.ObjectTemplates {
	objTemp := utils.ObjectTemplates{}
	objTemp.ComplianceType = "musthave"
	objTemp.ObjectDefinition = objDef

	return objTemp
}

func CreatePlacementBinding(name string, namespace string, ruleName string, subjects []utils.Subject) utils.PlacementBinding {
	placementBinding := utils.PlacementBinding{}
	placementBinding.ApiVersion = "policy.open-cluster-management.io/v1"
	placementBinding.Kind = "PlacementBinding"
	placementBinding.Metadata.Name = name + "-placementbinding"
	placementBinding.Metadata.Namespace = namespace
	placementBinding.PlacementRef.Name = ruleName
	placementBinding.PlacementRef.Kind = "PlacementRule"
	placementBinding.PlacementRef.ApiGroup = "apps.open-cluster-management.io"
	placementBinding.Subjects = subjects

	return placementBinding
}

func CreatePolicySubject(policyName string) utils.Subject {
	subject := utils.Subject{}
	subject.ApiGroup = "policy.open-cluster-management.io"
	subject.Kind = "Policy"
	subject.Name = policyName

	return subject
}

func CreatePlacementRule(name string, namespace string, matchKey string, matchOper string, matchValue string) utils.PlacementRule {
	placmentRule := utils.PlacementRule{}
	placmentRule.ApiVersion = "apps.open-cluster-management.io/v1"
	placmentRule.Kind = "PlacementRule"
	placmentRule.Metadata.Name = name + "-placementrule"
	placmentRule.Metadata.Namespace = namespace
	expressions := make(map[string]string)
	if matchOper == utils.ExistOper {
		expressions["key"] = matchKey
		expressions["operator"] = matchOper
	} else {
		expressions["key"] = matchKey
		expressions["operator"] = matchOper
		expressions["values"] = matchValue
	}
	placmentRule.Spec.ClusterSelector.MatchExpressions = expressions

	return placmentRule
}
