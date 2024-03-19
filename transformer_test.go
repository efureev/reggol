package reggol

import (
	"errors"
	"testing"
)

func TestTextTransformer(t *testing.T) {
	tests := []struct {
		name   string
		fields Fields
		want   string
	}{
		{"one str field", Fields{`Key1`: `Value1`}, `Key1=Value1`},
		{"several str fields1", Fields{`Key1`: `Value1`, `Key2`: `Value2`}, `Key1=Value1, Key2=Value2`},
		{"several str fields2", Fields{`Key1`: `Value1`, `Key3`: `3221`}, `Key1=Value1, Key3=3221`},
	}
	eventData := newEventData(TraceLevel)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventData.fields = tt.fields

			trans := NewTextTransformer(``)
			trans.HideLevel()
			trans.HideTimestamp()

			res := trans.Transform(eventData)

			if got, want := string(res), tt.want; got != want {
				t.Errorf("Fields = `%v`, want `%v`", got, want)
			}
		})
	}
}

func TestTextTransformerCreate(t *testing.T) {
	t.Run(`create Transformer`, func(t *testing.T) {
		tr := NewTextTransformer(``)

		if !tr.displayLevel {
			t.Errorf("`displayLevel` must be TRUE, given %s", `FALSE`)
		}

		if !tr.displayTimestamp {
			t.Errorf("`displayTimestamp` must be TRUE, given %s", `FALSE`)
		}

		tr.HideTimestamp()

		if !tr.displayLevel {
			t.Errorf("`displayLevel` must be TRUE, given %s", `FALSE`)
		}

		if tr.displayTimestamp {
			t.Errorf("`displayTimestamp` must be FALSE, given %s", `TRUE`)
		}

		tr.HideLevel()

		if tr.displayLevel {
			t.Errorf("`displayLevel` must be FALSE, given %s", `TRUE`)
		}

		if tr.displayTimestamp {
			t.Errorf("`displayTimestamp` must be FALSE, given %s", `TRUE`)
		}
	})
}

func TestTextTransformerMessage(t *testing.T) {
	t.Run(`Transform message`, func(t *testing.T) {
		eventData := newEventData(WarnLevel)
		eventData.message = `test message`

		trans := NewTextTransformer(``)
		trans.HideTimestamp()
		trans.HideLevel()

		res := trans.Transform(eventData)

		if got, want := string(res), MessageFieldName+`=`+eventData.message; got != want {
			t.Errorf("output data = `%v`, want `%v`", got, want)
		}
	})
}

func TestTextTransformerError(t *testing.T) {
	t.Run(`Transform err`, func(t *testing.T) {
		eventData := newEventData(WarnLevel)
		eventData.err = errors.New(`test error`)

		trans := NewTextTransformer(``)
		trans.HideTimestamp()
		trans.HideLevel()

		res := trans.Transform(eventData)

		if got, want := string(res), ErrorFieldName+`=`+eventData.err.Error(); got != want {
			t.Errorf("output data = `%v`, want `%v`", got, want)
		}
	})
}
