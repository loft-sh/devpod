package ansi

import (
	"bytes"

	"github.com/charmbracelet/x/ansi/parser"
)

// Sequence represents an ANSI sequence. This can be a control sequence, escape
// sequence, a printable character, etc.
type Sequence interface {
	// String returns the string representation of the sequence.
	String() string
	// Bytes returns the byte representation of the sequence.
	Bytes() []byte
	// Clone returns a copy of the sequence.
	Clone() Sequence
}

// Rune represents a printable character.
type Rune rune

var _ Sequence = Rune(0)

// Bytes implements Sequence.
func (r Rune) Bytes() []byte {
	return []byte(string(r))
}

// String implements Sequence.
func (r Rune) String() string {
	return string(r)
}

// Clone implements Sequence.
func (r Rune) Clone() Sequence {
	return r
}

// ControlCode represents a control code character. This is a character that
// is not printable and is used to control the terminal. This would be a
// character in the C0 or C1 set in the range of 0x00-0x1F and 0x80-0x9F.
type ControlCode byte

var _ Sequence = ControlCode(0)

// Bytes implements Sequence.
func (c ControlCode) Bytes() []byte {
	return []byte{byte(c)}
}

// String implements Sequence.
func (c ControlCode) String() string {
	return string(c)
}

// Clone implements Sequence.
func (c ControlCode) Clone() Sequence {
	return c
}

// EscSequence represents an escape sequence.
type EscSequence int

var _ Sequence = EscSequence(0)

// buffer returns the buffer of the escape sequence.
func (e EscSequence) buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteByte('\x1b')
	if i := parser.Intermediate(int(e)); i != 0 {
		b.WriteByte(byte(i))
	}
	b.WriteByte(byte(e.Command()))
	return &b
}

// Bytes implements Sequence.
func (e EscSequence) Bytes() []byte {
	return e.buffer().Bytes()
}

// String implements Sequence.
func (e EscSequence) String() string {
	return e.buffer().String()
}

// Clone implements Sequence.
func (e EscSequence) Clone() Sequence {
	return e
}

// Command returns the command byte of the escape sequence.
func (e EscSequence) Command() int {
	return parser.Command(int(e))
}

// Intermediate returns the intermediate byte of the escape sequence.
func (e EscSequence) Intermediate() int {
	return parser.Intermediate(int(e))
}

// SosSequence represents a SOS sequence.
type SosSequence struct {
	// Data contains the raw data of the sequence.
	Data []byte
}

var _ Sequence = &SosSequence{}

// Clone implements Sequence.
func (s SosSequence) Clone() Sequence {
	return SosSequence{Data: append([]byte(nil), s.Data...)}
}

// Bytes implements Sequence.
func (s SosSequence) Bytes() []byte {
	return s.buffer().Bytes()
}

// String implements Sequence.
func (s SosSequence) String() string {
	return s.buffer().String()
}

func (s SosSequence) buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteByte('\x1b')
	b.WriteByte('X')
	b.Write(s.Data)
	b.WriteString("\x1b\\")
	return &b
}

// PmSequence represents a PM sequence.
type PmSequence struct {
	// Data contains the raw data of the sequence.
	Data []byte
}

var _ Sequence = &PmSequence{}

// Clone implements Sequence.
func (s PmSequence) Clone() Sequence {
	return PmSequence{Data: append([]byte(nil), s.Data...)}
}

// Bytes implements Sequence.
func (s PmSequence) Bytes() []byte {
	return s.buffer().Bytes()
}

// String implements Sequence.
func (s PmSequence) String() string {
	return s.buffer().String()
}

// buffer returns the buffer of the PM sequence.
func (s PmSequence) buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteByte('\x1b')
	b.WriteByte('^')
	b.Write(s.Data)
	b.WriteString("\x1b\\")
	return &b
}

// ApcSequence represents an APC sequence.
type ApcSequence struct {
	// Data contains the raw data of the sequence.
	Data []byte
}

var _ Sequence = &ApcSequence{}

// Clone implements Sequence.
func (s ApcSequence) Clone() Sequence {
	return ApcSequence{Data: append([]byte(nil), s.Data...)}
}

// Bytes implements Sequence.
func (s ApcSequence) Bytes() []byte {
	return s.buffer().Bytes()
}

// String implements Sequence.
func (s ApcSequence) String() string {
	return s.buffer().String()
}

// buffer returns the buffer of the APC sequence.
func (s ApcSequence) buffer() *bytes.Buffer {
	var b bytes.Buffer
	b.WriteByte('\x1b')
	b.WriteByte('_')
	b.Write(s.Data)
	b.WriteString("\x1b\\")
	return &b
}
