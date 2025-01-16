package native

import (
	"encoding/binary"

	"golang.org/x/sys/cpu"
)

var Endian binary.ByteOrder

// IsBigEndian records whether the GOARCH's byte order is big endian.
//
// Deprecated: use golang.org/x/sys/cpu.IsBigEndian instead, now that it exists.
const IsBigEndian = cpu.IsBigEndian

func init() {
	if IsBigEndian {
		Endian = binary.BigEndian
	} else {
		Endian = binary.LittleEndian
	}
}
