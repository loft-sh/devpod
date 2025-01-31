// Copyright 2015-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !plan9 && !windows
// +build !plan9,!windows

package termios

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"golang.org/x/sys/unix"
)

// GTTY returns the TTY struct for a given fd. It is like a New in
// many packages but the name GTTY is a tradition.
func GTTY(fd int) (*TTY, error) {
	term, err := unix.IoctlGetTermios(fd, gets)
	if err != nil {
		return nil, err
	}
	w, err := unix.IoctlGetWinsize(fd, getWinSize)
	if err != nil {
		return nil, err
	}

	t := TTY{Opts: make(map[string]bool), CC: make(map[string]uint8)}
	for n, b := range boolFields {
		val := uint32(reflect.ValueOf(term).Elem().Field(b.word).Uint()) & b.mask
		t.Opts[n] = val != 0
	}

	for n, c := range cc {
		t.CC[n] = term.Cc[c]
	}

	// back in the day, you could have different i and o speeds.
	// since about 1975, this has not been a thing. It's still in POSIX
	// evidently. WTF?
	t.Ispeed = int(term.Ispeed)
	t.Ospeed = int(term.Ospeed)
	t.Row = int(w.Row)
	t.Col = int(w.Col)

	return &t, nil
}

// STTY uses a TTY * to set TTY settings on an fd.
// It returns a new TTY struct for the fd after the changes are made,
// and an error. It does not change the original TTY struct.
func (t *TTY) STTY(fd int) (*TTY, error) {
	// Get a unix.Termios which we can partially fill in.
	term, err := unix.IoctlGetTermios(fd, gets)
	if err != nil {
		return nil, err
	}

	for n, b := range boolFields {
		set := t.Opts[n]
		i := reflect.ValueOf(term).Elem().Field(b.word).Uint()
		if set {
			i |= uint64(b.mask)
		} else {
			i &= ^uint64(b.mask)
		}
		reflect.ValueOf(term).Elem().Field(b.word).SetUint(i)
	}

	for n, c := range cc {
		term.Cc[c] = t.CC[n]
	}

	term.Ispeed = speed(t.Ispeed)
	term.Ospeed = speed(t.Ospeed)

	if err := unix.IoctlSetTermios(fd, sets, term); err != nil {
		return nil, err
	}

	w := &unix.Winsize{Row: uint16(t.Row), Col: uint16(t.Col)}
	if err := unix.IoctlSetWinsize(fd, setWinSize, w); err != nil {
		return nil, err
	}

	return GTTY(fd)
}

// String will stringify a TTY, including printing out the options all in the same order.
// The options are presented in the order:
// integer options as name:value
// boolean options which are set, printed as name, sorted by name
// boolean options which are clear, printed as ~name, sorted by name
// This ordering makes it a bit more readable: integer value, sorted set values, sorted clear values
func (t *TTY) String() string {
	s := fmt.Sprintf("speed:%v ", t.Ispeed)
	s += fmt.Sprintf("rows:%d cols:%d", t.Row, t.Col)

	var intopts []string
	for n, c := range t.CC {
		intopts = append(intopts, fmt.Sprintf("%v:%#02x", n, c))
	}
	sort.Strings(intopts)

	var trueopts, falseopts []string
	for n, set := range t.Opts {
		if set {
			trueopts = append(trueopts, n)
		} else {
			falseopts = append(falseopts, "~"+n)
		}
	}
	sort.Strings(trueopts)
	sort.Strings(falseopts)

	for _, v := range append(intopts, append(trueopts, falseopts...)...) {
		s += fmt.Sprintf(" %s", v)
	}
	return s
}

func intarg(s []string, bits int) (int, error) {
	if len(s) < 2 {
		return -1, fmt.Errorf("%s requires an arg", s[0])
	}
	i, err := strconv.ParseUint(s[1], 0, bits)
	if err != nil {
		return -1, fmt.Errorf("%s is not a number", s)
	}
	return int(i), nil
}

// SetOpts sets opts in a TTY given an array of key-value pairs and
// booleans. The arguments are a variety of key-value pairs and booleans.
// booleans are cleared if the first char is a -, set otherwise.
func (t *TTY) SetOpts(opts []string) error {
	var err error
	for i := 0; i < len(opts) && err == nil; i++ {
		o := opts[i]
		switch o {
		case "rows":
			t.Row, err = intarg(opts[i:], 16)
			i++
			continue
		case "cols":
			t.Col, err = intarg(opts[i:], 16)
			i++
			continue
		case "speed":
			// 32 may sound crazy but ... baud can be REALLY large
			t.Ispeed, err = intarg(opts[i:], 32)
			i++
			continue
		}

		// see if it's one of the control char options.
		if _, ok := cc[opts[i]]; ok {
			var opt int
			if opt, err = intarg(opts[i:], 8); err != nil {
				return err
			}

			t.CC[opts[i]] = uint8(opt)
			i++
			continue
		}

		// At this point, it has to be one of the boolean ones
		// or we're done here.
		set := true
		if o[0] == '~' {
			set = false
			o = o[1:]
		}
		if _, ok := boolFields[o]; !ok {
			return fmt.Errorf("opt %v is not valid", o)
		}
		t.Opts[o] = set
		if err != nil {
			return err
		}
	}
	return err
}

// Raw sets a TTY into raw mode, returning a TTY struct
func Raw(fd int) (*TTY, error) {
	t, err := GTTY(fd)
	if err != nil {
		return nil, err
	}

	t.SetOpts([]string{"~ignbrk", "~brkint", "~parmrk", "~istrip", "~inlcr", "~igncr", "~icrnl", "~ixon", "~opost", "~echo", "~echonl", "~icanon", "~isig", "~iexten", "~parenb" /*"cs8", */, "min", "1", "time", "0"})

	return t.STTY(fd)
}
