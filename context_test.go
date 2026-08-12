package reggol

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestWithBindsFields(t *testing.T) {
	var buf bytes.Buffer

	child := newTestLogger(&buf).With().
		Str("service", "auth").
		Int("shard", 7).
		Logger()

	child.Info().Str("req", "x1").Msg("handled")

	got := buf.String()
	for _, want := range []string{"service=auth", "shard=7", "req=x1", "handled"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

// TestBoundFieldsPrecedeEventFields pins the documented ordering: bound fields
// come first, sorting applies within each group rather than across them.
func TestBoundFieldsPrecedeEventFields(t *testing.T) {
	var buf bytes.Buffer

	child := newTestLogger(&buf).With().Str("zzz", "bound").Logger()
	child.Info().Str("aaa", "event").Send()

	got := buf.String()

	boundAt := strings.Index(got, "zzz=bound")
	eventAt := strings.Index(got, "aaa=event")

	if boundAt < 0 || eventAt < 0 {
		t.Fatalf("both fields expected in %q", got)
	}

	if boundAt > eventAt {
		t.Fatalf("bound field should precede the event field: %q", got)
	}
}

func TestWithIsChainable(t *testing.T) {
	var buf bytes.Buffer

	root := newTestLogger(&buf)
	a := root.With().Str("a", "1").Logger()
	b := a.With().Str("b", "2").Logger()

	b.Info().Send()

	got := buf.String()
	if !strings.Contains(got, "a=1") || !strings.Contains(got, "b=2") {
		t.Fatalf("chained fields lost: %q", got)
	}

	buf.Reset()
	a.Info().Send()

	if strings.Contains(buf.String(), "b=2") {
		t.Fatalf("child leaked into parent: %q", buf.String())
	}
}

func TestWithNoFieldsReturnsSameLogger(t *testing.T) {
	var buf bytes.Buffer

	root := newTestLogger(&buf)
	same := root.With().Logger()

	same.Info().Msg("m")

	if !strings.Contains(buf.String(), "m") {
		t.Fatalf("logger broken by an empty With: %q", buf.String())
	}
}

// TestWithFallsBackWithoutPrefixEncoder covers an encoder that does not
// implement PrefixEncoder: bound fields must still appear, just re-encoded.
func TestWithFallsBackWithoutPrefixEncoder(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(plainEncoder{}), WithLevel(TraceLevel)).
		With().Str("bound", "yes").Logger()

	l.Info().Str("event", "also").Send()

	got := buf.String()
	if !strings.Contains(got, "bound=yes") || !strings.Contains(got, "event=also") {
		t.Fatalf("fallback path lost fields: %q", got)
	}
}

// plainEncoder implements Encoder but not PrefixEncoder.
type plainEncoder struct{}

func (plainEncoder) AppendEvent(dst []byte, d *EventData) []byte {
	for i, f := range d.Fields() {
		if i > 0 {
			dst = append(dst, ' ')
		}

		dst = append(dst, f.Key...)
		dst = append(dst, '=')
		dst = f.Val.AppendTo(dst)
	}

	return append(dst, '\n')
}

func TestContextCarriesLogger(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf).With().Str("bound", "1").Logger()

	ctx := l.WithContext(context.Background())

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("logger not found in context")
	}

	got.Info().Msg("from ctx")

	if !strings.Contains(buf.String(), "bound=1") {
		t.Fatalf("context logger lost its fields: %q", buf.String())
	}
}

func TestFromContextWithoutLogger(t *testing.T) {
	l, ok := FromContext(context.Background())
	if ok {
		t.Fatal("did not expect a logger")
	}

	// The fallback must be safe to use.
	l.Info().Msg("ignored")

	if _, ok := FromContext(nil); ok { //nolint:staticcheck // nil context is the case under test
		t.Fatal("nil context must not yield a logger")
	}
}

func TestContextExtractorsRun(t *testing.T) {
	var buf bytes.Buffer

	type traceKey struct{}

	l := New(&buf,
		WithEncoder(NewTextEncoder(WithoutTimestamp())),
		WithLevel(TraceLevel),
		WithContextExtractor(func(ctx context.Context, e *Event) {
			if v, ok := ctx.Value(traceKey{}).(string); ok {
				e.Str("trace_id", v)
			}
		}),
	)

	ctx := context.WithValue(context.Background(), traceKey{}, "abc123")
	l.Ctx(ctx, InfoLevel).Msg("handled")

	if !strings.Contains(buf.String(), "trace_id=abc123") {
		t.Fatalf("extractor did not run: %q", buf.String())
	}
}

func TestEventCtxRoundTrip(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLogger(&buf)

	type key struct{}

	ctx := context.WithValue(context.Background(), key{}, "v")

	e := l.Info().Ctx(ctx)
	if e.GetCtx().Value(key{}) != "v" {
		t.Fatal("context not carried on the event")
	}

	e.Msg("m")

	var nilEvent *Event
	if nilEvent.GetCtx() == nil {
		t.Fatal("GetCtx on a nil event must return a usable context")
	}
}
