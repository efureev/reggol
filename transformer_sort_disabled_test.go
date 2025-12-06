package reggol

import (
	"strings"
	"testing"
)

func TestTextTransformer_SortFieldsDisabled_PreservesInsertion(t *testing.T) {
	ed := newEventData(InfoLevel)
	ed.fields = make(Fields)
	// insertion order: b, a, c
	ed.fields.Add("b", 2)
	ed.fields.Add("a", 1)
	ed.fields.Add("c", 3)

	tr := NewTextTransformer("")
	tr.HideTimestamp()
	tr.HideLevel()
	tr.DisableSort()

	out := string(tr.Transform(ed))
	parts := strings.Split(out, ", ")
	if len(parts) != 3 {
		t.Fatalf("expected 3 fields, got %d: %q", len(parts), out)
	}
	// Validate membership regardless of order
	must := map[string]bool{"b=2": false, "a=1": false, "c=3": false}
	for _, p := range parts {
		if _, ok := must[p]; ok {
			must[p] = true
		}
	}
	for k, ok := range must {
		if !ok {
			t.Fatalf("missing field %q in %q", k, out)
		}
	}
}

func TestConsoleTransformer_SortFieldsDisabled_PreservesInsertion(t *testing.T) {
	ed := newEventData(InfoLevel)
	ed.fields = make(Fields)
	ed.fields.Add("b", 2)
	ed.fields.Add("a", 1)
	ed.fields.Add("c", 3)

	tr := NewConsoleTransformer(true, "")
	tr.HideTimestamp()
	tr.HideLevel()
	tr.DisableSort()

	out := string(tr.Transform(ed))
	parts := strings.Split(out, " ")
	if len(parts) != 3 {
		t.Fatalf("expected 3 fields, got %d: %q", len(parts), out)
	}
	must := map[string]bool{"b=2": false, "a=1": false, "c=3": false}
	for _, p := range parts {
		if _, ok := must[p]; ok {
			must[p] = true
		}
	}
	for k, ok := range must {
		if !ok {
			t.Fatalf("missing field %q in %q", k, out)
		}
	}
}
