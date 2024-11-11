package catppuccingo

import (
	"image/color"
	"strings"
)

// Flavour is an interface implemented by all Catppuccin variations.
type Flavour interface {
	Rosewater() Color
	Flamingo() Color
	Pink() Color
	Mauve() Color
	Red() Color
	Maroon() Color
	Peach() Color
	Yellow() Color
	Green() Color
	Teal() Color
	Sky() Color
	Sapphire() Color
	Blue() Color
	Lavender() Color
	Text() Color
	Subtext1() Color
	Subtext0() Color
	Overlay2() Color
	Overlay1() Color
	Overlay0() Color
	Surface2() Color
	Surface1() Color
	Surface0() Color
	Crust() Color
	Mantle() Color
	Base() Color
	Name() string
}

// Theme is a type alias of Flavour to keep compatibility with previous versions.
type Theme = Flavour

// Color is a color in Hex, RGB, and HSL.
type Color struct {
	Hex string
	RGB [3]uint32
	HSL [3]float32
}

// RGBA implements color.Color
func (c Color) RGBA() (r uint32, g uint32, b uint32, a uint32) {
	return c.RGB[0], c.RGB[1], c.RGB[2], 1
}

var _ color.Color = Color{}

// Variant returns the Theme variant by name.
func Variant(flavour string) Theme {
	for _, t := range []Theme{
		Mocha,
		Frappe,
		Macchiato,
		Latte,
	} {
		if strings.EqualFold(t.Name(), flavour) {
			return t
		}
	}
	return nil
}
