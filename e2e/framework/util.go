package framework

import (
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
)

func CopyToTempDir(relativePath string) (string, error) {
	dir, err := os.MkdirTemp("", "temp-*")
	if err != nil {
		return "", err
	}

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}

	err = copy.Copy(relativePath, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	err = os.Chdir(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}

func CleanupTempDir(initialDir, tempDir string) {
	err := os.RemoveAll(tempDir)
	ExpectNoError(err)

	err = os.Chdir(initialDir)
	ExpectNoError(err)
}
