// The content of this file was copied from https://stackoverflow.com/questions/51779243/copy-a-folder-in-go
package fileutils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

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

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
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

		err = os.Lchown(destPath, int(stat.Uid), int(stat.Gid))
		if err != nil {
			return err
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

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func CopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}
