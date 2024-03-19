package reggol

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultTimeFormat = time.Kitchen
	TimeFieldFormat   = time.RFC3339
)

type Formatter func(interface{}) string

type Transformer interface {
	Transform(data EventData) []byte
}

// AbstractTransformer is a abstract transformer for plain text.
type AbstractTransformer struct {
	displayTimestamp bool
	displayLevel     bool

	fieldsDelimiter string
	timeFormat      string

	FormatLevelFn      Formatter
	FormatTimestampFn  Formatter
	FormatFieldFn      Formatter
	FormatFieldNameFn  Formatter
	FormatFieldValueFn Formatter
	FormatErrorFn      Formatter
	FormatMessageFn    Formatter

	BeforeTransformFn func(data EventData)
	AfterTransformFn  func(data EventData)
}

func (st *AbstractTransformer) HideLevel() {
	st.displayLevel = false
}

func (st *AbstractTransformer) HideTimestamp() {
	st.displayTimestamp = false
}

func (st AbstractTransformer) formatError(err error) string {
	if st.FormatErrorFn != nil {
		return (st.FormatErrorFn)(err)
	}

	return err.Error()
}

func (st AbstractTransformer) formatLevel(lvl Level) string {
	if st.FormatLevelFn != nil {
		return (st.FormatLevelFn)(lvl)
	}

	var l string

	fl, ok := FormattedLevels[lvl]

	if ok {
		l = fl
	} else {
		l = "???"
	}

	return l
}

func (st AbstractTransformer) formatTimestamp(ts time.Time) string {
	if st.FormatTimestampFn != nil {
		return (st.FormatTimestampFn)(ts)
	}

	if st.timeFormat == "" {
		st.timeFormat = defaultTimeFormat
	}

	return ts.Local().Format(st.timeFormat)
}

func (st AbstractTransformer) formatMessage(msg string) string {
	if st.FormatMessageFn != nil {
		return (st.FormatMessageFn)(msg)
	}

	return msg
}

func (st AbstractTransformer) formatBlocks(blocks Blocks) string {
	list := make([]string, len(blocks))

	for i, block := range blocks {
		list[i] = block.Value()
	}

	return `blocks=[` + strings.Join(list, `, `) + `]`
}

func (st AbstractTransformer) formatField(name string, value any) string {
	if st.FormatFieldFn != nil {
		return (st.FormatFieldFn)([2]string{st.formatFieldName(name), st.formatFieldValue(value)})
	}

	return fmt.Sprintf(`%s=%s`, st.formatFieldName(name), st.formatFieldValue(value))
}

func (st AbstractTransformer) formatFieldName(name string) string {
	if st.FormatFieldNameFn != nil {
		return (st.FormatFieldNameFn)(name)
	}

	return name
}

func (st AbstractTransformer) formatFieldValue(i any) string {
	if st.FormatFieldValueFn != nil {
		return (st.FormatFieldValueFn)(i)
	}

	switch i.(type) {
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", i)
	case string:
		return fmt.Sprintf("%s", i)
	default:
		return fmt.Sprintf("%v", i)
	}
}
