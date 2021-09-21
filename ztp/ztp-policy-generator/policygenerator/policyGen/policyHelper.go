package policyGen

import (
	"errors"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	"strings"
)

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

func CreatePlacementRule(name string, namespace string, matchKeyValue map[string]string) utils.PlacementRule {
	placementRule := utils.PlacementRule{}
	placementRule.ApiVersion = "apps.open-cluster-management.io/v1"
	placementRule.Kind = "PlacementRule"
	placementRule.Metadata.Name = name + "-placementrules"
	placementRule.Metadata.Namespace = namespace
	expressions := make([]map[string]interface{}, 0)

	for key, value := range matchKeyValue {
		expression := make(map[string]interface{})
		expression["key"] = key
		expression["operator"] = utils.InOper
		expression["values"] = strings.Split(value, ",")
		expressions = append(expressions, expression)
	}

	placementRule.Spec.ClusterSelector.MatchExpressions = expressions

	return placementRule
}

func CheckNameLength(namespace string, name string) error {
	// the policy (namespace.name + name) must not exceed 63 chars based on ACM documentation.
	if len(namespace+"."+name) > 63 {
		err := errors.New("Namespace.Name + ResourceName is exceeding the 63 chars limit: " + namespace + "." + name)
		return err
	}
	return nil
}
