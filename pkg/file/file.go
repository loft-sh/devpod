package file

import (
	"os"
	"path/filepath"
)

func Chown(userName string, target string) error {
	return chown(userName, target)
}

func MkdirAll(userName string, dir string, perm os.FileMode) error {
	_, err := os.Stat(dir)
	if err == nil {
		return nil
	}

	err = os.MkdirAll(dir, perm)
	if err != nil {
		return err
	}

	return chown(userName, dir)
}

func IsLocalDir(name string) (bool, string) {
	_, err := os.Stat(name)
	if err == nil {
		absPath, _ := filepath.Abs(name)
		if absPath != "" {
			return true, absPath
		}
	}

	return false, name
}
