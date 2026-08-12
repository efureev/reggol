package reggol

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// fixedTime keeps encoder output deterministic.
var fixedTime = time.Date(2026, 8, 12, 13, 45, 30, 0, time.UTC)

// encodeEvent renders one event with the given encoder, bypassing the logger so
// that the timestamp is fixed.
func encodeEvent(enc Encoder, build func(*EventData)) string {
	d := &EventData{ts: fixedTime, level: InfoLevel}
	build(d)

	return string(enc.AppendEvent(nil, d))
}

func TestConsoleEncoderGolden(t *testing.T) {
	enc := NewConsoleEncoder(
		WithColorMode(ColorNever, nil),
		WithConsoleOptions(WithTimeFormat(time.RFC3339)),
	)

	for _, tc := range []struct {
		name  string
		build func(*EventData)
		want  string
	}{
		{
			name:  "message only",
			build: func(d *EventData) { d.message = msgHello },
			want:  "2026-08-12T13:45:30Z INF " + msgHello + "\n",
		},
		{
			name:  "no level",
			build: func(d *EventData) { d.level = NoLevel; d.message = "plain" },
			want:  "2026-08-12T13:45:30Z plain\n",
		},
		{
			name: "fields are sorted",
			build: func(d *EventData) {
				d.message = "m"
				d.fields = []Field{Int("z", 1), String("a", "x")}
			},
			want: "2026-08-12T13:45:30Z INF m a=x z=1\n",
		},
		{
			name: "blocks precede message",
			build: func(d *EventData) {
				d.message = "ok"
				d.blocks = Blocks{{Text: "API"}, {Text: "GET"}}
			},
			want: "2026-08-12T13:45:30Z INF API GET ok\n",
		},
		{
			name: "error keeps the message",
			build: func(d *EventData) {
				d.level = ErrorLevel
				d.message = "context"
				d.fields = []Field{Err(errors.New("boom"))}
			},
			want: "2026-08-12T13:45:30Z ERR context boom\n",
		},
		{
			name: "empty message with fields",
			build: func(d *EventData) {
				d.fields = []Field{String("k", "v")}
			},
			want: "2026-08-12T13:45:30Z INF k=v\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := encodeEvent(enc, tc.build); got != tc.want {
				t.Fatalf("\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestTextEncoderGolden(t *testing.T) {
	enc := NewTextEncoder(WithTimeFormat(time.RFC3339))

	for _, tc := range []struct {
		name  string
		build func(*EventData)
		want  string
	}{
		{
			name:  "message only",
			build: func(d *EventData) { d.message = msgHello },
			want:  "ts=2026-08-12T13:45:30Z, level=info, message=" + msgHello + "\n",
		},
		{
			name: "fields",
			build: func(d *EventData) {
				d.message = "m"
				d.fields = []Field{Int("n", 7), Bool("ok", true)}
			},
			want: "ts=2026-08-12T13:45:30Z, level=info, message=m, n=7, ok=true\n",
		},
		{
			name: "blocks",
			build: func(d *EventData) {
				d.message = "m"
				d.blocks = Blocks{{Text: "A"}, {Text: "B"}}
			},
			want: "ts=2026-08-12T13:45:30Z, level=info, blocks=[A, B], message=m\n",
		},
		{
			name: "error is a normal field",
			build: func(d *EventData) {
				d.level = ErrorLevel
				d.message = "context"
				d.fields = []Field{Err(errors.New("boom"))}
			},
			want: "ts=2026-08-12T13:45:30Z, level=error, message=context, error=boom\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := encodeEvent(enc, tc.build); got != tc.want {
				t.Fatalf("\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestJSONEncoderGolden(t *testing.T) {
	enc := NewJSONEncoder(WithTimeFormat(time.RFC3339))

	for _, tc := range []struct {
		name  string
		build func(*EventData)
		want  string
	}{
		{
			name:  "message only",
			build: func(d *EventData) { d.message = msgHello },
			want:  `{"ts":"2026-08-12T13:45:30Z","level":"info","message":"` + msgHello + `"}` + "\n",
		},
		{
			name: "typed fields",
			build: func(d *EventData) {
				d.message = "m"
				d.fields = []Field{Int("n", 7), Bool("ok", true), Float64("f", 1.5)}
			},
			want: `{"ts":"2026-08-12T13:45:30Z","level":"info","message":"m","f":1.5,"n":7,"ok":true}` + "\n",
		},
		{
			name: "group nests",
			build: func(d *EventData) {
				d.fields = []Field{Group("g", String("a", "1"), Int("b", 2))}
			},
			want: `{"ts":"2026-08-12T13:45:30Z","level":"info","g":{"a":"1","b":2}}` + "\n",
		},
		{
			name: "nil error is null",
			build: func(d *EventData) {
				d.fields = []Field{AnErr("e", nil)}
			},
			want: `{"ts":"2026-08-12T13:45:30Z","level":"info","e":null}` + "\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := encodeEvent(enc, tc.build); got != tc.want {
				t.Fatalf("\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestJSONEncoderProducesValidJSON is the property that matters more than any
// single golden line: whatever is logged, the output must parse.
func TestJSONEncoderProducesValidJSON(t *testing.T) {
	nasty := []string{
		`quote"`, `back\slash`, "tab\t", "newline\n", "carriage\r",
		"bell\x07", "null\x00", "\x1b[31mansi", "emoji 🙂", "combining é",
		string([]byte{0xff, 0xfe}), strings.Repeat("x", 1000),
	}

	enc := NewJSONEncoder()

	for _, s := range nasty {
		var buf bytes.Buffer

		l := New(&buf, WithEncoder(enc), WithLevel(TraceLevel))
		l.Info().Str("k", s).Str(s, "v").Msg(s)

		var out map[string]any
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("invalid JSON for input %q: %v\nraw: %s", s, err, buf.String())
		}

		if !utf8.Valid(buf.Bytes()) {
			t.Fatalf("output is not valid UTF-8 for input %q", s)
		}
	}
}

func TestJSONEncoderHandlesNonFiniteFloats(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(NewJSONEncoder()), WithLevel(TraceLevel))
	l.Info().
		Float64("nan", math.NaN()).
		Float64("inf", math.Inf(1)).
		Float64("ninf", math.Inf(-1)).
		Send()

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("non-finite floats produced invalid JSON: %v\nraw: %s", err, buf.String())
	}

	for _, k := range []string{"nan", "inf", "ninf"} {
		if out[k] != nil {
			t.Fatalf("%s = %v, want null", k, out[k])
		}
	}
}

func TestEncoderOptions(t *testing.T) {
	t.Run("without timestamp", func(t *testing.T) {
		got := encodeEvent(NewTextEncoder(WithoutTimestamp()), func(d *EventData) { d.message = "m" })
		if strings.Contains(got, "ts=") {
			t.Fatalf("timestamp not hidden: %q", got)
		}
	})

	t.Run("without level", func(t *testing.T) {
		got := encodeEvent(NewTextEncoder(WithoutLevel()), func(d *EventData) { d.message = "m" })
		if strings.Contains(got, "level=") {
			t.Fatalf("level not hidden: %q", got)
		}
	})

	t.Run("without sort keeps insertion order", func(t *testing.T) {
		got := encodeEvent(
			NewTextEncoder(WithoutTimestamp(), WithoutLevel(), WithoutSort()),
			func(d *EventData) { d.fields = []Field{Int("z", 1), Int("a", 2)} },
		)

		if got != "z=1, a=2\n" {
			t.Fatalf("order changed: %q", got)
		}
	})
}

func TestFormattingHooks(t *testing.T) {
	enc := NewTextEncoder(
		WithoutTimestamp(),
		WithLevelFormatter(func(dst []byte, l Level) []byte {
			return append(dst, strings.ToUpper(l.String())...)
		}),
		WithMessageFormatter(func(dst []byte, msg string) []byte {
			return append(dst, "<"+msg+">"...)
		}),
		WithKeyFormatter(func(dst []byte, key string) []byte {
			return append(dst, "k_"+key...)
		}),
	)

	got := encodeEvent(enc, func(d *EventData) {
		d.message = "m"
		d.fields = []Field{Int("n", 1)}
	})

	want := "level=INFO, message=<m>, k_n=1\n"
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestConsoleColorIsEmittedOnlyWhenAsked(t *testing.T) {
	withColor := encodeEvent(
		NewConsoleEncoder(WithColorMode(ColorAlways, nil)),
		func(d *EventData) { d.message = "m" },
	)

	if !strings.Contains(withColor, "\x1b[") {
		t.Fatalf("ColorAlways produced no escape sequences: %q", withColor)
	}

	withoutColor := encodeEvent(
		NewConsoleEncoder(WithColorMode(ColorNever, nil)),
		func(d *EventData) { d.message = "m" },
	)

	if strings.Contains(withoutColor, "\x1b[") {
		t.Fatalf("ColorNever emitted escape sequences: %q", withoutColor)
	}
}

// TestColorAutoOnNonTerminal covers the v0 gap where redirecting output to a
// file still produced ANSI sequences.
func TestColorAutoOnNonTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")

	if resolveColor(ColorAuto, &bytes.Buffer{}) {
		t.Fatal("a bytes.Buffer is not a terminal")
	}
}

func TestColorEnvOverrides(t *testing.T) {
	t.Run("FORCE_COLOR wins", func(t *testing.T) {
		t.Setenv("FORCE_COLOR", "1")

		if !resolveColor(ColorAuto, &bytes.Buffer{}) {
			t.Fatal("FORCE_COLOR ignored")
		}
	})

	t.Run("TERM=dumb disables", func(t *testing.T) {
		t.Setenv("TERM", "dumb")

		if resolveColor(ColorAuto, &bytes.Buffer{}) {
			t.Fatal("TERM=dumb ignored")
		}
	})
}
