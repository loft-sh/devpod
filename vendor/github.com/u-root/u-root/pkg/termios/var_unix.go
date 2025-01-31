// Copyright 2015-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !plan9
// +build !plan9

package termios

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// Termios is a struct for Termios operations.
type Termios struct {
	unix.Termios
}

type bit struct {
	word int
	mask uint32
}

var (
	boolFields = map[string]*bit{
		// Input processing
		"ignbrk":  {word: I, mask: syscall.IGNBRK},
		"brkint":  {word: I, mask: syscall.BRKINT},
		"ignpar":  {word: I, mask: syscall.IGNPAR},
		"parmrk":  {word: I, mask: syscall.PARMRK},
		"inpck":   {word: I, mask: syscall.INPCK},
		"istrip":  {word: I, mask: syscall.ISTRIP},
		"inlcr":   {word: I, mask: syscall.INLCR},
		"igncr":   {word: I, mask: syscall.IGNCR},
		"icrnl":   {word: I, mask: syscall.ICRNL},
		"ixon":    {word: I, mask: syscall.IXON},
		"ixany":   {word: I, mask: syscall.IXANY},
		"ixoff":   {word: I, mask: syscall.IXOFF},
		"imaxbel": {word: I, mask: syscall.IMAXBEL},

		// Outputprocessing
		"opost":  {word: O, mask: syscall.OPOST},
		"onlcr":  {word: O, mask: syscall.ONLCR},
		"ocrnl":  {word: O, mask: syscall.OCRNL},
		"onocr":  {word: O, mask: syscall.ONOCR},
		"onlret": {word: O, mask: syscall.ONLRET},

		// Localprocessing
		"isig":    {word: L, mask: syscall.ISIG},
		"icanon":  {word: L, mask: syscall.ICANON},
		"echo":    {word: L, mask: syscall.ECHO},
		"echoe":   {word: L, mask: syscall.ECHOE},
		"echok":   {word: L, mask: syscall.ECHOK},
		"echonl":  {word: L, mask: syscall.ECHONL},
		"noflsh":  {word: L, mask: syscall.NOFLSH},
		"tostop":  {word: L, mask: syscall.TOSTOP},
		"echoctl": {word: L, mask: syscall.ECHOCTL},
		"echoprt": {word: L, mask: syscall.ECHOPRT},
		"echoke":  {word: L, mask: syscall.ECHOKE},
		"flusho":  {word: L, mask: syscall.FLUSHO},
		"pendin":  {word: L, mask: syscall.PENDIN},
		"iexten":  {word: L, mask: syscall.IEXTEN},

		// Controlprocessing

		"cstopb": {word: C, mask: syscall.CSTOPB},
		"cread":  {word: C, mask: syscall.CREAD},
		"parenb": {word: C, mask: syscall.PARENB},
		"parodd": {word: C, mask: syscall.PARODD},
		"hupcl":  {word: C, mask: syscall.HUPCL},
		"clocal": {word: C, mask: syscall.CLOCAL},
	}
	cc = map[string]int{
		"min":   5,
		"time":  0,
		"lnext": syscall.VLNEXT,
		//"flush": syscall.VFLUSH,
		"intr":  syscall.VINTR,
		"quit":  syscall.VQUIT,
		"erase": syscall.VERASE,
		"kill":  syscall.VKILL,
		"eof":   syscall.VEOF,
		"eol":   syscall.VEOL,
		"eol2":  syscall.VEOL2,
		//"swtch": syscall.VSWTCH,
		"start": syscall.VSTART,
		"stop":  syscall.VSTOP,
		"susp":  syscall.VSUSP,
		//"rprnt": syscall.VRPRNT,
		"werase": syscall.VWERASE,
	}
)

// These consts describe the offsets into the termios struct of various elements.
const (
	I = iota // Input control
	O        // Output control
	C        // Control
	L        // Line control
)
