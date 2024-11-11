package ansi

// RequestXTVersion is a control sequence that requests the terminal's XTVERSION. It responds with a DSR sequence identifying the version.
//
//	CSI > Ps q
//	DCS > | text ST
//
// See https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-PC-Style-Function-Keys
const RequestXTVersion = "\x1b[>0q"

// RequestPrimaryDeviceAttributes is a control sequence that requests the
// terminal's primary device attributes (DA1).
//
//	CSI c
//
// See https://vt100.net/docs/vt510-rm/DA1.html
const RequestPrimaryDeviceAttributes = "\x1b[c"
