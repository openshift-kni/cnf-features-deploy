package policyGen

import (
	"errors"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	"strings"
)

func CreateAcmPolicy(name string, namespace string, policyObjDefArr []utils.PolicyObjectDefinition) utils.AcmPolicy {
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
	policy.Spec.PolicyTemplates = policyObjDefArr

	return policy
}

func CreateAcmConfigPolicy(name string, objTempArr []utils.ObjectTemplates) utils.AcmConfigurationPolicy {
	configPolicy := utils.AcmConfigurationPolicy{}
	configPolicy.ApiVersion = "policy.open-cluster-management.io/v1"
	configPolicy.Kind = "ConfigurationPolicy"
	configPolicy.Metadata.Name = name + "-config"
	configPolicy.Spec.RemediationAction = "enforce"
	configPolicy.Spec.Severity = "low"
	exclude := make([]string, 1)
	exclude[0] = "kube-*"
	configPolicy.Spec.NamespaceSelector.Exclude = exclude
	include := make([]string, 1)
	include[0] = "*"
	configPolicy.Spec.NamespaceSelector.Include = include
	configPolicy.Spec.ObjectTemplates = objTempArr

	return configPolicy
}

func CreateObjTemplates(objDef map[string]interface{}) utils.ObjectTemplates {
	objTemp := utils.ObjectTemplates{}
	// Using mustonlyhave compliance type to ensures the object in GIT exactly matches what is enforced on the cluster.
	objTemp.ComplianceType = "mustonlyhave"
	objTemp.ObjectDefinition = objDef

	return objTemp
}

func CreatePolicyObjectDefinition(acmConfigPolicy utils.AcmConfigurationPolicy) utils.PolicyObjectDefinition {
	policyObjDef := utils.PolicyObjectDefinition{}
	policyObjDef.ObjDef = acmConfigPolicy

	return policyObjDef
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
	expression := make(map[string]interface{})
	expression["key"] = matchKey
	expression["operator"] = matchOper
	if matchOper != utils.ExistOper {
		expression["values"] = strings.Split(matchValue, ",")
	}
	expressions := make([]map[string]interface{}, 1)
	expressions[0] = expression
	placmentRule.Spec.ClusterSelector.MatchExpressions = expressions

	return placmentRule
}

func CheckNameLength(namespace string, name string) error {
	// the policy (namespace.name + name) must not exceed 63 chars based on ACM documentation.
	if len(namespace+"."+name) > 63 {
		err := errors.New("Namespace.Name + ResourceName is exceeding the 63 chars limit: " + namespace + "." + name)
		return err
	}
	return nil
}
