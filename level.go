package reggol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gh.tarampamp.am/colors"
)

// Level defines log levels.
type Level int8

var (
	// LevelTraceValue is the value used for the trace level field.
	LevelTraceValue = "trace"
	// LevelDebugValue is the value used for the debug level field.
	LevelDebugValue = "debug"
	// LevelInfoValue is the value used for the info level field.
	LevelInfoValue = "info"
	// LevelWarnValue is the value used for the warn level field.
	LevelWarnValue = "warn"
	// LevelErrorValue is the value used for the error level field.
	LevelErrorValue = "error"
	// LevelFatalValue is the value used for the fatal level field.
	LevelFatalValue = "fatal"
	// LevelPanicValue is the value used for the panic level field.
	LevelPanicValue = "panic"

	LevelFieldMarshalFunc = func(l Level) string {
		return l.String()
	}

	// (colors.FgYellow | colors.Bold))
	// LevelColors are used by ConsoleWriter's consoleDefaultFormatLevel to color
	// log levels.
	LevelColors = map[Level]colors.TextStyle{
		TraceLevel: colors.FgBlue,
		DebugLevel: 0,
		InfoLevel:  colors.FgGreen,  // colorGreen,
		WarnLevel:  colors.FgYellow, // colorYellow,
		ErrorLevel: colors.FgRed,    // colorRed,
		FatalLevel: colors.FgRed,    // colorRed,
		PanicLevel: colors.FgRed,    // colorRed,
	}

	// FormattedLevels are used by ConsoleWriter's consoleDefaultFormatLevel
	// for a short level name.
	FormattedLevels = map[Level]string{
		TraceLevel: "TRC",
		DebugLevel: "DBG",
		InfoLevel:  "INF",
		WarnLevel:  "WRN",
		ErrorLevel: "ERR",
		FatalLevel: "FTL",
		PanicLevel: "PNC",
	}
)

const (
	// DebugLevel defines debug log level.
	DebugLevel Level = iota
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

	// TraceLevel defines trace log level.
	TraceLevel Level = -1
	// Values less than TraceLevel are handled as numbers.

)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return LevelTraceValue
	case DebugLevel:
		return LevelDebugValue
	case InfoLevel:
		return LevelInfoValue
	case WarnLevel:
		return LevelWarnValue
	case ErrorLevel:
		return LevelErrorValue
	case FatalLevel:
		return LevelFatalValue
	case PanicLevel:
		return LevelPanicValue
	case Disabled:
		return "disabled"
	case NoLevel:
		return ""
	}

	return strconv.Itoa(int(l))
}

// ParseLevel converts a level string into a zerolog Level value.
// returns an error if the input string does not match known values.
func ParseLevel(levelStr string) (Level, error) {
	switch {
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(TraceLevel)):
		return TraceLevel, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(DebugLevel)):
		return DebugLevel, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(InfoLevel)):
		return InfoLevel, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(WarnLevel)):
		return WarnLevel, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(ErrorLevel)):
		return ErrorLevel, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(FatalLevel)):
		return FatalLevel, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(PanicLevel)):
		return PanicLevel, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(Disabled)):
		return Disabled, nil
	case strings.EqualFold(levelStr, LevelFieldMarshalFunc(NoLevel)):
		return NoLevel, nil
	}

	i, err := strconv.Atoi(levelStr)

	if err != nil {
		return NoLevel, fmt.Errorf("Unknown Level String: '%s', defaulting to NoLevel", levelStr)
	}

	if i > 127 || i < -128 {
		return NoLevel, fmt.Errorf("Out-Of-Bounds Level: '%d', defaulting to NoLevel", i)
	}

	return Level(i), nil
}

// UnmarshalText implements encoding.TextUnmarshaler to allow for easy reading from toml/yaml/json formats.
func (l *Level) UnmarshalText(text []byte) error {
	if l == nil {
		return errors.New("can't unmarshal a nil *Level")
	}

	var err error

	*l, err = ParseLevel(string(text))

	return err
}

// MarshalText implements encoding.TextMarshaler to allow for easy writing into toml/yaml/json formats.
func (l Level) MarshalText() ([]byte, error) {
	return []byte(LevelFieldMarshalFunc(l)), nil
}
