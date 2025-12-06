package reggol

import (
	"strings"
	"testing"
)

func TestTextTransformer_SortFieldsEnabled(t *testing.T) {
	ed := newEventData(InfoLevel)
	ed.fields = make(Fields)
	ed.fields.Add("b", 2)
	ed.fields.Add("a", 1)

	tr := NewTextTransformer("")
	tr.HideTimestamp()
	tr.HideLevel()
	tr.SetSortFields(true)

	out := string(tr.Transform(ed))
	if !strings.HasPrefix(out, "a=1") {
		t.Fatalf("expected sorted fields with 'a' first, got %q", out)
	}
}

func TestConsoleTransformer_SortFieldsEnabled(t *testing.T) {
	ed := newEventData(InfoLevel)
	ed.fields = make(Fields)
	ed.fields.Add("b", 2)
	ed.fields.Add("a", 1)

	tr := NewConsoleTransformer(true, "")
	tr.HideTimestamp()
	tr.HideLevel()
	tr.SetSortFields(true)

	out := string(tr.Transform(ed))
	// when only fields exist, they should be sorted as 'a' then 'b'
	if !strings.HasPrefix(out, "a=1 ") && out != "a=1" {
		t.Fatalf("expected sorted fields starting with 'a', got %q", out)
	}
}
