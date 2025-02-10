package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/otiai10/copy"
)

func GetTimeout() time.Duration {
	if runtime.GOOS == "windows" {
		return 600 * time.Second
	}

	return 60 * time.Second
}

func CreateTempDir() (string, error) {
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

	return dir, nil
}

func CopyToTempDirWithoutChdir(relativePath string) (string, error) {
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

	return dir, nil
}

func CopyToTempDirInDir(baseDir, relativePath string) (string, error) {
	// Create temporary directory
	dir, err := os.MkdirTemp(baseDir, "temp-*")
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

func CopyToTempDir(relativePath string) (string, error) {
	return CopyToTempDirInDir("", relativePath)
}

func CleanupTempDir(initialDir, tempDir string) {
	err := os.Chdir(initialDir)
	ExpectNoError(err)

	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Println("WARN:", err)
	}
}

func CleanString(input string) string {
	input = strings.ReplaceAll(input, "\\", "")
	return strings.ReplaceAll(input, "/", "")
}
