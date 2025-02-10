package catppuccingo

type mocha struct{}

// Mocha flavour variant.
var Mocha Flavour = mocha{}

func (mocha) Name() string { return "mocha" }

func (mocha) Rosewater() Color {
	return Color{
		Hex: "#f5e0dc",
		RGB: [3]uint32{245, 224, 220},
		HSL: [3]float32{10, 0.56, 0.91},
	}
}

func (mocha) Flamingo() Color {
	return Color{
		Hex: "#f2cdcd",
		RGB: [3]uint32{242, 205, 205},
		HSL: [3]float32{0, 0.59, 0.88},
	}
}

func (mocha) Pink() Color {
	return Color{
		Hex: "#f5c2e7",
		RGB: [3]uint32{245, 194, 231},
		HSL: [3]float32{316, 0.72, 0.86},
	}
}

func (mocha) Mauve() Color {
	return Color{
		Hex: "#cba6f7",
		RGB: [3]uint32{203, 166, 247},
		HSL: [3]float32{267, 0.84, 0.81},
	}
}

func (mocha) Red() Color {
	return Color{
		Hex: "#f38ba8",
		RGB: [3]uint32{243, 139, 168},
		HSL: [3]float32{343, 0.81, 0.75},
	}
}

func (mocha) Maroon() Color {
	return Color{
		Hex: "#eba0ac",
		RGB: [3]uint32{235, 160, 172},
		HSL: [3]float32{350, 0.65, 0.77},
	}
}

func (mocha) Peach() Color {
	return Color{
		Hex: "#fab387",
		RGB: [3]uint32{250, 179, 135},
		HSL: [3]float32{23, 0.92, 0.75},
	}
}

func (mocha) Yellow() Color {
	return Color{
		Hex: "#f9e2af",
		RGB: [3]uint32{249, 226, 175},
		HSL: [3]float32{41, 0.86, 0.83},
	}
}

func (mocha) Green() Color {
	return Color{
		Hex: "#a6e3a1",
		RGB: [3]uint32{166, 227, 161},
		HSL: [3]float32{115, 0.54, 0.76},
	}
}

func (mocha) Teal() Color {
	return Color{
		Hex: "#94e2d5",
		RGB: [3]uint32{148, 226, 213},
		HSL: [3]float32{170, 0.57, 0.73},
	}
}

func (mocha) Sky() Color {
	return Color{
		Hex: "#89dceb",
		RGB: [3]uint32{137, 220, 235},
		HSL: [3]float32{189, 0.71, 0.73},
	}
}

func (mocha) Sapphire() Color {
	return Color{
		Hex: "#74c7ec",
		RGB: [3]uint32{116, 199, 236},
		HSL: [3]float32{199, 0.76, 0.69},
	}
}

func (mocha) Blue() Color {
	return Color{
		Hex: "#89b4fa",
		RGB: [3]uint32{137, 180, 250},
		HSL: [3]float32{217, 0.92, 0.76},
	}
}

func (mocha) Lavender() Color {
	return Color{
		Hex: "#b4befe",
		RGB: [3]uint32{180, 190, 254},
		HSL: [3]float32{232, 0.97, 0.85},
	}
}

func (mocha) Text() Color {
	return Color{
		Hex: "#cdd6f4",
		RGB: [3]uint32{205, 214, 244},
		HSL: [3]float32{226, 0.64, 0.88},
	}
}

func (mocha) Subtext1() Color {
	return Color{
		Hex: "#bac2de",
		RGB: [3]uint32{186, 194, 222},
		HSL: [3]float32{227, 0.35, 0.80},
	}
}

func (mocha) Subtext0() Color {
	return Color{
		Hex: "#a6adc8",
		RGB: [3]uint32{166, 173, 200},
		HSL: [3]float32{228, 0.24, 0.72},
	}
}

func (mocha) Overlay2() Color {
	return Color{
		Hex: "#9399b2",
		RGB: [3]uint32{147, 153, 178},
		HSL: [3]float32{228, 0.17, 0.64},
	}
}

func (mocha) Overlay1() Color {
	return Color{
		Hex: "#7f849c",
		RGB: [3]uint32{127, 132, 156},
		HSL: [3]float32{230, 0.13, 0.55},
	}
}

func (mocha) Overlay0() Color {
	return Color{
		Hex: "#6c7086",
		RGB: [3]uint32{108, 112, 134},
		HSL: [3]float32{231, 0.11, 0.47},
	}
}

func (mocha) Surface2() Color {
	return Color{
		Hex: "#585b70",
		RGB: [3]uint32{88, 91, 112},
		HSL: [3]float32{233, 0.12, 0.39},
	}
}

func (mocha) Surface1() Color {
	return Color{
		Hex: "#45475a",
		RGB: [3]uint32{69, 71, 90},
		HSL: [3]float32{234, 0.13, 0.31},
	}
}

func (mocha) Surface0() Color {
	return Color{
		Hex: "#313244",
		RGB: [3]uint32{49, 50, 68},
		HSL: [3]float32{237, 0.16, 0.23},
	}
}

func (mocha) Base() Color {
	return Color{
		Hex: "#1e1e2e",
		RGB: [3]uint32{30, 30, 46},
		HSL: [3]float32{240, 0.21, 0.15},
	}
}

func (mocha) Mantle() Color {
	return Color{
		Hex: "#181825",
		RGB: [3]uint32{24, 24, 37},
		HSL: [3]float32{240, 0.21, 0.12},
	}
}

func (mocha) Crust() Color {
	return Color{
		Hex: "#11111b",
		RGB: [3]uint32{17, 17, 27},
		HSL: [3]float32{240, 0.23, 0.9},
	}
}
