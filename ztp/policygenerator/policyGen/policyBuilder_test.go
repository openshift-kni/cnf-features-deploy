package policyGen

import (
	"testing"

	utils "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const defaultComplianceType = utils.DefaultComplianceType

func extractCRsFromPolicies(t *testing.T, policies map[string]interface{}) []utils.ObjectTemplates {
	// The policies map contains entries such as:
	// test1/test1-gen-sub-policy This is the one we want
	// test1/test1-placementrules
	// test1/test1-placementbinding
	assert.Equal(t, len(policies), 3, "Expect a single policy with placement rule/binding for testing")
	for _, value := range policies {
		// This is the configuration policy
		policy, ok := value.(utils.AcmPolicy)
		if !ok {
			continue
		}
		// This is the policy-templates array
		assert.Equal(t, len(policy.Spec.PolicyTemplates), 1)
		// Extract the object-template from the object-definitions. The
		// object-template contains the actual CR
		objects := policy.Spec.PolicyTemplates[0].ObjDef.Spec.ObjectTemplates
		// Return the first (and only) non-placement entry
		return objects
	}
	return nil
}

// Test cases for override of complianceType for Namespace kinds. Namespace as the first object here.
func TestComplianceTypeDefault(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 3)

	assert.Equal(t, objects[0].ComplianceType, defaultComplianceType)
	assert.Equal(t, objects[0].ObjectDefinition["kind"], "Namespace")

	assert.Equal(t, objects[1].ComplianceType, defaultComplianceType)
	assert.Equal(t, objects[1].ObjectDefinition["kind"], "Subscription")

	assert.Equal(t, objects[2].ComplianceType, defaultComplianceType)
	assert.Equal(t, objects[2].ObjectDefinition["kind"], "OperatorGroup")
}

// Test cases for override of complianceType for Namespace kinds
func TestNamespaceCompliance(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
      complianceType: "musthave"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 4)

	assert.Equal(t, objects[0].ComplianceType, defaultComplianceType)
	assert.Equal(t, objects[0].ObjectDefinition["kind"], "Subscription")

	assert.Equal(t, objects[1].ComplianceType, "musthave")
	assert.Equal(t, objects[1].ObjectDefinition["kind"], "Namespace")

	assert.Equal(t, objects[2].ComplianceType, defaultComplianceType)
	assert.Equal(t, objects[2].ObjectDefinition["kind"], "OperatorGroup")

	// We only override the first one
	assert.Equal(t, objects[3].ComplianceType, defaultComplianceType)
	assert.Equal(t, objects[3].ObjectDefinition["kind"], "Namespace")
}

// Test cases for override of complianceType for Namespace kinds
func TestNamespaceComplianceMultiple(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
      complianceType: "musthave"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
      complianceType: "musthave"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
      complianceType: "musthave"
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
      complianceType: "mustonlyhave"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 4)

	assert.Equal(t, objects[0].ComplianceType, "musthave")
	assert.Equal(t, objects[0].ObjectDefinition["kind"], "Namespace")

	assert.Equal(t, objects[1].ComplianceType, "musthave")
	assert.Equal(t, objects[1].ObjectDefinition["kind"], "Subscription")

	assert.Equal(t, objects[2].ComplianceType, "musthave")
	assert.Equal(t, objects[2].ObjectDefinition["kind"], "OperatorGroup")

	assert.Equal(t, objects[3].ComplianceType, "mustonlyhave")
	assert.Equal(t, objects[3].ObjectDefinition["kind"], "Namespace")
}

// Test cases for override of complianceType for Namespace kinds. Namespace as the first object here.
func TestComplianceTypeGlobal(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  complianceType: mustonlyhave
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
      complianceType: mustonlyhave
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
      complianceType: musthave
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 3)

	assert.Equal(t, objects[0].ComplianceType, "mustonlyhave")
	assert.Equal(t, objects[0].ObjectDefinition["kind"], "Namespace")

	assert.Equal(t, objects[1].ComplianceType, "musthave")
	assert.Equal(t, objects[1].ObjectDefinition["kind"], "Subscription")

	assert.Equal(t, objects[2].ComplianceType, "mustonlyhave")
	assert.Equal(t, objects[2].ObjectDefinition["kind"], "OperatorGroup")

	// Switch the global value and check again
	pgt.Spec.ComplianceType = "musthave"
	policies, err = pBuilder.Build(pgt)

	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")

	objects = extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 3)

	assert.Equal(t, objects[0].ComplianceType, "mustonlyhave")
	assert.Equal(t, objects[0].ObjectDefinition["kind"], "Namespace")

	assert.Equal(t, objects[1].ComplianceType, "musthave")
	assert.Equal(t, objects[1].ObjectDefinition["kind"], "Subscription")

	assert.Equal(t, objects[2].ComplianceType, "musthave")
	assert.Equal(t, objects[2].ObjectDefinition["kind"], "OperatorGroup")

}

func TestNamespaceRemediationActionDefault(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")
	policy := policies["test1/test1-gen-sub-policy"].(utils.AcmPolicy)
	assert.Equal(t, policy.Spec.RemediationAction, "inform")
	assert.Equal(t, policy.Spec.PolicyTemplates[0].ObjDef.Spec.RemediationAction, "inform")
}

func TestNamespaceRemediationActionPGTLevel(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  remediationAction: "enforce"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")
	policy := policies["test1/test1-gen-sub-policy"].(utils.AcmPolicy)
	assert.Equal(t, policy.Spec.RemediationAction, "enforce")
	assert.Equal(t, policy.Spec.PolicyTemplates[0].ObjDef.Spec.RemediationAction, "enforce")
}

func TestNamespaceRemediationActionOverride(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  remediationAction: "inform"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
      remediationAction: "enforce"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
      remediationAction: "enforce"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
      remediationAction: "enforce"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")
	policy := policies["test1/test1-gen-sub-policy"].(utils.AcmPolicy)
	assert.Equal(t, policy.Spec.RemediationAction, "enforce")
	assert.Equal(t, policy.Spec.PolicyTemplates[0].ObjDef.Spec.RemediationAction, "enforce")
}

func TestNamespaceRemediationActionConflict(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  remediationAction: "enforce"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
      remediationAction: "inform"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
      remediationAction: "enforce"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "remediationAction conflict for policyName")
	assert.NotNil(t, policies)
}

func TestNamespaceRemediationActionOverrideOnce(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  remediationAction: "inform"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
      remediationAction: "enforce"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")
	policy := policies["test1/test1-gen-sub-policy"].(utils.AcmPolicy)
	assert.Equal(t, policy.Spec.RemediationAction, "enforce")
	assert.Equal(t, policy.Spec.PolicyTemplates[0].ObjDef.Spec.RemediationAction, "enforce")
}

func TestPolicyZtpDeployWaveAnnotation(t *testing.T) {
	// sources have same wave
	input1 := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input1), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")
	policy := policies["test1/test1-gen-sub-policy"].(utils.AcmPolicy)
	assert.Equal(t, policy.Metadata.Annotations[utils.ZtpDeployWaveAnnotation], "1")

	// overwrite a mismatched wave in the source
	input2 := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test2"
  namespace: "test2"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy"
    - fileName: GenericConfig.yaml
      policyName: "gen-policy"
      metadata:
        annotations:
          ran.openshift.io/ztp-deploy-wave: "1"
`
	// Read in the test PGT
	pgt = utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input2), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler = utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder = NewPolicyBuilder(fHandler)
	policies, err = pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test2/test2-gen-policy")
	policy = policies["test2/test2-gen-policy"].(utils.AcmPolicy)
	assert.Equal(t, policy.Metadata.Annotations[utils.ZtpDeployWaveAnnotation], "1")
}

func TestPolicyZtpDeployWaveAnnotationWithMultipleWaves(t *testing.T) {
	// one source has different wave with others
	input1 := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy"
    - fileName: GenericConfig.yaml
      policyName: "gen-policy"
`

	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input1), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "doesn't match with Policy")
	assert.NotNil(t, policies)

	// one source doesn't have wave but others have
	input2 := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test2"
  namespace: "test2"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy"
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy"
`
	// Read in the test PGT
	pgt = utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input2), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler = utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder = NewPolicyBuilder(fHandler)
	policies, err = pBuilder.Build(pgt)

	// Validate the run
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "doesn't match with Policy")
	assert.NotNil(t, policies)

	// overwrite a wave to be different with others
	input3 := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test3"
  namespace: "test3"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy"
      metadata:
        annotations:
          ran.openshift.io/ztp-deploy-wave: "100"
`
	// Read in the test PGT
	pgt = utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input3), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler = utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

	// Run the PGT through the generator
	pBuilder = NewPolicyBuilder(fHandler)
	policies, err = pBuilder.Build(pgt)

	// Validate the run
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "doesn't match with Policy")
	assert.NotNil(t, policies)
}
