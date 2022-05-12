package siteConfig

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

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

func GetExtraManifestResourceFiles(manifestsPath string) ([]os.FileInfo, error) {
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(ex)

	var files []os.FileInfo

	files, err = GetFiles(resolveFilePath(manifestsPath, dir))

	if err != nil {
		dir, err = os.Getwd()

		if err != nil {
			return nil, err
		}

		files, err = GetFiles(resolveFilePath(manifestsPath, dir))
	}
	return files, err
}
