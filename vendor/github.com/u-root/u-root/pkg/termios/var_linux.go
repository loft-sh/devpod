// Copyright 2015-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package termios

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// baud2unixB convert a baudrate to the corresponding unix const.
var baud2unixB = map[int]uint32{
	50:      unix.B50,
	75:      unix.B75,
	110:     unix.B110,
	134:     unix.B134,
	150:     unix.B150,
	200:     unix.B200,
	300:     unix.B300,
	600:     unix.B600,
	1200:    unix.B1200,
	1800:    unix.B1800,
	2400:    unix.B2400,
	4800:    unix.B4800,
	9600:    unix.B9600,
	19200:   unix.B19200,
	38400:   unix.B38400,
	57600:   unix.B57600,
	115200:  unix.B115200,
	230400:  unix.B230400,
	460800:  unix.B460800,
	500000:  unix.B500000,
	576000:  unix.B576000,
	921600:  unix.B921600,
	1000000: unix.B1000000,
	1152000: unix.B1152000,
	1500000: unix.B1500000,
	2000000: unix.B2000000,
	2500000: unix.B2500000,
	3000000: unix.B3000000,
	3500000: unix.B3500000,
	4000000: unix.B4000000,
}

// init adds constants that are linux-specific
func init() {
	extra := map[string]*bit{
		"iuclc": {word: I, mask: syscall.IUCLC},
		"olcuc": {word: O, mask: syscall.OLCUC},
		"xcase": {word: L, mask: syscall.XCASE},
		// not in FreeBSD
		"iutf8": {word: I, mask: syscall.IUTF8},
		"ofill": {word: O, mask: syscall.OFILL},
		"ofdel": {word: O, mask: syscall.OFDEL},
	}
	for k, v := range extra {
		boolFields[k] = v
	}
}
