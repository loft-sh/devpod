//go:build windows

package copy

import (
	"os"
)

func IsUID(info os.FileInfo, uid uint32) bool {
	return true
}

func Lchown(info os.FileInfo, sourcePath, destPath string) error {
	return nil
}
