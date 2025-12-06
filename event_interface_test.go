package reggol

import "testing"

type demo struct{ A int }

func TestEventInterfaceAddsField(t *testing.T) {
	ed := newEventData(InfoLevel)
	ed.fields = make(Fields)
	// build via Event API to stay close to real usage
	w := LevelWriterAdapter{TransformWriterAdapter{Writer: discardWriter{}, Trans: NewTextTransformer("")}}
	e := newEvent(w, InfoLevel)
	e.data = ed

	e.Interface("obj", demo{A: 1})

	tr := NewTextTransformer("")
	tr.HideTimestamp()
	tr.HideLevel()

	out := tr.Transform(e.data)
	got := string(out)
	// Value is formatted with fmt: obj={1}
	if got != "obj={1}" {
		t.Fatalf("unexpected output: %q", got)
	}
}

// discardWriter is a no-op writer used only to satisfy interfaces in tests.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
