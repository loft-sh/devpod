//go:build linux || darwin || unix

package copy

import (
	"fmt"
	"os"
	"syscall"
)

func IsUID(info os.FileInfo, uid uint32) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uid
}

func Lchown(info os.FileInfo, sourcePath, destPath string) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
	}
	if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
		return err
	}
	return nil
}
