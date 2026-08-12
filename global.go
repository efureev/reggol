package reggol

import (
	"sync/atomic"
)

// Package-level state, all accessed atomically so that changing it at runtime
// is not a data race.
//
//nolint:gochecknoglobals // package configuration, guarded by atomics
var (
	// gLevel holds the global minimum log level.
	gLevel atomic.Int32
	// gExitCode holds the process exit code used by Fatal.
	gExitCode atomic.Int32
	// gErrorHandler holds the optional handler for write failures.
	gErrorHandler atomic.Pointer[func(error)]
)

//nolint:gochecknoinits // atomic defaults cannot be expressed as initializers
func init() {
	gLevel.Store(int32(InfoLevel))
	gExitCode.Store(1)
}

// SetGlobalLevel sets the global override for the log level. Raising it raises
// the effective minimum for every logger.
//
// Set it to Disabled to silence logging process-wide.
func SetGlobalLevel(l Level) { gLevel.Store(int32(l)) }

// GlobalLevel returns the current global log level.
func GlobalLevel() Level { return Level(gLevel.Load()) }

// SetExitCode sets the process exit code used by Logger.Fatal.
func SetExitCode(code int) { gExitCode.Store(int32(code)) }

// ExitCode returns the process exit code used by Logger.Fatal.
func ExitCode() int { return int(gExitCode.Load()) }

// SetErrorHandler installs a handler invoked when writing an event fails.
//
// Passing nil restores the default, which reports the failure on stderr.
func SetErrorHandler(fn func(error)) {
	if fn == nil {
		gErrorHandler.Store(nil)

		return
	}

	gErrorHandler.Store(&fn)
}

// errorHandler returns the installed handler, or nil.
func errorHandler() func(error) {
	if fn := gErrorHandler.Load(); fn != nil {
		return *fn
	}

	return nil
}
