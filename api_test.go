package reggol

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// msgHello is reused across assertions.
const msgHello = "hello"

// TestEventSettersCoverEveryType walks every typed setter on Event and pins the
// bytes it produces. In v0 most of these had no test at all.
func TestEventSettersCoverEveryType(t *testing.T) {
	ts := time.Date(2026, 8, 12, 13, 45, 30, 0, time.UTC)

	for _, tc := range []struct {
		name string
		set  func(*Event) *Event
		want string
	}{
		{"Str", func(e *Event) *Event { return e.Str("k", "v") }, "k=v"},
		{"Int", func(e *Event) *Event { return e.Int("k", -7) }, "k=-7"},
		{"Int64", func(e *Event) *Event { return e.Int64("k", 1<<40) }, "k=1099511627776"},
		{"Uint64", func(e *Event) *Event { return e.Uint64("k", 1<<63) }, "k=9223372036854775808"},
		{"Float64", func(e *Event) *Event { return e.Float64("k", 2.5) }, "k=2.5"},
		{"Bool", func(e *Event) *Event { return e.Bool("k", true) }, "k=true"},
		{"Dur", func(e *Event) *Event { return e.Dur("k", 90*time.Second) }, "k=1m30s"},
		{"Time", func(e *Event) *Event { return e.Time("k", ts) }, "k=2026-08-12T13:45:30Z"},
		{"Bytes", func(e *Event) *Event { return e.Bytes("k", []byte("raw")) }, "k=raw"},
		{"Interface", func(e *Event) *Event { return e.Interface("k", 12) }, "k=12"},
		{"Any", func(e *Event) *Event { return e.Any("k", "s") }, "k=s"},
		{"IPAddr", func(e *Event) *Event { return e.IPAddr("k", net.IPv4(10, 0, 0, 1)) }, "k=10.0.0.1"},
		{"AnErr", func(e *Event) *Event { return e.AnErr("k", errors.New("x")) }, "k=x"},
		{"Field", func(e *Event) *Event { return e.Field(String("k", "f")) }, "k=f"},
		{"Fields", func(e *Event) *Event { return e.Fields(String("k", "a"), Int("n", 1)) }, "k=a, n=1"},
		{"Err", func(e *Event) *Event { return e.Err(errors.New("boom")) }, "error=boom"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			l := New(&buf,
				WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())),
				WithLevel(TraceLevel),
			)

			tc.set(l.Info()).Send()

			got := strings.TrimSuffix(buf.String(), "\n")
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestNilErrorSettersAreNoOps documents that a nil error adds nothing.
func TestNilErrorSettersAreNoOps(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())), WithLevel(TraceLevel))

	l.Info().Err(nil).AnErr("k", nil).Send()

	if got := strings.TrimSuffix(buf.String(), "\n"); got != "" {
		t.Fatalf("nil errors produced %q", got)
	}
}

// TestContextSettersCoverEveryType mirrors the Event table for bound fields.
func TestContextSettersCoverEveryType(t *testing.T) {
	ts := time.Date(2026, 8, 12, 13, 45, 30, 0, time.UTC)

	for _, tc := range []struct {
		name string
		bind func(Context) Context
		want string
	}{
		{"Str", func(c Context) Context { return c.Str("k", "v") }, "k=v"},
		{"Int", func(c Context) Context { return c.Int("k", 3) }, "k=3"},
		{"Int64", func(c Context) Context { return c.Int64("k", 4) }, "k=4"},
		{"Uint64", func(c Context) Context { return c.Uint64("k", 5) }, "k=5"},
		{"Float64", func(c Context) Context { return c.Float64("k", 1.25) }, "k=1.25"},
		{"Bool", func(c Context) Context { return c.Bool("k", false) }, "k=false"},
		{"Dur", func(c Context) Context { return c.Dur("k", time.Minute) }, "k=1m0s"},
		{"Time", func(c Context) Context { return c.Time("k", ts) }, "k=2026-08-12T13:45:30Z"},
		{"Bytes", func(c Context) Context { return c.Bytes("k", []byte("b")) }, "k=b"},
		{"Err", func(c Context) Context { return c.Err(errors.New("e")) }, "error=e"},
		{"AnErr", func(c Context) Context { return c.AnErr("k", errors.New("e")) }, "k=e"},
		{"Stringer", func(c Context) Context { return c.Stringer("k", net.IPv4(1, 2, 3, 4)) }, "k=1.2.3.4"},
		{"Any", func(c Context) Context { return c.Any("k", 9) }, "k=9"},
		{"Interface", func(c Context) Context { return c.Interface("k", "i") }, "k=i"},
		{"Field", func(c Context) Context { return c.Field(String("k", "f")) }, "k=f"},
		{"Fields", func(c Context) Context { return c.Fields(String("k", "a")) }, "k=a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			l := New(&buf,
				WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())),
				WithLevel(TraceLevel),
			)

			tc.bind(l.With()).Logger().Info().Send()

			got := strings.TrimSuffix(buf.String(), "\n")
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEventDataAccessors(t *testing.T) {
	ts := time.Date(2026, 8, 12, 13, 45, 30, 0, time.UTC)

	var captured *EventData

	enc := NewTextEncoder(WithBeforeEncode(func(d *EventData) { captured = d }))

	var buf bytes.Buffer

	l := New(&buf, WithEncoder(enc), WithLevel(TraceLevel))
	l.Warn().Timestamp(ts).Str("k", "v").BlockText("B").Msg(msgHello)

	if captured == nil {
		t.Fatal("BeforeEncode was not called")
	}

	if got := captured.Level(); got != WarnLevel {
		t.Errorf("Level = %v, want warn", got)
	}

	if got := captured.Time(); !got.Equal(ts) {
		t.Errorf("Time = %v, want %v", got, ts)
	}

	if got := captured.Message(); got != msgHello {
		t.Errorf("Message = %q, want hello", got)
	}

	if got := captured.Fields(); len(got) != 1 || got[0].Key != "k" {
		t.Errorf("Fields = %v", got)
	}

	if got := captured.Blocks(); len(got) != 1 || got[0].Text != "B" {
		t.Errorf("Blocks = %v", got)
	}

	if got := captured.Prefix(); got != nil {
		t.Errorf("Prefix on a root logger = %q, want nil", got)
	}
}

func TestAfterEncodeRuns(t *testing.T) {
	called := false

	enc := NewTextEncoder(WithAfterEncode(func(*EventData) { called = true }))

	var buf bytes.Buffer

	New(&buf, WithEncoder(enc), WithLevel(TraceLevel)).Info().Msg("m")

	if !called {
		t.Fatal("AfterEncode was not called")
	}
}

func TestBlocksHelpers(t *testing.T) {
	var bb Blocks

	bb.Add("one").AddBlock(NewBlock("two", strings.ToUpper))

	if len(bb) != 2 {
		t.Fatalf("len = %d, want 2", len(bb))
	}

	if got := bb[0].Value(); got != "one" {
		t.Errorf("plain block = %q", got)
	}

	if got := bb[1].Value(); got != "TWO" {
		t.Errorf("decorated block = %q", got)
	}
}

func TestJSONPrefixEncoding(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(NewJSONEncoder(WithoutTimestamp())), WithLevel(TraceLevel)).
		With().Str("service", "auth").Int("shard", 2).Logger()

	l.Info().Str("req", "x").Send()

	want := `{"level":"info","service":"auth","shard":2,"req":"x"}` + "\n"
	if got := buf.String(); got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestConsolePrefixEncoding(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf,
		WithEncoder(NewConsoleEncoder(
			WithColorMode(ColorNever, nil),
			WithConsoleOptions(WithoutTimestamp()),
		)),
		WithLevel(TraceLevel),
	).With().Str("service", "auth").Logger()

	l.Info().Str("req", "x").Msg("m")

	want := "INF m service=auth req=x\n"
	if got := buf.String(); got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestSetLevelColor(t *testing.T) {
	enc := NewConsoleEncoder(WithColorMode(ColorAlways, nil))
	enc.SetLevelColor(InfoLevel, ColorFgMagenta)

	got := encodeEvent(enc, func(d *EventData) { d.message = []byte("m") })
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("no escape sequence: %q", got)
	}

	// A colorless encoder must ignore the override rather than start emitting.
	plain := NewConsoleEncoder(WithColorMode(ColorNever, nil))
	plain.SetLevelColor(InfoLevel, ColorFgMagenta)

	if out := encodeEvent(plain, func(d *EventData) { d.message = []byte("m") }); strings.Contains(out, "\x1b[") {
		t.Fatalf("colorless encoder emitted color: %q", out)
	}
}

func TestErrorHandlerReceivesWriteFailures(t *testing.T) {
	prev := errorHandler()
	t.Cleanup(func() { SetErrorHandler(prev) })

	var got error

	SetErrorHandler(func(err error) { got = err })

	want := errors.New("disk on fire")
	l := New(failingWriter{err: want}, WithEncoder(NewTextEncoder()), WithLevel(TraceLevel))
	l.Info().Msg("m")

	if !errors.Is(got, want) {
		t.Fatalf("handler got %v, want %v", got, want)
	}

	SetErrorHandler(nil)

	if errorHandler() != nil {
		t.Fatal("SetErrorHandler(nil) should clear the handler")
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestExitCodeAccessors(t *testing.T) {
	prev := ExitCode()
	t.Cleanup(func() { SetExitCode(prev) })

	SetExitCode(7)

	if got := ExitCode(); got != 7 {
		t.Fatalf("ExitCode = %d, want 7", got)
	}
}

func TestMultiWriterFansOut(t *testing.T) {
	var a, b bytes.Buffer

	l := New(nil, WithEncoder(NewTextEncoder(WithoutTimestamp())), WithLevel(TraceLevel))
	l = l.Output(&teeWriter{w: MultiWriter(&a, &b)})

	l.Info().Msg("both")

	if !strings.Contains(a.String(), "both") || !strings.Contains(b.String(), "both") {
		t.Fatalf("fan-out failed: a=%q b=%q", a.String(), b.String())
	}
}

// teeWriter adapts a Writer back to io.Writer so it can be passed to Output.
type teeWriter struct{ w Writer }

func (t *teeWriter) Write(p []byte) (int, error) { return t.w.WriteLevel(NoLevel, p) }

func TestNewWithNilWriterDiscards(t *testing.T) {
	l := New(nil, WithEncoder(NewTextEncoder()), WithLevel(TraceLevel))
	l.Info().Msg("goes nowhere")
}

func TestEncoderAccessor(t *testing.T) {
	enc := NewTextEncoder()

	if got := New(nil, WithEncoder(enc)).Encoder(); got != enc {
		t.Fatal("Encoder did not return the configured encoder")
	}

	// A nil encoder must be ignored rather than installed.
	if got := New(nil, WithEncoder(nil)).Encoder(); got == nil {
		t.Fatal("nil encoder should have been ignored")
	}
}

func TestSyncWriterPassesThroughClose(t *testing.T) {
	c := &closableBuffer{}

	sw, ok := SyncWriter(c).(interface{ Close() error })
	if !ok {
		t.Fatal("SyncWriter should expose Close")
	}

	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if !c.closed {
		t.Fatal("Close did not reach the underlying writer")
	}

	if SyncWriter(nil) == nil {
		t.Fatal("SyncWriter(nil) must return a usable writer")
	}
}

type closableBuffer struct {
	bytes.Buffer

	closed bool
}

func (c *closableBuffer) Close() error {
	c.closed = true

	return nil
}
