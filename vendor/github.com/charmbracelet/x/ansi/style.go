package ansi

import (
	"image/color"
	"strconv"
	"strings"
)

// ResetStyle is a SGR (Select Graphic Rendition) style sequence that resets
// all attributes.
// See: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
const ResetStyle = "\x1b[m"

// Attr is a SGR (Select Graphic Rendition) style attribute.
type Attr = string

// Style represents an ANSI SGR (Select Graphic Rendition) style.
type Style []Attr

// String returns the ANSI SGR (Select Graphic Rendition) style sequence for
// the given style.
func (s Style) String() string {
	if len(s) == 0 {
		return ResetStyle
	}
	return "\x1b[" + strings.Join(s, ";") + "m"
}

// Styled returns a styled string with the given style applied.
func (s Style) Styled(str string) string {
	if len(s) == 0 {
		return str
	}
	return s.String() + str + ResetStyle
}

// Reset appends the reset style attribute to the style.
func (s Style) Reset() Style {
	return append(s, ResetAttr)
}

// Bold appends the bold style attribute to the style.
func (s Style) Bold() Style {
	return append(s, BoldAttr)
}

// Faint appends the faint style attribute to the style.
func (s Style) Faint() Style {
	return append(s, FaintAttr)
}

// Italic appends the italic style attribute to the style.
func (s Style) Italic() Style {
	return append(s, ItalicAttr)
}

// Underline appends the underline style attribute to the style.
func (s Style) Underline() Style {
	return append(s, UnderlineAttr)
}

// DoubleUnderline appends the double underline style attribute to the style.
func (s Style) DoubleUnderline() Style {
	return append(s, DoubleUnderlineAttr)
}

// CurlyUnderline appends the curly underline style attribute to the style.
func (s Style) CurlyUnderline() Style {
	return append(s, CurlyUnderlineAttr)
}

// DottedUnderline appends the dotted underline style attribute to the style.
func (s Style) DottedUnderline() Style {
	return append(s, DottedUnderlineAttr)
}

// DashedUnderline appends the dashed underline style attribute to the style.
func (s Style) DashedUnderline() Style {
	return append(s, DashedUnderlineAttr)
}

// SlowBlink appends the slow blink style attribute to the style.
func (s Style) SlowBlink() Style {
	return append(s, SlowBlinkAttr)
}

// RapidBlink appends the rapid blink style attribute to the style.
func (s Style) RapidBlink() Style {
	return append(s, RapidBlinkAttr)
}

// Reverse appends the reverse style attribute to the style.
func (s Style) Reverse() Style {
	return append(s, ReverseAttr)
}

// Conceal appends the conceal style attribute to the style.
func (s Style) Conceal() Style {
	return append(s, ConcealAttr)
}

// Strikethrough appends the strikethrough style attribute to the style.
func (s Style) Strikethrough() Style {
	return append(s, StrikethroughAttr)
}

// NoBold appends the no bold style attribute to the style.
func (s Style) NoBold() Style {
	return append(s, NoBoldAttr)
}

// NormalIntensity appends the normal intensity style attribute to the style.
func (s Style) NormalIntensity() Style {
	return append(s, NormalIntensityAttr)
}

// NoItalic appends the no italic style attribute to the style.
func (s Style) NoItalic() Style {
	return append(s, NoItalicAttr)
}

// NoUnderline appends the no underline style attribute to the style.
func (s Style) NoUnderline() Style {
	return append(s, NoUnderlineAttr)
}

// NoBlink appends the no blink style attribute to the style.
func (s Style) NoBlink() Style {
	return append(s, NoBlinkAttr)
}

// NoReverse appends the no reverse style attribute to the style.
func (s Style) NoReverse() Style {
	return append(s, NoReverseAttr)
}

// NoConceal appends the no conceal style attribute to the style.
func (s Style) NoConceal() Style {
	return append(s, NoConcealAttr)
}

// NoStrikethrough appends the no strikethrough style attribute to the style.
func (s Style) NoStrikethrough() Style {
	return append(s, NoStrikethroughAttr)
}

// DefaultForegroundColor appends the default foreground color style attribute to the style.
func (s Style) DefaultForegroundColor() Style {
	return append(s, DefaultForegroundColorAttr)
}

// DefaultBackgroundColor appends the default background color style attribute to the style.
func (s Style) DefaultBackgroundColor() Style {
	return append(s, DefaultBackgroundColorAttr)
}

// DefaultUnderlineColor appends the default underline color style attribute to the style.
func (s Style) DefaultUnderlineColor() Style {
	return append(s, DefaultUnderlineColorAttr)
}

// ForegroundColor appends the foreground color style attribute to the style.
func (s Style) ForegroundColor(c Color) Style {
	return append(s, ForegroundColorAttr(c))
}

// BackgroundColor appends the background color style attribute to the style.
func (s Style) BackgroundColor(c Color) Style {
	return append(s, BackgroundColorAttr(c))
}

// UnderlineColor appends the underline color style attribute to the style.
func (s Style) UnderlineColor(c Color) Style {
	return append(s, UnderlineColorAttr(c))
}

// SGR (Select Graphic Rendition) style attributes.
// See: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
const (
	ResetAttr                  Attr = "0"
	BoldAttr                   Attr = "1"
	FaintAttr                  Attr = "2"
	ItalicAttr                 Attr = "3"
	UnderlineAttr              Attr = "4"
	DoubleUnderlineAttr        Attr = "4:2"
	CurlyUnderlineAttr         Attr = "4:3"
	DottedUnderlineAttr        Attr = "4:4"
	DashedUnderlineAttr        Attr = "4:5"
	SlowBlinkAttr              Attr = "5"
	RapidBlinkAttr             Attr = "6"
	ReverseAttr                Attr = "7"
	ConcealAttr                Attr = "8"
	StrikethroughAttr          Attr = "9"
	NoBoldAttr                 Attr = "21" // Some terminals treat this as double underline.
	NormalIntensityAttr        Attr = "22"
	NoItalicAttr               Attr = "23"
	NoUnderlineAttr            Attr = "24"
	NoBlinkAttr                Attr = "25"
	NoReverseAttr              Attr = "27"
	NoConcealAttr              Attr = "28"
	NoStrikethroughAttr        Attr = "29"
	DefaultForegroundColorAttr Attr = "39"
	DefaultBackgroundColorAttr Attr = "49"
	DefaultUnderlineColorAttr  Attr = "59"
)

// ForegroundColorAttr returns the style SGR attribute for the given foreground
// color.
// See: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func ForegroundColorAttr(c Color) Attr {
	switch c := c.(type) {
	case BasicColor:
		// 3-bit or 4-bit ANSI foreground
		// "3<n>" or "9<n>" where n is the color number from 0 to 7
		if c < 8 {
			return "3" + string('0'+c)
		} else if c < 16 {
			return "9" + string('0'+c-8)
		}
	case ExtendedColor:
		// 256-color ANSI foreground
		// "38;5;<n>"
		return "38;5;" + strconv.FormatUint(uint64(c), 10)
	case TrueColor, color.Color:
		// 24-bit "true color" foreground
		// "38;2;<r>;<g>;<b>"
		r, g, b, _ := c.RGBA()
		return "38;2;" +
			strconv.FormatUint(uint64(shift(r)), 10) + ";" +
			strconv.FormatUint(uint64(shift(g)), 10) + ";" +
			strconv.FormatUint(uint64(shift(b)), 10)
	}
	return DefaultForegroundColorAttr
}

// BackgroundColorAttr returns the style SGR attribute for the given background
// color.
// See: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func BackgroundColorAttr(c Color) Attr {
	switch c := c.(type) {
	case BasicColor:
		// 3-bit or 4-bit ANSI foreground
		// "4<n>" or "10<n>" where n is the color number from 0 to 7
		if c < 8 {
			return "4" + string('0'+c)
		} else {
			return "10" + string('0'+c-8)
		}
	case ExtendedColor:
		// 256-color ANSI foreground
		// "48;5;<n>"
		return "48;5;" + strconv.FormatUint(uint64(c), 10)
	case TrueColor, color.Color:
		// 24-bit "true color" foreground
		// "38;2;<r>;<g>;<b>"
		r, g, b, _ := c.RGBA()
		return "48;2;" +
			strconv.FormatUint(uint64(shift(r)), 10) + ";" +
			strconv.FormatUint(uint64(shift(g)), 10) + ";" +
			strconv.FormatUint(uint64(shift(b)), 10)
	}
	return DefaultBackgroundColorAttr
}

// UnderlineColorAttr returns the style SGR attribute for the given underline
// color.
// See: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func UnderlineColorAttr(c Color) Attr {
	switch c := c.(type) {
	// NOTE: we can't use 3-bit and 4-bit ANSI color codes with underline
	// color, use 256-color instead.
	//
	// 256-color ANSI underline color
	// "58;5;<n>"
	case BasicColor:
		return "58;5;" + strconv.FormatUint(uint64(c), 10)
	case ExtendedColor:
		return "58;5;" + strconv.FormatUint(uint64(c), 10)
	case TrueColor, color.Color:
		// 24-bit "true color" foreground
		// "38;2;<r>;<g>;<b>"
		r, g, b, _ := c.RGBA()
		return "58;2;" +
			strconv.FormatUint(uint64(shift(r)), 10) + ";" +
			strconv.FormatUint(uint64(shift(g)), 10) + ";" +
			strconv.FormatUint(uint64(shift(b)), 10)
	}
	return DefaultUnderlineColorAttr
}

func shift(v uint32) uint32 {
	if v > 0xff {
		return v >> 8
	}
	return v
}
