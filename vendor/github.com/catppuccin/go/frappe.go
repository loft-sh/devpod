package catppuccingo

// frappe variant.
type frappe struct{}

// Frappe flavour variant.
var Frappe Flavour = frappe{}

func (frappe) Name() string { return "frappe" }

func (frappe) Rosewater() Color {
	return Color{
		Hex: "#f2d5cf",
		RGB: [3]uint32{242, 213, 207},
		HSL: [3]float32{10, 0.57, 0.88},
	}
}

func (frappe) Flamingo() Color {
	return Color{
		Hex: "#eebebe",
		RGB: [3]uint32{238, 190, 190},
		HSL: [3]float32{0, 0.59, 0.84},
	}
}

func (frappe) Pink() Color {
	return Color{
		Hex: "#f4b8e4",
		RGB: [3]uint32{244, 184, 228},
		HSL: [3]float32{316, 0.73, 0.84},
	}
}

func (frappe) Mauve() Color {
	return Color{
		Hex: "#ca9ee6",
		RGB: [3]uint32{202, 158, 230},
		HSL: [3]float32{277, 0.59, 0.76},
	}
}

func (frappe) Red() Color {
	return Color{
		Hex: "#e78284",
		RGB: [3]uint32{231, 130, 132},
		HSL: [3]float32{359, 0.68, 0.71},
	}
}

func (frappe) Maroon() Color {
	return Color{
		Hex: "#ea999c",
		RGB: [3]uint32{234, 153, 156},
		HSL: [3]float32{358, 0.66, 0.76},
	}
}

func (frappe) Peach() Color {
	return Color{
		Hex: "#ef9f76",
		RGB: [3]uint32{239, 159, 118},
		HSL: [3]float32{20, 0.79, 0.70},
	}
}

func (frappe) Yellow() Color {
	return Color{
		Hex: "#e5c890",
		RGB: [3]uint32{229, 200, 144},
		HSL: [3]float32{40, 0.62, 0.73},
	}
}

func (frappe) Green() Color {
	return Color{
		Hex: "#a6d189",
		RGB: [3]uint32{166, 209, 137},
		HSL: [3]float32{96, 0.44, 0.68},
	}
}

func (frappe) Teal() Color {
	return Color{
		Hex: "#81c8be",
		RGB: [3]uint32{129, 200, 190},
		HSL: [3]float32{172, 0.39, 0.65},
	}
}

func (frappe) Sky() Color {
	return Color{
		Hex: "#99d1db",
		RGB: [3]uint32{153, 209, 219},
		HSL: [3]float32{189, 0.48, 0.73},
	}
}

func (frappe) Sapphire() Color {
	return Color{
		Hex: "#85c1dc",
		RGB: [3]uint32{133, 193, 220},
		HSL: [3]float32{199, 0.55, 0.69},
	}
}

func (frappe) Blue() Color {
	return Color{
		Hex: "#8caaee",
		RGB: [3]uint32{140, 170, 238},
		HSL: [3]float32{222, 0.74, 0.74},
	}
}

func (frappe) Lavender() Color {
	return Color{
		Hex: "#babbf1",
		RGB: [3]uint32{186, 187, 241},
		HSL: [3]float32{239, 0.66, 0.84},
	}
}

func (frappe) Text() Color {
	return Color{
		Hex: "#c6d0f5",
		RGB: [3]uint32{198, 208, 245},
		HSL: [3]float32{227, 0.70, 0.87},
	}
}

func (frappe) Subtext1() Color {
	return Color{
		Hex: "#b5bfe2",
		RGB: [3]uint32{181, 191, 226},
		HSL: [3]float32{227, 0.44, 0.80},
	}
}

func (frappe) Subtext0() Color {
	return Color{
		Hex: "#a5adce",
		RGB: [3]uint32{165, 173, 206},
		HSL: [3]float32{228, 0.29, 0.73},
	}
}

func (frappe) Overlay2() Color {
	return Color{
		Hex: "#949cbb",
		RGB: [3]uint32{148, 156, 187},
		HSL: [3]float32{228, 0.22, 0.66},
	}
}

func (frappe) Overlay1() Color {
	return Color{
		Hex: "#838ba7",
		RGB: [3]uint32{131, 139, 167},
		HSL: [3]float32{227, 0.17, 0.58},
	}
}

func (frappe) Overlay0() Color {
	return Color{
		Hex: "#737994",
		RGB: [3]uint32{115, 121, 148},
		HSL: [3]float32{229, 0.13, 0.52},
	}
}

func (frappe) Surface2() Color {
	return Color{
		Hex: "#626880",
		RGB: [3]uint32{98, 104, 128},
		HSL: [3]float32{228, 0.13, 0.44},
	}
}

func (frappe) Surface1() Color {
	return Color{
		Hex: "#51576d",
		RGB: [3]uint32{81, 87, 109},
		HSL: [3]float32{227, 0.15, 0.37},
	}
}

func (frappe) Surface0() Color {
	return Color{
		Hex: "#414559",
		RGB: [3]uint32{65, 69, 89},
		HSL: [3]float32{230, 0.16, 0.30},
	}
}

func (frappe) Base() Color {
	return Color{
		Hex: "#303446",
		RGB: [3]uint32{48, 52, 70},
		HSL: [3]float32{229, 0.19, 0.23},
	}
}

func (frappe) Mantle() Color {
	return Color{
		Hex: "#292c3c",
		RGB: [3]uint32{41, 44, 60},
		HSL: [3]float32{231, 0.19, 0.20},
	}
}

func (frappe) Crust() Color {
	return Color{
		Hex: "#232634",
		RGB: [3]uint32{35, 38, 52},
		HSL: [3]float32{229, 0.20, 0.17},
	}
}
