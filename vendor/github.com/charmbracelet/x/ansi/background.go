package ansi

import (
	"image/color"
)

// SetForegroundColor returns a sequence that sets the default terminal
// foreground color.
//
//	OSC 10 ; color ST
//	OSC 10 ; color BEL
//
// Where color is the encoded color number.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
func SetForegroundColor(c color.Color) string {
	return "\x1b]10;" + colorToHexString(c) + "\x07"
}

// RequestForegroundColor is a sequence that requests the current default
// terminal foreground color.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
const RequestForegroundColor = "\x1b]10;?\x07"

// SetBackgroundColor returns a sequence that sets the default terminal
// background color.
//
//	OSC 11 ; color ST
//	OSC 11 ; color BEL
//
// Where color is the encoded color number.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
func SetBackgroundColor(c color.Color) string {
	return "\x1b]11;" + colorToHexString(c) + "\x07"
}

// RequestBackgroundColor is a sequence that requests the current default
// terminal background color.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
const RequestBackgroundColor = "\x1b]11;?\x07"

// SetCursorColor returns a sequence that sets the terminal cursor color.
//
//	OSC 12 ; color ST
//	OSC 12 ; color BEL
//
// Where color is the encoded color number.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
func SetCursorColor(c color.Color) string {
	return "\x1b]12;" + colorToHexString(c) + "\x07"
}

// RequestCursorColor is a sequence that requests the current terminal cursor
// color.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
const RequestCursorColor = "\x1b]12;?\x07"
