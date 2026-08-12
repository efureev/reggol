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
	caller     bool
	callerSkip int
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

// WithCaller records the call site on every event.
//
// Off by default, and deliberately so: capturing the program counter costs
// about 56 ns, which roughly doubles the price of a record. Nothing is
// allocated either way.
func WithCaller() Option {
	return func(l *Logger) { l.caller = true }
}

// WithCallerSkip sets how many extra frames to skip when recording the call
// site.
//
// Needed only by code that wraps reggol behind its own helpers: without it the
// reported position is the wrapper, not the code that called it.
func WithCallerSkip(n int) Option {
	return func(l *Logger) { l.callerSkip = n }
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

// AddCallerSkip returns a copy of the logger skipping n more frames when
// recording the call site.
//
// This is what a wrapper adds for each layer it puts between its users and
// reggol; the log/ facade uses it for exactly that reason.
func (l Logger) AddCallerSkip(n int) Logger {
	l.callerSkip += n

	return l
}

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

// event is the single constructor every public entry point funnels through.
//
// Every one of them must call it **directly**, exactly one hop away: the caller
// skip count is a constant, and routing one public method through another —
// Err through Error, Ctx through WithLevel — would silently shift the reported
// call site into reggol itself. TestCallerReportsCallSite guards this.
func (l Logger) event(lvl Level, doneFn func(string)) *Event {
	if !l.should(lvl) {
		if doneFn != nil {
			doneFn("")
		}

		return nil
	}

	e := newEvent(l.w, l.enc, lvl)
	e.doneFn = doneFn
	e.data.prefix = l.ctx

	if l.caller {
		e.data.pc = capturePC(callerSkipEvent + l.callerSkip)
	}

	if len(l.ctxFields) > 0 && l.ctx == nil {
		e.data.fields = append(e.data.fields, l.ctxFields...)
	}

	return e
}

// exitFn returns the completion callback used by Fatal.
func (l Logger) exitFn() func(string) {
	return func(string) {
		if c, ok := l.w.(io.Closer); ok {
			_ = c.Close()
		}

		os.Exit(ExitCode())
	}
}

// panicFn is the completion callback used by Panic.
func panicFn(msg string) { panic(msg) }

// Trace starts a trace-level event.
func (l Logger) Trace() *Event { return l.event(TraceLevel, nil) }

// Debug starts a debug-level event.
func (l Logger) Debug() *Event { return l.event(DebugLevel, nil) }

// Info starts an info-level event.
func (l Logger) Info() *Event { return l.event(InfoLevel, nil) }

// Warn starts a warn-level event.
func (l Logger) Warn() *Event { return l.event(WarnLevel, nil) }

// Error starts an error-level event.
func (l Logger) Error() *Event { return l.event(ErrorLevel, nil) }

// Log starts an event with no level.
func (l Logger) Log() *Event { return l.event(NoLevel, nil) }

// Err starts an error-level event carrying err, or an info-level event when err
// is nil.
func (l Logger) Err(err error) *Event {
	if err != nil {
		return l.event(ErrorLevel, nil).Err(err)
	}

	return l.event(InfoLevel, nil)
}

// Fatal starts a fatal-level event. Writing it flushes the writer and exits the
// process with ExitCode.
//
// The process exits even when the level filters the event out. Suppressing the
// record is a logging decision; suppressing the termination would let a raised
// log level silently change control flow.
func (l Logger) Fatal() *Event {
	return l.event(FatalLevel, l.exitFn())
}

// Panic starts a panic-level event. Writing it panics with the message.
func (l Logger) Panic() *Event {
	return l.event(PanicLevel, panicFn)
}

// WithLevel starts an event at an arbitrary level.
func (l Logger) WithLevel(lvl Level) *Event {
	switch lvl {
	case FatalLevel:
		return l.event(FatalLevel, l.exitFn())
	case PanicLevel:
		return l.event(PanicLevel, panicFn)
	case Disabled:
		return nil
	case TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, NoLevel:
		return l.event(lvl, nil)
	default:
		return l.event(lvl, nil)
	}
}

// Ctx starts an event at lvl with ctx attached, running any context extractors.
func (l Logger) Ctx(ctx context.Context, lvl Level) *Event {
	var e *Event

	switch lvl {
	case FatalLevel:
		e = l.event(FatalLevel, l.exitFn())
	case PanicLevel:
		e = l.event(PanicLevel, panicFn)
	case Disabled:
		return nil
	case TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, NoLevel:
		e = l.event(lvl, nil)
	default:
		e = l.event(lvl, nil)
	}

	if e == nil {
		return nil
	}

	e.data.goCtx = ctx

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

	l.event(NoLevel, nil).Msg(string(p))

	return n, nil
}

// Print logs at debug level in the manner of fmt.Print.
func (l Logger) Print(v ...any) {
	if e := l.event(DebugLevel, nil); e.Enabled() {
		e.Msg(fmt.Sprint(v...))
	}
}

// Printf logs at debug level in the manner of fmt.Printf.
func (l Logger) Printf(format string, v ...any) {
	if e := l.event(DebugLevel, nil); e.Enabled() {
		e.Msgf(format, v...)
	}
}

// Println logs at debug level in the manner of fmt.Println.
func (l Logger) Println(v ...any) {
	if e := l.event(DebugLevel, nil); e.Enabled() {
		e.Msg(fmt.Sprintln(v...))
	}
}
