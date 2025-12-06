package reggol

import (
	"bytes"
	"strings"
	"testing"
)

// helper to build a deterministic text-logger (no ts/level).
func newDeterministicTextLogger(buf *bytes.Buffer) Logger {
	tr := NewTextTransformer("")
	tr.HideTimestamp()
	tr.HideLevel()
	twa := TransformWriterAdapter{Writer: buf, Trans: tr}

	return New(twa).Level(DebugLevel)
}

func TestLogger_Write_TrimsNewline(t *testing.T) {
	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)

	// Logger implements io.Writer; should trim trailing newline and log message once
	_, _ = logger.Write([]byte("hello world\n"))

	got := buf.String()
	want := "message=hello world\n"
	if got != want {
		t.Fatalf("unexpected output: got %q want %q", got, want)
	}
}

func TestLogger_WithLevel_Routing(t *testing.T) {
	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)

	logger.WithLevel(NoLevel).Msg("nolvl")
	if !strings.Contains(buf.String(), "message=nolvl\n") {
		t.Fatalf("expected message for NoLevel, got %q", buf.String())
	}

	buf.Reset()
	ev := logger.WithLevel(Disabled)
	if ev != nil {
		t.Fatalf("expected nil event for Disabled level")
	}

	if buf.Len() != 0 {
		t.Fatalf("expected no output for Disabled, got %q", buf.String())
	}
}

func TestLogger_Nop_Disabled(t *testing.T) {
	var buf bytes.Buffer
	// Route output through deterministic text transformer
	tr := NewTextTransformer("")
	tr.HideTimestamp()
	tr.HideLevel()
	logger := Nop()
	// Replace writer to capture output, keep Disabled level
	logger.w = LevelWriterAdapter{TransformWriterAdapter{Writer: &buf, Trans: tr}}

	logger.Info().Msg("hidden")
	if buf.Len() != 0 {
		t.Fatalf("expected no output for Nop logger, got %q", buf.String())
	}
}

func TestLogger_Panic_WritesThenPanics(t *testing.T) {
	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}

		//nolint:forcetypeassert
		if r.(string) != "boom" { // panic(msg)
			t.Fatalf("unexpected panic value: %v", r)
		}

		if got := buf.String(); got != "message=boom\n" {
			t.Fatalf("unexpected output: %q", got)
		}
	}()

	logger.Panic().Msg("boom")
}

func TestGlobalVsLocalLevelPrecedence(t *testing.T) {
	// save and restore global level
	prev := GlobalLevel()
	t.Cleanup(func() { SetGlobalLevel(prev) })

	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf).Level(DebugLevel)

	SetGlobalLevel(WarnLevel)
	logger.Info().Msg("hidden by global")
	if buf.Len() != 0 {
		t.Fatalf("info should be filtered by global warn, got %q", buf.String())
	}

	logger.Warn().Msg("visible")
	if got := buf.String(); got != "message=visible\n" {
		t.Fatalf("warn should pass, got %q", got)
	}
}

func TestRaisingGlobalLevelAffectsExistingLogger(t *testing.T) {
	prev := GlobalLevel()
	t.Cleanup(func() { SetGlobalLevel(prev) })

	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)

	SetGlobalLevel(InfoLevel)
	logger.Info().Msg("seen")
	if got := buf.String(); got != "message=seen\n" {
		t.Fatalf("expected first info visible, got %q", got)
	}

	SetGlobalLevel(ErrorLevel) // raise
	logger.Info().Msg("hidden")
	if got := buf.String(); got != "message=seen\n" {
		t.Fatalf("info after raising global should be hidden, got %q", got)
	}
}

func TestGlobalDisabledSilencesAll(t *testing.T) {
	prev := GlobalLevel()
	t.Cleanup(func() { SetGlobalLevel(prev) })
	SetGlobalLevel(Disabled)

	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)
	logger.Error().Msg("hidden")
	if buf.Len() != 0 {
		t.Fatalf("expected silence when global Disabled, got %q", buf.String())
	}
}

func TestLogger_ErrNil_RoutesToInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)

	logger.Err(nil).Msg("ok")
	if got := buf.String(); got != "message=ok\n" {
		t.Fatalf("Err(nil) should route to info message, got %q", got)
	}
}
