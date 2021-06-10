package main

import (
	"github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

const TEST_DIRECTORY = "testFiles"
const OUT_DIRECTORY = "out"
const TEMPLATE_DIRECTORY = "templates"
const SOURCE_POLICIES_DIRECTORY = "sourcePolicies"
const COMMON_DIRECTORY = "common"
const GROUP_DIRECTORY = "groups"
const SITE_DIRECTORY = "sites"
const CUSTOM_RESOURCE_DIRECTORY = "customResource"
const POLICIES = "-policies"
const POLICIES_PLACEMENT_RULE = "policies-placementrule.yaml"
const POLICIES_PLACEMENT_BINDING = "policies-placementbinding.yaml"
const SPEC = "spec"
const POLICY_TEMPLATES = "policy-templates"
const OBJECT_TEMPLATES = "object-templates"
const OBJECT_DEFINITION = "objectDefinition"

/* Section Test Setup Functions Starts */
func testSetup(t *testing.T) {
	os.RemoveAll(getOutPath(t))
}

func testCleanup(t *testing.T) {
	os.RemoveAll(getOutPath(t))
}

/* Section Test Setup Functions Ends */

/* Section Test Functions Starts */

/*func TestExample(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}*/

func TestNamespace(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestOperatorGroup(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestSubscription(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestMachineConfigPool(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestSriovNetwork(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestPtpConfig(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

/* Section Test Functions Ends */

/* Section Test Trigger Functions Starts */
func generateACMResourceDefinitions(t *testing.T) {
	InitiatePolicyGen(getTemplatePath(t), getSourcePolicyPath(t), getOutPath(t), true, false, false)
}

func generateCustomResourceDefinitions(t *testing.T) {
	InitiatePolicyGen(getTemplatePath(t), getSourcePolicyPath(t), getOutPath(t), true, true, false)
}

/* Section Test Trigger Functions Ends */

/* Section Assertions Functions Starts */
func ACMResourceDefinitionAssertions(t *testing.T) {
	dirName, fileName := computeACMCustomResourceDirectoryNameAndFileName(t)
	dir, file := path.Split(dirName)
	parentDir := getOutPath(t)
	if len(dir) > 0 {
		parentDir = path.Join(getOutPath(t), dir)
		dirName = file
	}
	assert.True(t, checkDirectoryExists(parentDir, dirName))
	var dirPath = path.Join(parentDir, dirName)
	assert.True(t, checkFileExists(dirPath, fileName))
	placementRuleAndBindingAssertions(dirPath, t)
	var ACMResourceFilePath = path.Join(dirPath, fileName)
	checkIfSourceYamlEqualsACMObjectDefinition(ACMResourceFilePath, t)

}

func placementRuleAndBindingAssertions(dirPath string, t *testing.T) {
	_, file := path.Split(dirPath)

	assert.True(t, checkFileExists(dirPath, file+"-"+POLICIES_PLACEMENT_RULE))
	assert.True(t, checkFileExists(dirPath, file+"-"+POLICIES_PLACEMENT_BINDING))
}

func CustomResourceDefinitionAssertions(t *testing.T) {
	dirName := computeCustomResourceDirectoryName(t)
	fileName := computeCustomResourceFileName(t)
	dir, file := path.Split(dirName)
	parentDir := getOutPath(t)
	if len(dir) > 0 {
		parentDir = path.Join(getOutPath(t), dir)
		dirName = file
	}
	assert.True(t, checkDirectoryExists(parentDir, dirName))
	var customResourceDir = path.Join(parentDir, dirName)
	assert.True(t, checkFileExists(customResourceDir, fileName))
	var customResourceFilePath = path.Join(customResourceDir, fileName)
	assert.True(t, checkIfYamlFilesAreEqual(customResourceFilePath, t))
}

func checkIfYamlFilesAreEqual(generatedFilePath string, t *testing.T) bool {
	sourceFileData := getSourcePolicyWithSubstitutions(t)
	generatedFileData := readFileToMap(generatedFilePath, t)
	return reflect.DeepEqual(sourceFileData, generatedFileData)
}

func checkIfSourceYamlEqualsACMObjectDefinition(generatedFilePath string, t *testing.T) bool {
	sourceFileData := getSourcePolicyWithSubstitutions(t)
	generatedFileData := getACMObjectDefinitionFromGeneratedYaml(generatedFilePath, t)
	return reflect.DeepEqual(sourceFileData, generatedFileData)
}

func checkDirectoryExists(parentDirPath string, childDir string) bool {
	var dirFs, err = ioutil.ReadDir(parentDirPath)
	if err == nil {
		for i := range dirFs {
			if dirFs[i].IsDir() && dirFs[i].Name() == childDir {
				return true
			}
		}
	}
	return false
}

func checkFileExists(parentDirPath string, fileName string) bool {
	var dirFs, err = ioutil.ReadDir(parentDirPath)
	if err == nil {
		for i := range dirFs {
			if !dirFs[i].IsDir() && dirFs[i].Name() == fileName {
				return true
			}
		}
	}
	return false
}

/* Section Assertions Functions Ends */

/* Section Getter Functions Starts */

func getOutPath(t *testing.T) string {
	cwd, _ := os.Getwd()
	testName := t.Name()
	outPath := path.Join(cwd, TEST_DIRECTORY, testName, OUT_DIRECTORY)
	return outPath
}

func getSourcePolicyPath(t *testing.T) string {
	cwd, _ := os.Getwd()
	testName := t.Name()
	sourcePolicyPath := path.Join(cwd, TEST_DIRECTORY, testName, SOURCE_POLICIES_DIRECTORY)
	return sourcePolicyPath
}

func getTemplatePath(t *testing.T) string {
	cwd, _ := os.Getwd()
	testName := t.Name()
	templatePath := path.Join(cwd, TEST_DIRECTORY, testName, TEMPLATE_DIRECTORY)
	return templatePath
}

func getTestCRDFileName(t *testing.T) string {
	testName := t.Name()
	return testName + utils.FileExt
}
func readFileToTemplateObject(t *testing.T) utils.PolicyGenTemplate {
	filePath := path.Join(getTemplatePath(t), getTestCRDFileName(t))
	fileData := utils.PolicyGenTemplate{}
	file1, err := ioutil.ReadFile(filePath)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = yaml.Unmarshal(file1, &fileData)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	return fileData
}

func computeACMCustomResourceDirectoryNameAndFileName(t *testing.T) (string, string) {
	dirName := ""
	fileName := ""
	fileData := readFileToTemplateObject(t)

	if fileData.Metadata.Labels.Common {
		dirName = COMMON_DIRECTORY
		fileName += COMMON_DIRECTORY
	} else if fileData.Metadata.Labels.GroupName != utils.NotApplicable {
		dirName = GROUP_DIRECTORY
		dirName = path.Join(dirName, fileData.Metadata.Labels.GroupName)
		fileName += fileData.Metadata.Labels.GroupName
	} else if fileData.Metadata.Labels.SiteName != utils.NotApplicable {
		dirName = SITE_DIRECTORY
		dirName = path.Join(dirName, fileData.Metadata.Labels.SiteName)
		fileName += fileData.Metadata.Labels.SiteName
	}

	if len(fileData.SourceFiles[0].FileName) > 0 {
		fileName += "-" + fileData.SourceFiles[0].PolicyName
	}

	fileName += utils.FileExt
	return dirName, fileName
}

func computeCustomResourceDirectoryName(t *testing.T) string {
	dirName := ""
	fileData := readFileToTemplateObject(t)
	dirName = CUSTOM_RESOURCE_DIRECTORY
	if fileData.Metadata.Labels.Common {
		commonDirName := COMMON_DIRECTORY + POLICIES
		dirName = path.Join(dirName, commonDirName)
	} else if fileData.Metadata.Labels.GroupName != utils.NotApplicable {
		groupPoliciesDirName := fileData.Metadata.Labels.GroupName + POLICIES
		dirName = path.Join(dirName, groupPoliciesDirName)
	} else if fileData.Metadata.Labels.SiteName != utils.NotApplicable {
		sitePoliciesDirName := fileData.Metadata.Labels.SiteName + POLICIES
		dirName = path.Join(dirName, sitePoliciesDirName)
	}
	return dirName
}

func computeCustomResourceFileName(t *testing.T) string {
	fileName := ""

	fileData := getSourcePolicyWithSubstitutions(t)

	if fileData["kind"] != nil && len(fileData["kind"].(string)) > 0 && fileData["kind"] != utils.NotApplicable {
		fileName += fileData["kind"].(string)
	}
	if fileData["metadata"] != nil {
		fileMetadata := fileData["metadata"].(map[string]interface{})

		if fileMetadata["name"] != nil && len(fileMetadata["name"].(string)) > 0 && fileMetadata["name"].(string) != utils.NotApplicable {
			fileName += "-" + fileMetadata["name"].(string)
		}
		if fileMetadata["namespace"] != nil && len(fileMetadata["namespace"].(string)) > 0 && fileMetadata["namespace"] != utils.NotApplicable {
			fileName += "-" + fileMetadata["namespace"].(string)
		}
	}
	fileName += utils.FileExt
	return fileName
}

func getSourcePolicyWithSubstitutions(t *testing.T) map[string]interface{} {
	filePath := path.Join(getSourcePolicyPath(t), getTestCRDFileName(t))
	fileData := readFileToMap(filePath, t)
	policyTemp := getPolicyTemplateObject(t)
	for i := 0; i < len(policyTemp.SourceFiles); i++ {
		sFile := policyTemp.SourceFiles[i]
		if sFile.Metadata.Name != "" && sFile.Metadata.Name != utils.NotApplicable {
			fileData["metadata"].(map[string]interface{})["name"] = sFile.Metadata.Name
		}
		if len(sFile.Metadata.Labels) > 0 {
			fileData["metadata"].(map[string]interface{})["labels"] = sFile.Metadata.Labels
		}
		if sFile.Spec != nil {
			specMap := fileData["spec"].(map[string]interface{})
			substituteMapData(specMap, sFile.Spec)
			fileData["spec"] = specMap
		}
	}
	return fileData
}

func getPolicyTemplateObject(t *testing.T) utils.PolicyGenTemplate {
	filePath := path.Join(getTemplatePath(t), getTestCRDFileName(t))
	fileData := utils.PolicyGenTemplate{}
	file1, err := ioutil.ReadFile(filePath)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	err = yaml.Unmarshal(file1, &fileData)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	return fileData
}

func getACMObjectDefinitionFromGeneratedYaml(generatedFilePath string, t *testing.T) map[string]interface{} {
	fileData := readFileToMap(generatedFilePath, t)
	return fileData[SPEC].(map[string]interface{})[POLICY_TEMPLATES].([]interface{})[0].(map[string]interface{})[OBJECT_DEFINITION].(map[string]interface{})[SPEC].(map[string]interface{})[OBJECT_TEMPLATES].([]interface{})[0].(map[string]interface{})[OBJECT_DEFINITION].(map[string]interface{})
}

/* Section Getter Functions Ends */

/* Section Helper Functions Starts */
func readFileToMap(filePath string, t *testing.T) map[string]interface{} {
	fileData := make(map[string]interface{})
	file1, err := ioutil.ReadFile(filePath)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	err = yaml.Unmarshal(file1, &fileData)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	return fileData
}

func substituteMapData(sourceMap map[string]interface{}, valueMap map[string]interface{}) map[string]interface{} {
	for key, value := range valueMap {
		if reflect.TypeOf(value).Kind() == reflect.Map {
			sourceMap[key] = substituteMapData(sourceMap[key].(map[string]interface{}), value.(map[string]interface{}))
		} else if reflect.TypeOf(value).Kind() == reflect.Slice {
			valueArr := value.([]interface{})
			sourceMapArr := make([]interface{}, 1)
			for i := 0; i < len(valueArr); i++ {
				sourceMapArr[i] = substituteMapData(sourceMap[key].([]interface{})[i].(map[string]interface{}), valueArr[i].(map[string]interface{}))
			}
			sourceMap[key] = sourceMapArr

		} else {
			sourceMap[key] = value
		}
	}
	for key, value := range sourceMap {
		if value == nil ||
			(value != nil && reflect.ValueOf(value).Kind() == reflect.String &&
				(value.(string) == "" || value.(string) == utils.NotApplicable)) {
			delete(sourceMap, key)
		}
	}
	return sourceMap
}

/* Section Helper Functions Ends */
