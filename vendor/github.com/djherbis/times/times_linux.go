// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// http://golang.org/src/os/stat_linux.go

package times

import (
	"errors"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// HasChangeTime and HasBirthTime are true if and only if
// the target OS supports them.
const (
	HasChangeTime = true
	HasBirthTime  = false
)

type timespec struct {
	atime
	mtime
	ctime
	nobtime
}

type timespecBtime struct {
	atime
	mtime
	ctime
	btime
}

var (
	supportsStatx int32 = 1
	statxFunc           = unix.Statx
)

func isStatXSupported() bool {
	return atomic.LoadInt32(&supportsStatx) == 1
}

func isStatXUnsupported(err error) bool {
	// linux 4.10 and earlier does not support Statx syscall
	if err != nil && errors.Is(err, unix.ENOSYS) {
		atomic.StoreInt32(&supportsStatx, 0)
		return true
	}
	return false
}

// Stat returns the Timespec for the given filename.
func Stat(name string) (Timespec, error) {
	if isStatXSupported() {
		ts, err := statX(name)
		if err == nil {
			return ts, nil
		}
		if !isStatXUnsupported(err) {
			return nil, err
		}
		// Fallback.
	}
	return stat(name, os.Stat)
}

func statX(name string) (Timespec, error) {
	// https://man7.org/linux/man-pages/man2/statx.2.html
	var statx unix.Statx_t
	err := statxFunc(unix.AT_FDCWD, name, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_ATIME|unix.STATX_MTIME|unix.STATX_CTIME|unix.STATX_BTIME, &statx)
	if err != nil {
		return nil, err
	}
	return extractTimes(&statx), nil
}

// Lstat returns the Timespec for the given filename, and does not follow Symlinks.
func Lstat(name string) (Timespec, error) {
	if isStatXSupported() {
		ts, err := lstatx(name)
		if err == nil {
			return ts, nil
		}
		if !isStatXUnsupported(err) {
			return nil, err
		}
		// Fallback.
	}
	return stat(name, os.Lstat)
}

func lstatx(name string) (Timespec, error) {
	// https://man7.org/linux/man-pages/man2/statx.2.html
	var statX unix.Statx_t
	err := statxFunc(unix.AT_FDCWD, name, unix.AT_STATX_SYNC_AS_STAT|unix.AT_SYMLINK_NOFOLLOW, unix.STATX_ATIME|unix.STATX_MTIME|unix.STATX_CTIME|unix.STATX_BTIME, &statX)
	if err != nil {
		return nil, err
	}
	return extractTimes(&statX), nil
}

func statXFile(file *os.File) (Timespec, error) {
	sc, err := file.SyscallConn()
	if err != nil {
		return nil, err
	}

	var statx unix.Statx_t
	var statxErr error
	err = sc.Control(func(fd uintptr) {
		// https://man7.org/linux/man-pages/man2/statx.2.html
		statxErr = statxFunc(int(fd), "", unix.AT_EMPTY_PATH|unix.AT_STATX_SYNC_AS_STAT, unix.STATX_ATIME|unix.STATX_MTIME|unix.STATX_CTIME|unix.STATX_BTIME, &statx)
	})
	if err != nil {
		return nil, err
	}

	if statxErr != nil {
		return nil, statxErr
	}

	return extractTimes(&statx), nil
}

// StatFile returns the Timespec for the given *os.File.
func StatFile(file *os.File) (Timespec, error) {
	if isStatXSupported() {
		ts, err := statXFile(file)
		if err == nil {
			return ts, nil
		}
		if !isStatXUnsupported(err) {
			return nil, err
		}
		// Fallback.
	}
	return statFile(file)
}

func statFile(file *os.File) (Timespec, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return getTimespec(fi), nil
}

func statxTimestampToTime(ts unix.StatxTimestamp) time.Time {
	return time.Unix(ts.Sec, int64(ts.Nsec))
}

func extractTimes(statx *unix.Statx_t) Timespec {
	if statx.Mask&unix.STATX_BTIME == unix.STATX_BTIME {
		var t timespecBtime
		t.atime.v = statxTimestampToTime(statx.Atime)
		t.mtime.v = statxTimestampToTime(statx.Mtime)
		t.ctime.v = statxTimestampToTime(statx.Ctime)
		t.btime.v = statxTimestampToTime(statx.Btime)
		return t
	}

	var t timespec
	t.atime.v = statxTimestampToTime(statx.Atime)
	t.mtime.v = statxTimestampToTime(statx.Mtime)
	t.ctime.v = statxTimestampToTime(statx.Ctime)
	return t
}

func timespecToTime(ts syscall.Timespec) time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec))
}

func getTimespec(fi os.FileInfo) (t timespec) {
	stat := fi.Sys().(*syscall.Stat_t)
	t.atime.v = timespecToTime(stat.Atim)
	t.mtime.v = timespecToTime(stat.Mtim)
	t.ctime.v = timespecToTime(stat.Ctim)
	return t
}
