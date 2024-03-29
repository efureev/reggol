package reggol

import (
	"bytes"
	"errors"
	"testing"
)

func TestLogNew(t *testing.T) {
	t.Run(`New`, func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := New(buf)

		e1 := logger.Debug().Str(`key1`, `val1`)
		e1.Push()

		//nolint:forcetypeassert
		ts1 := logger.w.Transformer().(TextTransformer).formatTimestamp(e1.data.ts)

		e2 := logger.Debug().Str(`key2`, `val2`)
		e2.Push()

		//nolint:forcetypeassert
		ts2 := logger.w.Transformer().(TextTransformer).formatTimestamp(e2.data.ts)

		if expected, given := "ts="+ts1+", level=debug, key1=val1\nts="+ts2+", level=debug, key2=val2\n",
			buf.String(); expected != given {
			t.Errorf("Expected: `%s`, given: `%s`", expected, given)
		}
	})

	t.Run(`New From Console`, func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := New(NewConsoleWriter(func(w *ConsoleWriter) { w.Out = buf }))

		e1 := logger.Debug().Str(`key1`, `val1`)
		e1.Push()

		//nolint:forcetypeassert
		ts1 := logger.w.Transformer().(ConsoleTransformer).formatTimestamp(e1.data.ts)

		e2 := logger.Debug().Str(`key2`, `val2`)
		e2.Push()

		//nolint:forcetypeassert
		ts2 := logger.w.Transformer().(ConsoleTransformer).formatTimestamp(e2.data.ts)

		lvl := ColorBold.Wrap(`DBG`)

		if expected, given := ts1+" "+lvl+" key1=val1\n"+ts2+" "+lvl+" key2=val2\n", buf.String(); expected != given {
			t.Errorf("Expected: `%s`, given: `%s`", expected, given)
		}
	})

	t.Run(`Console: Int`, func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := New(
			NewConsoleWriter(
				func(w *ConsoleWriter) { w.Out = buf },
				func(w *ConsoleWriter) {
					tr := NewConsoleTransformer(true, ``)
					tr.HideTimestamp()
					w.Trans = tr
				},
			),
		)

		logger.Info().Int(`key1`, 32).Push()

		if expected, given := "INF key1=32\n", buf.String(); expected != given {
			t.Errorf("Expected: `%s`, given: `%s`", expected, given)
		}
	})

	t.Run(`Console: Bool`, func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := New(
			NewConsoleWriter(
				func(w *ConsoleWriter) { w.Out = buf },
				func(w *ConsoleWriter) {
					tr := NewConsoleTransformer(true, ``)
					tr.HideTimestamp()
					w.Trans = tr
				},
			),
		)

		logger.Info().Bool(`enabled`, true).Str(`key1`, `rer`).Push()

		if expected, given := "INF enabled=true key1=rer\n", buf.String(); expected != given {
			t.Errorf("Expected: `%s`, given: `%s`", expected, given)
		}
	})

	t.Run(`Console: Err`, func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := New(
			NewConsoleWriter(
				func(w *ConsoleWriter) { w.Out = buf },
				func(w *ConsoleWriter) {
					tr := NewConsoleTransformer(false, ``)
					tr.HideTimestamp()
					w.Trans = tr
				},
			),
		)

		err := errors.New(`test error`)
		logger.Err(err).Push()
		lvl := (ColorFgRed | ColorBold).Wrap(`ERR`)
		errMsg := ColorFgRed.Wrap(`test error`)
		if expected, given := lvl+" "+errMsg+"\n", buf.String(); expected != given {
			t.Errorf("Expected: `%s`, given: `%s`", expected, given)
		}

		// buf.WriteTo(os.Stdout)
	})

	t.Run(`Console: Error`, func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := New(
			NewConsoleWriter(
				func(w *ConsoleWriter) { w.Out = buf },
				func(w *ConsoleWriter) {
					tr := NewConsoleTransformer(false, ``)
					tr.HideTimestamp()
					w.Trans = tr
				},
			),
		)

		logger.Error().Msg(`test message`)
		lvl := (ColorFgRed | ColorBold).Wrap(`ERR`)

		if expected, given := lvl+" test message\n", buf.String(); expected != given {
			t.Errorf("Expected: `%s`, given: `%s`", expected, given)
		}

		// buf.WriteTo(os.Stdout)
	})
}
