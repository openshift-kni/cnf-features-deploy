package policyGen

import (
	"fmt"
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
	tests := []struct {
		input        string
		expectedWave map[string]string
	}{{
		// single policy with wave
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericConfig.yaml
      policyName: "single-policy"
`,
		expectedWave: map[string]string{
			"test/test-single-policy": "2",
		},
	}, {
		// single policy with no wave
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "single-policy"
`,
		expectedWave: map[string]string{
			"test/test-single-policy": "",
		},
	}, {
		// single policy with no wave
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "single-policy"
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "single-policy"
      metadata:
        name: instance2
        namespace: openshift-generic
`,
		expectedWave: map[string]string{
			"test/test-single-policy": "",
		},
	}, {
		// single policy with overridden wave
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "single-policy"
      metadata:
        annotations:
          ran.openshift.io/ztp-deploy-wave: "99"
`,
		expectedWave: map[string]string{
			"test/test-single-policy": "99",
		},
	}, {
		// multiple sources with the same wave
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
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
`,
		expectedWave: map[string]string{
			"test/test-gen-policy": "1",
		},
	}, {
		// multiple sources set to the same wave
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
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
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy"
      metadata:
        annotations:
          ran.openshift.io/ztp-deploy-wave: "1"
`,
		expectedWave: map[string]string{
			"test/test-gen-policy": "1",
		},
	}, {
		// multiple policies with different waves
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericNamespace.yaml
      policyName: "gen-policy-1"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy-1"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy-1"
    - fileName: GenericConfig.yaml
      policyName: "gen-policy-2"
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy-none"
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy-99"
      metadata:
        annotations:
          ran.openshift.io/ztp-deploy-wave: "99"
`,
		expectedWave: map[string]string{
			"test/test-gen-policy-1":    "1",
			"test/test-gen-policy-2":    "2",
			"test/test-gen-policy-none": "",
			"test/test-gen-policy-99":   "99",
		},
	}, {
		// one source doesn't have wave but others have
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy"
    - fileName: GenericNamespace.yaml
      policyName: "gen-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy"
`,
		expectedWave: map[string]string{
			"test/test-gen-policy": "1",
		},
	}, {
		// one source doesn't have wave but others have
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
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
`,
		expectedWave: map[string]string{
			"test/test-gen-policy": "1",
		},
	}, {
		// two source don't have wave but others have
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-policy"
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy"
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy"
      metadata:
        name: instance2
        namespace: openshift-generic
`,
		expectedWave: map[string]string{
			"test/test-gen-policy": "1",
		},
	}}

	for _, test := range tests {
		// Read in the test PGT
		pgt := utils.PolicyGenTemplate{}
		err := yaml.Unmarshal([]byte(test.input), &pgt)
		assert.NoError(t, err)

		// Set up the files handler to pick up local source-crs and skip any output
		fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

		// Run the PGT through the generator
		pBuilder := NewPolicyBuilder(fHandler)
		policies, err := pBuilder.Build(pgt)

		// Validate the run
		assert.NoError(t, err)
		assert.NotNil(t, policies)
		for policyName, expectedWave := range test.expectedWave {
			policy, found := policies[policyName].(utils.AcmPolicy)
			assert.True(t, found)
			wave, waveIsSet := policy.Metadata.Annotations[utils.ZtpDeployWaveAnnotation]
			if expectedWave == "" {
				assert.False(t, waveIsSet)
			} else {
				assert.Equal(t, wave, expectedWave)
			}

			// verify the wave has been removed from the built CRs wrapped in the policy
			objects := policy.Spec.PolicyTemplates[0].ObjDef.Spec.ObjectTemplates
			for _, obj := range objects {
				metadata, _ := obj.ObjectDefinition["metadata"].(map[string]interface{})
				annotations, ok := metadata["annotations"].(map[string]interface{})
				if ok {
					assert.NotContains(t, annotations, utils.ZtpDeployWaveAnnotation)
				}
			}
		}
	}
}

func TestPolicyZtpDeployWaveAnnotationWithMismatchedWaves(t *testing.T) {
	tests := []struct {
		input       string
		policyWave  string
		problemWave string
	}{{
		// one source has different wave with others
		input: `
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
`,
		policyWave:  "1",
		problemWave: "2",
	}, {
		// one source has different wave with others
		input: `
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
    - fileName: GenericConfigWithoutWave.yaml
      policyName: "gen-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-policy"
    - fileName: GenericConfig.yaml
      policyName: "gen-policy"
`,
		policyWave:  "1",
		problemWave: "2",
	}, {
		// overwrite a wave to be different with others
		input: `
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
`,
		policyWave:  "1",
		problemWave: "100",
	}}

	for _, test := range tests {
		// Read in the test PGT
		pgt := utils.PolicyGenTemplate{}
		err := yaml.Unmarshal([]byte(test.input), &pgt)
		assert.NoError(t, err)

		// Set up the files handler to pick up local source-crs and skip any output
		fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

		// Run the PGT through the generator
		pBuilder := NewPolicyBuilder(fHandler)
		policies, err := pBuilder.Build(pgt)

		// Validate the run
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "doesn't match with Policy")
		assert.Contains(t, err.Error(), fmt.Sprintf("(wave %s)", test.policyWave))
		assert.Contains(t, err.Error(), fmt.Sprintf("(wave %s)", test.problemWave))
		assert.NotNil(t, policies)
	}
}

func TestBindingRules(t *testing.T) {
	testcases := []struct {
		input    string
		expected []map[string]interface{}
	}{{
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    labelKey1: ""
    labelKey2: "labelValue2"
  bindingExcludedRules:
    labelKey3: "labelValue3"
    labelKey4: ""
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
`,
		expected: []map[string]interface{}{
			{
				"key":      "labelKey1",
				"operator": "Exists",
			},
			{
				"key":      "labelKey2",
				"operator": "In",
				"values":   []string{"labelValue2"},
			},
			{
				"key":      "labelKey3",
				"operator": "NotIn",
				"values":   []string{"labelValue3"},
			},
			{
				"key":      "labelKey4",
				"operator": "DoesNotExist",
			},
		},
	}, {
		input: `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    labelKey1: ""
  bindingExcludedRules:
    labelKey1: "labelValue1"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
`,
		expected: []map[string]interface{}{
			{
				"key":      "labelKey1",
				"operator": "Exists",
			},
			{
				"key":      "labelKey1",
				"operator": "NotIn",
				"values":   []string{"labelValue1"},
			},
		},
	}}
	for _, tc := range testcases {

		policies, _ := buildTest(t, tc.input)
		assert.Contains(t, policies, "test/test-placementrules")

		placementRule := policies["test/test-placementrules"].(utils.PlacementRule)
		assert.ElementsMatch(t, placementRule.Spec.ClusterSelector.MatchExpressions, tc.expected)
	}
}

func TestBindingRulesWithIncludedClustersOnly(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    labelKey1: labelValue1
    labelKey2: ""
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
`
	policies, _ := buildTest(t, input)
	assert.Contains(t, policies, "test1/test1-placementrules")

	placementRule := policies["test1/test1-placementrules"].(utils.PlacementRule)
	exceptedExpressions := []map[string]interface{}{
		{
			"key":      "labelKey2",
			"operator": "Exists",
		},
		{
			"key":      "labelKey1",
			"operator": "In",
			"values":   []string{"labelValue1"},
		},
	}

	assert.ElementsMatch(t, placementRule.Spec.ClusterSelector.MatchExpressions, exceptedExpressions)
}

func TestBindingRulesWithExcludedClustersOnly(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingExcludedRules:
    labelKey1: ""
    labelKey2: "labelValue2"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericSubscription.yaml
      policyName: "gen-sub-policy"
    - fileName: GenericOperatorGroup.yaml
      policyName: "gen-sub-policy"
`
	policies, _ := buildTest(t, input)
	assert.Contains(t, policies, "test1/test1-placementrules")

	placementRule := policies["test1/test1-placementrules"].(utils.PlacementRule)
	exceptedExpressions := []map[string]interface{}{
		{
			"key":      "labelKey1",
			"operator": "DoesNotExist",
		},
		{
			"key":      "labelKey2",
			"operator": "NotIn",
			"values":   []string{"labelValue2"},
		},
	}

	assert.ElementsMatch(t, placementRule.Spec.ClusterSelector.MatchExpressions, exceptedExpressions)
}

func TestBindingRulesWithDuplicateKey(t *testing.T) {
	inputs := []string{
		`
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    labelKey1: ""
  bindingExcludedRules:
    labelKey1: ""
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
`,
		`
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    labelKey1: "labelValue1"
  bindingExcludedRules:
    labelKey1: ""
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
`,
		`
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    labelKey1: "labelValue1"
    labelKey2: "labelValue2"
  bindingExcludedRules:
    labelKey2: "labelValue2"
  sourceFiles:
    # Create operators policies that will be installed in all clusters
    - fileName: GenericNamespace.yaml
      policyName: "gen-sub-policy"
`,
	}
	for _, input := range inputs {
		// Read in the test PGT
		pgt := utils.PolicyGenTemplate{}
		err := yaml.Unmarshal([]byte(input), &pgt)
		assert.NoError(t, err)

		// Set up the files handler to pick up local source-crs and skip any output
		fHandler := utils.NewFilesHandler("./testData/GenericSourceFiles", "/dev/null", "/dev/null")

		// Run the PGT through the generator
		pBuilder := NewPolicyBuilder(fHandler)
		policies, err := pBuilder.Build(pgt)

		assert.NotContains(t, policies, "test/test-placementrules")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid bindingRules and bindingExcludedRules found")
	}
}

// Test cases for when source-cr contains trailing separators
func TestTrailingSeperators(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test"
  namespace: "test"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericCRWithTrailingSeparators.yaml
      policyName: "gen-cr-policy"
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

	assert.Contains(t, policies, "test/test-gen-cr-policy")

	// Validate that the number of extracted objects is 1
	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, 1, len(objects))

	assert.Equal(t, defaultComplianceType, objects[0].ComplianceType)
}
