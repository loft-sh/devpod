//go:build linux

package inject

import "golang.org/x/sys/unix"

func isNoExec(path string) (bool, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		return false, err
	}

	return stat.Flags&unix.ST_NOEXEC != 0, nil
}
