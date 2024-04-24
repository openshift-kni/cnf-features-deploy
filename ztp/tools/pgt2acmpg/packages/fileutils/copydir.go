// The content of this file was copied from https://stackoverflow.com/questions/51779243/copy-a-folder-in-go
package fileutils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyDirectory Recursively content of a directory
func CopyDirectory(scrDir, dest string) error {
	err := CreateIfNotExists(dest, DefaultDirWritePermissions)
	if err != nil {
		return err
	}
	var entries []fs.DirEntry
	entries, err = os.ReadDir(scrDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		var fileInfo fs.FileInfo
		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			err = CreateIfNotExists(destPath, DefaultDirWritePermissions)
			if err != nil {
				return err
			}
			err = CopyDirectory(sourcePath, destPath)
			if err != nil {
				return err
			}
		case os.ModeSymlink:
			err = CopySymLink(sourcePath, destPath)
			if err != nil {
				return err
			}
		default:
			_, err = Copy(sourcePath, destPath)
			if err != nil {
				return err
			}
		}

		fInfo, err := entry.Info()
		if err != nil {
			return err
		}

		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, fInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

// Exists Checks if a file exists
func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// CreateIfNotExists Creates a directory if it does not exist
func CreateIfNotExists(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

// CopySymLink Copies a symlink
func CopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}
