package utils

import (
	"errors"
	"github.com/google/go-cmp/cmp"
	"io/ioutil"
	"log"
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

	file := fHandler.OutDir + "/" + filePath

	//if os.Getenv("PGT_DIFF_MODE") != "rec" {
	//	err2 := fHandler.createDirAndWriteFile(filePath, content, path, file)
	//	if err2 != nil {
	//		return err2
	//	}
	//	return nil
	//}

	if _, err := os.Stat(file); err == nil {
		// path/to/whatever exists
		// do diff
		log.Printf("exits: %s", file)
		dat, _ := os.ReadFile(file)
		srcDiff(content, dat)

	} else if errors.Is(err, os.ErrNotExist) {
		checkAgain := fHandler.OutDir + "/" + filePath[:strings.LastIndex(filePath, "/")]
		if _, err := os.Stat(checkAgain); err == nil {
			items, _ := os.ReadDir(checkAgain)
			for _, item := range items {
				t := fHandler.OutDir + "/" + filePath[:strings.LastIndex(filePath, "/")] + "/" + item.Name()
				log.Printf("attempting to read: %s", t)
				f, _ := os.ReadFile(t)
				srcDiff(content, f)
			}
		} else {
			err2 := fHandler.createDirAndWriteFile(filePath, content, path, file)
			if err2 != nil {
				return err2
			}
		}

	} else {
		// Schrodinger: file may or may not exist. See err for details.

		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence

	}

	return nil
}

func (fHandler *FilesHandler) createDirAndWriteFile(filePath string, content []byte, path string, file string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0775)
	}
	file = fHandler.OutDir + "/" + filePath[:strings.LastIndex(filePath, "/")] + "/" + filePath[strings.LastIndex(filePath, "/")+1:]
	log.Printf("Not using: %s", file)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(string(content)); err != nil {
		return err
	}
	return nil
}

func srcDiff(content []byte, dat []byte) {
	if os.Getenv("PGT_DIFF_MODE") != "rec" {

	}
	if diff := cmp.Diff(content, dat); diff != "" {
		log.Printf("------diff start --------")
		log.Printf(string(content))
		log.Printf(string(dat))
		log.Printf("Diff() mismatch (-new +current):\n%s", diff)
		log.Printf("------diff end --------")
	}
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
		log.Printf("Couldn't find CR git source-cr: %s", fileName)
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
		ret, err = fHandler.ReadFile(dir + "/" + SourceCRsPath + "/" + fileName)
	}
	return ret, err
}
