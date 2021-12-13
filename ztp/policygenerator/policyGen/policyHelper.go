package policyGen

import (
	"errors"
	"strings"

	utils "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"
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

// Create a new ObjectTemplate for the given resource with values
// populated from the reference ACM template (pulled from
// policyBuilder)
func BuildObjectTemplate(resource generatedCR) utils.ObjectTemplates {
	objTemplate := utils.ObjectTemplates{}

	// BZ 2009233 Namespaces, Subscriptions and OperatorGroups will be updated by OLM
	// with labels and annotations. A "mustonlyhave" ACM policy will fight with OLM
	// over these annotations/labels. Allow the user to set the compliance type to
	// avoid this condition. The most specific complianceType setting given by the
	// user will take precedence. Default to musthave so that we realize the CPU
	// reductions unless explicitly told otherwise
	complianceType := resource.globalComplianceType
	if resource.pgtSourceFile.ComplianceType != utils.UnsetStringValue {
		complianceType = resource.pgtSourceFile.ComplianceType
	}
	objTemplate.ComplianceType = complianceType
	objTemplate.ObjectDefinition = resource.builtCR

	return objTemplate
}

// We are using Deploywaves to order policies deployment.
// Each resource needs to be applied via ACM enforce policy controlled
// by ClusterGroupUpgrade operator should have a Deploywave annotation.
// For example,
//   metadata:
//     annotations:
//       "ran.openshift.io/ztp-deploy-wave": "1"
// Resources with same waves can be applied simultaneously in one
// policy, otherwise, they should be applied via separated policies
// in order.
func SetPolicyDeployWave(policyMeta utils.MetaData, resource generatedCR, isFirstOjbTemp bool) error {
	crMetadata, _ := resource.builtCR["metadata"].(map[string]interface{})
	crAnnotations, _ := crMetadata["annotations"].(map[string]interface{})
	crDeployWave, foundCrDeployWave := crAnnotations[utils.ZtpDeployWaveAnnotation].(string)

	if isFirstOjbTemp {
		// first resource added in policy, assign its wave to policy if exists
		if foundCrDeployWave {
			policyMeta.Annotations[utils.ZtpDeployWaveAnnotation] = crDeployWave
		}
	} else {
		// subsequent resource added in policy
		if policyDeployWave, foundPolicyWave := policyMeta.Annotations[utils.ZtpDeployWaveAnnotation]; foundPolicyWave {
			if foundCrDeployWave && (policyDeployWave == crDeployWave) {
				// policy wave is same with the resource wave
				return nil
			}
		} else if !foundCrDeployWave {
			// no wave for policy and resource
			return nil
		}

		// resource wave doesn't match with policy wave
		// resource has no wave but policy has or vice versa
		return errors.New(utils.ZtpDeployWaveAnnotation + " annotation in Resource " +
			resource.pgtSourceFile.FileName + " doesn't match with Policy " + policyMeta.Name)
	}
	return nil
}
