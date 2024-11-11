package ansi

import (
	"bytes"
	"strconv"

	"github.com/charmbracelet/x/ansi/parser"
)

// DcsSequence represents a Device Control String (DCS) escape sequence.
//
// The DCS sequence is used to send device control strings to the terminal. The
// sequence starts with the C1 control code character DCS (0x9B) or ESC P in
// 7-bit environments, followed by parameter bytes, intermediate bytes, a
// command byte, followed by data bytes, and ends with the C1 control code
// character ST (0x9C) or ESC \ in 7-bit environments.
//
// This follows the parameter string format.
// See ECMA-48 § 5.4.1
type DcsSequence struct {
	// Params contains the raw parameters of the sequence.
	// This is a slice of integers, where each integer is a 32-bit integer
	// containing the parameter value in the lower 31 bits and a flag in the
	// most significant bit indicating whether there are more sub-parameters.
	Params []int

	// Data contains the string raw data of the sequence.
	// This is the data between the final byte and the escape sequence terminator.
	Data []byte

	// Cmd contains the raw command of the sequence.
	// The command is a 32-bit integer containing the DCS command byte in the
	// lower 8 bits, the private marker in the next 8 bits, and the intermediate
	// byte in the next 8 bits.
	//
	//  DCS > 0 ; 1 $ r <data> ST
	//
	// Is represented as:
	//
	//  'r' | '>' << 8 | '$' << 16
	Cmd int
}

var _ Sequence = DcsSequence{}

// Marker returns the marker byte of the DCS sequence.
// This is always gonna be one of the following '<' '=' '>' '?' and in the
// range of 0x3C-0x3F.
// Zero is returned if the sequence does not have a marker.
func (s DcsSequence) Marker() int {
	return parser.Marker(s.Cmd)
}

// Intermediate returns the intermediate byte of the DCS sequence.
// An intermediate byte is in the range of 0x20-0x2F. This includes these
// characters from ' ', '!', '"', '#', '$', '%', '&', ”', '(', ')', '*', '+',
// ',', '-', '.', '/'.
// Zero is returned if the sequence does not have an intermediate byte.
func (s DcsSequence) Intermediate() int {
	return parser.Intermediate(s.Cmd)
}

// Command returns the command byte of the CSI sequence.
func (s DcsSequence) Command() int {
	return parser.Command(s.Cmd)
}

// Param returns the parameter at the given index.
// It returns -1 if the parameter does not exist.
func (s DcsSequence) Param(i int) int {
	return parser.Param(s.Params, i)
}

// HasMore returns true if the parameter has more sub-parameters.
func (s DcsSequence) HasMore(i int) bool {
	return parser.HasMore(s.Params, i)
}

// Subparams returns the sub-parameters of the given parameter.
// It returns nil if the parameter does not exist.
func (s DcsSequence) Subparams(i int) []int {
	return parser.Subparams(s.Params, i)
}

// Len returns the number of parameters in the sequence.
// This will return the number of parameters in the sequence, excluding any
// sub-parameters.
func (s DcsSequence) Len() int {
	return parser.Len(s.Params)
}

// Range iterates over the parameters of the sequence and calls the given
// function for each parameter.
// The function should return false to stop the iteration.
func (s DcsSequence) Range(fn func(i int, param int, hasMore bool) bool) {
	parser.Range(s.Params, fn)
}

// Clone returns a copy of the DCS sequence.
func (s DcsSequence) Clone() Sequence {
	return DcsSequence{
		Params: append([]int(nil), s.Params...),
		Data:   append([]byte(nil), s.Data...),
		Cmd:    s.Cmd,
	}
}

// String returns a string representation of the sequence.
// The string will always be in the 7-bit format i.e (ESC P p..p i..i f <data> ESC \).
func (s DcsSequence) String() string {
	return s.buffer().String()
}

// buffer returns a buffer containing the sequence.
func (s DcsSequence) buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteString("\x1bP")
	if m := s.Marker(); m != 0 {
		b.WriteByte(byte(m))
	}
	s.Range(func(i, param int, hasMore bool) bool {
		if param >= -1 {
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
	b.Write(s.Data)
	b.WriteByte(ESC)
	b.WriteByte('\\')
	return &b
}

// Bytes returns the byte representation of the sequence.
// The bytes will always be in the 7-bit format i.e (ESC P p..p i..i F <data> ESC \).
func (s DcsSequence) Bytes() []byte {
	return s.buffer().Bytes()
}
