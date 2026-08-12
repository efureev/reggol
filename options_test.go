package reggol

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// hookTime is the timestamp the hook tests render.
var hookTime = time.Date(2026, 8, 12, 13, 45, 30, 0, time.UTC)

// renderWith builds a logger over enc and returns the single record it writes.
func renderWith(t *testing.T, enc Encoder, emit func(Logger)) string {
	t.Helper()

	var buf bytes.Buffer

	emit(New(&buf, WithEncoder(enc), WithLevel(TraceLevel)))

	return strings.TrimSuffix(buf.String(), "\n")
}

// TestEncoderHooksAreInvoked walks every formatting hook and proves it reaches
// the output, rather than merely that the option compiles.
func TestEncoderHooksAreInvoked(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	for _, tc := range []struct {
		name string
		opts []EncoderOption
		emit func(Logger)
		want string
	}{
		{
			name: "level",
			opts: []EncoderOption{WithLevelFormatter(func(dst []byte, l Level) []byte {
				return append(dst, "<"+l.String()+">"...)
			})},
			emit: func(l Logger) { l.Warn().Send() },
			want: "level=<warn>",
		},
		{
			name: "time",
			opts: []EncoderOption{WithTimeFormatter(func(dst []byte, tm time.Time) []byte {
				return strconv.AppendInt(dst, tm.Unix(), 10)
			})},
			emit: func(l Logger) { l.Info().Timestamp(hookTime).Send() },
			want: "ts=" + strconv.FormatInt(hookTime.Unix(), 10),
		},
		{
			name: "key",
			opts: []EncoderOption{WithKeyFormatter(func(dst []byte, key string) []byte {
				return append(dst, strings.ToUpper(key)...)
			})},
			emit: func(l Logger) { l.Info().Str("host", "db-1").Send() },
			want: "HOST=db-1",
		},
		{
			name: "value",
			opts: []EncoderOption{WithValueFormatter(func(dst []byte, v Value) []byte {
				return append(dst, "["+v.String()+"]"...)
			})},
			emit: func(l Logger) { l.Info().Str("host", "db-1").Send() },
			want: "host=[db-1]",
		},
		{
			name: "field replaces key and value together",
			opts: []EncoderOption{WithFieldFormatter(func(dst []byte, key string, v Value) []byte {
				return append(dst, key+"->"+v.String()...)
			})},
			emit: func(l Logger) { l.Info().Str("host", "db-1").Send() },
			want: "host->db-1",
		},
		{
			name: "message",
			opts: []EncoderOption{WithMessageFormatter(func(dst, msg []byte) []byte {
				dst = append(dst, "«"...)
				dst = append(dst, msg...)

				return append(dst, "»"...)
			})},
			emit: func(l Logger) { l.Info().Msg("hello") },
			want: "message=«hello»",
		},
		{
			name: "blocks",
			opts: []EncoderOption{WithBlocksFormatter(func(dst []byte, b Blocks) []byte {
				dst = append(dst, "tags="...)

				for i := range b {
					if i > 0 {
						dst = append(dst, '/')
					}

					dst = append(dst, b[i].Value()...)
				}

				return dst
			})},
			emit: func(l Logger) { l.Info().Blocks("API", "GET").Send() },
			want: "tags=API/GET",
		},
		{
			name: "caller",
			opts: []EncoderOption{WithCallerFormatter(func(dst []byte, _ string, line int) []byte {
				dst = append(dst, "line#"...)

				return strconv.AppendInt(dst, int64(line), 10)
			})},
			emit: func(l Logger) { l.Info().Caller().Send() },
			want: "caller=line#",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := append([]EncoderOption{WithoutTimestamp()}, tc.opts...)
			if tc.name == "time" {
				opts = tc.opts // this one needs the timestamp shown
			}

			got := renderWith(t, NewTextEncoder(opts...), tc.emit)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("hook output missing\n  want: %s\n   got: %s", tc.want, got)
			}
		})
	}
}

// TestFieldFormatterInEveryEncoder covers the per-encoder branch: each one has
// its own appendField, and each must honor the hook.
func TestFieldFormatterInEveryEncoder(t *testing.T) {
	hook := WithFieldFormatter(func(dst []byte, key string, v Value) []byte {
		return append(dst, key+"~"+v.String()...)
	})

	for _, tc := range []struct {
		name string
		enc  Encoder
	}{
		{"text", NewTextEncoder(WithoutTimestamp(), hook)},
		{"json", NewJSONEncoder(WithoutTimestamp(), hook)},
		{"console", NewConsoleEncoder(
			WithColorMode(ColorNever, nil),
			WithConsoleOptions(WithoutTimestamp(), hook),
		)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := renderWith(t, tc.enc, func(l Logger) { l.Info().Str("k", "v").Send() })

			if !strings.Contains(got, "k~v") {
				t.Fatalf("field hook ignored by %s: %s", tc.name, got)
			}
		})
	}
}

// TestConsoleLevelFormatter covers the console encoder's own appendLevel, which
// is separate from the text one because it also applies color.
func TestConsoleLevelFormatter(t *testing.T) {
	enc := NewConsoleEncoder(
		WithColorMode(ColorNever, nil),
		WithConsoleOptions(WithoutTimestamp(), WithLevelFormatter(func(dst []byte, l Level) []byte {
			return append(dst, strings.ToUpper(l.String())...)
		})),
	)

	got := renderWith(t, enc, func(l Logger) { l.Warn().Msg("m") })
	if !strings.HasPrefix(got, "WARN ") {
		t.Fatalf("console level hook ignored: %q", got)
	}
}

func TestWithKeyNames(t *testing.T) {
	enc := NewJSONEncoder(WithKeyNames("t", "lvl", "text"))

	got := renderWith(t, enc, func(l Logger) { l.Info().Timestamp(hookTime).Msg("m") })

	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}

	for _, key := range []string{"t", "lvl", "text"} {
		if _, ok := out[key]; !ok {
			t.Errorf("key %q missing from %s", key, got)
		}
	}

	for _, key := range []string{TimestampFieldName, LevelFieldName, MessageFieldName} {
		if _, ok := out[key]; ok {
			t.Errorf("default key %q should have been replaced: %s", key, got)
		}
	}
}

// TestWithKeyNamesIgnoresEmpty pins that an empty name leaves the default alone,
// so callers can override just one of the three.
func TestWithKeyNamesIgnoresEmpty(t *testing.T) {
	enc := NewTextEncoder(WithoutTimestamp(), WithKeyNames("", "", "text"))

	got := renderWith(t, enc, func(l Logger) { l.Info().Msg("m") })

	if !strings.Contains(got, "level=info") || !strings.Contains(got, "text=m") {
		t.Fatalf("partial override went wrong: %s", got)
	}
}

func TestStyleWithFgAndWithBg(t *testing.T) {
	base := Style{Attrs: ColorBold}

	fg := base.WithFg(ColorRGB(255, 0, 0))
	if _, _, _, ok := fg.Fg.RGB(); !ok {
		t.Fatal("WithFg did not set the foreground")
	}

	if fg.Bg != 0 || fg.Attrs != ColorBold {
		t.Fatalf("WithFg touched the rest of the style: %+v", fg)
	}

	bg := base.WithBg(Color256(17))
	if _, ok := bg.Bg.Index(); !ok {
		t.Fatal("WithBg did not set the background")
	}

	if bg.Fg != 0 {
		t.Fatalf("WithBg touched the foreground: %+v", bg)
	}

	// The receiver is a value: the original must be untouched.
	if base.Fg != 0 || base.Bg != 0 {
		t.Fatalf("the original style was mutated: %+v", base)
	}
}

func TestWithCallerSkipOption(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	var buf bytes.Buffer

	// One extra frame declared, so the position should be this test's caller
	// rather than this line — asserting only that it moved.
	l := New(&buf,
		WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())),
		WithLevel(TraceLevel),
		WithCaller(),
		WithCallerSkip(1),
	)

	l.Info().Msg("m")

	got := buf.String()
	if !strings.Contains(got, CallerFieldName+"=") {
		t.Fatalf("no call site recorded: %q", got)
	}

	if strings.Contains(got, "options_test.go") {
		t.Fatalf("the extra skip had no effect, still pointing here: %q", got)
	}
}

func TestWithContextOnNilContext(t *testing.T) {
	l := Nop()

	//nolint:staticcheck // passing nil is the case under test
	ctx := l.WithContext(nil)
	if ctx == nil {
		t.Fatal("WithContext(nil) must return a usable context")
	}

	if _, ok := FromContext(ctx); !ok {
		t.Fatal("the logger was not stored")
	}

	if ctx.Err() != nil {
		t.Fatalf("unexpected context error: %v", ctx.Err())
	}
}

func TestStyleOfSplitsColorsAndAttributes(t *testing.T) {
	s := StyleOf(ColorFgRed | ColorBgBlue | ColorBold)

	if s.Fg == 0 || s.Bg == 0 {
		t.Fatalf("colors were not extracted: %+v", s)
	}

	if !s.Attrs.Has(ColorBold) {
		t.Fatalf("attributes were lost: %+v", s)
	}

	// A mask with no color bits leaves both unset.
	plain := StyleOf(ColorUnderline)
	if plain.Fg != 0 || plain.Bg != 0 {
		t.Fatalf("colors invented from an attribute-only mask: %+v", plain)
	}
}

// TestJSONSanitizesInvalidUTF8 drives the slow escaping path, which only runs
// for input that is not valid UTF-8 and has to survive every byte class.
func TestJSONSanitizesInvalidUTF8(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"lone invalid byte", "\xff"},
		{"invalid after text", "ok\xff"},
		{"invalid with quote", "a\"\xff"},
		{"invalid with control", "a\x00\xff"},
		{"invalid with newline and tab", "a\n\t\xff"},
		{"valid multibyte then invalid", "héllo\xff"},
		{"truncated multibyte", "a\xe2\x82"},
		{"only invalid bytes", "\xff\xfe\xfd"},
		{"invalid between valid", "a\xffб"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if utf8.ValidString(tc.in) {
				t.Fatalf("fixture %q is valid UTF-8, it would not reach the slow path", tc.in)
			}

			got := renderWith(t, NewJSONEncoder(WithoutTimestamp()),
				func(l Logger) { l.Info().Str("k", tc.in).Msg(tc.in) })

			var out map[string]any
			if err := json.Unmarshal([]byte(got), &out); err != nil {
				t.Fatalf("invalid JSON for %q: %v\nraw: %s", tc.in, err, got)
			}

			if !utf8.ValidString(got) {
				t.Fatalf("output is not valid UTF-8: %q", got)
			}

			// The encoder substitutes the replacement character itself rather
			// than a \ufffd escape: both are valid JSON, and the character is
			// shorter and already valid UTF-8.
			if !strings.ContainsRune(got, utf8.RuneError) {
				t.Fatalf("the invalid bytes were not replaced: %q", got)
			}
		})
	}
}

func TestWithLevelOnCustomLevel(t *testing.T) {
	// Both thresholds have to admit a level this low: the logger's own and the
	// process-wide one.
	withGlobalLevel(t, Level(-10))

	// A numeric level below TraceLevel takes the default branch of WithLevel.
	// The logger's own minimum has to be lowered too, or the event is filtered
	// out before it reaches the encoder.
	var buf bytes.Buffer

	New(&buf,
		WithEncoder(NewTextEncoder(WithoutTimestamp())),
		WithLevel(Level(-10)),
	).WithLevel(Level(-5)).Msg("m")

	got := buf.String()

	if !strings.Contains(got, "level=-5") {
		t.Fatalf("custom level not rendered: %s", got)
	}
}

func TestBeforeAndAfterEncodeRunInOrder(t *testing.T) {
	var order []string

	enc := NewTextEncoder(
		WithoutTimestamp(),
		WithBeforeEncode(func(*EventData) { order = append(order, "before") }),
		WithAfterEncode(func(*EventData) { order = append(order, "after") }),
	)

	renderWith(t, enc, func(l Logger) { l.Info().Msg("m") })

	if len(order) != 2 || order[0] != "before" || order[1] != "after" {
		t.Fatalf("callbacks ran as %v", order)
	}
}

func TestContextExtractorOptionAccumulates(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	type keyA struct{}

	type keyB struct{}

	var buf bytes.Buffer

	l := New(&buf,
		WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())),
		WithLevel(TraceLevel),
		WithContextExtractor(func(ctx context.Context, e *Event) {
			if v, ok := ctx.Value(keyA{}).(string); ok {
				e.Str("a", v)
			}
		}),
		WithContextExtractor(func(ctx context.Context, e *Event) {
			if v, ok := ctx.Value(keyB{}).(string); ok {
				e.Str("b", v)
			}
		}),
		// A nil extractor must be ignored rather than panic later.
		WithContextExtractor(nil),
	)

	ctx := context.WithValue(context.WithValue(t.Context(), keyA{}, "1"), keyB{}, "2")
	l.Ctx(ctx, InfoLevel).Send()

	got := buf.String()
	if !strings.Contains(got, "a=1") || !strings.Contains(got, "b=2") {
		t.Fatalf("both extractors should have run: %q", got)
	}
}
