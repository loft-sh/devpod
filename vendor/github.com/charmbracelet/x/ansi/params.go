package ansi

import (
	"bytes"
)

// Params parses and returns a list of control sequence parameters.
//
// Parameters are positive integers separated by semicolons. Empty parameters
// default to zero. Parameters can have sub-parameters separated by colons.
//
// Any non-parameter bytes are ignored. This includes bytes that are not in the
// range of 0x30-0x3B.
//
// See ECMA-48 § 5.4.1.
func Params(p []byte) [][]uint {
	if len(p) == 0 {
		return [][]uint{}
	}

	// Filter out non-parameter bytes i.e. non 0x30-0x3B.
	p = bytes.TrimFunc(p, func(r rune) bool {
		return r < 0x30 || r > 0x3B
	})

	parts := bytes.Split(p, []byte{';'})
	params := make([][]uint, len(parts))
	for i, part := range parts {
		sparts := bytes.Split(part, []byte{':'})
		params[i] = make([]uint, len(sparts))
		for j, spart := range sparts {
			params[i][j] = bytesToUint16(spart)
		}
	}

	return params
}

func bytesToUint16(b []byte) uint {
	var n uint
	for _, c := range b {
		n = n*10 + uint(c-'0')
	}
	return n
}
