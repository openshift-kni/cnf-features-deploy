package utils

import (
	"errors"
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
	err := os.WriteFile(fHandler.OutDir+"/"+filePath, content, 0644)

	return err
}

func (fHandler *FilesHandler) getFiles(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (fHandler *FilesHandler) ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func (fHandler *FilesHandler) GetTempFiles() ([]os.DirEntry, error) {
	return fHandler.getFiles(fHandler.PgtDir)
}

func (fHandler *FilesHandler) ReadTempFile(fileName string) ([]byte, error) {
	return fHandler.ReadFile(fHandler.PgtDir + "/" + fileName)
}

func (fHandler *FilesHandler) GetSourceFiles(subDir string) ([]os.DirEntry, error) {
	return fHandler.getFiles(fHandler.sourceDir + "/" + subDir)
}

func (fHandler *FilesHandler) ReadSourceFile(fileName string) ([]byte, error) {
	if fHandler.sourceDir == SourceCRsPath {
		return fHandler.ReadSourceCRFile(fileName)
	}
	return fHandler.ReadFile(fHandler.sourceDir + "/" + fileName)
}

func (fHandler *FilesHandler) ReadSourceCRFile(fileName string) ([]byte, error) {
	var (
		gitDir   = ""
		localDir = ""
		err      error
		fileByte []byte
	)

	// current working directory in git
	gitDir, err = os.Getwd()
	if err != nil {
		return nil, err
	}
	sourceCRPathGit := gitDir + "/" + SourceCRsPath + "/" + fileName
	fileByte, err = os.ReadFile(sourceCRPathGit)
	if errors.Is(err, os.ErrNotExist) {
		// path of the local executable
		localDir, err = os.Executable()
		if err != nil {
			return nil, err
		}
		dir := filepath.Dir(localDir)
		sourceCRPathLocal := dir + "/" + SourceCRsPath + "/" + fileName
		fileByte, err := os.ReadFile(sourceCRPathLocal)
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New(fileName + " is not found both in Git path: " + sourceCRPathGit + " and in the ztp container path: " + sourceCRPathLocal)
		}
		return fileByte, err
	}
	return fileByte, err
}
