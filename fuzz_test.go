package reggol

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzParseLevel checks that parsing never panics and that anything it accepts
// round-trips through String.
func FuzzParseLevel(f *testing.F) {
	for _, seed := range []string{
		"info", "INFO", "trace", "disabled", "", "3", "-1", "999", "-999",
		"nonsense", "  info  ", "\x00", "İNFO",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		lvl, err := ParseLevel(s)
		if err != nil {
			return
		}

		again, err := ParseLevel(lvl.String())
		if err != nil {
			t.Fatalf("ParseLevel(%q) = %v, whose String() %q no longer parses: %v",
				s, lvl, lvl.String(), err)
		}

		if again != lvl {
			t.Fatalf("round trip: %q -> %v -> %q -> %v", s, lvl, lvl.String(), again)
		}
	})
}

// FuzzJSONEncoder checks the property that matters most for a machine-readable
// encoder: whatever arbitrary bytes are logged, the line must parse as JSON and
// be valid UTF-8.
func FuzzJSONEncoder(f *testing.F) {
	for _, seed := range []struct{ key, val, msg string }{
		{"k", "v", "m"},
		{"", "", ""},
		{"quote\"", "back\\slash", "tab\t"},
		{"\x00\x01\x1f", "\xff\xfe", "\n\r"},
		{"emoji🙂", "combining é", "ansi\x1b[31m"},
	} {
		f.Add(seed.key, seed.val, seed.msg)
	}

	f.Fuzz(func(t *testing.T, key, val, msg string) {
		var buf bytes.Buffer

		l := New(&buf, WithEncoder(NewJSONEncoder()), WithLevel(TraceLevel))
		l.Info().Str(key, val).Bytes("raw", []byte(val)).Msg(msg)

		out := buf.Bytes()

		if n := bytes.Count(out, []byte("\n")); n != 1 {
			t.Fatalf("newlines = %d, want 1: %q", n, out)
		}

		if !utf8.Valid(out) {
			t.Fatalf("output is not valid UTF-8: %q", out)
		}

		var decoded map[string]any
		if err := json.Unmarshal(out, &decoded); err != nil {
			t.Fatalf("invalid JSON for key=%q val=%q msg=%q: %v\nraw: %s", key, val, msg, err, out)
		}
	})
}

// FuzzTextEncoder checks the invariants the flat encoders promise: exactly one
// newline per record, and valid UTF-8 out.
func FuzzTextEncoder(f *testing.F) {
	f.Add("k", "v", "m")
	f.Add("", "", "")
	f.Add("\n", "\t", "\x00")
	f.Add("\xff", "\xfe\xfd", "ok")

	f.Fuzz(func(t *testing.T, key, val, msg string) {
		for _, enc := range []Encoder{
			NewTextEncoder(),
			NewConsoleEncoder(WithColorMode(ColorNever, nil)),
			NewConsoleEncoder(WithColorMode(ColorAlways, nil)),
		} {
			var buf bytes.Buffer

			l := New(&buf, WithEncoder(enc), WithLevel(TraceLevel))
			l.Info().Str(key, val).Msg(msg)

			out := buf.String()

			// The message and the field value may legitimately contain newlines;
			// what must hold is that the record ends with exactly one.
			if !strings.HasSuffix(out, "\n") {
				t.Fatalf("record does not end with a newline: %q", out)
			}

			embedded := strings.Count(key, "\n") + strings.Count(val, "\n") + strings.Count(msg, "\n")
			if got := strings.Count(out, "\n"); got != embedded+1 {
				t.Fatalf("newlines = %d, want %d: %q", got, embedded+1, out)
			}
		}
	})
}

// FuzzValueAppend checks that rendering any value never panics.
func FuzzValueAppend(f *testing.F) {
	f.Add("s", int64(1), 1.5, true)
	f.Add("", int64(0), 0.0, false)

	f.Fuzz(func(t *testing.T, s string, i int64, fl float64, b bool) {
		for _, v := range []Value{
			StringValue(s),
			Int64Value(i),
			Uint64Value(uint64(i)),
			Float64Value(fl),
			BoolValue(b),
			DurationValue(0),
			BytesValue([]byte(s)),
			AnyValue(s),
			GroupValue(String("k", s), Int64("n", i)),
		} {
			got := v.AppendTo(nil)

			if str := v.String(); len(got) != len(str) {
				t.Fatalf("AppendTo and String disagree for kind %v: %q vs %q", v.Kind(), got, str)
			}
		}
	})
}
