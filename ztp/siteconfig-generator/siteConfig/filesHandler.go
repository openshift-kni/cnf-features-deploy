package siteConfig

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

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

func ReadExtraManifestResourceFile(filePath string) ([]byte, error) {
	var dir = ""
	var err error = nil
	var ret []byte

	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir = filepath.Dir(ex)
	ret, err = ReadFile(dir + "/" + filePath)

	// added fail safe for test runs as `os.Executable()` will fail for tests
	if err != nil {

		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
		ret, err = ReadFile(dir + "/" + filePath)
	}
	return ret, err
}

func GetExtraManifestResourceFiles(manifestsPath string) ([]os.FileInfo, error) {
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(ex)
	files, err := GetFiles(dir + "/" + manifestsPath)
	if err != nil {
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
		files, err = GetFiles(dir + "/" + manifestsPath)
	}
	return files, err
}
