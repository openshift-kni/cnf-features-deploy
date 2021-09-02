package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type FilesHandler struct {
	sourceDir string
	tempDir   string
	outDir    string
}

func NewFilesHandler(sourceDir string, tempDir string, outDir string) *FilesHandler {
	return &FilesHandler{sourceDir: sourceDir, tempDir: tempDir, outDir: outDir}
}

func (fHandler *FilesHandler) WriteFile(filePath string, content []byte) error {
	path := fHandler.outDir + "/" + filePath[:strings.LastIndex(filePath, "/")]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0775)
	}
	err := ioutil.WriteFile(fHandler.outDir+"/"+filePath, content, 0644)

	return err
}

func (fHandler *FilesHandler) getFiles(path string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(path)
}

func (fHandler *FilesHandler) readFile(filePath string) ([]byte, error) {
	return ioutil.ReadFile(filePath)
}

func (fHandler *FilesHandler) GetTempFiles() ([]os.FileInfo, error) {
	return fHandler.getFiles(fHandler.tempDir)
}

func (fHandler *FilesHandler) ReadTempFile(fileName string) ([]byte, error) {
	return fHandler.readFile(fHandler.tempDir + "/" + fileName)
}

func (fHandler *FilesHandler) GetSourceFiles(subDir string) ([]os.FileInfo, error) {
	return fHandler.getFiles(fHandler.sourceDir + "/" + subDir)
}

func (fHandler *FilesHandler) ReadSourceFile(fileName string) ([]byte, error) {
	return fHandler.readFile(fHandler.sourceDir + "/" + fileName)
}

func (fHandler *FilesHandler) ReadResourceFile(fileName string) ([]byte, error) {
	var dir = ""
	var err error = nil
	var ret []byte

	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir = filepath.Dir(ex)
	ret, err = fHandler.readFile(dir + "/" + ResourcesDir + "/" + fileName)

	// added fail safe for test runs as `os.Executable()` will fail for tests
	if err != nil {

		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
		ret, err = fHandler.readFile(dir + "/" + ResourcesDir + "/" + fileName)
	}
	return ret, err
}
