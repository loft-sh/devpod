//go:build linux || darwin || unix

package file

import (
	"os"
	"os/user"
	"strconv"
)

func chown(userName string, target string) error {
	if userName == "" {
		return nil
	}

	u, err := user.Lookup(userName)
	if err != nil {
		return err
	}

	uid, _ := strconv.ParseInt(u.Uid, 10, 64)
	gid, _ := strconv.ParseInt(u.Gid, 10, 64)
	if uid < 0 {
		return nil
	}
	if gid < 0 {
		gid = 0
	}

	return os.Chown(target, int(uid), int(gid))
}
