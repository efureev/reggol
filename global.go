package reggol

import "sync/atomic"

var (
	// ExitCode is used by Logger.Fatal() as process exit code.
	//
	//nolint:gochecknoglobals // package-level configuration knob
	ExitCode = 1
	// gLevel holds the global minimum log level (default: InfoLevel).
	// Using a plain int32 allows zero-cost initialization without init().
	//
	//nolint:gochecknoglobals // accessed via atomic ops
	gLevel int32 = int32(InfoLevel)
)

// SetGlobalLevel sets the global override for log level. If this
// values is raised, all Loggers will use at least this value.
//
// To globally disable logs, set GlobalLevel to Disabled.
func SetGlobalLevel(l Level) {
	atomic.StoreInt32(&gLevel, int32(l))
}

// GlobalLevel returns the current global log level.
func GlobalLevel() Level {
	return Level(atomic.LoadInt32(&gLevel))
}
