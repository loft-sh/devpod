/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

package conn

import (
	"net"

	"golang.org/x/sys/unix"
)

const (
	// TODO: upstream to x/sys/unix
	socketOptionLevelUDP   = 17
	socketOptionUDPSegment = 103
	socketOptionUDPGRO     = 104
)

func supportsUDPOffload(conn *net.UDPConn) (txOffload, rxOffload bool) {
	rc, err := conn.SyscallConn()
	if err != nil {
		return
	}
	err = rc.Control(func(fd uintptr) {
		_, errSyscall := unix.GetsockoptInt(int(fd), unix.IPPROTO_UDP, socketOptionUDPSegment)
		if errSyscall != nil {
			return
		}
		txOffload = true
		opt, errSyscall := unix.GetsockoptInt(int(fd), unix.IPPROTO_UDP, socketOptionUDPGRO)
		if errSyscall != nil {
			return
		}
		rxOffload = opt == 1
	})
	if err != nil {
		return false, false
	}
	return txOffload, rxOffload
}
