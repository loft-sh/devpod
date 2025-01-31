// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package termios

import "golang.org/x/sys/unix"

const (
	gets       = unix.TIOCGETA
	sets       = unix.TIOCSETA
	getWinSize = unix.TIOCGWINSZ
	setWinSize = unix.TIOCSWINSZ
)

func speed(speed int) int32 {
	return int32(speed)
}
