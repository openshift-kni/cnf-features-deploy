package siteConfig

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type DirContainFiles struct {
	Directory string
	Files     []os.FileInfo
}

func resolveFilePath(filePath string, basedir string) string {
	if _, errAbsPath := os.Stat(filePath); errAbsPath == nil {
		return filePath
	}
	return basedir + "/" + filePath
}

func GetFiles(path string) ([]os.FileInfo, error) {
	fileInfo, err := os.Stat(path)

	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return ioutil.ReadDir(path)
	}

	return []os.FileInfo{fileInfo}, nil
}

func ReadFile(filePath string) ([]byte, error) {
	return ioutil.ReadFile(filePath)
}

func WriteFile(filePath string, outDir string, content []byte) error {
	path := outDir + "/" + filePath[:strings.LastIndex(filePath, "/")]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0775)
	}
	err := ioutil.WriteFile(outDir+"/"+filePath, content, 0644)

	return err
}

func ReadExtraManifestResourceFile(filePath string) ([]byte, error) {
	var dir = ""
	var err error = nil
	var ret []byte

	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir = filepath.Dir(ex)

	ret, err = ReadFile(resolveFilePath(filePath, dir))

	// added fail safe for test runs as `os.Executable()` will fail for tests
	if err != nil {

		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}

		ret, err = ReadFile(resolveFilePath(filePath, dir))
	}
	return ret, err
}

func GetExtraManifestResourceDir(manifestsPath string) (string, error) {

	ex, err := os.Executable()
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(ex)

	return resolveFilePath(manifestsPath, dir), err
}

func GetExtraManifestResourceFiles(manifestsPath string) ([]os.FileInfo, error) {

	var files []os.FileInfo

	dirPath, err := GetExtraManifestResourceDir(manifestsPath)
	if err != nil {
		return files, err
	}

	files, err = GetFiles(dirPath)
	if err != nil {
		dir, err := os.Getwd()

		if err != nil {
			return nil, err
		}

		files, err = GetFiles(resolveFilePath(manifestsPath, dir))
	}
	return files, err
}
