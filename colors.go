package reggol

import (
	"gh.tarampamp.am/colors"
)

// TextStyle is an ANSI text style.
type TextStyle = colors.TextStyle

// ANSI text styles, aliased from the colors package.
//
// These are aliases rather than re-declarations on purpose: a re-declared
// `1 << iota` block would keep compiling while silently drifting from upstream
// values, producing wrong escape sequences with nothing to catch it.
const (
	ColorFgBlack   = colors.FgBlack
	ColorFgRed     = colors.FgRed
	ColorFgGreen   = colors.FgGreen
	ColorFgYellow  = colors.FgYellow
	ColorFgBlue    = colors.FgBlue
	ColorFgMagenta = colors.FgMagenta
	ColorFgCyan    = colors.FgCyan
	ColorFgWhite   = colors.FgWhite
	ColorFgDefault = colors.FgDefault
	ColorFgBright  = colors.FgBright

	ColorBgBlack   = colors.BgBlack
	ColorBgRed     = colors.BgRed
	ColorBgGreen   = colors.BgGreen
	ColorBgYellow  = colors.BgYellow
	ColorBgBlue    = colors.BgBlue
	ColorBgMagenta = colors.BgMagenta
	ColorBgCyan    = colors.BgCyan
	ColorBgWhite   = colors.BgWhite
	ColorBgDefault = colors.BgDefault
	ColorBgBright  = colors.BgBright

	ColorBold      = colors.Bold
	ColorFaint     = colors.Faint
	ColorItalic    = colors.Italic
	ColorUnderline = colors.Underline
	ColorBlinking  = colors.Blinking
	ColorReverse   = colors.Reverse
	ColorInvisible = colors.Invisible
	ColorStrike    = colors.Strike
	ColorReset     = colors.Reset
)

// style holds the escape sequences for one text style, resolved once at encoder
// construction instead of on every log line.
type style struct {
	start string
	reset string
}

// newStyle resolves s into its escape sequences.
//
// colors.ColorCodes reports the raw sequences regardless of the package-level
// Enabled() flag, which is deliberate here: reggol decides about color per
// encoder, from the writer it actually writes to, not from a global guess about
// os.Stdout.
func newStyle(s TextStyle) style {
	if s == 0 {
		return style{}
	}

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
