package testSource

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path"
	"reflect"
	"testing"
)

func ACMResourceDefinitionAssertions(t *testing.T) {
	dirName, fileName := ComputeACMCustomResourceDirectoryNameAndFileName(t)
	dir, file := path.Split(dirName)
	parentDir := GetOutPath(t)
	if len(dir) > 0 {
		parentDir = path.Join(GetOutPath(t), dir)
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

	assert.True(t, checkFileExists(dirPath, file+"-"+PLACEMENT_RULES))
	assert.True(t, checkFileExists(dirPath, file+"-"+PLACEMENT_BINDING))
}

func CustomResourceDefinitionAssertions(t *testing.T) {
	dirName := computeCustomResourceDirectoryName(t)
	fileName := computeCustomResourceFileName(t)
	dir, file := path.Split(dirName)
	parentDir := GetOutPath(t)
	if len(dir) > 0 {
		parentDir = path.Join(GetOutPath(t), dir)
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
