package utils

import (
	"io/ioutil"
	"strings"
	"os"
)

type filesHandler struct {
	sourcePoliciesDir string
	policyGenTempDir string
	outDir string
}

func NewFilesHandler(sourcePoliciesDir string, policyGenTempDir string, outDir string) *filesHandler {
	return &filesHandler{sourcePoliciesDir:sourcePoliciesDir, policyGenTempDir:policyGenTempDir, outDir:outDir}
}

func (fHandler *filesHandler) WriteFile(filePath string, content []byte) {
	path := fHandler.outDir + "/" + filePath[:strings.LastIndex(filePath, "/")]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0775)
	}

	err := ioutil.WriteFile( fHandler.outDir + "/" + filePath, content, 0644)
	if err != nil {
		panic(err)
	}
}

func (fHandler *filesHandler) GetPolicyGenTemplates() []os.FileInfo {
	files, err := ioutil.ReadDir(fHandler.policyGenTempDir)
	if err != nil {
		panic(err)
	}
	return files
}

func (fHandler *filesHandler) ReadPolicyGenTempFile(fileName string) []byte {
	file, err := ioutil.ReadFile(fHandler.policyGenTempDir + "/" + fileName)
	if err != nil {
		panic(err)
	}
	return file
}
