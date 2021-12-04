package policyGen

import (
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// Take the string input and build the policies by calling PolicyBuilder.Build
// and using the generic source-cr test data. Return the output of Build()
func buildTest(t *testing.T, input string) (map[string]interface{}, error) {
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
	return policies, err
}

// Validates the top level structure of the spec and returns topSimple, topList, and subMap
func validateBaselineStructure(t *testing.T, objDefSpec interface{}) (
	string,
	[]interface{},
	map[string]interface{},
	map[string]interface{},
) {
	spec := objDefSpec.(map[string]interface{})
	assert.NotNil(t, spec["topSimple"])

	assert.NotNil(t, spec["topList"])
	assert.NotNil(t, spec["topMap"])
	topMap := spec["topMap"].(map[string]interface{})
	assert.NotNil(t, topMap["subMap"])
	subMap := topMap["subMap"].(map[string]interface{})
	return spec["topSimple"].(string), spec["topList"].([]interface{}), topMap, subMap
}

// Test baseline case where user does not provide overlay
func TestNoOverlay(t *testing.T) {
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
    - fileName: GenericCR.yaml
      policyName: "gen-policy1"
`
	policies, _ := buildTest(t, input)

	assert.Contains(t, policies, "test1/test1-gen-policy1")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 1)

	objDef := objects[0].ObjectDefinition
	assert.NotNil(t, objDef)
	assert.Equal(t, objDef["kind"], "JustForTest")
	assert.NotNil(t, objDef["spec"])
	assert.Nil(t, objDef["data"])

	topSimple, topList, topMap, subMap := validateBaselineStructure(t, objDef["spec"])
	assert.Equal(t, topSimple, "tbd")
	assert.Equal(t, len(topList), 3)
	assert.Equal(t, topList[0], "a")
	assert.Equal(t, topList[1], "b")
	assert.Equal(t, topList[2], "c")
	assert.Equal(t, len(topMap), 1)
	assert.Equal(t, subMap["key1"], "value1")
	assert.Equal(t, subMap["key2"], "value2")
	assert.NotNil(t, subMap["subSub"])
	subSub := subMap["subSub"].(map[string]interface{})
	assert.Equal(t, subSub["x"], "y")
}

// Test case where user provides overlay of existing content in source-cr
func TestOverlay(t *testing.T) {
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
    - fileName: GenericCR.yaml
      policyName: "gen-policy1"
      spec:
        topSimple: hello
        topList:
          - d
        topMap:
          subMap:
            key1: newvalue
`
	policies, _ := buildTest(t, input)

	assert.Contains(t, policies, "test1/test1-gen-policy1")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 1)

	objDef := objects[0].ObjectDefinition
	assert.NotNil(t, objDef)
	assert.Equal(t, objDef["kind"], "JustForTest")
	assert.NotNil(t, objDef["spec"])

	topSimple, topList, topMap, subMap := validateBaselineStructure(t, objDef["spec"])
	assert.Equal(t, topSimple, "hello")
	assert.Equal(t, len(topList), 1)
	assert.Equal(t, topList[0], "d")
	assert.Equal(t, len(topMap), 1)
	assert.Equal(t, subMap["key1"], "newvalue")
	assert.Equal(t, subMap["key2"], "value2")
}

// Validate that an overlay at a level below other content updates
// only the lowest level and the source-cr content at the higer levels
// remains.
func TestOverlayDeep(t *testing.T) {
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
    - fileName: GenericCR.yaml
      policyName: "gen-policy1"
      spec:
        topMap:
          subMap:
            subSub:
              x: new
`
	policies, _ := buildTest(t, input)

	assert.Contains(t, policies, "test1/test1-gen-policy1")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 1)

	objDef := objects[0].ObjectDefinition
	assert.NotNil(t, objDef)
	assert.Equal(t, objDef["kind"], "JustForTest")
	assert.NotNil(t, objDef["spec"])

	topSimple, topList, topMap, subMap := validateBaselineStructure(t, objDef["spec"])
	assert.Equal(t, topSimple, "tbd")
	assert.Equal(t, len(topList), 3)
	assert.Equal(t, topList[0], "a")
	assert.Equal(t, len(topMap), 1)
	assert.Equal(t, subMap["key1"], "value1")
	assert.Equal(t, subMap["key2"], "value2")
	assert.NotNil(t, subMap["subSub"])
	subSub := subMap["subSub"].(map[string]interface{})
	assert.Equal(t, subSub["x"], "new")
}

// Test case where user provides overlay which adds new content at various
// levels
func TestAdditions(t *testing.T) {
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
    - fileName: GenericCR.yaml
      policyName: "gen-policy1"
      spec:
        newTopLevelItem: here
        topMap:
          newSubEntry: newsub
          subMap:
            newKey: newValue
`
	policies, _ := buildTest(t, input)

	assert.Contains(t, policies, "test1/test1-gen-policy1")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 1)

	objDef := objects[0].ObjectDefinition
	assert.NotNil(t, objDef)
	assert.Equal(t, objDef["kind"], "JustForTest")
	assert.NotNil(t, objDef["spec"])

	topSimple, topList, topMap, subMap := validateBaselineStructure(t, objDef["spec"])
	assert.Equal(t, topSimple, "tbd")
	assert.Equal(t, objDef["spec"].(map[string]interface{})["newTopLevelItem"], "here")
	assert.Equal(t, len(topList), 3)
	assert.Equal(t, topList[0], "a")
	assert.Equal(t, topList[1], "b")
	assert.Equal(t, topList[2], "c")
	assert.Equal(t, len(topMap), 2)
	assert.Equal(t, subMap["key1"], "value1")
	assert.Equal(t, subMap["key2"], "value2")
	assert.Equal(t, subMap["newKey"], "newValue")
}

// Test case where user provides overlay which adds a section (spec/data) which
// was not in the source-cr
func TestAddedSection(t *testing.T) {
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
    - fileName: GenericCR.yaml
      policyName: "gen-policy1"
      data:
        item1: value
`
	policies, _ := buildTest(t, input)

	assert.Contains(t, policies, "test1/test1-gen-policy1")

	objects := extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 1)

	objDef := objects[0].ObjectDefinition
	assert.NotNil(t, objDef)
	assert.Equal(t, objDef["kind"], "JustForTest")
	assert.NotNil(t, objDef["spec"])

	// Make sure the baseline content is OK
	topSimple, topList, topMap, subMap := validateBaselineStructure(t, objDef["spec"])
	assert.Equal(t, topSimple, "tbd")
	assert.Equal(t, len(topList), 3)
	assert.Equal(t, topList[0], "a")
	assert.Equal(t, topList[1], "b")
	assert.Equal(t, topList[2], "c")
	assert.Equal(t, len(topMap), 1)
	assert.Equal(t, subMap["key1"], "value1")
	assert.Equal(t, subMap["key2"], "value2")

	// Validate the new section
	assert.NotNil(t, objDef["data"])
	data := objDef["data"].(map[string]interface{})
	assert.Equal(t, data["item1"], "value")

	/////////
	// And the reverse test for adding a spec section
	/////////
	input = `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericDataCR.yaml
      policyName: "gen-policy1"
`
	policies, _ = buildTest(t, input)

	assert.Contains(t, policies, "test1/test1-gen-policy1")

	objects = extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 1)
	objDef = objects[0].ObjectDefinition
	assert.NotNil(t, objDef["data"])
	assert.Equal(t, objDef["data"].(map[string]interface{})["justData"], true)
	assert.Nil(t, objDef["spec"])

	input = `
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "test1"
  namespace: "test1"
spec:
  bindingRules:
    justfortest: "true"
  sourceFiles:
    - fileName: GenericDataCR.yaml
      policyName: "gen-policy1"
      spec:
        key5: value5
`
	policies, _ = buildTest(t, input)

	assert.Contains(t, policies, "test1/test1-gen-policy1")

	objects = extractCRsFromPolicies(t, policies)
	assert.Equal(t, len(objects), 1)
	objDef = objects[0].ObjectDefinition
	assert.NotNil(t, objDef["data"])
	assert.Equal(t, objDef["data"].(map[string]interface{})["justData"], true)
	assert.NotNil(t, objDef["spec"])
	assert.Equal(t, objDef["spec"].(map[string]interface{})["key5"], "value5")
}
