package reggol

import (
	"bytes"
	"sort"
	"strings"
	"time"

	"gh.tarampamp.am/colors"
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
		},
		noColor: noColor,
	}
}

func (ct ConsoleTransformer) IsNoColor() bool {
	return ct.noColor
}

func (ct ConsoleTransformer) formatTimestamp(ts time.Time) string {
	val := ct.AbstractTransformer.formatTimestamp(ts)

	return colorize(val, colors.FgBlack|colors.FgBright, ct.noColor)
}

func (ct ConsoleTransformer) formatLevel(lvl Level) string {
	val := ct.AbstractTransformer.formatLevel(lvl)

	return colorize(val, LevelColors[lvl]|colors.Bold, ct.noColor)
}

func (ct ConsoleTransformer) formatMessage(msg string) string {
	return ct.AbstractTransformer.formatMessage(msg)
}

func (ct ConsoleTransformer) formatBlocks(blocks Blocks) string {
	list := make([]string, len(blocks))
	for _, block := range blocks {
		list = append(list, block.Value())
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

	//nolint:prealloc
	var list []string

	// timestamp
	if ct.displayTimestamp {
		list = append(list, ct.formatTimestamp(data.ts))
	}

	// level
	if ct.displayLevel {
		list = append(list, ct.formatLevel(data.level))
	}

	// blocks
	if data.blocks != nil || len(data.blocks) > 0 {
		list = append(list, ct.formatBlocks(data.blocks))
	}

	// Error
	if data.err != nil {
		list = append(list, ct.formatError(data.err))
	} else if data.message != `` {
		// Message
		list = append(list, ct.formatMessage(data.message))
	}

	// fields
	keys := make([]string, 0, len(data.fields))
	for k := range data.fields {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		list = append(list, ct.formatField(k, data.fields[k]))
	}

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
