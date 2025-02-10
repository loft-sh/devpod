package ansi

import (
	"bytes"
	"strings"
)

// OscSequence represents an OSC sequence.
//
// The sequence starts with a OSC sequence, OSC (0x9D) in a 8-bit environment
// or ESC ] (0x1B 0x5D) in a 7-bit environment, followed by positive integer identifier,
// then by arbitrary data terminated by a ST (0x9C) in a 8-bit environment,
// ESC \ (0x1B 0x5C) in a 7-bit environment, or BEL (0x07) for backwards compatibility.
//
//	OSC Ps ; Pt ST
//	OSC Ps ; Pt BEL
//
// See ECMA-48 § 5.7.
type OscSequence struct {
	// Data contains the raw data of the sequence including the identifier
	// command.
	Data []byte

	// Cmd contains the raw command of the sequence.
	Cmd int
}

var _ Sequence = OscSequence{}

// Command returns the command of the OSC sequence.
func (s OscSequence) Command() int {
	return s.Cmd
}

// Params returns the parameters of the OSC sequence split by ';'.
// The first element is the identifier command.
func (s OscSequence) Params() []string {
	return strings.Split(string(s.Data), ";")
}

// Clone returns a copy of the OSC sequence.
func (s OscSequence) Clone() Sequence {
	return OscSequence{
		Data: append([]byte(nil), s.Data...),
		Cmd:  s.Cmd,
	}
}

// String returns the string representation of the OSC sequence.
// To be more compatible with different terminal, this will always return a
// 7-bit formatted sequence, terminated by BEL.
func (s OscSequence) String() string {
	return s.buffer().String()
}

// Bytes returns the byte representation of the OSC sequence.
// To be more compatible with different terminal, this will always return a
// 7-bit formatted sequence, terminated by BEL.
func (s OscSequence) Bytes() []byte {
	return s.buffer().Bytes()
}

func (s OscSequence) buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteString("\x1b]")
	b.Write(s.Data)
	b.WriteByte(BEL)
	return &b
}
