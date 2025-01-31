// Copyright 2015-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package termios implements basic termios operations including getting
// a tty struct, termio struct, a winsize struct, and setting raw mode.
// To get a TTY, call termios.New.
// To get a Termios, call tty.Get(); to set it, call tty.Set(*Termios)
// To set raw mode and then restore, one can do:
// tty := termios.NewTTY()
// restorer, err := tty.Raw()
// do things
// tty.Set(restorer)
package termios

type (
	// TTY is an os-independent version of the combined info in termios and window size structs.
	// It is used to get/set info to the termios functions as well as marshal/unmarshal data
	// in JSON format for dump and loading.
	TTY struct {
		Ispeed int
		Ospeed int
		Row    int
		Col    int

		CC map[string]uint8

		Opts map[string]bool
	}
)

// Raw sets the tty into raw mode.
func (t *TTYIO) Raw() (*Termios, error) {
	restorer, err := t.Get()
	if err != nil {
		return nil, err
	}

	raw := MakeRaw(restorer)

	if err := t.Set(raw); err != nil {
		return nil, err
	}
	return restorer, nil
}

// Serial configure the serial TTY at given baudrate with ECHO and character conversion (CRNL, ERASE, KILL)
func (t *TTYIO) Serial(baud int) (*Termios, error) {
	restorer, err := t.Get()
	if err != nil {
		return nil, err
	}

	serial := MakeSerialDefault(restorer)

	if baud != 0 {
		if err := t.Set(serial); err != nil {
			return nil, err
		}

		serial, err = MakeSerialBaud(serial, baud)
		if err != nil {
			return restorer, err
		}
	}

	err = t.Set(serial)
	return restorer, err
}

func (t *TTYIO) Read(b []byte) (int, error) {
	return t.f.Read(b)
}

func (t *TTYIO) Write(b []byte) (int, error) {
	return t.f.Write(b)
}
