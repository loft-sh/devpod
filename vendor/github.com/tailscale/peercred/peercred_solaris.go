// Copyright (c) 2021 AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package peercred

import (
	"fmt"
	"net"
	"strconv"

	"golang.org/x/sys/unix"
)

func init() {
	osGet = getSolaris
}

func getSolaris(c net.Conn) (*Creds, error) {
	switch c := c.(type) {
	case *net.UnixConn:
		return getUnix(c)
	case *net.TCPConn:
		// TODO: Need ideas
	}
	return nil, ErrUnsupportedConnType
}

func getUnix(c *net.UnixConn) (*Creds, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("SyscallConn: %w", err)
	}

	var creds *unix.Ucred
	cerr := raw.Control(func(fd uintptr) {
		creds, err = unix.GetPeerUcred(fd)
		if err != nil {
			err = fmt.Errorf("unix.GetPeerUcred: %w", err)
			return
		}
	})
	if cerr != nil {
		return nil, fmt.Errorf("raw.Control: %w", cerr)
	}
	if err != nil {
		return nil, err
	}
	return &Creds{
		pid: creds.Getpid(),
		uid: strconv.FormatUint(uint64(creds.Geteuid()), 10),
	}, nil
}
