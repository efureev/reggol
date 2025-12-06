package reggol

import (
	"bytes"
	"strings"
	"testing"
)

func countNewlines(s string) int {
	return strings.Count(s, "\n")
}

func TestConsoleWriterAddsSingleNewline(t *testing.T) {
	var buf bytes.Buffer
	cw := NewConsoleWriter(func(w *ConsoleWriter) { w.Out = &buf })
	lw := LevelWriterAdapter{cw}

	e := newEvent(lw, InfoLevel)
	e.Msg("hello")

	got := buf.String()

	if n := countNewlines(got); n != 1 {
		t.Fatalf("expected exactly 1 newline, got %d; output=%q", n, got)
	}
}

func TestTransformWriterAdapterAddsSingleNewline(t *testing.T) {
	var buf bytes.Buffer
	tr := NewTextTransformer("")
	twa := TransformWriterAdapter{Writer: &buf, Trans: tr}
	lw := LevelWriterAdapter{twa}

	e := newEvent(lw, InfoLevel)
	e.Msg("hello")

	got := buf.String()

	if n := countNewlines(got); n != 1 {
		t.Fatalf("expected exactly 1 newline, got %d; output=%q", n, got)
	}
}
