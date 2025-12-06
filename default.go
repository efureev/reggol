package reggol

const (
	ErrorFieldName     = `error`
	TimestampFieldName = `ts`
	LevelFieldName     = `level`
	MessageFieldName   = `message`
)

//nolint:gochecknoglobals // exposed for package customization
var ErrorMarshalFunc = func(err error) interface{} { return err }

// ErrorHandler is an optional global error handler for write failures.
//
//nolint:gochecknoglobals // conventional global callback
var ErrorHandler func(err error)
