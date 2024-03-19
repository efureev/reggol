package reggol

import (
	"bytes"
	"fmt"
	"sort"
)

// TextTransformer is a simple transformer for plain text.
type TextTransformer struct {
	AbstractTransformer
}

func NewTextTransformer(timeFormat string) TextTransformer {
	return TextTransformer{
		AbstractTransformer{
			timeFormat:       timeFormat,
			fieldsDelimiter:  `, `,
			displayTimestamp: true,
			displayLevel:     true,
		},
	}
}

func (tt TextTransformer) formatLevel(lvl Level) string {
	return lvl.String()
}

func (tt TextTransformer) Transform(data EventData) []byte {
	//nolint:prealloc
	var list []string

	// timestamp
	if tt.displayTimestamp {
		list = append(list, fmt.Sprintf(`%s=%s`, tt.formatFieldName(TimestampFieldName), tt.formatTimestamp(data.ts)))
	}

	// level
	if tt.displayLevel {
		list = append(list, fmt.Sprintf(`%s=%s`, tt.formatFieldName(LevelFieldName), tt.formatLevel(data.level)))
	}

	// Error
	if data.err != nil {
		list = append(list, fmt.Sprintf(`%s=%s`, tt.formatFieldName(ErrorFieldName), tt.formatError(data.err)))
	} else if data.message != `` {
		// Message
		list = append(list, fmt.Sprintf(`%s=%s`, tt.formatFieldName(MessageFieldName), tt.formatMessage(data.message)))
	}

	// fields
	keys := make([]string, 0, len(data.fields))

	for k := range data.fields {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		list = append(list, fmt.Sprintf(`%s=%s`, tt.formatFieldName(k), tt.formatFieldValue(data.fields[k])))
	}

	b := bytes.Buffer{}
	lastIdx := len(list) - 1

	for i, item := range list {
		b.WriteString(item)

		if i != lastIdx {
			b.WriteString(tt.fieldsDelimiter)
		}
	}

	return b.Bytes()
}
