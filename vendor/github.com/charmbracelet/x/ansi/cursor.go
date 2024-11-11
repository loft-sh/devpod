package ansi

import "strconv"

// SaveCursor (DECSC) is an escape sequence that saves the current cursor
// position.
//
//	ESC 7
//
// See: https://vt100.net/docs/vt510-rm/DECSC.html
const SaveCursor = "\x1b7"

// RestoreCursor (DECRC) is an escape sequence that restores the cursor
// position.
//
//	ESC 8
//
// See: https://vt100.net/docs/vt510-rm/DECRC.html
const RestoreCursor = "\x1b8"

// RequestCursorPosition (CPR) is an escape sequence that requests the current
// cursor position.
//
//	CSI 6 n
//
// The terminal will report the cursor position as a CSI sequence in the
// following format:
//
//	CSI Pl ; Pc R
//
// Where Pl is the line number and Pc is the column number.
// See: https://vt100.net/docs/vt510-rm/CPR.html
const RequestCursorPosition = "\x1b[6n"

// RequestExtendedCursorPosition (DECXCPR) is a sequence for requesting the
// cursor position report including the current page number.
//
//	CSI ? 6 n
//
// The terminal will report the cursor position as a CSI sequence in the
// following format:
//
//	CSI ? Pl ; Pc ; Pp R
//
// Where Pl is the line number, Pc is the column number, and Pp is the page
// number.
// See: https://vt100.net/docs/vt510-rm/DECXCPR.html
const RequestExtendedCursorPosition = "\x1b[?6n"

// CursorUp (CUU) returns a sequence for moving the cursor up n cells.
//
//	CSI n A
//
// See: https://vt100.net/docs/vt510-rm/CUU.html
func CursorUp(n int) string {
	var s string
	if n > 1 {
		s = strconv.Itoa(n)
	}
	return "\x1b[" + s + "A"
}

// CursorUp1 is a sequence for moving the cursor up one cell.
//
// This is equivalent to CursorUp(1).
const CursorUp1 = "\x1b[A"

// CursorDown (CUD) returns a sequence for moving the cursor down n cells.
//
//	CSI n B
//
// See: https://vt100.net/docs/vt510-rm/CUD.html
func CursorDown(n int) string {
	var s string
	if n > 1 {
		s = strconv.Itoa(n)
	}
	return "\x1b[" + s + "B"
}

// CursorDown1 is a sequence for moving the cursor down one cell.
//
// This is equivalent to CursorDown(1).
const CursorDown1 = "\x1b[B"

// CursorRight (CUF) returns a sequence for moving the cursor right n cells.
//
//	CSI n C
//
// See: https://vt100.net/docs/vt510-rm/CUF.html
func CursorRight(n int) string {
	var s string
	if n > 1 {
		s = strconv.Itoa(n)
	}
	return "\x1b[" + s + "C"
}

// CursorRight1 is a sequence for moving the cursor right one cell.
//
// This is equivalent to CursorRight(1).
const CursorRight1 = "\x1b[C"

// CursorLeft (CUB) returns a sequence for moving the cursor left n cells.
//
//	CSI n D
//
// See: https://vt100.net/docs/vt510-rm/CUB.html
func CursorLeft(n int) string {
	var s string
	if n > 1 {
		s = strconv.Itoa(n)
	}
	return "\x1b[" + s + "D"
}

// CursorLeft1 is a sequence for moving the cursor left one cell.
//
// This is equivalent to CursorLeft(1).
const CursorLeft1 = "\x1b[D"

// CursorNextLine (CNL) returns a sequence for moving the cursor to the
// beginning of the next line n times.
//
//	CSI n E
//
// See: https://vt100.net/docs/vt510-rm/CNL.html
func CursorNextLine(n int) string {
	var s string
	if n > 1 {
		s = strconv.Itoa(n)
	}
	return "\x1b[" + s + "E"
}

// CursorPreviousLine (CPL) returns a sequence for moving the cursor to the
// beginning of the previous line n times.
//
//	CSI n F
//
// See: https://vt100.net/docs/vt510-rm/CPL.html
func CursorPreviousLine(n int) string {
	var s string
	if n > 1 {
		s = strconv.Itoa(n)
	}
	return "\x1b[" + s + "F"
}

// MoveCursor (CUP) returns a sequence for moving the cursor to the given row
// and column.
//
//	CSI n ; m H
//
// See: https://vt100.net/docs/vt510-rm/CUP.html
func MoveCursor(row, col int) string {
	if row < 0 {
		row = 0
	}
	if col < 0 {
		col = 0
	}
	return "\x1b[" + strconv.Itoa(row) + ";" + strconv.Itoa(col) + "H"
}

// MoveCursorOrigin is a sequence for moving the cursor to the upper left
// corner of the screen. This is equivalent to MoveCursor(1, 1).
const MoveCursorOrigin = "\x1b[1;1H"

// SaveCursorPosition (SCP or SCOSC) is a sequence for saving the cursor
// position.
//
//	CSI s
//
// This acts like Save, except the page number where the cursor is located is
// not saved.
//
// See: https://vt100.net/docs/vt510-rm/SCOSC.html
const SaveCursorPosition = "\x1b[s"

// RestoreCursorPosition (RCP or SCORC) is a sequence for restoring the cursor
// position.
//
//	CSI u
//
// This acts like Restore, except the cursor stays on the same page where the
// cursor was saved.
//
// See: https://vt100.net/docs/vt510-rm/SCORC.html
const RestoreCursorPosition = "\x1b[u"

// SetCursorStyle (DECSCUSR) returns a sequence for changing the cursor style.
//
//	CSI Ps SP q
//
// Where Ps is the cursor style:
//
//	0: Blinking block
//	1: Blinking block (default)
//	2: Steady block
//	3: Blinking underline
//	4: Steady underline
//	5: Blinking bar (xterm)
//	6: Steady bar (xterm)
//
// See: https://vt100.net/docs/vt510-rm/DECSCUSR.html
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h4-Functions-using-CSI-_-ordered-by-the-final-character-lparen-s-rparen:CSI-Ps-SP-q.1D81
func SetCursorStyle(style int) string {
	if style < 0 {
		style = 0
	}
	return "\x1b[" + strconv.Itoa(style) + " q"
}

// SetPointerShape returns a sequence for changing the mouse pointer cursor
// shape. Use "default" for the default pointer shape.
//
//	OSC 22 ; Pt ST
//	OSC 22 ; Pt BEL
//
// Where Pt is the pointer shape name. The name can be anything that the
// operating system can understand. Some common names are:
//
//   - copy
//   - crosshair
//   - default
//   - ew-resize
//   - n-resize
//   - text
//   - wait
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Operating-System-Commands
func SetPointerShape(shape string) string {
	return "\x1b]22;" + shape + "\x07"
}
