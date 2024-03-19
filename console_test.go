package reggol

import (
	"errors"
	"testing"
)

func TestConsoleWriterTextTransformer(t *testing.T) {
	t.Run(`Create Console Writer with Text Transformer`, func(t *testing.T) {
		consoleWriter := NewConsoleWriter()

		e := newEvent(LevelWriterAdapter{consoleWriter}, WarnLevel).
			Str(`key_1`, `val_1`).
			Int(`key_3`, 32)

		if err := e.write(); err != nil {
			t.Errorf(`error: %s`, err)
		}
	})
}

func TestConsoleWriterConsoleTransformer(t *testing.T) {
	t.Run(`Create Console Writer with Console Transformer`, func(t *testing.T) {
		consoleWriter := NewConsoleWriter(func(w *ConsoleWriter) { w.Trans = NewConsoleTransformer(false, ``) })

		e := newEvent(LevelWriterAdapter{consoleWriter}, WarnLevel).
			Str(`key_1`, `val_1`).
			Int(`key_3`, 32)

		if err := e.write(); err != nil {
			t.Errorf(`error: %s`, err)
		}
	})
}

func TestConsoleWriterConsoleTransformer2(t *testing.T) {
	t.Run(`Create Console Writer with Console Transformer: Error`, func(t *testing.T) {
		consoleWriter := NewConsoleWriter(func(w *ConsoleWriter) { w.Trans = NewConsoleTransformer(false, ``) })

		e := newEvent(LevelWriterAdapter{consoleWriter}, WarnLevel).
			Err(errors.New(`text Error`))

		if err := e.write(); err != nil {
			t.Errorf(`error: %s`, err)
		}
	})
}
