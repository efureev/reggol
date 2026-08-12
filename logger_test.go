package reggol

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// withGlobalLevel lowers the process-wide minimum for the duration of a test.
//
// The global default is InfoLevel, so trace and debug events are filtered no
// matter what a single logger is configured to accept.
func withGlobalLevel(t *testing.T, lvl Level) {
	t.Helper()

	prev := GlobalLevel()
	t.Cleanup(func() { SetGlobalLevel(prev) })
	SetGlobalLevel(lvl)
}

// newTestLogger returns a logger writing into buf with a deterministic encoder:
// no timestamp, no color.
func newTestLogger(buf *bytes.Buffer, opts ...EncoderOption) Logger {
	opts = append([]EncoderOption{WithoutTimestamp()}, opts...)

	enc := NewConsoleEncoder(
		WithColorMode(ColorNever, nil),
		WithConsoleOptions(opts...),
	)

	return New(buf, WithEncoder(enc), WithLevel(TraceLevel))
}

// TestNewIsUsableWithoutAVariable pins the v0 defect where New returned a value
// while the level methods lived on *Logger, making this exact expression — the
// one printed in the README — fail to compile.
func TestNewIsUsableWithoutAVariable(t *testing.T) {
	var buf bytes.Buffer

	enc := NewTextEncoder(WithoutTimestamp())
	New(&buf, WithEncoder(enc), WithLevel(TraceLevel)).Warn().Msg("disk almost full")

	if got := buf.String(); !strings.Contains(got, "disk almost full") {
		t.Fatalf("message missing from %q", got)
	}
}

// TestErrDoesNotDisplaceMessage pins the v0 defect where attaching an error
// silently dropped the message text.
func TestErrDoesNotDisplaceMessage(t *testing.T) {
	for _, tc := range []struct {
		name string
		enc  Encoder
	}{
		{"console", NewConsoleEncoder(WithColorMode(ColorNever, nil), WithConsoleOptions(WithoutTimestamp()))},
		{"text", NewTextEncoder(WithoutTimestamp())},
		{"json", NewJSONEncoder(WithoutTimestamp())},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			l := New(&buf, WithEncoder(tc.enc), WithLevel(TraceLevel))
			l.Error().Err(errors.New("boom")).Str("k", "v").Msg("important context")

			got := buf.String()

			for _, want := range []string{"important context", "boom", "k"} {
				if !strings.Contains(got, want) {
					t.Errorf("output %q is missing %q", got, want)
				}
			}
		})
	}
}

// TestAnErrHonoursKey pins the v0 defect where the explicit key was discarded.
func TestAnErrHonoursKey(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf)
	l.Error().AnErr("custom_key", errors.New("boom")).Send()

	got := buf.String()
	if !strings.Contains(got, "custom_key=boom") {
		t.Fatalf("expected custom_key=boom, got %q", got)
	}
}

func TestLevelFiltering(t *testing.T) {
	for _, tc := range []struct {
		name    string
		min     Level
		emit    func(Logger)
		wantOut bool
	}{
		{"below minimum is dropped", WarnLevel, func(l Logger) { l.Info().Msg("x") }, false},
		{"at minimum passes", WarnLevel, func(l Logger) { l.Warn().Msg("x") }, true},
		{"above minimum passes", WarnLevel, func(l Logger) { l.Error().Msg("x") }, true},
		{"disabled drops everything", Disabled, func(l Logger) { l.Error().Msg("x") }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			tc.emit(newTestLogger(&buf).Level(tc.min))

			if got := buf.Len() > 0; got != tc.wantOut {
				t.Fatalf("output=%v, want %v (buffer=%q)", got, tc.wantOut, buf.String())
			}
		})
	}
}

func TestGlobalLevelRaisesMinimum(t *testing.T) {
	prev := GlobalLevel()
	t.Cleanup(func() { SetGlobalLevel(prev) })

	var buf bytes.Buffer

	l := newTestLogger(&buf)

	SetGlobalLevel(ErrorLevel)
	l.Info().Msg("hidden")

	if buf.Len() != 0 {
		t.Fatalf("global level ignored, got %q", buf.String())
	}

	l.Error().Msg("visible")

	if !strings.Contains(buf.String(), "visible") {
		t.Fatalf("error should pass the global level, got %q", buf.String())
	}
}

func TestNopLoggerWritesNothing(t *testing.T) {
	l := Nop()
	l.Error().Str("k", "v").Msg("nothing")

	if l.GetLevel() != Disabled {
		t.Fatalf("Nop level = %v, want Disabled", l.GetLevel())
	}
}

func TestExactlyOneNewlinePerEvent(t *testing.T) {
	for _, tc := range []struct {
		name string
		enc  Encoder
	}{
		{"console", NewConsoleEncoder(WithColorMode(ColorNever, nil))},
		{"text", NewTextEncoder()},
		{"json", NewJSONEncoder()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			l := New(&buf, WithEncoder(tc.enc), WithLevel(TraceLevel))
			l.Info().Msg("one")
			l.Info().Msg("two")

			if n := bytes.Count(buf.Bytes(), []byte("\n")); n != 2 {
				t.Fatalf("newlines=%d, want 2; output=%q", n, buf.String())
			}
		})
	}
}

func TestLoggerImplementsIOWriter(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf)

	n, err := l.Write([]byte("from stdlib\n"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if n != len("from stdlib\n") {
		t.Fatalf("n=%d, want %d", n, len("from stdlib\n"))
	}

	got := buf.String()
	if !strings.Contains(got, "from stdlib") {
		t.Fatalf("missing payload in %q", got)
	}

	if strings.Count(got, "\n") != 1 {
		t.Fatalf("trailing newline was not trimmed: %q", got)
	}
}

func TestDiscardWritesNothing(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf)

	e := l.Info()
	if e.Discard() != nil {
		t.Fatal("Discard should return nil")
	}

	if buf.Len() != 0 {
		t.Fatalf("discarded event was written: %q", buf.String())
	}
}

func TestNilEventIsSafe(t *testing.T) {
	l := newTestLogger(&bytes.Buffer{}).Level(Disabled)

	// Every chained call must tolerate a nil *Event.
	l.Info().Str("k", "v").Int("n", 1).Err(errors.New("x")).Blocks("b").Msg("never")

	if e := l.Info(); e.Enabled() {
		t.Fatal("event on a disabled logger should not be enabled")
	}
}

func TestWithLevelCoversEveryLevel(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	for _, lvl := range []Level{TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, NoLevel} {
		t.Run(lvl.String(), func(t *testing.T) {
			var buf bytes.Buffer

			l := New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp())), WithLevel(TraceLevel))
			l.WithLevel(lvl).Msg("m")

			got := buf.String()
			if !strings.Contains(got, "message=m") {
				t.Fatalf("missing message in %q", got)
			}

			if lvl != NoLevel && !strings.Contains(got, "level="+lvl.String()) {
				t.Fatalf("missing level %q in %q", lvl, got)
			}
		})
	}

	if e := newTestLogger(&bytes.Buffer{}).WithLevel(Disabled); e != nil {
		t.Fatal("WithLevel(Disabled) should return nil")
	}
}

func TestBlocksRenderInOrder(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf)
	l.Info().Blocks("API", "GET /users").Msg("ok")

	got := buf.String()
	if !strings.Contains(got, "API GET /users ok") {
		t.Fatalf("blocks not rendered before the message: %q", got)
	}
}

func TestBlockDecoratorIsApplied(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf)
	l.Info().Block(NewBlock("auth", func(s string) string { return "[" + s + "]" })).Msg("verified")

	if got := buf.String(); !strings.Contains(got, "[auth] verified") {
		t.Fatalf("decorator not applied: %q", got)
	}
}

func TestDuplicateKeysAreBothEmitted(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf)
	l.Info().Str("k", "a").Str("k", "b").Send()

	got := buf.String()
	if strings.Count(got, "k=") != 2 {
		t.Fatalf("expected both duplicate keys, got %q", got)
	}
}

func TestPrintFamilyLogsAtDebug(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	var buf bytes.Buffer

	l := newTestLogger(&buf)

	l.Print("a", "b")
	l.Printf("n=%d", 42)
	l.Println("line")

	got := buf.String()
	for _, want := range []string{"ab", "n=42", "line"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestErrHelperPicksLevel(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp())), WithLevel(TraceLevel))

	l.Err(nil).Msg("no error")

	if !strings.Contains(buf.String(), "level=info") {
		t.Fatalf("nil error should log at info: %q", buf.String())
	}

	buf.Reset()
	l.Err(errors.New("bad")).Msg("with error")

	if !strings.Contains(buf.String(), "level=error") {
		t.Fatalf("non-nil error should log at error: %q", buf.String())
	}
}
