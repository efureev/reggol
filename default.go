package reggol

var (
	ErrorFieldName     = `error`
	TimestampFieldName = `ts`
	LevelFieldName     = `level`
	MessageFieldName   = `message`

	ErrorMarshalFunc = func(err error) interface{} {
		return err
	}

	ErrorHandler func(err error)
)
