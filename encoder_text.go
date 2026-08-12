package reggol

// TextEncoder renders events as a flat, greppable key=value line.
type TextEncoder struct {
	baseEncoder
}

// NewTextEncoder creates a text encoder.
func NewTextEncoder(opts ...EncoderOption) *TextEncoder {
	t := &TextEncoder{baseEncoder: newBaseEncoder(DefaultTextTimeFormat)}

	for _, opt := range opts {
		opt(&t.baseEncoder)
	}

	return t
}

// AppendEvent implements Encoder.
func (t *TextEncoder) AppendEvent(dst []byte, d *EventData) []byte {
	if t.BeforeEncode != nil {
		t.BeforeEncode(d)
	}

	sep := false

	if t.showTimestamp && !d.ts.IsZero() {
		dst = appendTextSep(dst, &sep)
		dst = append(dst, t.timeKey...)
		dst = append(dst, '=')
		dst = t.appendTime(dst, d.ts)
	}

	if t.showLevel && d.level != NoLevel {
		dst = appendTextSep(dst, &sep)
		dst = append(dst, t.levelKey...)
		dst = append(dst, '=')
		dst = t.appendLevel(dst, d.level)
	}

	if len(d.blocks) > 0 {
		dst = appendTextSep(dst, &sep)
		dst = t.appendBlocks(dst, d.blocks)
	}

	if d.message != "" {
		dst = appendTextSep(dst, &sep)
		dst = append(dst, t.messageKey...)
		dst = append(dst, '=')
		dst = t.appendMessage(dst, d.message)
	}

	if len(d.ctx) > 0 {
		dst = appendTextSep(dst, &sep)
		dst = append(dst, d.ctx...)
	}

	for _, f := range t.orderedFields(d) {
		dst = appendTextSep(dst, &sep)
		dst = t.appendField(dst, f.Key, f.Val)
	}

	if t.AfterEncode != nil {
		t.AfterEncode(d)
	}

	return append(dst, '\n')
}

// AppendPrefix implements PrefixEncoder.
func (t *TextEncoder) AppendPrefix(dst []byte, fields []Field) []byte {
	for i, f := range fields {
		if i > 0 {
			dst = append(dst, ", "...)
		}

		dst = t.appendField(dst, f.Key, f.Val)
	}

	return dst
}

func (t *TextEncoder) appendLevel(dst []byte, l Level) []byte {
	if t.FormatLevel != nil {
		return t.FormatLevel(dst, l)
	}

	return append(dst, l.String()...)
}

func (t *TextEncoder) appendField(dst []byte, key string, v Value) []byte {
	if t.FormatField != nil {
		return t.FormatField(dst, key, v)
	}

	dst = t.appendKey(dst, key)
	dst = append(dst, '=')

	return t.appendValue(dst, v)
}

func (t *TextEncoder) appendBlocks(dst []byte, blocks Blocks) []byte {
	if t.FormatBlocks != nil {
		return t.FormatBlocks(dst, blocks)
	}

	dst = append(dst, t.blocksKey...)
	dst = append(dst, "=["...)

	for i := range blocks {
		if i > 0 {
			dst = append(dst, ", "...)
		}

		dst = blocks[i].appendTo(dst)
	}

	return append(dst, ']')
}

// appendTextSep inserts the field delimiter before every element but the first.
func appendTextSep(dst []byte, sep *bool) []byte {
	if *sep {
		return append(dst, ", "...)
	}

	*sep = true

	return dst
}
