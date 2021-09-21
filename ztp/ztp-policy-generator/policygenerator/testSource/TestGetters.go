package testSource

import (
	"github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func GetOutPath(t *testing.T) string {
	cwd, _ := os.Getwd()
	testName := t.Name()
	outPath := path.Join(cwd, TEST_DIRECTORY, testName, OUT_DIRECTORY)
	return outPath
}

func GetSourcePolicyPath(t *testing.T) string {
	cwd, _ := os.Getwd()
	testName := t.Name()
	sourcePolicyPath := path.Join(cwd, TEST_DIRECTORY, testName, SOURCE_POLICIES_DIRECTORY)
	return sourcePolicyPath
}

func GetTemplatePath(t *testing.T) string {
	cwd, _ := os.Getwd()
	testName := t.Name()
	templatePath := path.Join(cwd, TEST_DIRECTORY, testName, TEMPLATE_DIRECTORY)
	return templatePath
}

func GetTestCRDFileName(t *testing.T) string {
	testName := t.Name()
	return testName + utils.FileExt
}
func ReadFileToTemplateObject(t *testing.T) utils.PolicyGenTemplate {
	filePath := path.Join(GetTemplatePath(t), GetTestCRDFileName(t))
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

func ComputeACMCustomResourceDirectoryNameAndFileName(t *testing.T) (string, string) {
	dirName := ""
	fileName := ""
	fileData := ReadFileToTemplateObject(t)

	dirName = fileData.Metadata.Name
	fileName += fileData.Metadata.Name

	if len(fileData.Spec.SourceFiles[0].FileName) > 0 {
		fileName += "-" + fileData.Spec.SourceFiles[0].PolicyName
	}

	fileName += utils.FileExt
	return dirName, fileName
}

func computeCustomResourceDirectoryName(t *testing.T) string {
	dirName := ""
	fileData := ReadFileToTemplateObject(t)
	dirName = CUSTOM_RESOURCE_DIRECTORY
	childDirName := fileData.Metadata.Name
	dirName = path.Join(dirName, childDirName)
	return dirName
}

func computeCustomResourceFileName(t *testing.T) string {
	fileName := ""

	fileData := getSourcePolicyWithSubstitutions(t)

	if fileData["kind"] != nil && len(fileData["kind"].(string)) > 0 && fileData["kind"] != NOT_APPLICABLE {
		fileName += fileData["kind"].(string)
	}
	if fileData["metadata"] != nil {
		fileMetadata := fileData["metadata"].(map[string]interface{})

		if fileMetadata["name"] != nil && len(fileMetadata["name"].(string)) > 0 && fileMetadata["name"].(string) != NOT_APPLICABLE {
			fileName += "-" + fileMetadata["name"].(string)
		}
		if fileMetadata["namespace"] != nil && len(fileMetadata["namespace"].(string)) > 0 && fileMetadata["namespace"] != NOT_APPLICABLE {
			fileName += "-" + fileMetadata["namespace"].(string)
		}
	}
	fileName += utils.FileExt
	return fileName
}

func getSourcePolicyWithSubstitutions(t *testing.T) map[string]interface{} {
	filePath := path.Join(GetSourcePolicyPath(t), GetTestCRDFileName(t))
	fileData := readFileToMap(filePath, t)
	policyTemp := getPolicyTemplateObject(t)
	for i := 0; i < len(policyTemp.Spec.SourceFiles); i++ {
		sFile := policyTemp.Spec.SourceFiles[i]
		if sFile.Metadata.Name != "" && sFile.Metadata.Name != NOT_APPLICABLE {
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
	filePath := path.Join(GetTemplatePath(t), GetTestCRDFileName(t))
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
