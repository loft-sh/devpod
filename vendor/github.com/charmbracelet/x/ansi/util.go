package ansi

import (
	"fmt"
	"image/color"
)

// colorToHexString returns a hex string representation of a color.
func colorToHexString(c color.Color) string {
	if c == nil {
		return ""
	}
	shift := func(v uint32) uint32 {
		if v > 0xff {
			return v >> 8
		}
		return v
	}
	r, g, b, _ := c.RGBA()
	r, g, b = shift(r), shift(g), shift(b)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// rgbToHex converts red, green, and blue values to a hexadecimal value.
//
//	hex := rgbToHex(0, 0, 255) // 0x0000FF
func rgbToHex(r, g, b uint32) uint32 {
	return r<<16 + g<<8 + b
}
