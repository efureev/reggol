package reggol

import (
	"math"
	"strconv"
	"unicode/utf8"
)

// JSONEncoder renders events as one JSON object per line.
//
// Nothing from encoding/json is used on the hot path: values are appended
// directly, which keeps the encoder allocation-free.
type JSONEncoder struct {
	baseEncoder
}

// NewJSONEncoder creates a JSON encoder.
func NewJSONEncoder(opts ...EncoderOption) *JSONEncoder {
	j := &JSONEncoder{baseEncoder: newBaseEncoder(DefaultJSONTimeFormat)}

	for _, opt := range opts {
		opt(&j.baseEncoder)
	}

	return j
}

// AppendEvent implements Encoder.
func (j *JSONEncoder) AppendEvent(dst []byte, d *EventData) []byte {
	if j.BeforeEncode != nil {
		j.BeforeEncode(d)
	}

	dst = append(dst, '{')
	sep := false

	if j.showTimestamp && !d.ts.IsZero() {
		dst = appendJSONSep(dst, &sep)
		dst = appendJSONKey(dst, j.timeKey)
		dst = append(dst, '"')
		dst = j.appendTime(dst, d.ts)
		dst = append(dst, '"')
	}

	if j.showLevel && d.level != NoLevel {
		dst = appendJSONSep(dst, &sep)
		dst = appendJSONKey(dst, j.levelKey)
		dst = appendJSONString(dst, d.level.String())
	}

	if hasCaller(d) {
		dst = appendJSONSep(dst, &sep)
		dst = appendJSONKey(dst, CallerFieldName)
		dst = append(dst, '"')
		dst = j.appendCaller(dst, d)
		dst = append(dst, '"')
	}

	if len(d.blocks) > 0 {
		dst = appendJSONSep(dst, &sep)
		dst = appendJSONKey(dst, j.blocksKey)
		dst = append(dst, '[')

		for i := range d.blocks {
			if i > 0 {
				dst = append(dst, ',')
			}

			dst = appendJSONString(dst, d.blocks[i].Value())
		}

		dst = append(dst, ']')
	}

	if len(d.message) > 0 {
		dst = appendJSONSep(dst, &sep)
		dst = appendJSONKey(dst, j.messageKey)
		dst = appendJSONBytes(dst, d.message)
	}

	if len(d.prefix) > 0 {
		dst = appendJSONSep(dst, &sep)
		dst = append(dst, d.prefix...)
	}

	for _, f := range j.orderedFields(d) {
		dst = appendJSONSep(dst, &sep)
		dst = j.appendField(dst, f.Key, f.Val)
	}

	dst = append(dst, '}')

	if j.AfterEncode != nil {
		j.AfterEncode(d)
	}

	return append(dst, '\n')
}

// AppendPrefix implements PrefixEncoder.
func (j *JSONEncoder) AppendPrefix(dst []byte, fields []Field) []byte {
	for i, f := range fields {
		if i > 0 {
			dst = append(dst, ',')
		}

		dst = j.appendField(dst, f.Key, f.Val)
	}

	return dst
}

func (j *JSONEncoder) appendField(dst []byte, key string, v Value) []byte {
	if j.FormatField != nil {
		return j.FormatField(dst, key, v)
	}

	dst = appendJSONKey(dst, key)

	return appendJSONValue(dst, v)
}

//nolint:exhaustive // KindAny and unknown kinds share the default branch
func appendJSONValue(dst []byte, v Value) []byte {
	switch v.Kind() {
	case KindString:
		return appendJSONString(dst, v.String())
	case KindInt64:
		return strconv.AppendInt(dst, v.Int64(), base10)
	case KindUint64:
		return strconv.AppendUint(dst, v.Uint64(), base10)
	case KindFloat64:
		return appendJSONFloat(dst, v.Float64())
	case KindBool:
		return strconv.AppendBool(dst, v.Bool())
	case KindDuration:
		return strconv.AppendInt(dst, int64(v.Duration()), base10)
	case KindTime:
		dst = append(dst, '"')
		dst = v.Time().AppendFormat(dst, DefaultJSONTimeFormat)

		return append(dst, '"')
	case KindBytes:
		return appendJSONString(dst, string(v.Bytes()))
	case KindError:
		if err := v.Err(); err != nil {
			return appendJSONString(dst, err.Error())
		}

		return append(dst, "null"...)
	case KindGroup:
		dst = append(dst, '{')

		for i, f := range v.Group() {
			if i > 0 {
				dst = append(dst, ',')
			}

			dst = appendJSONKey(dst, f.Key)
			dst = appendJSONValue(dst, f.Val)
		}

		return append(dst, '}')
	default:
		if v.Any() == nil {
			return append(dst, "null"...)
		}

		return appendJSONString(dst, string(v.AppendTo(nil)))
	}
}

// appendJSONFloat renders a float, mapping the values JSON cannot express onto
// null rather than emitting invalid documents.
func appendJSONFloat(dst []byte, f float64) []byte {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return append(dst, "null"...)
	}

	return strconv.AppendFloat(dst, f, floatFormat, floatPrecision, float64Bits)
}

func appendJSONSep(dst []byte, sep *bool) []byte {
	if *sep {
		return append(dst, ',')
	}

	*sep = true

	return dst
}

func appendJSONKey(dst []byte, key string) []byte {
	dst = appendJSONString(dst, key)

	return append(dst, ':')
}

const hexDigits = "0123456789abcdef"

// appendJSONString appends s as a quoted JSON string.
func appendJSONString(dst []byte, s string) []byte {
	return appendJSONText(dst, s, utf8.ValidString(s))
}

// appendJSONBytes appends b as a quoted JSON string.
//
// The message travels as bytes so that Msgf can format into the event's pooled
// buffer without allocating a string.
func appendJSONBytes(dst, b []byte) []byte {
	return appendJSONText(dst, b, utf8.Valid(b))
}

// appendJSONText quotes and escapes text per RFC 8259.
//
// Validity is decided once for the whole input rather than rune by rune: when
// the text is valid UTF-8 — which it is for essentially every log record —
// multi-byte sequences need no inspection at all and are copied verbatim, so
// the loop only looks for the ASCII bytes that require escaping.
func appendJSONText[T ~string | ~[]byte](dst []byte, s T, valid bool) []byte {
	if !valid {
		return appendJSONSanitized(dst, string(s))
	}

	dst = append(dst, '"')

	start := 0

	for i := range len(s) {
		b := s[i]
		if b >= utf8.RuneSelf || isSafeJSONByte(b) {
			continue
		}

		dst = append(dst, s[start:i]...)
		dst = appendJSONEscape(dst, b)
		start = i + 1
	}

	dst = append(dst, s[start:]...)

	return append(dst, '"')
}

// appendJSONSanitized handles text that is not valid UTF-8, replacing the bad
// bytes with U+FFFD so that the output is still a valid document.
//
// This path allocates, and deliberately so: invalid UTF-8 in a log record is
// pathological, and keeping the common path free of rune decoding is worth more
// than optimizing the broken one.
func appendJSONSanitized(dst []byte, s string) []byte {
	dst = append(dst, '"')

	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if isSafeJSONByte(b) {
				dst = append(dst, b)
			} else {
				dst = appendJSONEscape(dst, b)
			}

			i++

			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, "\ufffd"...)
		} else {
			dst = append(dst, s[i:i+size]...)
		}

		i += size
	}

	return append(dst, '"')
}

func isSafeJSONByte(b byte) bool {
	return b >= 0x20 && b != '"' && b != '\\'
}

func appendJSONEscape(dst []byte, b byte) []byte {
	switch b {
	case '"':
		return append(dst, '\\', '"')
	case '\\':
		return append(dst, '\\', '\\')
	case '\n':
		return append(dst, '\\', 'n')
	case '\r':
		return append(dst, '\\', 'r')
	case '\t':
		return append(dst, '\\', 't')
	case '\b':
		return append(dst, '\\', 'b')
	case '\f':
		return append(dst, '\\', 'f')
	default:
		return append(dst, '\\', 'u', '0', '0', hexDigits[b>>4], hexDigits[b&0xF])
	}
}
