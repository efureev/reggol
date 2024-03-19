package reggol

import (
	"bytes"
	"io"
)

type TransformWriter interface {
	io.Writer
	Transformer() Transformer
}

type TransformWriterAdapter struct {
	io.Writer
	Trans Transformer
}

func (lw TransformWriterAdapter) Transformer() Transformer {
	return lw.Trans
}

func (lw TransformWriterAdapter) Write(p []byte) (n int, err error) {
	buf := bytes.Buffer{}
	buf.Write(p)
	err = buf.WriteByte('\n')

	if err != nil {
		return n, err
	}

	return lw.Writer.Write(buf.Bytes())
}

type LevelWriter interface {
	TransformWriter
	WriteLevel(data EventData) (n int, err error)
}

// LevelWriterAdapter adapts an io.Writer to support the LevelWriter interface.
type LevelWriterAdapter struct {
	TransformWriter
}

// WriteLevel simply writes everything to the adapted writer, ignoring the level.
func (lw LevelWriterAdapter) WriteLevel(data EventData) (n int, err error) {
	return lw.Write(lw.Transformer().Transform(data))
}

// Call the underlying writer's Close method if it is an io.Closer. Otherwise does nothing.
func (lw LevelWriterAdapter) Close() error {
	if closer, ok := lw.TransformWriter.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
