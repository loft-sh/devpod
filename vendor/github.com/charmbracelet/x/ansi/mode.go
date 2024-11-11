package ansi

// This file define uses multiple sequences to set (SM), reset (RM), and request
// (DECRQM) different ANSI and DEC modes.
//
// See: https://vt100.net/docs/vt510-rm/SM.html
// See: https://vt100.net/docs/vt510-rm/RM.html
// See: https://vt100.net/docs/vt510-rm/DECRQM.html
//
// The terminal then responds to the request with a Report Mode function
// (DECRPM) in the format:
//
// ANSI format:
//
//  CSI Pa ; Ps ; $ y
//
// DEC format:
//
//  CSI ? Pa ; Ps $ y
//
// Where Pa is the mode number, and Ps is the mode value.
// See: https://vt100.net/docs/vt510-rm/DECRPM.html

// Application Cursor Keys (DECCKM) is a mode that determines whether the
// cursor keys send ANSI cursor sequences or application sequences.
//
// See: https://vt100.net/docs/vt510-rm/DECCKM.html
const (
	EnableCursorKeys  = "\x1b[?1h"
	DisableCursorKeys = "\x1b[?1l"
	RequestCursorKeys = "\x1b[?1$p"
)

// Text Cursor Enable Mode (DECTCEM) is a mode that shows/hides the cursor.
//
// See: https://vt100.net/docs/vt510-rm/DECTCEM.html
const (
	ShowCursor              = "\x1b[?25h"
	HideCursor              = "\x1b[?25l"
	RequestCursorVisibility = "\x1b[?25$p"
)

// VT Mouse Tracking is a mode that determines whether the mouse reports on
// button press and release.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	EnableMouse  = "\x1b[?1000h"
	DisableMouse = "\x1b[?1000l"
	RequestMouse = "\x1b[?1000$p"
)

// VT Hilite Mouse Tracking is a mode that determines whether the mouse reports on
// button presses, releases, and highlighted cells.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	EnableMouseHilite  = "\x1b[?1001h"
	DisableMouseHilite = "\x1b[?1001l"
	RequestMouseHilite = "\x1b[?1001$p"
)

// Cell Motion Mouse Tracking is a mode that determines whether the mouse
// reports on button press, release, and motion events.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	EnableMouseCellMotion  = "\x1b[?1002h"
	DisableMouseCellMotion = "\x1b[?1002l"
	RequestMouseCellMotion = "\x1b[?1002$p"
)

// All Mouse Tracking is a mode that determines whether the mouse reports on
// button press, release, motion, and highlight events.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	EnableMouseAllMotion  = "\x1b[?1003h"
	DisableMouseAllMotion = "\x1b[?1003l"
	RequestMouseAllMotion = "\x1b[?1003$p"
)

// Report Focus is a mode that makes the terminal report focus-in and focus-out events.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-FocusIn_FocusOut
const (
	EnableReportFocus  = "\x1b[?1004h"
	DisableReportFocus = "\x1b[?1004l"
	RequestReportFocus = "\x1b[?1004$p"
)

// SGR Mouse Extension is a mode that determines whether the mouse reports events
// formatted with SGR parameters.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	EnableMouseSgrExt  = "\x1b[?1006h"
	DisableMouseSgrExt = "\x1b[?1006l"
	RequestMouseSgrExt = "\x1b[?1006$p"
)

// Alternate Screen Buffer is a mode that determines whether the alternate screen
// buffer is active.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-The-Alternate-Screen-Buffer
const (
	EnableAltScreenBuffer  = "\x1b[?1049h"
	DisableAltScreenBuffer = "\x1b[?1049l"
	RequestAltScreenBuffer = "\x1b[?1049$p"
)

// Bracketed Paste Mode is a mode that determines whether pasted text is
// bracketed with escape sequences.
//
// See: https://cirw.in/blog/bracketed-paste
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Bracketed-Paste-Mode
const (
	EnableBracketedPaste  = "\x1b[?2004h"
	DisableBracketedPaste = "\x1b[?2004l"
	RequestBracketedPaste = "\x1b[?2004$p"
)

// Synchronized Output Mode is a mode that determines whether output is
// synchronized with the terminal.
//
// See: https://gist.github.com/christianparpart/d8a62cc1ab659194337d73e399004036
const (
	EnableSyncdOutput  = "\x1b[?2026h"
	DisableSyncdOutput = "\x1b[?2026l"
	RequestSyncdOutput = "\x1b[?2026$p"
)

// Grapheme Clustering Mode is a mode that determines whether the terminal
// should look for grapheme clusters instead of single runes in the rendered
// text. This makes the terminal properly render combining characters such as
// emojis.
//
// See: https://github.com/contour-terminal/terminal-unicode-core
const (
	EnableGraphemeClustering  = "\x1b[?2027h"
	DisableGraphemeClustering = "\x1b[?2027l"
	RequestGraphemeClustering = "\x1b[?2027$p"
)

// Win32Input is a mode that determines whether input is processed by the
// Win32 console and Conpty.
//
// See: https://github.com/microsoft/terminal/blob/main/doc/specs/%234999%20-%20Improved%20keyboard%20handling%20in%20Conpty.md
const (
	EnableWin32Input  = "\x1b[?9001h"
	DisableWin32Input = "\x1b[?9001l"
	RequestWin32Input = "\x1b[?9001$p"
)
