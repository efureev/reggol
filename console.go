package reggol

import (
	"bytes"
	"io"
	"os"
	"sync"
)

var (
	consoleBufPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 100))
		},
	}
)

type ConsoleWriter struct {
	// Out is the output destination.
	Out   io.Writer
	Trans Transformer
}

// NewConsoleWriter creates and initializes a new ConsoleWriter.
func NewConsoleWriter(options ...func(w *ConsoleWriter)) ConsoleWriter {
	w := ConsoleWriter{
		Out: os.Stdout,
	}

	for _, opt := range options {
		opt(&w)
	}

	if w.Trans == nil {
		w.Trans = NewConsoleTransformer(false, ``)
	}

	// Fix color on Windows
	//if w.Out == os.Stdout || w.Out == os.Stderr {
	//	w.Out = colorable.NewColorable(w.Out.(*os.File))
	//}

	return w
}

func (w ConsoleWriter) Transformer() Transformer {
	return w.Trans
}

func (w ConsoleWriter) WithTransformer(trans Transformer) ConsoleWriter {
	w.Trans = trans

	return w
}

func (w ConsoleWriter) Write(p []byte) (n int, err error) {

	var buf = consoleBufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		consoleBufPool.Put(buf)
	}()

	buf.Write(p)
	err = buf.WriteByte('\n')
	if err != nil {
		return n, err
	}

	_, err = buf.WriteTo(w.Out)

	return len(p), err
}

func (w ConsoleWriter) Close() error {
	if closer, ok := w.Out.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
