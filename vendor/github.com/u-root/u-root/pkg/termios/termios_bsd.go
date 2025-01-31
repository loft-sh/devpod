// Copyright 2015-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin || freebsd || openbsd
// +build darwin freebsd openbsd

package termios

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

type TTYIO struct {
	f *os.File
}

// Winsize embeds unix.Winsize.
type Winsize struct {
	unix.Winsize
}

// New creates a new TTYIO using /dev/tty
func New() (*TTYIO, error) {
	return NewWithDev("/dev/tty")
}

// NewWithDev creates a new TTYIO with the specified device
func NewWithDev(device string) (*TTYIO, error) {
	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &TTYIO{f: f}, nil
}

// NewTTYS returns a new TTYIO.
func NewTTYS(port string) (*TTYIO, error) {
	f, err := os.OpenFile(filepath.Join("/dev", port), unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK, 0o620)
	if err != nil {
		return nil, err
	}
	return &TTYIO{f: f}, nil
}

// GetTermios returns a filled-in Termios, from an fd.
func GetTermios(fd uintptr) (*Termios, error) {
	t, err := unix.IoctlGetTermios(int(fd), unix.TIOCGETA)
	if err != nil {
		return nil, err
	}
	return &Termios{Termios: *t}, nil
}

// Get terms a Termios from a TTYIO.
func (t *TTYIO) Get() (*Termios, error) {
	return GetTermios(t.f.Fd())
}

// SetTermios sets tty parameters for an fd from a Termios.
func SetTermios(fd uintptr, ti *Termios) error {
	return unix.IoctlSetTermios(int(fd), unix.TIOCSETA, &ti.Termios)
}

// Set sets tty parameters for a TTYIO from a Termios.
func (t *TTYIO) Set(ti *Termios) error {
	return SetTermios(t.f.Fd(), ti)
}

// GetWinSize gets window size from an fd.
func GetWinSize(fd uintptr) (*Winsize, error) {
	w, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	return &Winsize{Winsize: *w}, err
}

// GetWinSize gets window size from a TTYIO.
func (t *TTYIO) GetWinSize() (*Winsize, error) {
	return GetWinSize(t.f.Fd())
}

// SetWinSize sets window size for an fd from a Winsize.
func SetWinSize(fd uintptr, w *Winsize) error {
	return unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, &w.Winsize)
}

// SetWinSize sets window size for a TTYIO from a Winsize.
func (t *TTYIO) SetWinSize(w *Winsize) error {
	return SetWinSize(t.f.Fd(), w)
}

// Ctty sets the control tty into a Cmd, from a TTYIO.
func (t *TTYIO) Ctty(c *exec.Cmd) {
	c.Stdin, c.Stdout, c.Stderr = t.f, t.f, t.f
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setctty = true
	c.SysProcAttr.Setsid = true
	c.SysProcAttr.Ctty = int(t.f.Fd())
}

// MakeRaw modifies Termio state so, if it used for an fd or tty, it will set it to raw mode.
func MakeRaw(term *Termios) *Termios {
	raw := *term
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8

	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	return &raw
}

// MakeSerialBaud updates the Termios to set the baudrate
func MakeSerialBaud(term *Termios, baud int) (*Termios, error) {
	t := *term
	rate, ok := baud2unixB[baud]
	if !ok {
		return nil, fmt.Errorf("%d: Unrecognized baud rate", baud)
	}

	//	t.Cflag &^= unix.CBAUD
	t.Cflag |= toTermiosCflag(rate)
	t.Ispeed = rate
	t.Ospeed = rate

	return &t, nil
}

// MakeSerialDefault updates the Termios to typical serial configuration:
//   - Ignore all flow control (modem, hardware, software...)
//   - Translate carriage return to newline on input
//   - Enable canonical mode: Input is available line by line, with line editing
//     enabled (ERASE, KILL are supported)
//   - Local ECHO is added (and handled by line editing)
//   - Map newline to carriage return newline on output
func MakeSerialDefault(term *Termios) *Termios {
	t := *term
	/* Clear all except baud, stop bit and parity settings */
	t.Cflag &= /*unix.CBAUD | */ unix.CSTOPB | unix.PARENB | unix.PARODD
	/* Set: 8 bits; ignore Carrier Detect; enable receive */
	t.Cflag |= unix.CS8 | unix.CLOCAL | unix.CREAD
	t.Iflag = unix.ICRNL
	t.Lflag = unix.ICANON | unix.ISIG | unix.ECHO | unix.ECHOE | unix.ECHOK | unix.ECHOKE | unix.ECHOCTL
	/* non-raw output; add CR to each NL */
	t.Oflag = unix.OPOST | unix.ONLCR
	/* reads will block only if < 1 char is available */
	t.Cc[unix.VMIN] = 1
	/* no timeout (reads block forever) */
	t.Cc[unix.VTIME] = 0
	// t.Line = 0

	return &t
}
