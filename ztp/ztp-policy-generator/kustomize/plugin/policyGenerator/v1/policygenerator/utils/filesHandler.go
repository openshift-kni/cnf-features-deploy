package utils

import (
	"io/ioutil"
	"os"
	"strings"
	"path/filepath"
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
	ex, err := os.Executable()
	dir := filepath.Dir(ex)
	if err != nil {
		return nil, err
	}
	return fHandler.readFile( dir + "/" + ResourcesDir + "/" + fileName)
}
