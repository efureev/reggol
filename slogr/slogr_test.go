package slogr_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"testing/slogtest"

	"github.com/efureev/reggol"
	"github.com/efureev/reggol/slogr"
)

// newJSONLogger builds a reggol logger emitting slog-compatible JSON.
func newJSONLogger(buf *bytes.Buffer) reggol.Logger {
	enc := reggol.NewJSONEncoder(
		reggol.WithKeyNames(slog.TimeKey, slog.LevelKey, slog.MessageKey),
	)

	return reggol.New(buf, reggol.WithEncoder(enc), reggol.WithLevel(reggol.TraceLevel))
}

// TestSlogConformance runs the standard library's own handler test suite.
//
// It is the only check that covers the corners — empty groups, inlined groups,
// LogValuer resolution — thoroughly enough to trust the bridge.
func TestSlogConformance(t *testing.T) {
	var buf bytes.Buffer

	handler := slogr.NewHandler(newJSONLogger(&buf))

	results := func() []map[string]any {
		var out []map[string]any

		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}

			var flat map[string]any
			if err := json.Unmarshal([]byte(line), &flat); err != nil {
				t.Fatalf("invalid JSON %q: %v", line, err)
			}

			out = append(out, unflatten(flat))
		}

		return out
	}

	slogtest.Run(t, func(*testing.T) slog.Handler {
		buf.Reset()

		return handler
	}, func(*testing.T) map[string]any {
		got := results()
		if len(got) == 0 {
			return map[string]any{}
		}

		return got[len(got)-1]
	})
}

// unflatten turns dotted keys back into nested maps, which is how slogtest
// expects to inspect groups.
func unflatten(flat map[string]any) map[string]any {
	out := map[string]any{}

	for k, v := range flat {
		parts := strings.Split(k, ".")
		m := out

		for _, p := range parts[:len(parts)-1] {
			next, ok := m[p].(map[string]any)
			if !ok {
				next = map[string]any{}
				m[p] = next
			}

			m = next
		}

		m[parts[len(parts)-1]] = v
	}

	return out
}

func TestLevelMapping(t *testing.T) {
	for _, tc := range []struct {
		slogLevel slog.Level
		want      reggol.Level
	}{
		{slogr.LevelTrace, reggol.TraceLevel},
		{slog.LevelDebug, reggol.DebugLevel},
		{slog.LevelInfo, reggol.InfoLevel},
		{slog.LevelWarn, reggol.WarnLevel},
		{slog.LevelError, reggol.ErrorLevel},
		{slogr.LevelFatal, reggol.FatalLevel},
		{slogr.LevelPanic, reggol.PanicLevel},
	} {
		if got := slogr.FromSlogLevel(tc.slogLevel); got != tc.want {
			t.Errorf("FromSlogLevel(%v) = %v, want %v", tc.slogLevel, got, tc.want)
		}
	}

	for _, lvl := range []reggol.Level{
		reggol.TraceLevel, reggol.DebugLevel, reggol.InfoLevel,
		reggol.WarnLevel, reggol.ErrorLevel,
	} {
		if got := slogr.FromSlogLevel(slogr.ToSlogLevel(lvl)); got != lvl {
			t.Errorf("round trip of %v gave %v", lvl, got)
		}
	}
}

func TestHandlerRespectsLevel(t *testing.T) {
	var buf bytes.Buffer

	l := newJSONLogger(&buf).Level(reggol.WarnLevel)
	logger := slogr.New(l)

	logger.Info("dropped")

	if buf.Len() != 0 {
		t.Fatalf("info should be filtered: %q", buf.String())
	}

	logger.Warn("kept")

	if !strings.Contains(buf.String(), "kept") {
		t.Fatalf("warn should pass: %q", buf.String())
	}
}

func TestHandlerCarriesAttrsAndGroups(t *testing.T) {
	var buf bytes.Buffer

	logger := slogr.New(newJSONLogger(&buf)).
		With("service", "auth").
		WithGroup("req").
		With("id", 7)

	logger.Info("handled", slog.String("outcome", "ok"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}

	for k, want := range map[string]any{
		"service":     "auth",
		"req.id":      float64(7),
		"req.outcome": "ok",
		"msg":         "handled",
	} {
		if got[k] != want {
			t.Errorf("%s = %v (%T), want %v", k, got[k], got[k], want)
		}
	}
}

func TestHandlerPassesContext(t *testing.T) {
	type key struct{}

	var buf bytes.Buffer

	l := reggol.New(&buf,
		reggol.WithEncoder(reggol.NewTextEncoder(reggol.WithoutTimestamp())),
		reggol.WithLevel(reggol.TraceLevel),
		reggol.WithContextExtractor(func(ctx context.Context, e *reggol.Event) {
			if v, ok := ctx.Value(key{}).(string); ok {
				e.Str("trace_id", v)
			}
		}),
	)

	ctx := context.WithValue(context.Background(), key{}, "t-1")
	slogr.New(l).InfoContext(ctx, "hello")

	if !strings.Contains(buf.String(), "trace_id=t-1") {
		t.Fatalf("context did not reach the extractor: %q", buf.String())
	}
}
