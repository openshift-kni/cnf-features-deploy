package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type FilesHandler struct {
	sourceDir string
	PgtDir    string
	OutDir    string
}

func NewFilesHandler(sourceDir string, pgtDir string, OutDir string) *FilesHandler {
	return &FilesHandler{sourceDir: sourceDir, PgtDir: pgtDir, OutDir: OutDir}
}

func (fHandler *FilesHandler) WriteFile(filePath string, content []byte) error {
	path := fHandler.OutDir + "/" + filePath[:strings.LastIndex(filePath, "/")]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0775)
	}
	file := fHandler.OutDir + "/" + filePath

	//create new or append if exist
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = f.WriteString(string(content)); err != nil {
		return err
	}

	return err
}

func (fHandler *FilesHandler) getFiles(path string) ([]os.FileInfo, error) {
	// todo: update deprecated
	return ioutil.ReadDir(path)
}

func (fHandler *FilesHandler) ReadFile(filePath string) ([]byte, error) {
	// todo: update deprecated
	return ioutil.ReadFile(filePath)
}

func (fHandler *FilesHandler) GetTempFiles() ([]os.FileInfo, error) {
	return fHandler.getFiles(fHandler.PgtDir)
}

func (fHandler *FilesHandler) ReadTempFile(fileName string) ([]byte, error) {
	return fHandler.ReadFile(fHandler.PgtDir + "/" + fileName)
}

func (fHandler *FilesHandler) GetSourceFiles(subDir string) ([]os.FileInfo, error) {
	return fHandler.getFiles(fHandler.sourceDir + "/" + subDir)
}

func (fHandler *FilesHandler) ReadSourceFile(fileName string) ([]byte, error) {
	if fHandler.sourceDir == SourceCRsPath {
		return fHandler.ReadSourceCRFile(fileName)
	}
	return fHandler.ReadFile(fHandler.sourceDir + "/" + fileName)
}

func (fHandler *FilesHandler) ReadSourceCRFile(fileName string) ([]byte, error) {
	var dir = ""
	var err error = nil
	var ret []byte

	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir = filepath.Dir(ex)
	ret, err = fHandler.ReadFile(dir + "/" + SourceCRsPath + "/" + fileName)

	// added fail safe for test runs as `os.Executable()` will fail for tests
	if err != nil {
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
		ret, err = fHandler.ReadFile(dir + "/" + SourceCRsPath + "/" + fileName)
	}
	return ret, err
}
