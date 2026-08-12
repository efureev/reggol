package reggol

import (
	"strconv"
)

// TextStyle is a bitmask describing an ANSI text style: at most one foreground
// color, at most one background color, and any combination of attributes.
//
// The mapping to escape sequences is ECMA-48 SGR, unchanged since 1976, which
// is why reggol carries these hundred lines rather than a dependency.
type TextStyle uint32

// ANSI text styles.
//
// The numeric values are part of the API: code that stored one is entitled to
// keep working, and TestTextStyleValues pins every one of them.
const (
	ColorFgBlack   TextStyle = 1 << iota // black text
	ColorFgRed                           // red text
	ColorFgGreen                         // green text
	ColorFgYellow                        // yellow text
	ColorFgBlue                          // blue text
	ColorFgMagenta                       // magenta text
	ColorFgCyan                          // cyan text
	ColorFgWhite                         // white text
	ColorFgDefault                       // the terminal's default text color

	ColorFgBright // brighten the text color; on its own it selects nothing

	ColorBgBlack   // black background
	ColorBgRed     // red background
	ColorBgGreen   // green background
	ColorBgYellow  // yellow background
	ColorBgBlue    // blue background
	ColorBgMagenta // magenta background
	ColorBgCyan    // cyan background
	ColorBgWhite   // white background
	ColorBgDefault // the terminal's default background

	ColorBgBright // brighten the background color

	ColorBold      // bold text
	ColorFaint     // faint text
	ColorItalic    // italic text
	ColorUnderline // underlined text
	ColorBlinking  // blinking text
	ColorReverse   // swapped foreground and background
	ColorInvisible // invisible text
	ColorStrike    // struck-through text

	ColorReset // reset every style; overrides everything else
)

// Has reports whether every bit of z is set in ts.
func (ts TextStyle) Has(z TextStyle) bool { return ts&z != 0 }

// Add sets the given styles.
func (ts *TextStyle) Add(styles ...TextStyle) {
	for _, s := range styles {
		*ts |= s
	}
}

// Remove clears the given styles.
func (ts *TextStyle) Remove(styles ...TextStyle) {
	for _, s := range styles {
		*ts &^= s
	}
}

// SGR parameters. Grouped rather than inlined so that the relationship between
// a style and the code that undoes it is visible in one place.
// ansiEscape opens every SGR sequence.
const ansiEscape = 0x1b

const (
	sgrReset = 0

	sgrBold      = 1
	sgrFaint     = 2
	sgrItalic    = 3
	sgrUnderline = 4
	sgrBlinking  = 5
	sgrReverse   = 7
	sgrInvisible = 8
	sgrStrike    = 9

	sgrNormalIntensity = 22 // undoes both bold and faint
	sgrNoItalic        = 23
	sgrNoUnderline     = 24
	sgrNoBlinking      = 25
	sgrNoReverse       = 27
	sgrNoInvisible     = 28
	sgrNoStrike        = 29

	sgrFgBase    = 30
	sgrFgDefault = 39
	sgrBgBase    = 40
	sgrBgDefault = 49

	// sgrBrightOffset turns a base color code into its bright variant.
	sgrBrightOffset = 60
)

// attribute pairs an attribute bit with its SGR code and the code that undoes
// it. The order of this table is the order codes appear in the output.
//
//nolint:gochecknoglobals // immutable lookup table
var attributes = [...]struct {
	bit      TextStyle
	set, off int
}{
	{ColorBold, sgrBold, sgrNormalIntensity},
	{ColorFaint, sgrFaint, sgrNormalIntensity},
	{ColorItalic, sgrItalic, sgrNoItalic},
	{ColorUnderline, sgrUnderline, sgrNoUnderline},
	{ColorBlinking, sgrBlinking, sgrNoBlinking},
	{ColorReverse, sgrReverse, sgrNoReverse},
	{ColorInvisible, sgrInvisible, sgrNoInvisible},
	{ColorStrike, sgrStrike, sgrNoStrike},
}

// colorEntry maps a color bit to its offset from the base SGR code.
type colorEntry struct {
	bit    TextStyle
	offset int
}

// The color tables list bits in selection order: the first one present wins,
// which is what makes a mask holding two colors well defined rather than
// undefined. Default comes first so that it beats an explicit color.
//
//nolint:gochecknoglobals // immutable lookup tables
var (
	foregrounds = [...]colorEntry{
		{ColorFgDefault, sgrFgDefault - sgrFgBase},
		{ColorFgBlack, 0}, {ColorFgRed, 1}, {ColorFgGreen, 2}, {ColorFgYellow, 3},
		{ColorFgBlue, 4}, {ColorFgMagenta, 5}, {ColorFgCyan, 6}, {ColorFgWhite, 7},
	}

	backgrounds = [...]colorEntry{
		{ColorBgDefault, sgrBgDefault - sgrBgBase},
		{ColorBgBlack, 0}, {ColorBgRed, 1}, {ColorBgGreen, 2}, {ColorBgYellow, 3},
		{ColorBgBlue, 4}, {ColorBgMagenta, 5}, {ColorBgCyan, 6}, {ColorBgWhite, 7},
	}
)

// maxSGRCodes is the widest sequence a TextStyle can produce: eight attributes
// plus a foreground and a background.
const maxSGRCodes = len(attributes) + 2

// ColorCodes returns the escape sequences that open and close the style.
//
// Both are empty for the zero style, and reset is empty for ColorReset, which
// needs nothing to undo it.
//
// The order of the closing codes is the reverse of the opening ones — background,
// then foreground, then attributes last-set-first. That is not cosmetic: it is
// how the sequences nest, and reversing it changes the bytes.
func (ts TextStyle) ColorCodes() (start, reset string) {
	if ts == 0 {
		return "", ""
	}

	if ts.Has(ColorReset) {
		return renderSGR([]int{sgrReset}), ""
	}

	var (
		setCodes [maxSGRCodes]int
		offCodes [maxSGRCodes]int
		set, off int
	)

	for _, a := range attributes {
		if ts.Has(a.bit) {
			setCodes[set] = a.set
			set++

			offCodes[off] = a.off
			off++
		}
	}

	// Attributes undo in reverse: the last one set is the first one cleared.
	reverseInts(offCodes[:off])

	if code, ok := colorCode(ts, foregrounds[:], sgrFgBase, sgrFgDefault, ColorFgBright); ok {
		setCodes[set] = code
		set++

		if code != sgrFgDefault {
			off = prependInt(offCodes[:], off, sgrFgDefault)
		}
	}

	if code, ok := colorCode(ts, backgrounds[:], sgrBgBase, sgrBgDefault, ColorBgBright); ok {
		setCodes[set] = code
		set++

		if code != sgrBgDefault {
			off = prependInt(offCodes[:], off, sgrBgDefault)
		}
	}

	return renderSGR(setCodes[:set]), renderSGR(offCodes[:off])
}

// colorCode resolves the first color bit present in table, applying the
// bright modifier when its bit is set.
//
// The default color is never brightened: there is no bright variant of "the
// terminal's own color", and 99 would be nonsense.
func colorCode(ts TextStyle, table []colorEntry, base, defaultCode int, bright TextStyle) (int, bool) {
	for _, e := range table {
		if !ts.Has(e.bit) {
			continue
		}

		code := base + e.offset
		if code != defaultCode && ts.Has(bright) {
			code += sgrBrightOffset
		}

		return code, true
	}

	return 0, false
}

// renderSGR assembles `ESC [ a ; b ; c m`.
func renderSGR(codes []int) string {
	if len(codes) == 0 {
		return ""
	}

	buf := make([]byte, 0, 2+len(codes)*3+1)
	buf = append(buf, ansiEscape, '[')

	for i, c := range codes {
		if i > 0 {
			buf = append(buf, ';')
		}

		buf = strconv.AppendInt(buf, int64(c), base10)
	}

	return string(append(buf, 'm'))
}

// prependInt inserts v at the front of codes[:n] and returns the new length.
func prependInt(codes []int, n, v int) int {
	copy(codes[1:n+1], codes[:n])
	codes[0] = v

	return n + 1
}

func reverseInts(s []int) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// style holds the escape sequences for one text style, resolved once at encoder
// construction instead of on every log line.
type style struct {
	start string
	reset string
}

// newStyle resolves s into its escape sequences.
func newStyle(s TextStyle) style {
	start, reset := s.ColorCodes()

	return style{start: start, reset: reset}
}

// appendTo appends val to dst wrapped in the style.
func (s style) appendTo(dst []byte, val string) []byte {
	if s.start == "" {
		return append(dst, val...)
	}

	dst = append(dst, s.start...)
	dst = append(dst, val...)

	return append(dst, s.reset...)
}

// open appends the opening sequence, if any.
func (s style) open(dst []byte) []byte {
	if s.start == "" {
		return dst
	}

	return append(dst, s.start...)
}

// close appends the closing sequence, if any.
func (s style) close(dst []byte) []byte {
	if s.start == "" {
		return dst
	}

	return append(dst, s.reset...)
}
