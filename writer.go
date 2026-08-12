package reggol

import (
	"io"
	"sync"
)

// Writer accepts fully encoded events.
//
// The level is passed alongside the bytes so that a writer can route records by
// severity without decoding them.
type Writer interface {
	WriteLevel(l Level, p []byte) (n int, err error)
}

// LevelWriter is an alias for Writer, kept so that a signature can say which of
// the two roles it means.
//
// Nothing in reggol uses it; it is retained only because removing an exported
// name needs a major version.
type LevelWriter = Writer

// writerAdapter turns a plain io.Writer into a Writer.
type writerAdapter struct {
	w io.Writer
}

func (a writerAdapter) WriteLevel(_ Level, p []byte) (int, error) {
	return a.w.Write(p)
}

func (a writerAdapter) Close() error {
	if c, ok := a.w.(io.Closer); ok {
		return c.Close()
	}

	return nil
}

// toWriter adapts w to the Writer interface.
func toWriter(w io.Writer) Writer {
	if w == nil {
		return writerAdapter{w: io.Discard}
	}

	if lw, ok := w.(Writer); ok {
		return lw
	}

	return writerAdapter{w: w}
}

// syncWriter serializes concurrent writes to an underlying writer.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// SyncWriter wraps w so that concurrent writes are serialized.
//
// Without it, a logger is exactly as concurrency-safe as the writer underneath
// it — and most writers, including bytes.Buffer and any wrapper around one, are
// not safe at all. In v0 there was no way to make logging from several
// goroutines correct; this is that way.
func SyncWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}

	return &syncWriter{w: w}
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.w.Write(p)
}

// WriteLevel forwards to the underlying writer under the same lock, so that a
// level-aware writer stays serialized too.
func (s *syncWriter) WriteLevel(l Level, p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if lw, ok := s.w.(Writer); ok {
		return lw.WriteLevel(l, p)
	}

	return s.w.Write(p)
}

// Close closes the underlying writer if it is an io.Closer.
func (s *syncWriter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.w.(io.Closer); ok {
		return c.Close()
	}

	return nil
}

// MultiWriter fans an event out to several writers.
//
// The first error is reported; every writer is still attempted.
func MultiWriter(writers ...io.Writer) Writer {
	ws := make([]Writer, 0, len(writers))
	for _, w := range writers {
		ws = append(ws, toWriter(w))
	}

	return multiWriter(ws)
}

type multiWriter []Writer

func (m multiWriter) WriteLevel(l Level, p []byte) (int, error) {
	var (
		n        int
		firstErr error
	)

	for _, w := range m {
		written, err := w.WriteLevel(l, p)
		if err != nil && firstErr == nil {
			firstErr = err
		}

		n = written
	}

	return n, firstErr
}
