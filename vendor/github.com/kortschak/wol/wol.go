// Copyright ©2016 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wol provides a Wake On LAN function.
package wol

import (
	"bytes"
	"errors"
	"io"
	"net"
)

const magicLen = 6 + 16*6

// Wake sends a Wake On LAN magic packet for the given MAC address
// at the given remote address. If local is not nil, it is used as
// the local address for the connection to send on.
func Wake(mac net.HardwareAddr, pass []byte, local, remote *net.UDPAddr) error {
	if len(mac) != 6 {
		return errors.New("wol: bad MAC address")
	}
	switch len(pass) {
	default:
		return errors.New("wol: bad password length")
	case 0, 6:
	}
	var magic [magicLen + 6]byte
	copy(magic[:], []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	buf := bytes.NewBuffer(magic[:6])
	for i := 0; i < 16; i++ {
		buf.Write(mac)
	}
	buf.Write(pass)
	if buf.Len() != magicLen+len(pass) {
		panic("wol: unexpected packet length")
	}

	conn, err := net.DialUDP("udp", local, remote)
	if err != nil {
		return err
	}
	defer conn.Close()

	n, err := conn.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if n < magicLen+len(pass) {
		return io.ErrShortWrite
	}
	return nil
}
