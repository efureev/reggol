package reggol

import (
	"testing"
)

func TestConsoleTransformerBlocks_Empty(t *testing.T) {
	ct := NewConsoleTransformer(true, ``)
	ct.HideTimestamp()
	ct.HideLevel()

	ed := newEventData(InfoLevel)
	// no blocks, no msg, no err, no fields

	out := ct.Transform(ed)

	if got := string(out); got != "" {
		t.Fatalf("expected empty output with no blocks, got %q", got)
	}
}

func TestConsoleTransformerBlocks_NonEmpty(t *testing.T) {
	ct := NewConsoleTransformer(true, ``)
	ct.HideTimestamp()
	ct.HideLevel()

	ed := newEventData(InfoLevel)
	ed.blocks.Add("first").Add("second")

	out := ct.Transform(ed)

	if got, want := string(out), "first second"; got != want {
		t.Fatalf("blocks output mismatch: got %q, want %q", got, want)
	}
}
