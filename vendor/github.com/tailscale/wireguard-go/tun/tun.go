/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

package tun

import (
	"os"
)

type Event int

const (
	EventUp = 1 << iota
	EventDown
	EventMTUUpdate
)

type Device interface {
	// File returns the file descriptor of the device.
	File() *os.File

	// Read one or more packets from the Device (without any additional headers).
	// On a successful read it returns the number of packets read, and sets
	// packet lengths within the sizes slice. len(sizes) must be >= len(bufs).
	// A nonzero offset can be used to instruct the Device on where to begin
	// reading into each element of the bufs slice.
	Read(bufs [][]byte, sizes []int, offset int) (n int, err error)

	// Write one or more packets to the device (without any additional headers).
	// On a successful write it returns the number of packets written. A nonzero
	// offset can be used to instruct the Device on where to begin writing from
	// each packet contained within the bufs slice.
	Write(bufs [][]byte, offset int) (int, error)

	// MTU returns the MTU of the Device.
	MTU() (int, error)

	// Name returns the current name of the Device.
	Name() (string, error)

	// Events returns a channel of type Event, which is fed Device events.
	Events() <-chan Event

	// Close stops the Device and closes the Event channel.
	Close() error

	// BatchSize returns the preferred/max number of packets that can be read or
	// written in a single read/write call. BatchSize must not change over the
	// lifetime of a Device.
	BatchSize() int
}

// GRODevice is a Device extended with methods for disabling GRO. Certain OS
// versions may have offload bugs. Where these bugs negatively impact throughput
// or break connectivity entirely we can use these methods to disable the
// related offload.
//
// Linux has the following known, GRO bugs.
//
// torvalds/linux@e269d79c7d35aa3808b1f3c1737d63dab504ddc8 broke virtio_net
// TCP & UDP GRO causing GRO writes to return EINVAL. The bug was then
// resolved later in
// torvalds/linux@89add40066f9ed9abe5f7f886fe5789ff7e0c50e. The offending
// commit was pulled into various LTS releases.
//
// UDP GRO writes end up blackholing/dropping packets destined for a
// vxlan/geneve interface on kernel versions prior to 6.8.5.
type GRODevice interface {
	Device
	// DisableUDPGRO disables UDP GRO if it is enabled.
	DisableUDPGRO()
	// DisableTCPGRO disables TCP GRO if it is enabled.
	DisableTCPGRO()
}
