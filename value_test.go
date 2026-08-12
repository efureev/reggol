package reggol

import (
	"errors"
	"math"
	"net"
	"testing"
	"time"
)

func TestValueRoundTrip(t *testing.T) {
	now := time.Date(2026, 8, 12, 13, 45, 30, 123456789, time.UTC)

	for _, tc := range []struct {
		name string
		val  Value
		kind Kind
		text string
	}{
		{"string", StringValue("hello"), KindString, "hello"},
		{"empty string", StringValue(""), KindString, ""},
		{"int", IntValue(-42), KindInt64, "-42"},
		{"int64 max", Int64Value(math.MaxInt64), KindInt64, "9223372036854775807"},
		{"uint64 max", Uint64Value(math.MaxUint64), KindUint64, "18446744073709551615"},
		{"float", Float64Value(3.5), KindFloat64, "3.5"},
		{"bool true", BoolValue(true), KindBool, "true"},
		{"bool false", BoolValue(false), KindBool, "false"},
		{"duration", DurationValue(90 * time.Second), KindDuration, "1m30s"},
		{"time", TimeValue(now), KindTime, "2026-08-12T13:45:30Z"},
		{"error", ErrValue(errors.New("boom")), KindError, "boom"},
		{"nil error", ErrValue(nil), KindError, "<nil>"},
		{"bytes", BytesValue([]byte("raw")), KindBytes, "raw"},
		{"stringer", StringerValue(net.IPv4(10, 0, 0, 1)), KindStringer, "10.0.0.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.val.Kind(); got != tc.kind {
				t.Fatalf("Kind = %v, want %v", got, tc.kind)
			}

			if got := string(tc.val.AppendTo(nil)); got != tc.text {
				t.Fatalf("AppendTo = %q, want %q", got, tc.text)
			}

			if got := tc.val.String(); got != tc.text {
				t.Fatalf("String = %q, want %q", got, tc.text)
			}
		})
	}
}

// TestTimeValueIsLossless covers the decomposition into seconds, nanoseconds and
// location. A UnixNano-based encoding would break at both ends of the range,
// and the zero time is the case the benchmarks depend on.
func TestTimeValueIsLossless(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   time.Time
	}{
		{"zero", time.Time{}},
		{"unix epoch", time.Unix(0, 0).UTC()},
		{"now", time.Now()},
		{"year 1", time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"year 9999", time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)},
		{"pre-1678", time.Date(1500, 6, 15, 12, 0, 0, 0, time.UTC)},
		{"post-2262", time.Date(2500, 6, 15, 12, 0, 0, 0, time.UTC)},
		{"with nanos", time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := TimeValue(tc.in).Time()

			if !got.Equal(tc.in) {
				t.Fatalf("round trip = %v, want %v", got, tc.in)
			}

			if got.Nanosecond() != tc.in.Nanosecond() {
				t.Fatalf("nanoseconds = %d, want %d", got.Nanosecond(), tc.in.Nanosecond())
			}
		})
	}
}

func TestTimeValuePreservesLocation(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}

	in := time.Date(2026, 8, 12, 13, 0, 0, 0, loc)

	got := TimeValue(in).Time()
	if got.Location().String() != loc.String() {
		t.Fatalf("location = %v, want %v", got.Location(), loc)
	}

	if !got.Equal(in) {
		t.Fatalf("instant = %v, want %v", got, in)
	}
}

func TestAnyValuePicksSpecificKind(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   any
		want Kind
	}{
		{"string", "s", KindString},
		{"int", 1, KindInt64},
		{"int8", int8(1), KindInt64},
		{"int64", int64(1), KindInt64},
		{"uint", uint(1), KindUint64},
		{"uint64", uint64(1), KindUint64},
		{"float32", float32(1), KindFloat64},
		{"float64", 1.0, KindFloat64},
		{"bool", true, KindBool},
		{"duration", time.Second, KindDuration},
		{"time", time.Now(), KindTime},
		{"bytes", []byte("b"), KindBytes},
		{"error", errors.New("e"), KindError},
		{"nil", nil, KindAny},
		{"struct", struct{ A int }{1}, KindAny},
		{"value passthrough", StringValue("x"), KindString},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := AnyValue(tc.in).Kind(); got != tc.want {
				t.Fatalf("Kind = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValueAnyRoundTrip(t *testing.T) {
	// Integer kinds are normalized to int64, as slog does, so Any reports
	// int64 regardless of the width that went in.
	if got := AnyValue(42).Any(); got != int64(42) {
		t.Fatalf("Any = %T(%v), want int64(42)", got, got)
	}

	if got := AnyValue("s").Any(); got != "s" {
		t.Fatalf("Any = %v, want s", got)
	}

	if got := AnyValue(true).Any(); got != true {
		t.Fatalf("Any = %v, want true", got)
	}
}

func TestGroupValue(t *testing.T) {
	v := GroupValue(String("a", "1"), Int("b", 2))

	if v.Kind() != KindGroup {
		t.Fatalf("Kind = %v, want KindGroup", v.Kind())
	}

	if got := len(v.Group()); got != 2 {
		t.Fatalf("group size = %d, want 2", got)
	}

	if got := string(v.AppendTo(nil)); got != "[a=1 b=2]" {
		t.Fatalf("AppendTo = %q", got)
	}
}

func TestSortFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int
	}{
		{"insertion path", 5},
		{"general path", insertionSortThreshold + 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := make([]Field, 0, tc.size)
			for i := tc.size - 1; i >= 0; i-- {
				fields = append(fields, Int(string(rune('a'+i%26))+string(rune('0'+i/26)), i))
			}

			sortFields(fields)

			for i := 1; i < len(fields); i++ {
				if fields[i-1].Key > fields[i].Key {
					t.Fatalf("not sorted at %d: %q > %q", i, fields[i-1].Key, fields[i].Key)
				}
			}
		})
	}
}

// TestSortFieldsIsStable pins the duplicate-key semantics: both entries survive
// and keep their call order.
func TestSortFieldsIsStable(t *testing.T) {
	fields := []Field{String("k", "first"), String("a", "x"), String("k", "second")}

	sortFields(fields)

	if fields[1].Val.String() != "first" || fields[2].Val.String() != "second" {
		t.Fatalf("stability lost: %v", fields)
	}
}
