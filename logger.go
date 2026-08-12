package reggol

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Logger emits events through an encoder into a writer.
//
// Every method has a value receiver. That is not a style choice: in v0 the
// level methods were declared on *Logger while New returned a value, so
// `reggol.New(w).Info()` — the shape used in the README — did not compile,
// because the result of a call is not addressable.
//
// A Logger is immutable; Level and With return copies.
type Logger struct {
	w          Writer
	enc        Encoder
	ctx        []byte
	ctxFields  []Field
	extractors []ContextExtractor
	level      Level
}

// ContextExtractor pulls fields out of a context.Context onto an event.
type ContextExtractor func(ctx context.Context, e *Event)

// Option configures a Logger at construction.
type Option func(*Logger)

// WithEncoder sets the encoder. The console encoder is used when unset.
func WithEncoder(enc Encoder) Option {
	return func(l *Logger) {
		if enc != nil {
			l.enc = enc
		}
	}
}

// WithLevel sets the logger's minimum level.
func WithLevel(lvl Level) Option {
	return func(l *Logger) { l.level = lvl }
}

// WithContextExtractor appends a context extractor.
//
// With no extractors installed the context path costs nothing.
func WithContextExtractor(fn ContextExtractor) Option {
	return func(l *Logger) {
		if fn != nil {
			l.extractors = append(l.extractors, fn)
		}
	}
}

// New creates a Logger writing to w.
//
// Concurrency: the logger is as safe as w. Wrap w in SyncWriter when several
// goroutines log into a writer that is not itself synchronized.
func New(w io.Writer, opts ...Option) Logger {
	l := Logger{
		w:     toWriter(w),
		enc:   NewConsoleEncoder(),
		level: TraceLevel,
	}

	for _, opt := range opts {
		opt(&l)
	}

	return l
}

// Nop returns a logger that discards everything.
func Nop() Logger {
	return New(io.Discard, WithLevel(Disabled))
}

// Level returns a copy of the logger with the given minimum level.
func (l Logger) Level(lvl Level) Logger {
	l.level = lvl

	return l
}

// GetLevel returns the logger's minimum level.
func (l Logger) GetLevel() Level { return l.level }

// Encoder returns the logger's encoder.
func (l Logger) Encoder() Encoder { return l.enc }

// Output returns a copy of the logger writing to w.
func (l Logger) Output(w io.Writer) Logger {
	l.w = toWriter(w)

	return l
}

// should reports whether an event at lvl passes both the logger's own minimum
// and the global one.
//
// Kept small and branch-flat so that it inlines: the zero-cost disabled path
// depends on it.
func (l Logger) should(lvl Level) bool {
	if l.w == nil || lvl == Disabled {
		return false
	}

	return lvl >= l.level && lvl >= GlobalLevel()
}

func (l Logger) newEvent(lvl Level, doneFn func(string)) *Event {
	if !l.should(lvl) {
		if doneFn != nil {
			doneFn("")
		}

		return nil
	}

	e := newEvent(l.w, l.enc, lvl)
	e.doneFn = doneFn
	e.data.ctx = l.ctx

	if len(l.ctxFields) > 0 && l.ctx == nil {
		e.data.fields = append(e.data.fields, l.ctxFields...)
	}

	return e
}

// Trace starts a trace-level event.
func (l Logger) Trace() *Event { return l.newEvent(TraceLevel, nil) }

// Debug starts a debug-level event.
func (l Logger) Debug() *Event { return l.newEvent(DebugLevel, nil) }

// Info starts an info-level event.
func (l Logger) Info() *Event { return l.newEvent(InfoLevel, nil) }

// Warn starts a warn-level event.
func (l Logger) Warn() *Event { return l.newEvent(WarnLevel, nil) }

// Error starts an error-level event.
func (l Logger) Error() *Event { return l.newEvent(ErrorLevel, nil) }

// Log starts an event with no level.
func (l Logger) Log() *Event { return l.newEvent(NoLevel, nil) }

// Err starts an error-level event carrying err, or an info-level event when err
// is nil.
func (l Logger) Err(err error) *Event {
	if err != nil {
		return l.Error().Err(err)
	}

	return l.Info()
}

// Fatal starts a fatal-level event. Writing it flushes the writer and exits the
// process with ExitCode.
//
// The process exits even when the level filters the event out. Suppressing the
// record is a logging decision; suppressing the termination would let a raised
// log level silently change control flow.
func (l Logger) Fatal() *Event {
	return l.newEvent(FatalLevel, func(string) {
		if c, ok := l.w.(io.Closer); ok {
			_ = c.Close()
		}

		os.Exit(ExitCode())
	})
}

// Panic starts a panic-level event. Writing it panics with the message.
func (l Logger) Panic() *Event {
	return l.newEvent(PanicLevel, func(msg string) { panic(msg) })
}

// WithLevel starts an event at an arbitrary level.
func (l Logger) WithLevel(lvl Level) *Event {
	switch lvl {
	case FatalLevel:
		return l.Fatal()
	case PanicLevel:
		return l.Panic()
	case Disabled:
		return nil
	case TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, NoLevel:
		return l.newEvent(lvl, nil)
	default:
		return l.newEvent(lvl, nil)
	}
}

// Ctx starts an event at lvl with ctx attached, running any context extractors.
func (l Logger) Ctx(ctx context.Context, lvl Level) *Event {
	e := l.WithLevel(lvl)
	if e == nil {
		return nil
	}

	e.ctx = ctx

	for _, fn := range l.extractors {
		fn(ctx, e)
	}

	return e
}

// Write implements io.Writer, logging p as a message with no level.
//
// A trailing newline added by the standard library's log package is trimmed.
func (l Logger) Write(p []byte) (int, error) {
	n := len(p)

	if n > 0 && p[n-1] == '\n' {
		p = p[:n-1]
	}

	l.Log().Msg(string(p))

	return n, nil
}

// Print logs at debug level in the manner of fmt.Print.
func (l Logger) Print(v ...any) {
	if e := l.Debug(); e.Enabled() {
		e.Msg(fmt.Sprint(v...))
	}
}

// Printf logs at debug level in the manner of fmt.Printf.
func (l Logger) Printf(format string, v ...any) {
	if e := l.Debug(); e.Enabled() {
		e.Msgf(format, v...)
	}
}

// Println logs at debug level in the manner of fmt.Println.
func (l Logger) Println(v ...any) {
	if e := l.Debug(); e.Enabled() {
		e.Msg(fmt.Sprintln(v...))
	}
}
