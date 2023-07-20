//go:build !windows
// +build !windows

package setup

import (
	"fmt"
	"io/fs"
	"strconv"
	"syscall"
)

func GetUserInfo(info fs.FileInfo) (string, string, error) {
	var UID, GID string

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return UID, GID, fmt.Errorf("file read error")
	}

	UID = strconv.Itoa(int(stat.Uid))
	GID = strconv.Itoa(int(stat.Gid))

	return UID, GID, nil
}
