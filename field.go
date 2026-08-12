package reggol

import (
	"fmt"
	"time"
)

// Field is a single key/value pair attached to a log event.
//
// Fields are stored in insertion order in a slice rather than a map: this
// avoids hashing, avoids boxing values into an interface, and keeps sorting a
// choice rather than a necessity. One consequence is user-visible — repeating a
// key no longer replaces the earlier entry, both are emitted, as in slog.
type Field struct {
	Key string
	Val Value
}

// String builds a string field.
func String(key, val string) Field { return Field{Key: key, Val: StringValue(val)} }

// Int builds an int field.
func Int(key string, val int) Field { return Field{Key: key, Val: IntValue(val)} }

// Int64 builds an int64 field.
func Int64(key string, val int64) Field { return Field{Key: key, Val: Int64Value(val)} }

// Uint64 builds a uint64 field.
func Uint64(key string, val uint64) Field { return Field{Key: key, Val: Uint64Value(val)} }

// Float64 builds a float64 field.
func Float64(key string, val float64) Field { return Field{Key: key, Val: Float64Value(val)} }

// Bool builds a bool field.
func Bool(key string, val bool) Field { return Field{Key: key, Val: BoolValue(val)} }

// Dur builds a time.Duration field.
func Dur(key string, val time.Duration) Field { return Field{Key: key, Val: DurationValue(val)} }

// Time builds a time.Time field.
func Time(key string, val time.Time) Field { return Field{Key: key, Val: TimeValue(val)} }

// Bytes builds a byte-slice field.
func Bytes(key string, val []byte) Field { return Field{Key: key, Val: BytesValue(val)} }

// Err builds an error field under the conventional error key.
func Err(err error) Field { return Field{Key: ErrorFieldName, Val: ErrValue(err)} }

// AnErr builds an error field under an explicit key.
func AnErr(key string, err error) Field { return Field{Key: key, Val: ErrValue(err)} }

// Stringer builds a field from a fmt.Stringer.
func Stringer(key string, val fmt.Stringer) Field { return Field{Key: key, Val: StringerValue(val)} }

// Group builds a field holding nested fields.
func Group(key string, fields ...Field) Field { return Field{Key: key, Val: GroupValue(fields...)} }

// Any builds a field from an arbitrary value, picking the most specific Kind.
func Any(key string, val any) Field { return Field{Key: key, Val: AnyValue(val)} }

// sortFields orders fields by key, preserving the relative order of equal keys.
//
// Insertion sort wins decisively for the field counts a log line actually
// carries; the general sort is kept for the rare wide event.
func sortFields(fields []Field) {
	if len(fields) < minSortable {
		return
	}

	if len(fields) > insertionSortThreshold {
		sortFieldsLarge(fields)

		return
	}

	for i := 1; i < len(fields); i++ {
		f := fields[i]

		j := i - 1
		for j >= 0 && fields[j].Key > f.Key {
			fields[j+1] = fields[j]
			j--
		}

		fields[j+1] = f
	}
}

const (
	// insertionSortThreshold is the field count above which the general sort wins.
	insertionSortThreshold = 16
	// minSortable is the shortest slice worth sorting at all.
	minSortable = 2
)
