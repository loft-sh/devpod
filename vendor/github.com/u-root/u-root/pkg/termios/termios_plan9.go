// Copyright 2015-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build plan9
// +build plan9

package termios

import (
	"fmt"
	"os"
	"path/filepath"
)

// Termios is used to manipulate the control channel of a kernel.
type Termios struct{}

// Winsize holds the window size information, it is modeled on unix.Winsize.
type Winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// TTYIO is a wrapper that only allows Read and Write.
type TTYIO struct {
	f *os.File
}

// New creates a new TTYIO using /dev/tty
func New() (*TTYIO, error) {
	return NewWithDev("/dev/cons")
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
	f, err := os.OpenFile(filepath.Join("/dev", port), os.O_RDWR, 0o620)
	if err != nil {
		return nil, err
	}
	return &TTYIO{f: f}, nil
}

// GetTermios returns a filled-in Termios, from an fd.
func GetTermios(fd uintptr) (*Termios, error) {
	return &Termios{}, nil
}

// Get terms a Termios from a TTYIO.
func (t *TTYIO) Get() (*Termios, error) {
	return GetTermios(t.f.Fd())
}

// SetTermios sets tty parameters for an fd from a Termios.
func SetTermios(fd uintptr, ti *Termios) error {
	return fmt.Errorf("Plan 9: not yet")
}

// Set sets tty parameters for a TTYIO from a Termios.
func (t *TTYIO) Set(ti *Termios) error {
	return SetTermios(t.f.Fd(), ti)
}

// GetWinSize gets window size from an fd.
func GetWinSize(fd uintptr) (*Winsize, error) {
	return nil, fmt.Errorf("Plan 9: not yet")
}

// GetWinSize gets window size from a TTYIO.
func (t *TTYIO) GetWinSize() (*Winsize, error) {
	return GetWinSize(t.f.Fd())
}

// SetWinSize sets window size for an fd from a Winsize.
func SetWinSize(fd uintptr, w *Winsize) error {
	return fmt.Errorf("Plan 9: not yet")
}

// SetWinSize sets window size for a TTYIO from a Winsize.
func (t *TTYIO) SetWinSize(w *Winsize) error {
	return SetWinSize(t.f.Fd(), w)
}

// MakeRaw modifies Termio state so, if it used for an fd or tty, it will set it to raw mode.
func MakeRaw(term *Termios) *Termios {
	raw := *term
	return &raw
}

// MakeSerialBaud updates the Termios to set the baudrate
func MakeSerialBaud(term *Termios, baud int) (*Termios, error) {
	t := *term

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

	return &t
}
