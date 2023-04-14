package file

import "os"

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
