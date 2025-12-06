package reggol

import (
	"bytes"
	"errors"
	"testing"
)

func TestEvent_Discard(t *testing.T) {
	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)

	e := logger.Info().Discard()
	if e != nil {
		t.Fatalf("Discard should return nil to stop chaining")
	}
	if buf.Len() != 0 {
		t.Fatalf("discarded event must not write anything, got %q", buf.String())
	}
}

func TestEvent_Push_EmptyMessage_WritesOnlyNewline(t *testing.T) {
	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf) // no ts/level

	logger.Info().Push()

	if got := buf.String(); got != "\n" {
		t.Fatalf("expected single newline for empty push, got %q", got)
	}
}

func TestEvent_AnErr_Branches(t *testing.T) {
	prev := ErrorMarshalFunc
	t.Cleanup(func() { ErrorMarshalFunc = prev })

	var buf bytes.Buffer
	logger := newDeterministicTextLogger(&buf)

	// nil error -> no output until message
	buf.Reset()
	logger.Error().AnErr("error", nil).Msg("x")
	if got := buf.String(); got != "message=x\n" {
		t.Fatalf("nil err should not set error: %q", got)
	}

	// error -> stored in e.data.err -> console/text behavior handled by transformer
	buf.Reset()
	ErrorMarshalFunc = func(err error) interface{} { return err }
	logger.Error().AnErr("error", errors.New("boom")).Msg("")
	// With text transformer and no ts/level, error appears as error=<text>
	if got := buf.String(); got != "error=boom\n" {
		t.Fatalf("error branch expected 'error=boom', got %q", got)
	}

	// string via ErrorMarshalFunc
	buf.Reset()
	ErrorMarshalFunc = func(err error) interface{} { return "STR" }
	logger.Error().AnErr("error", errors.New("irrelevant")).Msg("")
	if got := buf.String(); got != "error=STR\n" {
		t.Fatalf("string branch expected 'error=STR', got %q", got)
	}

	// default branch: arbitrary object -> goes through Interface(key, m)
	buf.Reset()
	type custom struct{ A int }
	ErrorMarshalFunc = func(err error) interface{} { return custom{A: 7} }
	logger.Error().AnErr("errKey", errors.New("x")).Msg("")
	// formatted as obj via fmt: {7}
	if got := buf.String(); got != "errKey={7}\n" {
		t.Fatalf("default branch expected 'errKey={7}', got %q", got)
	}
}
