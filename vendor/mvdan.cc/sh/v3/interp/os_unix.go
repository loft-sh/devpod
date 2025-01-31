// Copyright (c) 2017, Andrey Nering <andrey.nering@gmail.com>
// See LICENSE for licensing information

//go:build unix

package interp

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

func mkfifo(path string, mode uint32) error {
	return unix.Mkfifo(path, mode)
}

// hasPermissionToDir returns if the OS current user has execute permission
// to the given directory
func hasPermissionToDir(info os.FileInfo) bool {
	user, err := user.Current()
	if err != nil {
		return false // unknown user; assume no permissions
	}
	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return false // on POSIX systems, Uid should always be a decimal number
	}
	if uid == 0 {
		return true // super-user
	}

	st, _ := info.Sys().(*syscall.Stat_t)
	if st == nil {
		panic("unexpected info.Sys type")
	}
	perm := info.Mode().Perm()
	// user (u)
	if perm&0o100 != 0 && st.Uid == uint32(uid) {
		return true
	}

	gid, _ := strconv.Atoi(user.Gid)
	// other users in group (g)
	if perm&0o010 != 0 && st.Uid != uint32(uid) && st.Gid == uint32(gid) {
		return true
	}
	// remaining users (o)
	if perm&0o001 != 0 && st.Uid != uint32(uid) && st.Gid != uint32(gid) {
		return true
	}

	return false
}
