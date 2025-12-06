package reggol

import (
	"bytes"
	"sort"
	"strings"
	"time"
)

// ConsoleTransformer is a transformer for console text.
type ConsoleTransformer struct {
	AbstractTransformer
	noColor bool
}

func NewConsoleTransformer(noColor bool, timeFormat string) ConsoleTransformer {
	return ConsoleTransformer{
		AbstractTransformer: AbstractTransformer{
			timeFormat:       timeFormat,
			fieldsDelimiter:  ` `,
			displayTimestamp: true,
			displayLevel:     true,
			SortFields:       true,
		},
		noColor: noColor,
	}
}

func (ct ConsoleTransformer) IsNoColor() bool {
	return ct.noColor
}

func (ct ConsoleTransformer) formatTimestamp(ts time.Time) string {
	val := ct.AbstractTransformer.formatTimestamp(ts)

	return colorize(val, ColorFgBlack|ColorFgBright, ct.noColor)
}

func (ct ConsoleTransformer) formatLevel(lvl Level) string {
	val := ct.AbstractTransformer.formatLevel(lvl)

	return colorize(val, LevelColors[lvl]|ColorBold, ct.noColor)
}

func (ct ConsoleTransformer) formatMessage(msg string) string {
	return ct.AbstractTransformer.formatMessage(msg)
}

func (ct ConsoleTransformer) formatBlocks(blocks Blocks) string {
	list := make([]string, len(blocks))
	for i, block := range blocks {
		list[i] = block.Value()
	}

	return strings.Join(list, ` `)
}

func (ct ConsoleTransformer) formatField(name string, value any) string {
	return ct.AbstractTransformer.formatField(name, value)
}

func (ct ConsoleTransformer) formatError(err error) string {
	val := ct.AbstractTransformer.formatError(err)

	return colorize(val, LevelColors[ErrorLevel], ct.noColor)
}

func (ct ConsoleTransformer) Transform(data EventData) []byte {
	if ct.AbstractTransformer.BeforeTransformFn != nil {
		(ct.AbstractTransformer.BeforeTransformFn)(data)
	}

	list := make([]string, 0, ct.preallocCap(data))

	ct.appendTimestamp(&list, data)
	ct.appendLevel(&list, data)
	ct.appendBlocks(&list, data)
	ct.appendMsgOrErr(&list, data)
	ct.appendFields(&list, data)

	b := bytes.Buffer{}

	lastIdx := len(list) - 1
	for i, item := range list {
		b.WriteString(item)

		if i != lastIdx {
			b.WriteString(ct.fieldsDelimiter)
		}
	}

	if ct.AbstractTransformer.AfterTransformFn != nil {
		(ct.AbstractTransformer.AfterTransformFn)(data)
	}

	return b.Bytes()
}

func (ct ConsoleTransformer) preallocCap(data EventData) int {
	capHint := 0

	if ct.displayTimestamp {
		capHint++
	}

	if ct.displayLevel && data.level != NoLevel {
		capHint++
	}

	if len(data.blocks) > 0 {
		capHint++
	}

	if data.err != nil || data.message != `` {
		capHint++
	}

	capHint += len(data.fields)

	return capHint
}

func (ct ConsoleTransformer) appendTimestamp(list *[]string, data EventData) {
	if ct.displayTimestamp {
		*list = append(*list, ct.formatTimestamp(data.ts))
	}
}

func (ct ConsoleTransformer) appendLevel(list *[]string, data EventData) {
	if ct.displayLevel && data.level != NoLevel {
		*list = append(*list, ct.formatLevel(data.level))
	}
}

func (ct ConsoleTransformer) appendBlocks(list *[]string, data EventData) {
	if len(data.blocks) > 0 {
		*list = append(*list, ct.formatBlocks(data.blocks))
	}
}

func (ct ConsoleTransformer) appendMsgOrErr(list *[]string, data EventData) {
	if data.err != nil {
		*list = append(*list, ct.formatError(data.err))

		return
	}

	if data.message != `` {
		*list = append(*list, ct.formatMessage(data.message))
	}
}

func (ct ConsoleTransformer) appendFields(list *[]string, data EventData) {
	if len(data.fields) == 0 {
		return
	}

	keys := make([]string, 0, len(data.fields))
	for k := range data.fields {
		keys = append(keys, k)
	}

	if ct.SortFields {
		sort.Strings(keys)
	}

	for _, k := range keys {
		*list = append(*list, ct.formatField(k, data.fields[k]))
	}
}
