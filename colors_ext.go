package reggol

import (
	"os"
	"strconv"
	"strings"
)

// ColorDepth is how many colors the destination terminal understands.
//
// It matters because emitting a 24-bit sequence to a terminal that only speaks
// 16 colors does not degrade gracefully — it prints the escape codes as text,
// in the middle of a log line.
type ColorDepth uint8

// Color depths.
const (
	// DepthAuto reads COLORTERM and TERM to decide.
	DepthAuto ColorDepth = iota
	// Depth16 is the basic ANSI set every terminal supports.
	Depth16
	// Depth256 is the xterm 256-color palette.
	Depth256
	// DepthTrueColor is 24-bit color.
	DepthTrueColor
)

// Color is a foreground or background color: one of the named ANSI colors, an
// index into the 256-color palette, or a 24-bit RGB value.
//
// The zero value means "unset", which is what lets a Style leave the foreground
// or background alone.
type Color uint32

// SGR selectors for the extended color forms.
const (
	sgrExtendedIndex = 5 // ESC[38;5;N — palette
	sgrExtendedRGB   = 2 // ESC[38;2;R;G;B — direct color
)

// Bit positions of the RGB components inside a Color's value.
const (
	redShift   = 16
	greenShift = 8
)

// A Color packs its kind into the high bits and the value into the low 24.
const (
	colorKindShift = 28
	colorValueMask = 1<<colorKindShift - 1

	colorUnset   uint32 = iota
	colorNamed          // value is the SGR code itself
	colorIndexed        // value is a palette index
	colorRGB            // value is 0xRRGGBB
)

func newColor(kind, value uint32) Color {
	return Color(kind<<colorKindShift | value&colorValueMask)
}

func (c Color) kind() uint32  { return uint32(c) >> colorKindShift }
func (c Color) value() uint32 { return uint32(c) & colorValueMask }

// Color256 returns a color from the xterm 256-color palette.
func Color256(index uint8) Color { return newColor(colorIndexed, uint32(index)) }

// ColorRGB returns a 24-bit color.
func ColorRGB(r, g, b uint8) Color {
	return newColor(colorRGB, uint32(r)<<redShift|uint32(g)<<greenShift|uint32(b))
}

// namedColor wraps a raw SGR color code.
func namedColor(sgr int) Color { return newColor(colorNamed, uint32(sgr)) }

// RGB reports the color's components. ok is false unless the color was built
// with ColorRGB.
func (c Color) RGB() (r, g, b uint8, ok bool) {
	if c.kind() != colorRGB {
		return 0, 0, 0, false
	}

	v := c.value()

	return uint8(v >> redShift), uint8(v >> greenShift), uint8(v), true
}

// Index reports the palette index. ok is false unless the color was built with
// Color256.
func (c Color) Index() (uint8, bool) {
	if c.kind() != colorIndexed {
		return 0, false
	}

	return uint8(c.value()), true
}

// Style describes how a piece of text is rendered.
//
// It exists alongside TextStyle rather than replacing it: a 32-bit mask cannot
// hold a 24-bit color, and widening TextStyle would break every caller that
// stored one.
type Style struct {
	Fg    Color
	Bg    Color
	Attrs TextStyle // only attribute bits are used; color bits are ignored
}

// StyleOf converts a TextStyle bitmask into a Style, splitting its color bits
// from its attribute bits.
func StyleOf(ts TextStyle) Style {
	s := Style{Attrs: ts}

	if code, ok := colorCode(ts, foregrounds[:], sgrFgBase, sgrFgDefault, ColorFgBright); ok {
		s.Fg = namedColor(code)
	}

	if code, ok := colorCode(ts, backgrounds[:], sgrBgBase, sgrBgDefault, ColorBgBright); ok {
		s.Bg = namedColor(code)
	}

	return s
}

// WithFg returns a copy of the style using the given foreground.
func (s Style) WithFg(c Color) Style {
	s.Fg = c

	return s
}

// WithBg returns a copy of the style using the given background.
func (s Style) WithBg(c Color) Style {
	s.Bg = c

	return s
}

// ColorCodes returns the escape sequences that open and close the style,
// reduced to what depth can express.
func (s Style) ColorCodes(depth ColorDepth) (start, reset string) {
	if depth == DepthAuto {
		depth = Depth16
	}

	var (
		setCodes []int
		offCodes []int
	)

	for _, a := range attributes {
		if s.Attrs.Has(a.bit) {
			setCodes = append(setCodes, a.set)
			offCodes = append([]int{a.off}, offCodes...)
		}
	}

	if codes, ok := s.Fg.sgr(depth, sgrFgBase); ok {
		setCodes = append(setCodes, codes...)

		if len(codes) != 1 || codes[0] != sgrFgDefault {
			offCodes = append([]int{sgrFgDefault}, offCodes...)
		}
	}

	if codes, ok := s.Bg.sgr(depth, sgrBgBase); ok {
		setCodes = append(setCodes, codes...)

		if len(codes) != 1 || codes[0] != sgrBgDefault {
			offCodes = append([]int{sgrBgDefault}, offCodes...)
		}
	}

	return renderSGR(setCodes), renderSGR(offCodes)
}

// sgr renders the color as SGR parameters for the given layer, downgrading it
// to fit depth.
//
// base is sgrFgBase or sgrBgBase; the extended forms use 38 and 48, which are
// base+8.
func (c Color) sgr(depth ColorDepth, base int) ([]int, bool) {
	const extendedOffset = 8 // 30 -> 38, 40 -> 48

	switch c.kind() {
	case colorUnset:
		return nil, false

	case colorNamed:
		return []int{int(c.value())}, true

	case colorIndexed:
		index := uint8(c.value())

		if depth == Depth16 {
			return []int{namedFromIndex(index, base)}, true
		}

		return []int{base + extendedOffset, sgrExtendedIndex, int(index)}, true

	case colorRGB:
		r, g, b, _ := c.RGB()

		switch depth {
		case DepthTrueColor:
			return []int{base + extendedOffset, sgrExtendedRGB, int(r), int(g), int(b)}, true
		case Depth256:
			return []int{base + extendedOffset, sgrExtendedIndex, int(indexFromRGB(r, g, b))}, true
		case DepthAuto, Depth16:
			return []int{namedFromIndex(indexFromRGB(r, g, b), base)}, true
		default:
			return []int{namedFromIndex(indexFromRGB(r, g, b), base)}, true
		}

	default:
		return nil, false
	}
}

// indexFromRGB maps a 24-bit color onto the xterm 256-color palette.
//
// Near-grey colors go to the 24-step grey ramp, everything else to the 6x6x6
// cube; this is the mapping every terminal library uses, and it keeps greys from
// picking up a color cast.
func indexFromRGB(r, g, b uint8) uint8 {
	const (
		cubeBase    = 16
		cubeSteps   = 6
		greyBase    = 232
		greySteps   = 24
		greyMax     = 0xee
		greyMin     = 0x08
		greySpread  = 10 // a color this close to grey is treated as grey
		cubeDivisor = 51 // 255 / (cubeSteps - 1)
	)

	if absDiff(r, g) < greySpread && absDiff(g, b) < greySpread && absDiff(r, b) < greySpread {
		const channels = 3

		grey := (int(r) + int(g) + int(b)) / channels

		switch {
		case grey < greyMin:
			return cubeBase // black
		case grey > greyMax:
			return cubeBase + cubeSteps*cubeSteps*cubeSteps - 1 // white
		}

		return uint8(greyBase + (grey-greyMin)*(greySteps-1)/(greyMax-greyMin))
	}

	quant := func(v uint8) int { return (int(v) + cubeDivisor/2) / cubeDivisor }

	return uint8(cubeBase + cubeSteps*cubeSteps*quant(r) + cubeSteps*quant(g) + quant(b))
}

// namedFromIndex reduces a palette index to a basic SGR color code.
func namedFromIndex(index uint8, base int) int {
	const (
		basicCount  = 8
		brightStart = 8
		brightLimit = 16
		cubeBase    = 16
		greyBase    = 232
	)

	switch {
	case index < brightStart:
		return base + int(index)
	case index < brightLimit:
		return base + sgrBrightOffset + int(index-brightStart)
	case index >= greyBase:
		// Grey ramp: dark half becomes black, light half white.
		const greyMidpoint = 243
		if index < greyMidpoint {
			return base
		}

		return base + sgrBrightOffset + 7 //nolint:mnd // bright white
	default:
		// Cube: recover the components and pick the nearest basic color.
		i := int(index) - cubeBase
		r, g, b := i/36, (i/6)%6, i%6 //nolint:mnd // 6x6x6 cube arithmetic

		const half = 3

		code := 0
		if r >= half {
			code |= 1
		}

		if g >= half {
			code |= 2
		}

		if b >= half {
			code |= 4
		}

		return base + code
	}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}

	return int(b - a)
}

// detectColorDepth reports what the environment says the terminal supports.
//
// COLORTERM is the de-facto signal for 24-bit support; TERM naming a 256-color
// variant is the older one. Anything else is assumed to be the basic set,
// because guessing high is what produces escape codes in the output.
func detectColorDepth() ColorDepth {
	switch strings.ToLower(os.Getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return DepthTrueColor
	}

	if term := os.Getenv("TERM"); strings.Contains(term, "256color") {
		return Depth256
	}

	return Depth16
}

// resolveDepth turns DepthAuto into a concrete depth.
func resolveDepth(d ColorDepth) ColorDepth {
	if d == DepthAuto {
		return detectColorDepth()
	}

	return d
}

// String implements fmt.Stringer for diagnostics.
func (d ColorDepth) String() string {
	switch d {
	case DepthAuto:
		return "auto"
	case Depth16:
		return "16"
	case Depth256:
		return "256"
	case DepthTrueColor:
		return "truecolor"
	default:
		return "ColorDepth(" + strconv.Itoa(int(d)) + ")"
	}
}
