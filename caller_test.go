package reggol

import (
	"bytes"
	"errors"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// here returns the position of the line that called it, formatted the way the
// encoders format a call site.
//
// Putting here() on the same line as the logging call makes the two line
// numbers identical, which is what lets the matrix below assert an exact
// position without hard-coding any line numbers.
func here() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return "<unknown>"
	}

	return shortCallerPath(file) + ":" + strconv.Itoa(line)
}

// hereAt returns the position of a line offset from the caller's own.
//
// The matrix above keeps the logging call and here() on one line; where that
// would hurt readability, hereAt names the offset explicitly instead.
func hereAt(offset int) string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return "<unknown>"
	}

	return shortCallerPath(file) + ":" + strconv.Itoa(line+offset)
}

// callerLogger returns a logger with call sites enabled and everything else
// stripped, so the output is exactly `caller message`.
func callerLogger(buf *bytes.Buffer) Logger {
	return New(buf,
		WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())),
		WithLevel(TraceLevel),
		WithCaller(),
	)
}

// TestCallerReportsCallSite is the guard for the skip constants.
//
// Every public entry point must funnel through Logger.event exactly one hop
// away. Routing one public method through another — Err through Error, Ctx
// through WithLevel — shifts the reported position into reggol itself, and this
// is the only thing that notices.
func TestCallerReportsCallSite(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	for _, tc := range []struct {
		name string
		emit func(Logger) string
	}{
		{"Trace", func(l Logger) string { l.Trace().Msg("m"); return here() }},
		{"Debug", func(l Logger) string { l.Debug().Msg("m"); return here() }},
		{"Info", func(l Logger) string { l.Info().Msg("m"); return here() }},
		{"Warn", func(l Logger) string { l.Warn().Msg("m"); return here() }},
		{"Error", func(l Logger) string { l.Error().Msg("m"); return here() }},
		{"Log", func(l Logger) string { l.Log().Msg("m"); return here() }},
		{"Err non-nil", func(l Logger) string { l.Err(errors.New("x")).Msg("m"); return here() }},
		{"Err nil", func(l Logger) string { l.Err(nil).Msg("m"); return here() }},
		{"WithLevel", func(l Logger) string { l.WithLevel(WarnLevel).Msg("m"); return here() }},
		{"WithLevel NoLevel", func(l Logger) string { l.WithLevel(NoLevel).Msg("m"); return here() }},
		{"Ctx", func(l Logger) string { l.Ctx(t.Context(), InfoLevel).Msg("m"); return here() }},
		{"Print", func(l Logger) string { l.Print("m"); return here() }},
		{"Printf", func(l Logger) string { l.Printf("m"); return here() }},
		{"Println", func(l Logger) string { l.Println("m"); return here() }},
		{"Write", func(l Logger) string { _, _ = l.Write([]byte("m")); return here() }},
		{"child logger", func(l Logger) string { l.With().Str("k", "v").Logger().Info().Msg("m"); return here() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			want := tc.emit(callerLogger(&buf))

			assertCaller(t, buf.String(), want)
		})
	}
}

// TestCallerPanicAndFatalPaths covers the two entry points that carry a
// completion callback, which is where the chains were most tangled.
func TestCallerPanicAndFatalPaths(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	t.Run("Panic", func(t *testing.T) {
		var buf bytes.Buffer

		func() {
			// The panic unwinds before here() could run, so this case asserts
			// the file rather than an exact line.
			defer func() { _ = recover() }()

			callerLogger(&buf).Panic().Msg("m")
		}()

		if !strings.Contains(buf.String(), "caller_test.go:") {
			t.Fatalf("Panic did not report a call site in this file: %q", buf.String())
		}
	})

	t.Run("WithLevel(Panic)", func(t *testing.T) {
		var buf bytes.Buffer

		func() {
			defer func() { _ = recover() }()

			callerLogger(&buf).WithLevel(PanicLevel).Msg("m")
		}()

		if !strings.Contains(buf.String(), "caller_test.go:") {
			t.Fatalf("WithLevel(Panic) did not report a call site in this file: %q", buf.String())
		}
	})
}

// TestEventCallerOptIn covers the per-event opt-in on a logger that does not
// capture call sites itself.
func TestEventCallerOptIn(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	var buf bytes.Buffer

	l := New(&buf,
		WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())),
		WithLevel(TraceLevel),
	)

	l.Info().Caller().Msg("m")

	want := hereAt(-2)

	assertCaller(t, buf.String(), want)

	// Without the opt-in nothing is reported.
	buf.Reset()
	l.Info().Msg("m")

	if strings.Contains(buf.String(), CallerFieldName) {
		t.Fatalf("caller reported without opting in: %q", buf.String())
	}
}

// TestAddCallerSkip covers what a wrapping library needs: without the extra
// skip the wrapper's own line is reported instead of its caller's.
func TestAddCallerSkip(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	var buf bytes.Buffer

	l := callerLogger(&buf)

	// A helper that logs on behalf of its caller.
	logVia := func(l Logger, msg string) { l.AddCallerSkip(1).Info().Msg(msg) }

	logVia(l, "m")

	want := hereAt(-2)

	assertCaller(t, buf.String(), want)

	buf.Reset()

	// Without the adjustment the helper's own line is reported.
	naive := func(l Logger, msg string) { l.Info().Msg(msg) }
	naive(l, "m")

	if strings.Contains(buf.String(), want) {
		t.Fatalf("skip adjustment had no effect: %q", buf.String())
	}
}

// TestCallerDisabledByDefault pins the opt-in contract.
func TestCallerDisabledByDefault(t *testing.T) {
	var buf bytes.Buffer

	New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp())), WithLevel(TraceLevel)).
		Info().Msg("m")

	if strings.Contains(buf.String(), CallerFieldName) {
		t.Fatalf("caller is on by default: %q", buf.String())
	}
}

func TestCallerInEveryEncoder(t *testing.T) {
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

			l := New(&buf, WithEncoder(tc.enc), WithLevel(TraceLevel), WithCaller())
			l.Info().Msg("m")

			want := hereAt(-2)

			if !strings.Contains(buf.String(), want) {
				t.Fatalf("missing %q in %q", want, buf.String())
			}
		})
	}
}

func TestCallerFormatterHook(t *testing.T) {
	var buf bytes.Buffer

	enc := NewTextEncoder(
		WithoutTimestamp(),
		WithoutLevel(),
		WithCallerFormatter(func(dst []byte, file string, line int) []byte {
			dst = append(dst, "at "...)
			dst = append(dst, filepath.Base(file)...)
			dst = append(dst, '#')

			return strconv.AppendInt(dst, int64(line), 10)
		}),
	)

	New(&buf, WithEncoder(enc), WithLevel(TraceLevel), WithCaller()).Info().Msg("m")

	if got := buf.String(); !strings.Contains(got, "caller=at caller_test.go#") {
		t.Fatalf("hook not applied: %q", got)
	}
}

func TestEventDataCallerAccessors(t *testing.T) {
	var captured *EventData

	enc := NewTextEncoder(WithBeforeEncode(func(d *EventData) { captured = d }))

	var buf bytes.Buffer

	New(&buf, WithEncoder(enc), WithLevel(TraceLevel), WithCaller()).Info().Msg("m")

	if captured == nil {
		t.Fatal("BeforeEncode was not called")
	}

	if captured.PC() == 0 {
		t.Fatal("PC was not recorded")
	}

	file, line, ok := captured.Caller()
	if !ok || !strings.HasSuffix(file, "caller_test.go") || line == 0 {
		t.Fatalf("Caller = %q:%d, ok=%v", file, line, ok)
	}

	if fn := captured.CallerFunction(); !strings.Contains(fn, "TestEventDataCallerAccessors") {
		t.Fatalf("CallerFunction = %q", fn)
	}
}

func TestShortCallerPath(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"/home/bob/src/myapp/api/handler.go", "api/handler.go"},
		{"api/handler.go", "api/handler.go"},
		{"handler.go", "handler.go"},
		{"/handler.go", "/handler.go"},
		{`C:\src\myapp\api\handler.go`, `api\handler.go`},
		{"", ""},
	} {
		if got := shortCallerPath(tc.in); got != tc.want {
			t.Errorf("shortCallerPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolvePCHandlesZero(t *testing.T) {
	if _, _, ok := resolvePC(0); ok {
		t.Fatal("a zero program counter must not resolve")
	}

	if got := funcForPC(0); got != "" {
		t.Fatalf("funcForPC(0) = %q, want empty", got)
	}

	if got := appendCallerPosition(nil, 0); len(got) != 0 {
		t.Fatalf("appendCallerPosition wrote %q for a zero counter", got)
	}
}

// TestCallerMatchesRuntimeFrames guards the shortcut in resolvePC.
//
// The runtime documentation points at CallersFrames, and reggol uses FuncForPC
// instead because it allocates nothing. The two agree only because resolvePC
// undoes the return-address offset; without that, an event started through Err,
// WithLevel or Ctx resolved to a line inside reggol itself. Every entry point is
// checked against the runtime's own answer so that a regression cannot hide.
func TestCallerMatchesRuntimeFrames(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	for _, tc := range []struct {
		name string
		emit func(Logger)
	}{
		{"Info", func(l Logger) { l.Info().Msg("m") }},
		{"Err", func(l Logger) { l.Err(errors.New("x")).Msg("m") }},
		{"WithLevel", func(l Logger) { l.WithLevel(WarnLevel).Msg("m") }},
		{"Ctx", func(l Logger) { l.Ctx(t.Context(), InfoLevel).Msg("m") }},
		{"Print", func(l Logger) { l.Print("m") }},
		{"child", func(l Logger) { l.With().Str("k", "v").Logger().Info().Msg("m") }},
		{"Event.Caller", func(l Logger) { l.Info().Caller().Msg("m") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var captured *EventData

			enc := NewTextEncoder(WithBeforeEncode(func(d *EventData) { captured = d }))

			var buf bytes.Buffer

			tc.emit(New(&buf, WithEncoder(enc), WithLevel(TraceLevel), WithCaller()))

			if captured == nil || captured.PC() == 0 {
				t.Fatal("no call site was captured")
			}

			wantFile, wantLine := referenceResolve(captured.PC())

			gotFile, gotLine, ok := resolvePC(captured.PC())
			if !ok {
				t.Fatal("resolvePC failed")
			}

			if gotFile != wantFile || gotLine != wantLine {
				t.Fatalf("resolvePC = %s:%d, runtime.CallersFrames = %s:%d",
					shortCallerPath(gotFile), gotLine, shortCallerPath(wantFile), wantLine)
			}
		})
	}
}

// referenceResolve is the runtime's own answer, used as the oracle.
func referenceResolve(pc uintptr) (string, int) {
	var arr [1]uintptr
	arr[0] = pc

	fr, _ := runtime.CallersFrames(arr[:]).Next()

	return fr.File, fr.Line
}

// assertCaller checks that the record carries exactly the expected position.
func assertCaller(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, CallerFieldName+"="+want) {
		t.Fatalf("call site mismatch\n  want: %s=%s\n   got: %s", CallerFieldName, want, strings.TrimSpace(got))
	}
}
