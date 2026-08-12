package reggol

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Level defines log levels.
type Level int8

// Log levels, ordered by severity. TraceLevel is the lowest; values below it
// are treated as numbers.
const (
	// TraceLevel defines trace log level.
	TraceLevel Level = iota - 1
	// DebugLevel defines debug log level.
	DebugLevel
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// NoLevel defines an absent log level.
	NoLevel
	// Disabled disables the logger.
	Disabled
)

// Level field values used by the text and JSON encoders.
const (
	LevelTraceValue = "trace"
	LevelDebugValue = "debug"
	LevelInfoValue  = "info"
	LevelWarnValue  = "warn"
	LevelErrorValue = "error"
	LevelFatalValue = "fatal"
	LevelPanicValue = "panic"
)

// Errors returned by ParseLevel.
var (
	// ErrUnknownLevel reports a level string that matches no known level.
	ErrUnknownLevel = errors.New("unknown level")
	// ErrLevelOutOfRange reports a numeric level outside the int8 range.
	ErrLevelOutOfRange = errors.New("level out of range")
)

// levelOffset maps a Level to a table index: TraceLevel (-1) becomes 0.
const levelOffset = 1

// levelCount is the number of addressable levels, TraceLevel..Disabled.
const levelCount = 9

// levelIndex reports whether lvl is a known level and its table index.
func levelIndex(lvl Level) (int, bool) {
	i := int(lvl) + levelOffset

	return i, i >= 0 && i < levelCount
}

// Indexed lookup tables replace maps: a log line pays an array index rather
// than a hash, and the tables cannot be mutated from another goroutine.
//
//nolint:gochecknoglobals // immutable lookup tables
var (
	levelNames = [levelCount]string{
		LevelTraceValue, LevelDebugValue, LevelInfoValue, LevelWarnValue,
		LevelErrorValue, LevelFatalValue, LevelPanicValue, "", "disabled",
	}

	levelLabels = [levelCount]string{
		"TRC", "DBG", "INF", "WRN", "ERR", "FTL", "PNC", "", "",
	}
)

// String returns the lowercase name of the level.
func (l Level) String() string {
	if i, ok := levelIndex(l); ok {
		return levelNames[i]
	}

	return strconv.Itoa(int(l))
}

// Label returns the short uppercase label used by the console encoder.
func (l Level) Label() string {
	if i, ok := levelIndex(l); ok && levelLabels[i] != "" {
		return levelLabels[i]
	}

	return "???"
}

// ParseLevel converts a level string into a Level value.
//
// Numeric strings are accepted so that levels below TraceLevel round-trip.
// Errors wrap ErrUnknownLevel or ErrLevelOutOfRange.
func ParseLevel(levelStr string) (Level, error) {
	for i := range levelCount {
		if levelNames[i] == "" {
			continue
		}

		if strings.EqualFold(levelStr, levelNames[i]) {
			return Level(i - levelOffset), nil
		}
	}

	if levelStr == "" {
		return NoLevel, nil
	}

	i, err := strconv.Atoi(levelStr)
	if err != nil {
		return NoLevel, fmt.Errorf("%w: %q", ErrUnknownLevel, levelStr)
	}

	if i > math.MaxInt8 || i < math.MinInt8 {
		return NoLevel, fmt.Errorf("%w: %d", ErrLevelOutOfRange, i)
	}

	return Level(i), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for toml/yaml/json.
func (l *Level) UnmarshalText(text []byte) error {
	if l == nil {
		return errors.New("reggol: cannot unmarshal into a nil *Level")
	}

	lvl, err := ParseLevel(string(text))
	if err != nil {
		return err
	}

	*l = lvl

	return nil
}

// MarshalText implements encoding.TextMarshaler for toml/yaml/json.
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}
