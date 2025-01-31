// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// https://github.com/golang/go/blob/master/src/os/stat_wasip1.go

//go:build wasip1
// +build wasip1

package times

import (
	"os"
	"syscall"
	"time"
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

func timespecToTime(sec, nsec int64) time.Time {
	return time.Unix(sec, nsec)
}

func getTimespec(fi os.FileInfo) (t timespec) {
	stat := fi.Sys().(*syscall.Stat_t)
	t.atime.v = timespecToTime(int64(stat.Atime), 0)
	t.mtime.v = timespecToTime(int64(stat.Mtime), 0)
	t.ctime.v = timespecToTime(int64(stat.Ctime), 0)
	return t
}
