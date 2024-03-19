package reggol

import "sync/atomic"

var (
	ExitCode = 1
	gLevel   = new(int32)
)

// SetGlobalLevel sets the global override for log level. If this
// values is raised, all Loggers will use at least this value.
//
// To globally disable logs, set GlobalLevel to Disabled.
func SetGlobalLevel(l Level) {
	atomic.StoreInt32(gLevel, int32(l))
}

// GlobalLevel returns the current global log level.
func GlobalLevel() Level {
	return Level(atomic.LoadInt32(gLevel))
}
