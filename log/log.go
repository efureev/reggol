// Package log provides a package-level logger, ready to use without setup.
//
// It writes human-readable output to stderr and is safe to use from several
// goroutines: the default writer is wrapped in reggol.SyncWriter.
package log

import (
	"context"
	"os"
	"sync/atomic"

	"github.com/efureev/reggol"
)

// current holds the package logger.
//
// v0 exported this as a plain variable, so replacing it raced with every
// goroutine reading it. Access now goes through an atomic pointer.
//
//nolint:gochecknoglobals // package-level logger, guarded by an atomic
var current atomic.Pointer[reggol.Logger]

//nolint:gochecknoinits // the default logger cannot be expressed as an initializer
func init() {
	out := reggol.SyncWriter(os.Stderr)
	l := reggol.New(out, reggol.WithEncoder(
		reggol.NewConsoleEncoder(reggol.WithColorMode(reggol.ColorAuto, os.Stderr)),
	))

	current.Store(&l)
}

// L returns the package logger.
func L() reggol.Logger { return *current.Load() }

// SetLogger replaces the package logger.
func SetLogger(l reggol.Logger) { current.Store(&l) }

// Level returns a child logger with the given minimum level.
func Level(level reggol.Level) reggol.Logger { return L().Level(level) }

// GetLevel returns the package logger's minimum level.
func GetLevel() reggol.Level { return L().GetLevel() }

// Output returns a child logger writing to w.
func Output(w interface{ Write([]byte) (int, error) }) reggol.Logger { return L().Output(w) }

// With starts building a child logger with bound fields.
func With() reggol.Context { return L().With() }

// Trace starts a trace-level event.
func Trace() *reggol.Event { return L().Trace() }

// Debug starts a debug-level event.
func Debug() *reggol.Event { return L().Debug() }

// Info starts an info-level event.
func Info() *reggol.Event { return L().Info() }

// Warn starts a warn-level event.
func Warn() *reggol.Event { return L().Warn() }

// Error starts an error-level event.
func Error() *reggol.Event { return L().Error() }

// Err starts an error-level event carrying err, or an info-level event when err
// is nil.
func Err(err error) *reggol.Event { return L().Err(err) }

// Fatal starts a fatal-level event; writing it exits the process.
func Fatal() *reggol.Event { return L().Fatal() }

// Panic starts a panic-level event; writing it panics.
func Panic() *reggol.Event { return L().Panic() }

// Log starts an event with no level.
func Log() *reggol.Event { return L().Log() }

// WithLevel starts an event at an arbitrary level.
func WithLevel(level reggol.Level) *reggol.Event { return L().WithLevel(level) }

// Ctx starts an event at the given level with ctx attached.
func Ctx(ctx context.Context, level reggol.Level) *reggol.Event { return L().Ctx(ctx, level) }

// WithContext returns a copy of ctx carrying the package logger.
func WithContext(ctx context.Context) context.Context { return L().WithContext(ctx) }

// Write implements io.Writer, logging p with no level.
func Write(p []byte) (int, error) { return L().Write(p) }

// Print logs at debug level in the manner of fmt.Print.
func Print(v ...any) { L().Print(v...) }

// Printf logs at debug level in the manner of fmt.Printf.
func Printf(format string, v ...any) { L().Printf(format, v...) }

// Println logs at debug level in the manner of fmt.Println.
func Println(v ...any) { L().Println(v...) }
