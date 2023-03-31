package framework

import (
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
)

func CopyToTempDir(relativePath string) (string, error) {
	// Create temporary directory
	dir, err := os.MkdirTemp("", "temp-*")
	if err != nil {
		return "", err
	}

	// Make sure temp dir path is an absolute path
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}

	// Copy the file files from relativePath to the temp dir
	err = copy.Copy(relativePath, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	// Set the temp director as the current directory
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
