package reggol

import (
	"io"
	"os"
)

// ColorMode controls whether the console encoder emits ANSI sequences.
type ColorMode uint8

// Color modes.
const (
	// ColorAuto enables color only when the destination looks like a terminal.
	ColorAuto ColorMode = iota
	// ColorAlways always emits ANSI sequences.
	ColorAlways
	// ColorNever never emits ANSI sequences.
	ColorNever
)

// ConsoleEncoder renders events for humans: short level labels, colors, blocks
// ahead of the message.
type ConsoleEncoder struct {
	baseEncoder

	color       bool
	levelStyles [levelCount]style
	timeStyle   style
	errorStyle  style
	keyStyle    style
}

// ConsoleOption configures a ConsoleEncoder.
type ConsoleOption func(*ConsoleEncoder)

// WithColorMode sets the color policy.
//
// This replaces v0's `NewConsoleTransformer(noColor bool, …)`, whose first
// argument read as the opposite of its meaning.
func WithColorMode(m ColorMode, out io.Writer) ConsoleOption {
	return func(c *ConsoleEncoder) { c.color = resolveColor(m, out) }
}

// WithConsoleOptions applies shared encoder options.
func WithConsoleOptions(opts ...EncoderOption) ConsoleOption {
	return func(c *ConsoleEncoder) {
		for _, opt := range opts {
			opt(&c.baseEncoder)
		}
	}
}

// NewConsoleEncoder creates a console encoder.
//
// Color defaults to ColorAuto against os.Stdout; pass WithColorMode to decide
// from the writer the logger actually uses.
func NewConsoleEncoder(opts ...ConsoleOption) *ConsoleEncoder {
	c := &ConsoleEncoder{
		baseEncoder: newBaseEncoder(DefaultConsoleTimeFormat),
		color:       resolveColor(ColorAuto, os.Stdout),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.buildStyles()

	return c
}

// defaultLevelColors maps a level to its style, indexed rather than hashed.
//
//nolint:gochecknoglobals // immutable lookup table
var defaultLevelColors = [levelCount]TextStyle{
	ColorFgBlue,               // trace
	0,                         // debug
	ColorFgGreen | ColorBold,  // info
	ColorFgYellow | ColorBold, // warn
	ColorFgRed | ColorBold,    // error
	ColorFgRed | ColorBold,    // fatal
	ColorFgRed | ColorBold,    // panic
	0,                         // no level
	0,                         // disabled
}

// buildStyles resolves every escape sequence once, so that a log line appends
// precomputed strings instead of recomputing styles per event.
func (c *ConsoleEncoder) buildStyles() {
	if !c.color {
		c.levelStyles = [levelCount]style{}
		c.timeStyle = style{}
		c.errorStyle = style{}
		c.keyStyle = style{}

		return
	}

	for i := range defaultLevelColors {
		c.levelStyles[i] = newStyle(defaultLevelColors[i])
	}

	c.timeStyle = newStyle(ColorFgBlack | ColorFgBright)
	c.errorStyle = newStyle(ColorFgRed)
	c.keyStyle = newStyle(ColorFgBlack | ColorFgBright)
}

// SetLevelColor overrides the style of a single level.
func (c *ConsoleEncoder) SetLevelColor(l Level, s TextStyle) {
	if i, ok := levelIndex(l); ok && c.color {
		c.levelStyles[i] = newStyle(s)
	}
}

// AppendEvent implements Encoder.
func (c *ConsoleEncoder) AppendEvent(dst []byte, d *EventData) []byte {
	if c.BeforeEncode != nil {
		c.BeforeEncode(d)
	}

	sep := false

	if c.showTimestamp && !d.ts.IsZero() {
		start := len(dst)
		dst = c.timeStyle.open(dst)
		dst = c.appendTime(dst, d.ts)
		dst = c.timeStyle.close(dst)

		sep = len(dst) > start
	}

	if c.showLevel && d.level != NoLevel {
		dst = appendSpace(dst, &sep)
		dst = c.appendLevel(dst, d.level)
	}

	if hasCaller(d) {
		dst = appendSpace(dst, &sep)
		dst = c.keyStyle.open(dst)
		dst = c.appendCaller(dst, d)
		dst = c.keyStyle.close(dst)
	}

	for i := range d.blocks {
		dst = appendSpace(dst, &sep)
		dst = d.blocks[i].appendTo(dst)
	}

	if len(d.message) > 0 {
		dst = appendSpace(dst, &sep)
		dst = c.appendMessage(dst, d.message)
	}

	if len(d.prefix) > 0 {
		dst = appendSpace(dst, &sep)
		dst = append(dst, d.prefix...)
	}

	for _, f := range c.orderedFields(d) {
		dst = appendSpace(dst, &sep)
		dst = c.appendField(dst, f.Key, f.Val)
	}

	if c.AfterEncode != nil {
		c.AfterEncode(d)
	}

	return append(dst, '\n')
}

// AppendPrefix implements PrefixEncoder.
func (c *ConsoleEncoder) AppendPrefix(dst []byte, fields []Field) []byte {
	for i, f := range fields {
		if i > 0 {
			dst = append(dst, ' ')
		}

		dst = c.appendField(dst, f.Key, f.Val)
	}

	return dst
}

func (c *ConsoleEncoder) appendLevel(dst []byte, l Level) []byte {
	if c.FormatLevel != nil {
		return c.FormatLevel(dst, l)
	}

	i, ok := levelIndex(l)
	if !ok {
		return append(dst, l.Label()...)
	}

	return c.levelStyles[i].appendTo(dst, l.Label())
}

// appendField renders one pair. The conventional error key is rendered as bare
// colored text, which is what makes an error read as an error in a terminal.
func (c *ConsoleEncoder) appendField(dst []byte, key string, v Value) []byte {
	if c.FormatField != nil {
		return c.FormatField(dst, key, v)
	}

	if key == ErrorFieldName && v.Kind() == KindError {
		start := len(dst)
		dst = c.errorStyle.open(dst)
		dst = c.appendValue(dst, v)

		if len(dst) > start {
			dst = c.errorStyle.close(dst)
		}

		return dst
	}

	dst = c.keyStyle.appendTo(dst, key)
	dst = append(dst, '=')

	return c.appendValue(dst, v)
}

// appendSpace inserts a separator before every element but the first.
func appendSpace(dst []byte, sep *bool) []byte {
	if *sep {
		return append(dst, ' ')
	}

	*sep = true

	return dst
}

// resolveColor decides whether to emit ANSI sequences.
//
// Detection is done against the writer the encoder actually writes to, and
// honors the NO_COLOR and FORCE_COLOR conventions. v0 had no detection at all,
// so redirecting output to a file produced escape sequences in the file.
func resolveColor(m ColorMode, out io.Writer) bool {
	switch m {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	case ColorAuto:
		return autoColor(out)
	default:
		return autoColor(out)
	}
}

func autoColor(out io.Writer) bool {
	if _, ok := os.LookupEnv("FORCE_COLOR"); ok {
		return true
	}

	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}

	if os.Getenv("TERM") == "dumb" {
		return false
	}

	return isTerminal(out)
}

// isTerminal reports whether out is a character device.
//
// os.File.Stat is enough for this and keeps the dependency list at one direct
// entry, which is a stated property of the library.
func isTerminal(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
