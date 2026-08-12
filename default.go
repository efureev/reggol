package reggol

import "time"

// Conventional field names used by the built-in encoders.
const (
	ErrorFieldName     = "error"
	TimestampFieldName = "ts"
	LevelFieldName     = "level"
	MessageFieldName   = "message"
	BlocksFieldName    = "blocks"
)

// Default time layouts.
//
// Unlike the v0 TimeFieldFormat these are the values actually in effect: the
// console favors a short human-readable clock, the machine-readable encoders
// favor RFC3339 with nanoseconds so that timestamps sort lexicographically.
const (
	// DefaultConsoleTimeFormat is the layout used by ConsoleEncoder.
	DefaultConsoleTimeFormat = time.Kitchen
	// DefaultTextTimeFormat is the layout used by TextEncoder.
	DefaultTextTimeFormat = time.RFC3339
	// DefaultJSONTimeFormat is the layout used by JSONEncoder.
	DefaultJSONTimeFormat = time.RFC3339Nano
	// TimeFieldFormat is the layout used to render time-valued fields.
	TimeFieldFormat = time.RFC3339
)
