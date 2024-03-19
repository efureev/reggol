package log

import (
	"fmt"
	"os"

	"github.com/efureev/reggol"
)

// Logger is the global logger.
//
//nolint:gochecknoglobals
var Logger = reggol.New(os.Stderr)

// Level creates a child logger with the minimum accepted level set to level.
func Level(level reggol.Level) reggol.Logger {
	return Logger.Level(level)
}

// Err starts a new message with error level with err as a field if not nil or
// with info level if err is nil.
//
// You must call Msg on the returned event in order to send the event.
func Err(err error) *reggol.Event {
	return Logger.Err(err)
}

// Trace starts a new message with trace level.
//
// You must call Msg on the returned event in order to send the event.
func Trace() *reggol.Event {
	return Logger.Trace()
}

// Debug starts a new message with debug level.
//
// You must call Msg on the returned event in order to send the event.
func Debug() *reggol.Event {
	return Logger.Debug()
}

// Info starts a new message with info level.
//
// You must call Msg on the returned event in order to send the event.
func Info() *reggol.Event {
	return Logger.Info()
}

// Warn starts a new message with warn level.
//
// You must call Msg on the returned event in order to send the event.
func Warn() *reggol.Event {
	return Logger.Warn()
}

// Error starts a new message with error level.
//
// You must call Msg on the returned event in order to send the event.
func Error() *reggol.Event {
	return Logger.Error()
}

// Fatal starts a new message with fatal level. The os.Exit(1) function
// is called by the Msg method.
//
// You must call Msg on the returned event in order to send the event.
func Fatal() *reggol.Event {
	return Logger.Fatal()
}

// Panic starts a new message with panic level. The message is also sent
// to the panic function.
//
// You must call Msg on the returned event in order to send the event.
func Panic() *reggol.Event {
	return Logger.Panic()
}

// WithLevel starts a new message with level.
//
// You must call Msg on the returned event in order to send the event.
func WithLevel(level reggol.Level) *reggol.Event {
	return Logger.WithLevel(level)
}

// Log starts a new message with no level. Setting zerolog.GlobalLevel to
// zerolog.Disabled will still disable events produced by this method.
//
// You must call Msg on the returned event in order to send the event.
func Log() *reggol.Event {
	return Logger.Log()
}

// Print sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	Logger.Debug().Msg(fmt.Sprint(v...))
}

// Printf sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	Logger.Debug().Msgf(format, v...)
}
