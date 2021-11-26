package policyGen

import (
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func extractCRsFromPolicies(t *testing.T, policies map[string]interface{}) []utils.ObjectTemplates {
	// The policies map contains entries such as:
	// test1/test1-gen-sub-policy This is the one we want
	// test1/test1-placementrules
	// test1/test1-placementbinding
	assert.Equal(t, len(policies), 3, "Expect a single policy with placement rule/binding for testing")
	for key, value := range policies {
		if strings.Contains(key, "placement") {
			continue
		}
		// This is the configuration policy
		policy := value.(utils.AcmPolicy)
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
	fHandler.SetResourceBaseDir("..")

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

	assert.Equal(t, objects[1].ComplianceType, "mustonlyhave")
	assert.Equal(t, objects[1].ObjectDefinition["kind"], "Subscription")

	assert.Equal(t, objects[2].ComplianceType, "mustonlyhave")
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
	fHandler.SetResourceBaseDir("..")

	// Run the PGT through the generator
	pBuilder := NewPolicyBuilder(fHandler)
	policies, err := pBuilder.Build(pgt)

	// Validate the run
	assert.Nil(t, err)
	assert.NotNil(t, policies)

	assert.Contains(t, policies, "test1/test1-gen-sub-policy")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 4)

	assert.Equal(t, objects[0].ComplianceType, "mustonlyhave")
	assert.Equal(t, objects[0].ObjectDefinition["kind"], "Subscription")

	assert.Equal(t, objects[1].ComplianceType, "musthave")
	assert.Equal(t, objects[1].ObjectDefinition["kind"], "Namespace")

	assert.Equal(t, objects[2].ComplianceType, "mustonlyhave")
	assert.Equal(t, objects[2].ObjectDefinition["kind"], "OperatorGroup")

	// We only override the first one
	assert.Equal(t, objects[3].ComplianceType, "mustonlyhave")
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
	fHandler.SetResourceBaseDir("..")

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
	fHandler.SetResourceBaseDir("..")

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
  remediationAction: "inform"
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
	fHandler.SetResourceBaseDir("..")

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
  remediationAction: "enforce"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
      remediationAction: "inform"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
      remediationAction: "inform"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
      remediationAction: "inform"
`
	// Read in the test PGT
	pgt := utils.PolicyGenTemplate{}
	_ = yaml.Unmarshal([]byte(input), &pgt)

	// Set up the files handler to pick up local source-crs and skip any output
	fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")
	fHandler.SetResourceBaseDir("..")

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
	fHandler.SetResourceBaseDir("..")

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
	fHandler.SetResourceBaseDir("..")

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
