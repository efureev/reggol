package slogr_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/efureev/reggol"
	"github.com/efureev/reggol/slogr"
)

// decode reads the single JSON record a handler produced.
func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("no record was emitted")
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		t.Fatalf("invalid JSON %q: %v", line, err)
	}

	return out
}

func TestFromHandlerCarriesEverything(t *testing.T) {
	var buf bytes.Buffer

	logger := slogr.FromHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger.Warn().
		Str("service", "auth").
		Int("shard", 7).
		Bool("cached", true).
		Float64("ratio", 1.5).
		Dur("took", 90*time.Second).
		Msg("handled")

	got := decode(t, &buf)

	for k, want := range map[string]any{
		"msg":     "handled",
		"level":   "WARN",
		"service": "auth",
		"shard":   float64(7),
		"cached":  true,
		"ratio":   1.5,
	} {
		if got[k] != want {
			t.Errorf("%s = %v (%T), want %v", k, got[k], got[k], want)
		}
	}

	if got["took"] == nil {
		t.Error("duration field missing")
	}
}

func TestFromHandlerMapsLevels(t *testing.T) {
	// reggol's process-wide threshold applies before the handler's, and it
	// defaults to InfoLevel — debug records would never reach the bridge.
	prev := reggol.GlobalLevel()
	t.Cleanup(func() { reggol.SetGlobalLevel(prev) })
	reggol.SetGlobalLevel(reggol.TraceLevel)

	for _, tc := range []struct {
		name string
		emit func(reggol.Logger)
		want string
	}{
		{"debug", func(l reggol.Logger) { l.Debug().Msg("m") }, "DEBUG"},
		{"info", func(l reggol.Logger) { l.Info().Msg("m") }, "INFO"},
		{"warn", func(l reggol.Logger) { l.Warn().Msg("m") }, "WARN"},
		{"error", func(l reggol.Logger) { l.Error().Msg("m") }, "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			logger := slogr.FromHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slogr.LevelTrace,
			}))

			tc.emit(logger)

			if got := decode(t, &buf)["level"]; got != tc.want {
				t.Fatalf("level = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestFromHandlerPassesChildFields covers the reason the forwarder must not
// implement PrefixEncoder: without a pre-encoded prefix, reggol keeps a child
// logger's bound fields structural, which is what a handler needs.
func TestFromHandlerPassesChildFields(t *testing.T) {
	var buf bytes.Buffer

	child := slogr.FromHandler(slog.NewJSONHandler(&buf, nil)).
		With().
		Str("service", "auth").
		Int("shard", 7).
		Logger()

	child.Info().Str("req", "x1").Msg("handled")

	got := decode(t, &buf)

	for k, want := range map[string]any{
		"service": "auth",
		"shard":   float64(7),
		"req":     "x1",
	} {
		if got[k] != want {
			t.Errorf("%s = %v, want %v", k, got[k], want)
		}
	}
}

func TestFromHandlerPassesContext(t *testing.T) {
	type key struct{}

	var seen string

	h := &contextProbe{onHandle: func(ctx context.Context) {
		if v, ok := ctx.Value(key{}).(string); ok {
			seen = v
		}
	}}

	ctx := context.WithValue(context.Background(), key{}, "t-1")
	slogr.FromHandler(h).Ctx(ctx, reggol.InfoLevel).Msg("m")

	if seen != "t-1" {
		t.Fatalf("handler saw %q, want the caller's context value", seen)
	}
}

func TestFromHandlerHonoursHandlerLevel(t *testing.T) {
	var buf bytes.Buffer

	logger := slogr.FromHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	logger.Info().Msg("dropped by the handler")

	if buf.Len() != 0 {
		t.Fatalf("handler level ignored: %q", buf.String())
	}

	logger.Error().Msg("kept")

	if !strings.Contains(buf.String(), "kept") {
		t.Fatalf("error should pass: %q", buf.String())
	}
}

func TestFromHandlerRendersBlocksAndErrors(t *testing.T) {
	var buf bytes.Buffer

	logger := slogr.FromHandler(slog.NewJSONHandler(&buf, nil))

	logger.Error().
		Blocks("API", "GET /users").
		Err(errors.New("boom")).
		Msg("failed")

	got := decode(t, &buf)

	blocks, ok := got[reggol.BlocksFieldName].([]any)
	if !ok || len(blocks) != 2 || blocks[0] != "API" {
		t.Fatalf("blocks = %v", got[reggol.BlocksFieldName])
	}

	if got["error"] != "boom" {
		t.Fatalf("error = %v, want boom", got["error"])
	}

	// The message must survive alongside the error, as everywhere else.
	if got["msg"] != "failed" {
		t.Fatalf("msg = %v, want failed", got["msg"])
	}
}

func TestFromHandlerGroups(t *testing.T) {
	var buf bytes.Buffer

	slogr.FromHandler(slog.NewJSONHandler(&buf, nil)).
		Info().
		Field(reggol.Group("g", reggol.String("a", "1"), reggol.Int("b", 2))).
		Msg("m")

	got := decode(t, &buf)

	group, ok := got["g"].(map[string]any)
	if !ok {
		t.Fatalf("g = %v, want a nested object", got["g"])
	}

	if group["a"] != "1" || group["b"] != float64(2) {
		t.Fatalf("group contents = %v", group)
	}
}

// TestRoundTrip sends records through both bridges: reggol → slog.Handler
// (FromHandler) where the handler is itself reggol-backed (NewHandler).
func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer

	sink := reggol.New(&buf,
		reggol.WithEncoder(reggol.NewTextEncoder(reggol.WithoutTimestamp())),
		reggol.WithLevel(reggol.TraceLevel),
	)

	logger := slogr.FromHandler(slogr.NewHandler(sink))
	logger.Info().Str("k", "v").Int("n", 1).Msg("round trip")

	got := buf.String()
	for _, want := range []string{"level=info", "message=round trip", "k=v", "n=1"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestFromHandlerNilIsNop(t *testing.T) {
	logger := slogr.FromHandler(nil)

	// Must not panic and must emit nothing.
	logger.Error().Str("k", "v").Msg("ignored")

	if logger.GetLevel() != reggol.Disabled {
		t.Fatalf("level = %v, want Disabled", logger.GetLevel())
	}
}

// contextProbe records the context each Handle call receives.
type contextProbe struct {
	onHandle func(context.Context)
}

func (p *contextProbe) Enabled(context.Context, slog.Level) bool { return true }

func (p *contextProbe) Handle(ctx context.Context, _ slog.Record) error {
	p.onHandle(ctx)

	return nil
}

func (p *contextProbe) WithAttrs([]slog.Attr) slog.Handler { return p }

func (p *contextProbe) WithGroup(string) slog.Handler { return p }

// TestFromHandlerCarriesCallSite closes the gap that motivated the whole
// feature: before this, AddSource produced nothing because the bridge passed a
// zero program counter.
func TestFromHandlerCarriesCallSite(t *testing.T) {
	var buf bytes.Buffer

	logger := slogr.FromHandler(
		slog.NewJSONHandler(&buf, &slog.HandlerOptions{AddSource: true}),
		reggol.WithCaller(),
	)

	logger.Info().Msg("where am I")

	got := decode(t, &buf)

	src, ok := got["source"].(map[string]any)
	if !ok {
		t.Fatalf("no source in %s", buf.String())
	}

	file, _ := src["file"].(string)
	if !strings.HasSuffix(file, "forward_test.go") {
		t.Fatalf("source.file = %v, want this test file", src["file"])
	}

	if line, _ := src["line"].(float64); line == 0 {
		t.Fatalf("source.line = %v", src["line"])
	}
}

// TestFromHandlerWithoutCallerHasNoSource pins the opt-in: capturing costs time,
// so it happens only when asked for.
func TestFromHandlerWithoutCallerHasNoSource(t *testing.T) {
	var buf bytes.Buffer

	slogr.FromHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{AddSource: true})).
		Info().Msg("m")

	if _, present := decode(t, &buf)["source"]; present {
		t.Fatalf("source reported without WithCaller: %s", buf.String())
	}
}

// TestHandlerUsesSlogCallSite covers the other direction: slog captured the
// position already, and the bridge must use it rather than a line inside slogr.
func TestHandlerUsesSlogCallSite(t *testing.T) {
	var buf bytes.Buffer

	sink := reggol.New(&buf,
		reggol.WithEncoder(reggol.NewTextEncoder(reggol.WithoutTimestamp(), reggol.WithoutLevel())),
		reggol.WithLevel(reggol.TraceLevel),
		reggol.WithCaller(),
	)

	slogr.New(sink).Info("m")

	got := buf.String()
	if !strings.Contains(got, "slogr_test") && !strings.Contains(got, "forward_test.go") {
		t.Fatalf("call site should point at this test, got %q", got)
	}

	if strings.Contains(got, "slogr/slogr.go") || strings.Contains(got, "slogr/forward.go") {
		t.Fatalf("call site points inside the bridge: %q", got)
	}
}
