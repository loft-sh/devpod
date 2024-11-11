package ansi

import (
	"unicode/utf8"
	"unsafe"

	"github.com/charmbracelet/x/ansi/parser"
)

// ParserDispatcher is a function that dispatches a sequence.
type ParserDispatcher func(Sequence)

// Parser represents a DEC ANSI compatible sequence parser.
//
// It uses a state machine to parse ANSI escape sequences and control
// characters. The parser is designed to be used with a terminal emulator or
// similar application that needs to parse ANSI escape sequences and control
// characters.
// See package [parser] for more information.
//
//go:generate go run ./gen.go
type Parser struct {
	// Params contains the raw parameters of the sequence.
	// These parameters used when constructing CSI and DCS sequences.
	Params []int

	// Data contains the raw data of the sequence.
	// These data used when constructing OSC, DCS, SOS, PM, and APC sequences.
	Data []byte

	// DataLen keeps track of the length of the data buffer.
	// If DataLen is -1, the data buffer is unlimited and will grow as needed.
	// Otherwise, DataLen is limited by the size of the Data buffer.
	DataLen int

	// ParamsLen keeps track of the number of parameters.
	// This is limited by the size of the Params buffer.
	//
	// This is also used when collecting UTF-8 runes to keep track of the
	// number of rune bytes collected.
	ParamsLen int

	// Cmd contains the raw command along with the private marker and
	// intermediate bytes of the sequence.
	// The first lower byte contains the command byte, the next byte contains
	// the private marker, and the next byte contains the intermediate byte.
	//
	// This is also used when collecting UTF-8 runes treating it as a slice of
	// 4 bytes.
	Cmd int

	// State is the current state of the parser.
	State byte
}

// NewParser returns a new parser with the given sizes allocated.
// If dataSize is zero, the underlying data buffer will be unlimited and will
// grow as needed.
func NewParser(paramsSize, dataSize int) *Parser {
	s := new(Parser)
	if dataSize <= 0 {
		dataSize = 0
		s.DataLen = -1
	}
	s.Params = make([]int, paramsSize)
	s.Data = make([]byte, dataSize)
	return s
}

// Reset resets the parser to its initial state.
func (p *Parser) Reset() {
	p.clear()
	p.State = parser.GroundState
}

// clear clears the parser parameters and command.
func (p *Parser) clear() {
	if len(p.Params) > 0 {
		p.Params[0] = parser.MissingParam
	}
	p.ParamsLen = 0
	p.Cmd = 0
}

// clearCmd clears the parser command, params len and data len.
func (p *Parser) clearCmd() {
	p.Cmd = 0
	p.ParamsLen = 0
	p.DataLen = 0
}

// StateName returns the name of the current state.
func (p *Parser) StateName() string {
	return parser.StateNames[p.State]
}

// Parse parses the given dispatcher and byte buffer.
func (p *Parser) Parse(dispatcher ParserDispatcher, b []byte) {
	for i := 0; i < len(b); i++ {
		p.Advance(dispatcher, b[i], i < len(b)-1)
	}
}

// Advance advances the parser with the given dispatcher and byte.
func (p *Parser) Advance(dispatcher ParserDispatcher, b byte, more bool) parser.Action {
	switch p.State {
	case parser.Utf8State:
		// We handle UTF-8 here.
		return p.advanceUtf8(dispatcher, b)
	default:
		return p.advance(dispatcher, b, more)
	}
}

func (p *Parser) collectRune(b byte) {
	if p.ParamsLen >= utf8.UTFMax {
		return
	}

	shift := p.ParamsLen * 8
	p.Cmd &^= 0xff << shift
	p.Cmd |= int(b) << shift
	p.ParamsLen++
}

func (p *Parser) advanceUtf8(dispatcher ParserDispatcher, b byte) parser.Action {
	// Collect UTF-8 rune bytes.
	p.collectRune(b)
	rw := utf8ByteLen(byte(p.Cmd & 0xff))
	if rw == -1 {
		// We panic here because the first byte comes from the state machine,
		// if this panics, it means there is a bug in the state machine!
		panic("invalid rune") // unreachable
	}

	if p.ParamsLen < rw {
		return parser.CollectAction
	}

	// We have enough bytes to decode the rune using unsafe
	r, _ := utf8.DecodeRune((*[utf8.UTFMax]byte)(unsafe.Pointer(&p.Cmd))[:rw])
	if dispatcher != nil {
		dispatcher(Rune(r))
	}

	p.State = parser.GroundState
	p.ParamsLen = 0

	return parser.PrintAction
}

func (p *Parser) advance(d ParserDispatcher, b byte, more bool) parser.Action {
	state, action := parser.Table.Transition(p.State, b)

	// We need to clear the parser state if the state changes from EscapeState.
	// This is because when we enter the EscapeState, we don't get a chance to
	// clear the parser state. For example, when a sequence terminates with a
	// ST (\x1b\\ or \x9c), we dispatch the current sequence and transition to
	// EscapeState. However, the parser state is not cleared in this case and
	// we need to clear it here before dispatching the esc sequence.
	if p.State != state {
		if p.State == parser.EscapeState {
			p.performAction(d, parser.ClearAction, state, b)
		}
		if action == parser.PutAction &&
			p.State == parser.DcsEntryState && state == parser.DcsStringState {
			// XXX: This is a special case where we need to start collecting
			// non-string parameterized data i.e. doesn't follow the ECMA-48 §
			// 5.4.1 string parameters format.
			p.performAction(d, parser.StartAction, state, 0)
		}
	}

	// Handle special cases
	switch {
	case b == ESC && p.State == parser.EscapeState:
		// Two ESCs in a row
		p.performAction(d, parser.ExecuteAction, state, b)
		if !more {
			// Two ESCs at the end of the buffer
			p.performAction(d, parser.ExecuteAction, state, b)
		}
	case b == ESC && !more:
		// Last byte is an ESC
		p.performAction(d, parser.ExecuteAction, state, b)
	case p.State == parser.EscapeState && b == 'P' && !more:
		// ESC P (DCS) at the end of the buffer
		p.performAction(d, parser.DispatchAction, state, b)
	case p.State == parser.EscapeState && b == 'X' && !more:
		// ESC X (SOS) at the end of the buffer
		p.performAction(d, parser.DispatchAction, state, b)
	case p.State == parser.EscapeState && b == '[' && !more:
		// ESC [ (CSI) at the end of the buffer
		p.performAction(d, parser.DispatchAction, state, b)
	case p.State == parser.EscapeState && b == ']' && !more:
		// ESC ] (OSC) at the end of the buffer
		p.performAction(d, parser.DispatchAction, state, b)
	case p.State == parser.EscapeState && b == '^' && !more:
		// ESC ^ (PM) at the end of the buffer
		p.performAction(d, parser.DispatchAction, state, b)
	case p.State == parser.EscapeState && b == '_' && !more:
		// ESC _ (APC) at the end of the buffer
		p.performAction(d, parser.DispatchAction, state, b)
	default:
		p.performAction(d, action, state, b)
	}

	p.State = state

	return action
}

func (p *Parser) performAction(dispatcher ParserDispatcher, action parser.Action, state parser.State, b byte) {
	switch action {
	case parser.IgnoreAction:
		break

	case parser.ClearAction:
		p.clear()

	case parser.PrintAction:
		if dispatcher != nil {
			dispatcher(Rune(b))
		}

	case parser.ExecuteAction:
		if dispatcher != nil {
			dispatcher(ControlCode(b))
		}

	case parser.MarkerAction:
		// Collect private marker
		// we only store the last marker
		p.Cmd &^= 0xff << parser.MarkerShift
		p.Cmd |= int(b) << parser.MarkerShift

	case parser.CollectAction:
		if state == parser.Utf8State {
			// Reset the UTF-8 counter
			p.ParamsLen = 0
			p.collectRune(b)
		} else {
			// Collect intermediate bytes
			// we only store the last intermediate byte
			p.Cmd &^= 0xff << parser.IntermedShift
			p.Cmd |= int(b) << parser.IntermedShift
		}

	case parser.ParamAction:
		// Collect parameters
		if p.ParamsLen >= len(p.Params) {
			break
		}

		if b >= '0' && b <= '9' {
			if p.Params[p.ParamsLen] == parser.MissingParam {
				p.Params[p.ParamsLen] = 0
			}

			p.Params[p.ParamsLen] *= 10
			p.Params[p.ParamsLen] += int(b - '0')
		}

		if b == ':' {
			p.Params[p.ParamsLen] |= parser.HasMoreFlag
		}

		if b == ';' || b == ':' {
			p.ParamsLen++
			if p.ParamsLen < len(p.Params) {
				p.Params[p.ParamsLen] = parser.MissingParam
			}
		}

	case parser.StartAction:
		if p.DataLen < 0 && p.Data != nil {
			p.Data = p.Data[:0]
		} else {
			p.DataLen = 0
		}
		if p.State >= parser.DcsEntryState && p.State <= parser.DcsStringState {
			// Collect the command byte for DCS
			p.Cmd |= int(b)
		} else {
			p.Cmd = parser.MissingCommand
		}

	case parser.PutAction:
		switch p.State {
		case parser.OscStringState:
			if b == ';' && p.Cmd == parser.MissingCommand {
				// Try to parse the command
				datalen := len(p.Data)
				if p.DataLen >= 0 {
					datalen = p.DataLen
				}
				for i := 0; i < datalen; i++ {
					d := p.Data[i]
					if d < '0' || d > '9' {
						break
					}
					if p.Cmd == parser.MissingCommand {
						p.Cmd = 0
					}
					p.Cmd *= 10
					p.Cmd += int(d - '0')
				}
			}
		}

		if p.DataLen < 0 {
			p.Data = append(p.Data, b)
		} else {
			if p.DataLen < len(p.Data) {
				p.Data[p.DataLen] = b
				p.DataLen++
			}
		}

	case parser.DispatchAction:
		// Increment the last parameter
		if p.ParamsLen > 0 && p.ParamsLen < len(p.Params)-1 ||
			p.ParamsLen == 0 && len(p.Params) > 0 && p.Params[0] != parser.MissingParam {
			p.ParamsLen++
		}

		if dispatcher == nil {
			break
		}

		var seq Sequence
		data := p.Data
		if p.DataLen >= 0 {
			data = data[:p.DataLen]
		}
		switch p.State {
		case parser.CsiEntryState, parser.CsiParamState, parser.CsiIntermediateState:
			p.Cmd |= int(b)
			seq = CsiSequence{Cmd: p.Cmd, Params: p.Params[:p.ParamsLen]}
		case parser.EscapeState, parser.EscapeIntermediateState:
			p.Cmd |= int(b)
			seq = EscSequence(p.Cmd)
		case parser.DcsEntryState, parser.DcsParamState, parser.DcsIntermediateState, parser.DcsStringState:
			seq = DcsSequence{Cmd: p.Cmd, Params: p.Params[:p.ParamsLen], Data: data}
		case parser.OscStringState:
			seq = OscSequence{Cmd: p.Cmd, Data: data}
		case parser.SosStringState:
			seq = SosSequence{Data: data}
		case parser.PmStringState:
			seq = PmSequence{Data: data}
		case parser.ApcStringState:
			seq = ApcSequence{Data: data}
		}

		dispatcher(seq)
	}
}

func utf8ByteLen(b byte) int {
	if b <= 0b0111_1111 { // 0x00-0x7F
		return 1
	} else if b >= 0b1100_0000 && b <= 0b1101_1111 { // 0xC0-0xDF
		return 2
	} else if b >= 0b1110_0000 && b <= 0b1110_1111 { // 0xE0-0xEF
		return 3
	} else if b >= 0b1111_0000 && b <= 0b1111_0111 { // 0xF0-0xF7
		return 4
	}
	return -1
}
