package reggol

import (
	"bytes"
	"errors"
	"testing"
)

func newBufferWriter(transformer ...Transformer) (*bytes.Buffer, LevelWriter) {
	var trans Transformer
	if transformer != nil {
		trans = transformer[0]
	} else {
		trans = NewTextTransformer(``)
	}
	var buf bytes.Buffer
	return &buf, LevelWriterAdapter{
		TransformWriter: TransformWriterAdapter{&buf, trans},
	}
}
func TestEvent_Fields(t *testing.T) {

	t.Run(`create event`, func(t *testing.T) {
		_, writer := newBufferWriter()

		e := newEvent(writer, DebugLevel).
			Str(`key_1`, `val_1`).
			Str(`key_2`, `val_2`).
			Int(`key_3`, 32)

		if len(e.data.fields) != 3 {
			t.Errorf("EventData.fields should has %d items, given %d", 3, len(e.data.fields))
		}

		if e.data.level != DebugLevel {
			t.Errorf("EventData should has %s level, given %s", DebugLevel, e.data.level)
		}
		if e.data.message != `` {
			t.Errorf("EventData should has an empty Message, given %s", e.data.message)
		}
	})
}

func TestEvent_Err(t *testing.T) {
	t.Run(`event with Error`, func(t *testing.T) {
		_, writer := newBufferWriter()
		e := newEvent(writer, InfoLevel).
			Err(errors.New(`test error`))

		if len(e.data.fields) != 0 {
			t.Errorf("EventData.fields should has 0 items, given %d", len(e.data.fields))
		}

		if e.data.level != InfoLevel {
			t.Errorf("EventData should has %s level, given %s", InfoLevel, e.data.level)
		}
		if e.data.message != `` {
			t.Errorf("EventData should has an empty Message, given %s", e.data.message)
		}

		if e.data.err == nil {
			t.Error("EventData.err should have an error, given nil")
		}
		if e.data.err.Error() != `test error` {
			t.Errorf("EventData.err should have an error message `test error`, given %s", e.data.err.Error())
		}
	})
}
