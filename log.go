package reggol

import (
	"fmt"
	"io"
	"os"
)

type Logger struct {
	w LevelWriter
	//context []byte
	//ctx     context.Context
	level Level
}

func New(w io.Writer) Logger {
	if w == nil {
		w = io.Discard
	}
	lw, ok := w.(LevelWriter)
	if !ok {
		writer, ok := w.(TransformWriter)
		if !ok {
			writer = TransformWriterAdapter{w, NewTextTransformer(``)}
		}

		lw = LevelWriterAdapter{writer}
	}

	return Logger{w: lw, level: TraceLevel}
}

// Nop returns a disabled logger for which all operation are no-op.
func Nop() Logger {
	return New(nil).Level(Disabled)
}

// Level creates a child logger with the minimum accepted level set to level.
func (l Logger) Level(lvl Level) Logger {
	l.level = lvl
	return l
}

// GetLevel returns the current Level of l.
func (l Logger) GetLevel() Level {
	return l.level
}

func (l Logger) Write(p []byte) (n int, err error) {
	n = len(p)
	if n > 0 && p[n-1] == '\n' {
		// Trim CR added by stdlog.
		p = p[0 : n-1]
	}
	l.Log().Msg(string(p))

	return
}

func (l *Logger) newEvent(level Level, doneFn func(string)) *Event {
	enabled := l.should(level)
	if !enabled {
		if doneFn != nil {
			doneFn("")
		}
		return nil
	}
	e := newEvent(l.w, level)
	e.doneFn = doneFn
	//e.ctx = l.ctx

	//if l.context != nil && len(l.context) > 1 {
	//	e.buf = enc.AppendObjectData(e.buf, l.context)
	//}
	//if l.stack {
	//	e.Stack()
	//}
	return e
}

// should returns true if the log event should be logged.
func (l *Logger) should(lvl Level) bool {
	if l.w == nil {
		return false
	}
	if lvl < l.level || lvl < GlobalLevel() {
		return false
	}

	return true
}

////

func (l *Logger) Log() *Event {
	return l.newEvent(NoLevel, nil)
}

func (l *Logger) Trace() *Event {
	return l.newEvent(TraceLevel, nil)
}

func (l *Logger) Debug() *Event {
	return l.newEvent(DebugLevel, nil)
}

func (l *Logger) Warn() *Event {
	return l.newEvent(WarnLevel, nil)
}

func (l *Logger) Info() *Event {
	return l.newEvent(InfoLevel, nil)
}

func (l *Logger) Error() *Event {
	return l.newEvent(ErrorLevel, nil)
}

func (l *Logger) Err(err error) *Event {
	if err != nil {
		return l.Error().Err(err)
	}

	return l.Info()
}

func (l *Logger) Fatal() *Event {
	return l.newEvent(FatalLevel, func(msg string) {
		if closer, ok := l.w.(io.Closer); ok {
			// Close the writer to flush any buffered message. Otherwise the message
			// will be lost as os.Exit() terminates the program immediately.
			closer.Close()
		}
		os.Exit(1)
	})
}

func (l *Logger) Panic() *Event {
	return l.newEvent(PanicLevel, func(msg string) { panic(msg) })
}

func (l *Logger) WithLevel(level Level) *Event {
	switch level {
	case TraceLevel:
		return l.Trace()
	case DebugLevel:
		return l.Debug()
	case InfoLevel:
		return l.Info()
	case WarnLevel:
		return l.Warn()
	case ErrorLevel:
		return l.Error()
	case FatalLevel:
		return l.newEvent(FatalLevel, nil)
	case PanicLevel:
		return l.newEvent(PanicLevel, nil)
	case NoLevel:
		return l.Log()
	case Disabled:
		return nil
	default:
		return l.newEvent(level, nil)
	}
}

func (l *Logger) Print(v ...interface{}) {
	if e := l.Debug(); e.Enabled() {
		e.Msg(fmt.Sprint(v...))
	}
}

func (l *Logger) Printf(format string, v ...interface{}) {
	if e := l.Debug(); e.Enabled() {
		e.Msg(fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Println(v ...interface{}) {
	if e := l.Debug(); e.Enabled() {
		e.Msg(fmt.Sprintln(v...))
	}
}
