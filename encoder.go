package reggol

import (
	"time"
)

// Encoder renders an event into a byte buffer.
//
// The contract is append-only: an encoder must append to dst and return the
// extended slice, never allocate an intermediate string or buffer of its own.
// It must also terminate the record with exactly one '\n' — the writer passes
// the bytes through untouched.
type Encoder interface {
	AppendEvent(dst []byte, d *EventData) []byte
}

// Formatting hooks.
//
// Each hook appends to a caller-owned buffer instead of returning a string, so
// customizing the output costs no allocation. This is the deliberate difference
// from the v0 `Formatter func(interface{}) string`, which boxed its argument and
// allocated its result on every field.
type (
	// LevelFormatter renders a level.
	LevelFormatter func(dst []byte, l Level) []byte
	// TimeFormatter renders a timestamp.
	TimeFormatter func(dst []byte, t time.Time) []byte
	// FieldFormatter renders a key/value pair.
	FieldFormatter func(dst []byte, key string, v Value) []byte
	// KeyFormatter renders a field key.
	KeyFormatter func(dst []byte, key string) []byte
	// ValueFormatter renders a field value.
	ValueFormatter func(dst []byte, v Value) []byte
	// MessageFormatter renders the message.
	MessageFormatter func(dst []byte, msg string) []byte
	// BlocksFormatter renders the blocks.
	BlocksFormatter func(dst []byte, b Blocks) []byte
)

// baseEncoder carries the options and hooks shared by the built-in encoders.
//
// Encoders embed it by value but expose only pointer receivers: a per-line copy
// of a struct holding this many function pointers is exactly the cost the
// rewrite set out to remove.
type baseEncoder struct {
	timeFormat string

	timeKey    string
	levelKey   string
	messageKey string
	blocksKey  string

	showTimestamp bool
	showLevel     bool
	sortFields    bool

	FormatLevel   LevelFormatter
	FormatTime    TimeFormatter
	FormatField   FieldFormatter
	FormatKey     KeyFormatter
	FormatValue   ValueFormatter
	FormatMessage MessageFormatter
	FormatBlocks  BlocksFormatter

	BeforeEncode func(d *EventData)
	AfterEncode  func(d *EventData)
}

func newBaseEncoder(timeFormat string) baseEncoder {
	return baseEncoder{
		timeFormat:    timeFormat,
		timeKey:       TimestampFieldName,
		levelKey:      LevelFieldName,
		messageKey:    MessageFieldName,
		blocksKey:     BlocksFieldName,
		showTimestamp: true,
		showLevel:     true,
		sortFields:    true,
	}
}

// WithKeyNames overrides the names of the built-in keys.
//
// The main use is interoperability: log/slog expects "time", "level" and "msg",
// and its conformance suite checks for exactly those.
func WithKeyNames(timeKey, levelKey, messageKey string) EncoderOption {
	return func(b *baseEncoder) {
		if timeKey != "" {
			b.timeKey = timeKey
		}

		if levelKey != "" {
			b.levelKey = levelKey
		}

		if messageKey != "" {
			b.messageKey = messageKey
		}
	}
}

// appendTime renders the timestamp, honoring a custom hook.
func (b *baseEncoder) appendTime(dst []byte, t time.Time) []byte {
	if b.FormatTime != nil {
		return b.FormatTime(dst, t)
	}

	return t.AppendFormat(dst, b.timeFormat)
}

// appendKey renders a field key, honoring a custom hook.
func (b *baseEncoder) appendKey(dst []byte, key string) []byte {
	if b.FormatKey != nil {
		return b.FormatKey(dst, key)
	}

	return append(dst, key...)
}

// appendValue renders a field value, honoring a custom hook.
func (b *baseEncoder) appendValue(dst []byte, v Value) []byte {
	if b.FormatValue != nil {
		return b.FormatValue(dst, v)
	}

	return v.AppendTo(dst)
}

// appendMessage renders the message, honoring a custom hook.
func (b *baseEncoder) appendMessage(dst []byte, msg string) []byte {
	if b.FormatMessage != nil {
		return b.FormatMessage(dst, msg)
	}

	return append(dst, msg...)
}

// orderedFields returns the event fields in output order.
//
// Sorting happens in place on the event's own slice, which is pooled, so no
// allocation is involved.
func (b *baseEncoder) orderedFields(d *EventData) []Field {
	if b.sortFields {
		sortFields(d.fields)
	}

	return d.fields
}

// EncoderOption configures a built-in encoder.
type EncoderOption func(*baseEncoder)

// WithTimeFormat sets the timestamp layout.
func WithTimeFormat(layout string) EncoderOption {
	return func(b *baseEncoder) { b.timeFormat = layout }
}

// WithoutTimestamp hides the timestamp.
func WithoutTimestamp() EncoderOption {
	return func(b *baseEncoder) { b.showTimestamp = false }
}

// WithoutLevel hides the level.
func WithoutLevel() EncoderOption {
	return func(b *baseEncoder) { b.showLevel = false }
}

// WithoutSort leaves fields in insertion order.
//
// Slightly faster, and useful when call-site order carries meaning; the trade
// is that output is no longer stable across runs for map-derived data.
func WithoutSort() EncoderOption {
	return func(b *baseEncoder) { b.sortFields = false }
}

// WithLevelFormatter overrides level rendering.
func WithLevelFormatter(fn LevelFormatter) EncoderOption {
	return func(b *baseEncoder) { b.FormatLevel = fn }
}

// WithTimeFormatter overrides timestamp rendering.
func WithTimeFormatter(fn TimeFormatter) EncoderOption {
	return func(b *baseEncoder) { b.FormatTime = fn }
}

// WithFieldFormatter overrides whole-field rendering.
func WithFieldFormatter(fn FieldFormatter) EncoderOption {
	return func(b *baseEncoder) { b.FormatField = fn }
}

// WithKeyFormatter overrides field-key rendering.
func WithKeyFormatter(fn KeyFormatter) EncoderOption {
	return func(b *baseEncoder) { b.FormatKey = fn }
}

// WithValueFormatter overrides field-value rendering.
func WithValueFormatter(fn ValueFormatter) EncoderOption {
	return func(b *baseEncoder) { b.FormatValue = fn }
}

// WithMessageFormatter overrides message rendering.
func WithMessageFormatter(fn MessageFormatter) EncoderOption {
	return func(b *baseEncoder) { b.FormatMessage = fn }
}

// WithBlocksFormatter overrides blocks rendering.
func WithBlocksFormatter(fn BlocksFormatter) EncoderOption {
	return func(b *baseEncoder) { b.FormatBlocks = fn }
}

// WithBeforeEncode installs a callback invoked before encoding.
func WithBeforeEncode(fn func(d *EventData)) EncoderOption {
	return func(b *baseEncoder) { b.BeforeEncode = fn }
}

// WithAfterEncode installs a callback invoked after encoding.
func WithAfterEncode(fn func(d *EventData)) EncoderOption {
	return func(b *baseEncoder) { b.AfterEncode = fn }
}
