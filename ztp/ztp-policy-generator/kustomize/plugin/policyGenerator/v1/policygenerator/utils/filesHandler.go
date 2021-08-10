package utils

import (
	"io/ioutil"
	"os"
	"strings"
)

type FilesHandler struct {
	sourcePoliciesDir string
	policyGenTempDir  string
	outDir            string
}

func NewFilesHandler(sourcePoliciesDir string, policyGenTempDir string, outDir string) *FilesHandler {
	return &FilesHandler{sourcePoliciesDir: sourcePoliciesDir, policyGenTempDir: policyGenTempDir, outDir: outDir}
}

func (fHandler *FilesHandler) WriteFile(filePath string, content []byte) {
	path := fHandler.outDir + "/" + filePath[:strings.LastIndex(filePath, "/")]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0775)
	}

	err := ioutil.WriteFile(fHandler.outDir+"/"+filePath, content, 0644)
	if err != nil {
		panic(err)
	}
}

func (fHandler *FilesHandler) getFiles(path string) []os.FileInfo {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}
	return files
}

func (fHandler *FilesHandler) readFile(filePath string) []byte {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return file
}

func (fHandler *FilesHandler) GetPolicyGenTemplates() []os.FileInfo {
	return fHandler.getFiles(fHandler.policyGenTempDir)
}

func (fHandler *FilesHandler) ReadPolicyGenTempFile(fileName string) []byte {
	return fHandler.readFile(fHandler.policyGenTempDir + "/" + fileName)
}

func (fHandler *FilesHandler) GetSourceFiles(subDir string) []os.FileInfo {
	return fHandler.getFiles(fHandler.sourcePoliciesDir + "/" + subDir)
}

func (fHandler *FilesHandler) ReadSourceFileCR(fileName string) []byte {
	return fHandler.readFile(fHandler.sourcePoliciesDir + "/" + fileName)
}
