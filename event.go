package reggol

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// Pool ceilings. A single oversized event must not permanently inflate every
// pooled buffer, which is what the long-standing TODO in v0 warned about and
// never implemented.
const (
	maxPooledBuf    = 64 << 10 // 64 KiB
	maxPooledMsg    = 8 << 10  // 8 KiB
	maxPooledFields = 64
	initialBufCap   = 256
	initialMsgCap   = 64
)

//nolint:gochecknoglobals // sync.Pool is the canonical shape for this
var eventPool = &sync.Pool{
	New: func() any {
		return &Event{
			buf:  make([]byte, 0, initialBufCap),
			data: EventData{message: make([]byte, 0, initialMsgCap)},
		}
	},
}

// EventData is the fully assembled state of one log event.
//
// Every accessor below is exported on purpose: in v0 the equivalent struct had
// only unexported fields and no accessors, which made it impossible for anyone
// outside the package to implement the encoder interface at all.
type EventData struct {
	ts      time.Time
	message []byte
	fields  []Field
	blocks  Blocks
	ctx     []byte
	level   Level
}

// Level returns the event level.
func (d *EventData) Level() Level { return d.level }

// Time returns the event timestamp.
func (d *EventData) Time() time.Time { return d.ts }

// Message returns the event message as a string.
//
// This allocates. Encoders and formatting hooks should use MessageBytes, which
// is what keeps Msgf allocation-free.
func (d *EventData) Message() string { return string(d.message) }

// MessageBytes returns the event message without copying.
//
// The bytes belong to the pooled event and are only valid until the terminal
// call returns; copy them if they must outlive it.
func (d *EventData) MessageBytes() []byte { return d.message }

// Fields returns the event fields in their current order.
func (d *EventData) Fields() []Field { return d.fields }

// Blocks returns the event blocks.
func (d *EventData) Blocks() Blocks { return d.blocks }

// Prefix returns the parent logger's pre-encoded fields.
//
// The bytes are already in the encoder's own syntax and are spliced in ahead of
// the event fields; this is what makes a child logger cost a memmove rather
// than a re-encode per line.
func (d *EventData) Prefix() []byte { return d.ctx }

// Event is a single log record under construction.
//
// An Event is pooled. Exactly one terminal call — Msg, Msgf or Send — must be
// made on it, after which the value must not be touched again.
type Event struct {
	w      Writer
	enc    Encoder
	buf    []byte
	data   EventData
	ctx    context.Context //nolint:containedctx // carried for context-aware hooks, never for cancellation
	doneFn func(msg string)
}

func newEvent(w Writer, enc Encoder, lvl Level) *Event {
	e, _ := eventPool.Get().(*Event)

	e.w = w
	e.enc = enc
	e.doneFn = nil
	e.ctx = nil
	e.buf = e.buf[:0]

	e.data.level = lvl
	e.data.ts = time.Now()
	e.data.message = e.data.message[:0]
	e.data.fields = e.data.fields[:0]
	e.data.blocks = e.data.blocks[:0]
	e.data.ctx = nil

	return e
}

func putEvent(e *Event) {
	if cap(e.buf) > maxPooledBuf ||
		cap(e.data.message) > maxPooledMsg ||
		cap(e.data.fields) > maxPooledFields {
		return
	}

	e.w = nil
	e.enc = nil
	e.ctx = nil
	e.doneFn = nil

	eventPool.Put(e)
}

// Enabled reports whether the event will be written.
func (e *Event) Enabled() bool {
	return e != nil && e.data.level != Disabled
}

// Discard drops the event without writing it and returns nil.
//
// The event goes back to the pool, unlike in v0 where discarding leaked it.
func (e *Event) Discard() *Event {
	if e == nil {
		return nil
	}

	putEvent(e)

	return nil
}

// Ctx attaches a context to the event, for consumption by context extractors.
func (e *Event) Ctx(ctx context.Context) *Event {
	if e == nil {
		return e
	}

	e.ctx = ctx

	return e
}

// GetCtx returns the context attached to the event, or context.Background.
func (e *Event) GetCtx() context.Context {
	if e == nil || e.ctx == nil {
		return context.Background()
	}

	return e.ctx
}

// Timestamp overrides the event's time.
//
// Setting the zero time suppresses the timestamp entirely, which is what an
// slog.Record with no time requires.
func (e *Event) Timestamp(t time.Time) *Event {
	if e == nil {
		return e
	}

	e.data.ts = t

	return e
}

// Field appends a pre-built field.
func (e *Event) Field(f Field) *Event {
	if e == nil {
		return e
	}

	e.data.fields = append(e.data.fields, f)

	return e
}

// Fields appends several pre-built fields.
func (e *Event) Fields(fields ...Field) *Event {
	if e == nil {
		return e
	}

	e.data.fields = append(e.data.fields, fields...)

	return e
}

// Str adds a string field.
func (e *Event) Str(key, val string) *Event { return e.Field(String(key, val)) }

// Int adds an int field.
func (e *Event) Int(key string, val int) *Event { return e.Field(Int(key, val)) }

// Int64 adds an int64 field.
func (e *Event) Int64(key string, val int64) *Event { return e.Field(Int64(key, val)) }

// Uint64 adds a uint64 field.
func (e *Event) Uint64(key string, val uint64) *Event { return e.Field(Uint64(key, val)) }

// Float64 adds a float64 field.
func (e *Event) Float64(key string, val float64) *Event { return e.Field(Float64(key, val)) }

// Bool adds a bool field.
func (e *Event) Bool(key string, val bool) *Event { return e.Field(Bool(key, val)) }

// Dur adds a duration field.
func (e *Event) Dur(key string, val time.Duration) *Event { return e.Field(Dur(key, val)) }

// Time adds a time field.
func (e *Event) Time(key string, val time.Time) *Event { return e.Field(Time(key, val)) }

// Bytes adds a byte-slice field.
func (e *Event) Bytes(key string, val []byte) *Event { return e.Field(Bytes(key, val)) }

// Interface adds a field holding an arbitrary value.
func (e *Event) Interface(key string, val any) *Event { return e.Field(Any(key, val)) }

// Any adds a field holding an arbitrary value.
func (e *Event) Any(key string, val any) *Event { return e.Field(Any(key, val)) }

// IPAddr adds an IP address field.
func (e *Event) IPAddr(key string, ip net.IP) *Event { return e.Field(Stringer(key, ip)) }

// Err adds an error under the conventional error key.
//
// Unlike v0 this never displaces the message: an event may carry both, and both
// are rendered.
func (e *Event) Err(err error) *Event {
	if e == nil || err == nil {
		return e
	}

	return e.Field(Err(err))
}

// AnErr adds an error under an explicit key.
//
// The key is honored, which it was not in v0.
func (e *Event) AnErr(key string, err error) *Event {
	if e == nil || err == nil {
		return e
	}

	return e.Field(AnErr(key, err))
}

// Block appends a block.
func (e *Event) Block(block Block) *Event {
	if e == nil {
		return e
	}

	e.data.blocks = append(e.data.blocks, block)

	return e
}

// BlockText appends a plain-text block.
func (e *Event) BlockText(text string) *Event { return e.Block(Block{Text: text}) }

// Blocks appends several plain-text blocks.
func (e *Event) Blocks(texts ...string) *Event {
	if e == nil {
		return e
	}

	for _, t := range texts {
		e.data.blocks = append(e.data.blocks, Block{Text: t})
	}

	return e
}

// Msg writes the event with the given message.
//
// This is a terminal call: the event returns to the pool and must not be used
// afterwards.
func (e *Event) Msg(msg string) {
	if e == nil {
		return
	}

	e.data.message = append(e.data.message[:0], msg...)

	e.write(msg)
}

// Msgf writes the event with a formatted message.
//
// The message is formatted straight into the event's pooled buffer, so unlike a
// fmt.Sprintf-based implementation this costs no allocation.
func (e *Event) Msgf(format string, v ...any) {
	if e == nil {
		return
	}

	e.data.message = fmt.Appendf(e.data.message[:0], format, v...)

	e.write("")
}

// Send writes the event with an empty message.
func (e *Event) Send() {
	if e == nil {
		return
	}

	e.data.message = e.data.message[:0]

	e.write("")
}

// write encodes and emits the event. doneMsg is passed to the completion
// callback used by Fatal and Panic; it is empty for the formatted path, where
// the rendered message lives in the buffer instead.
func (e *Event) write(doneMsg string) {
	if e.doneFn != nil {
		defer e.doneFn(e.doneMessage(doneMsg))
	}

	if e.data.level == Disabled || e.w == nil || e.enc == nil {
		putEvent(e)

		return
	}

	e.buf = e.enc.AppendEvent(e.buf[:0], &e.data)

	if _, err := e.w.WriteLevel(e.data.level, e.buf); err != nil {
		reportWriteError(err)
	}

	putEvent(e)
}

// doneMessage returns the text handed to the completion callback. Panic needs
// the rendered message, so the formatted path materializes it here — off the hot
// path, and only when a callback is installed.
func (e *Event) doneMessage(msg string) string {
	if msg != "" || len(e.data.message) == 0 {
		return msg
	}

	return string(e.data.message)
}

func reportWriteError(err error) {
	if fn := errorHandler(); fn != nil {
		fn(err)

		return
	}

	fmt.Fprintf(os.Stderr, "reggol: could not write event: %v\n", err)
}
