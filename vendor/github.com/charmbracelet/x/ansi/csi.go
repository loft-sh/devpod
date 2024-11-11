package ansi

import (
	"bytes"
	"strconv"

	"github.com/charmbracelet/x/ansi/parser"
)

// CsiSequence represents a control sequence introducer (CSI) sequence.
//
// The sequence starts with a CSI sequence, CSI (0x9B) in a 8-bit environment
// or ESC [ (0x1B 0x5B) in a 7-bit environment, followed by any number of
// parameters in the range of 0x30-0x3F, then by any number of intermediate
// byte in the range of 0x20-0x2F, then finally with a single final byte in the
// range of 0x20-0x7E.
//
//	CSI P..P I..I F
//
// See ECMA-48 § 5.4.
type CsiSequence struct {
	// Params contains the raw parameters of the sequence.
	// This is a slice of integers, where each integer is a 32-bit integer
	// containing the parameter value in the lower 31 bits and a flag in the
	// most significant bit indicating whether there are more sub-parameters.
	Params []int

	// Cmd contains the raw command of the sequence.
	// The command is a 32-bit integer containing the CSI command byte in the
	// lower 8 bits, the private marker in the next 8 bits, and the intermediate
	// byte in the next 8 bits.
	//
	//  CSI ? u
	//
	// Is represented as:
	//
	//  'u' | '?' << 8
	Cmd int
}

var _ Sequence = CsiSequence{}

// Marker returns the marker byte of the CSI sequence.
// This is always gonna be one of the following '<' '=' '>' '?' and in the
// range of 0x3C-0x3F.
// Zero is returned if the sequence does not have a marker.
func (s CsiSequence) Marker() int {
	return parser.Marker(s.Cmd)
}

// Intermediate returns the intermediate byte of the CSI sequence.
// An intermediate byte is in the range of 0x20-0x2F. This includes these
// characters from ' ', '!', '"', '#', '$', '%', '&', ”', '(', ')', '*', '+',
// ',', '-', '.', '/'.
// Zero is returned if the sequence does not have an intermediate byte.
func (s CsiSequence) Intermediate() int {
	return parser.Intermediate(s.Cmd)
}

// Command returns the command byte of the CSI sequence.
func (s CsiSequence) Command() int {
	return parser.Command(s.Cmd)
}

// Param returns the parameter at the given index.
// It returns -1 if the parameter does not exist.
func (s CsiSequence) Param(i int) int {
	return parser.Param(s.Params, i)
}

// HasMore returns true if the parameter has more sub-parameters.
func (s CsiSequence) HasMore(i int) bool {
	return parser.HasMore(s.Params, i)
}

// Subparams returns the sub-parameters of the given parameter.
// It returns nil if the parameter does not exist.
func (s CsiSequence) Subparams(i int) []int {
	return parser.Subparams(s.Params, i)
}

// Len returns the number of parameters in the sequence.
// This will return the number of parameters in the sequence, excluding any
// sub-parameters.
func (s CsiSequence) Len() int {
	return parser.Len(s.Params)
}

// Range iterates over the parameters of the sequence and calls the given
// function for each parameter.
// The function should return false to stop the iteration.
func (s CsiSequence) Range(fn func(i int, param int, hasMore bool) bool) {
	parser.Range(s.Params, fn)
}

// Clone returns a copy of the CSI sequence.
func (s CsiSequence) Clone() Sequence {
	return CsiSequence{
		Params: append([]int(nil), s.Params...),
		Cmd:    s.Cmd,
	}
}

// String returns a string representation of the sequence.
// The string will always be in the 7-bit format i.e (ESC [ P..P I..I F).
func (s CsiSequence) String() string {
	return s.buffer().String()
}

// buffer returns a buffer containing the sequence.
func (s CsiSequence) buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteString("\x1b[")
	if m := s.Marker(); m != 0 {
		b.WriteByte(byte(m))
	}
	s.Range(func(i, param int, hasMore bool) bool {
		if param >= 0 {
			b.WriteString(strconv.Itoa(param))
		}
		if i < len(s.Params)-1 {
			if hasMore {
				b.WriteByte(':')
			} else {
				b.WriteByte(';')
			}
		}
		return true
	})
	if i := s.Intermediate(); i != 0 {
		b.WriteByte(byte(i))
	}
	b.WriteByte(byte(s.Command()))
	return &b
}

// Bytes returns the byte representation of the sequence.
// The bytes will always be in the 7-bit format i.e (ESC [ P..P I..I F).
func (s CsiSequence) Bytes() []byte {
	return s.buffer().Bytes()
}
