package reggol

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Formatting parameters for strconv.
const (
	// base10 is the radix used for every integer rendering.
	base10 = 10
	// float64Bits is the bit size passed to strconv.AppendFloat.
	float64Bits = 64
	// floatFormat renders the shortest representation that round-trips.
	floatFormat = 'g'
	// floatPrecision asks for the shortest round-tripping precision.
	floatPrecision = -1
)

// Kind describes the type of data held by a Value.
//
// The set mirrors slog.Kind so that bridging to and from log/slog is a copy
// rather than a conversion through any.
type Kind uint8

// Kinds of values a field can hold.
const (
	KindAny Kind = iota
	KindString
	KindInt64
	KindUint64
	KindFloat64
	KindBool
	KindDuration
	KindTime
	KindBytes
	KindError
	KindStringer
	KindGroup
)

// String implements fmt.Stringer for diagnostics.
func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}

	return "Kind(" + strconv.Itoa(int(k)) + ")"
}

//nolint:gochecknoglobals // immutable lookup table
var kindNames = [...]string{
	"Any", "String", "Int64", "Uint64", "Float64", "Bool",
	"Duration", "Time", "Bytes", "Error", "Stringer", "Group",
}

// Value holds a log field value without boxing scalars into an interface.
//
// Scalars live in num (and aux, for the nanosecond part of a timestamp),
// strings live in str, and only genuinely dynamic values touch any. Storing an
// int, bool, duration or time therefore costs no allocation.
type Value struct {
	num  uint64
	aux  uint64
	str  string
	any  any
	kind Kind
}

// StringValue returns a Value for a string.
func StringValue(s string) Value {
	return Value{kind: KindString, str: s}
}

// IntValue returns a Value for an int.
func IntValue(i int) Value {
	return Int64Value(int64(i))
}

// Int64Value returns a Value for an int64.
func Int64Value(i int64) Value {
	return Value{kind: KindInt64, num: uint64(i)}
}

// Uint64Value returns a Value for a uint64.
func Uint64Value(u uint64) Value {
	return Value{kind: KindUint64, num: u}
}

// Float64Value returns a Value for a float64.
func Float64Value(f float64) Value {
	return Value{kind: KindFloat64, num: math.Float64bits(f)}
}

// BoolValue returns a Value for a bool.
func BoolValue(b bool) Value {
	var n uint64
	if b {
		n = 1
	}

	return Value{kind: KindBool, num: n}
}

// DurationValue returns a Value for a time.Duration.
func DurationValue(d time.Duration) Value {
	return Value{kind: KindDuration, num: uint64(d)}
}

// TimeValue returns a Value for a time.Time.
//
// The instant is decomposed into seconds, nanoseconds and *time.Location, all of
// which are stored without allocating: the monotonic reading is dropped, which
// is irrelevant for logging. Unlike a UnixNano-based encoding this is lossless
// across the whole representable range, including the zero time.
func TimeValue(t time.Time) Value {
	return Value{
		kind: KindTime,
		num:  uint64(t.Unix()),
		aux:  uint64(t.Nanosecond()),
		any:  t.Location(),
	}
}

// BytesValue returns a Value for a byte slice.
func BytesValue(b []byte) Value {
	return Value{kind: KindBytes, any: b}
}

// ErrValue returns a Value for an error.
func ErrValue(err error) Value {
	return Value{kind: KindError, any: err}
}

// StringerValue returns a Value for a fmt.Stringer.
func StringerValue(s fmt.Stringer) Value {
	return Value{kind: KindStringer, any: s}
}

// GroupValue returns a Value holding a group of fields.
func GroupValue(fields ...Field) Value {
	return Value{kind: KindGroup, any: fields}
}

// AnyValue returns a Value for an arbitrary value, choosing the most specific
// Kind available so that common types avoid the interface path entirely.
//
//nolint:cyclop // a flat type switch is the point
func AnyValue(v any) Value {
	switch x := v.(type) {
	case Value:
		return x
	case string:
		return StringValue(x)
	case int:
		return Int64Value(int64(x))
	case int8:
		return Int64Value(int64(x))
	case int16:
		return Int64Value(int64(x))
	case int32:
		return Int64Value(int64(x))
	case int64:
		return Int64Value(x)
	case uint:
		return Uint64Value(uint64(x))
	case uint8:
		return Uint64Value(uint64(x))
	case uint16:
		return Uint64Value(uint64(x))
	case uint32:
		return Uint64Value(uint64(x))
	case uint64:
		return Uint64Value(x)
	case float32:
		return Float64Value(float64(x))
	case float64:
		return Float64Value(x)
	case bool:
		return BoolValue(x)
	case time.Duration:
		return DurationValue(x)
	case time.Time:
		return TimeValue(x)
	case []byte:
		return BytesValue(x)
	case error:
		return ErrValue(x)
	case nil:
		return Value{kind: KindAny}
	default:
		return Value{kind: KindAny, any: v}
	}
}

// Kind reports the type of data held by v.
func (v Value) Kind() Kind { return v.kind }

// Int64 returns the value as an int64. Valid for KindInt64.
func (v Value) Int64() int64 { return int64(v.num) }

// Uint64 returns the value as a uint64. Valid for KindUint64.
func (v Value) Uint64() uint64 { return v.num }

// Float64 returns the value as a float64. Valid for KindFloat64.
func (v Value) Float64() float64 { return math.Float64frombits(v.num) }

// Bool returns the value as a bool. Valid for KindBool.
func (v Value) Bool() bool { return v.num != 0 }

// Duration returns the value as a time.Duration. Valid for KindDuration.
func (v Value) Duration() time.Duration { return time.Duration(v.num) }

// Time returns the value as a time.Time. Valid for KindTime.
func (v Value) Time() time.Time {
	loc, _ := v.any.(*time.Location)
	if loc == nil {
		loc = time.UTC
	}

	return time.Unix(int64(v.num), int64(v.aux)).In(loc)
}

// Bytes returns the value as a byte slice. Valid for KindBytes.
func (v Value) Bytes() []byte {
	b, _ := v.any.([]byte)

	return b
}

// Err returns the value as an error. Valid for KindError.
func (v Value) Err() error {
	err, _ := v.any.(error)

	return err
}

// Group returns the fields of a group value. Valid for KindGroup.
func (v Value) Group() []Field {
	f, _ := v.any.([]Field)

	return f
}

// Any returns the value as an interface, boxing scalars on demand.
//
//nolint:exhaustive // KindAny and unknown kinds share the default branch
func (v Value) Any() any {
	switch v.kind {
	case KindString:
		return v.str
	case KindInt64:
		return v.Int64()
	case KindUint64:
		return v.num
	case KindFloat64:
		return v.Float64()
	case KindBool:
		return v.Bool()
	case KindDuration:
		return v.Duration()
	case KindTime:
		return v.Time()
	case KindGroup:
		return v.Group()
	default:
		return v.any
	}
}

// String renders the value as text.
//
// It allocates; encoders use AppendTo instead.
func (v Value) String() string {
	if v.kind == KindString {
		return v.str
	}

	return string(v.AppendTo(nil))
}

// AppendTo appends the text representation of v to dst and returns the extended
// buffer. This is the allocation-free path used by every built-in encoder.
//
//nolint:exhaustive // KindAny and unknown kinds share the default branch
func (v Value) AppendTo(dst []byte) []byte {
	switch v.kind {
	case KindString:
		return append(dst, v.str...)
	case KindInt64:
		return strconv.AppendInt(dst, int64(v.num), base10)
	case KindUint64:
		return strconv.AppendUint(dst, v.num, base10)
	case KindFloat64:
		return strconv.AppendFloat(dst, v.Float64(), floatFormat, floatPrecision, float64Bits)
	case KindBool:
		return strconv.AppendBool(dst, v.num != 0)
	case KindDuration:
		return append(dst, v.Duration().String()...)
	case KindTime:
		return v.Time().AppendFormat(dst, TimeFieldFormat)
	case KindBytes:
		return append(dst, v.Bytes()...)
	case KindError:
		if err := v.Err(); err != nil {
			return append(dst, err.Error()...)
		}

		return append(dst, "<nil>"...)
	case KindStringer:
		if s, ok := v.any.(fmt.Stringer); ok && s != nil {
			return append(dst, s.String()...)
		}

		return append(dst, "<nil>"...)
	case KindGroup:
		return appendGroup(dst, v.Group())
	default:
		if v.any == nil {
			return append(dst, "<nil>"...)
		}

		return fmt.Append(dst, v.any)
	}
}

func appendGroup(dst []byte, fields []Field) []byte {
	dst = append(dst, '[')

	for i := range fields {
		if i > 0 {
			dst = append(dst, ' ')
		}

		dst = append(dst, fields[i].Key...)
		dst = append(dst, '=')
		dst = fields[i].Val.AppendTo(dst)
	}

	return append(dst, ']')
}
