package reggol

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

//nolint:gochecknoglobals
var eventPool = &sync.Pool{
	New: func() interface{} {
		return &Event{
			data: EventData{
				fields: make(Fields),
			},
		}
	},
}

type EventData struct {
	level   Level
	ts      time.Time
	fields  Fields
	blocks  Blocks
	err     error
	message string
}

type Event struct {
	w      LevelWriter
	data   EventData
	doneFn func(msg string)
	// ctx    context.Context // Optional Go context for event
}

func newEvent(w LevelWriter, level Level) *Event {
	//nolint:forcetypeassert
	e := eventPool.Get().(*Event)
	e.w = w
	e.data.ts = time.Now()
	e.data.level = level
	e.data.err = nil
	e.data.blocks = nil
	e.data.fields = make(Fields)
	// clear(e.data.fields) // # > 1.20
	// e.data = newEventData(level)

	return e
}

func newEventData(level Level) EventData {
	return EventData{
		level:  level,
		ts:     time.Now(),
		fields: make(Fields),
	}
}

func (e *Event) write() (err error) {
	if e == nil {
		return nil
	}

	if e.data.level != Disabled {
		if e.w != nil {
			_, err = e.w.WriteLevel(e.data)
		}
	}

	putEvent(e)

	return
}

func (e *Event) Enabled() bool {
	return e != nil && e.data.level != Disabled
}

func (e *Event) Discard() *Event {
	if e == nil {
		return e
	}

	e.data.level = Disabled

	return nil
}

func (e *Event) Block(block Block) *Event {
	if e == nil {
		return e
	}

	e.data.blocks.AddBlock(block)

	return e
}

func (e *Event) BlockText(msg string) *Event {
	if e == nil {
		return e
	}

	e.data.blocks.Add(msg)

	return e
}

func (e *Event) Blocks(msgs ...string) *Event {
	if e == nil {
		return e
	}

	for _, msg := range msgs {
		e.data.blocks.Add(msg)
	}

	return e
}

func (e *Event) Msg(msg string) {
	if e == nil {
		return
	}

	e.msg(msg)
}

func (e *Event) Msgf(format string, v ...interface{}) {
	if e == nil {
		return
	}

	e.msg(fmt.Sprintf(format, v...))
}

func (e *Event) msg(msg string) {
	e.data.message = msg

	if e.doneFn != nil {
		defer e.doneFn(msg)
	}

	if err := e.write(); err != nil {
		if ErrorHandler != nil {
			ErrorHandler(err)
		} else {
			fmt.Fprintf(os.Stderr, "reggol: could not write event: %v\n", err)
		}
	}
}

func (e *Event) Push() {
	if e == nil {
		return
	}

	e.msg("")
}

//nolint:wsl
func putEvent(e *Event) {
	// Proper usage of a sync.Pool requires each entry to have approximately
	// the same memory cost. To obtain this property when the stored type
	// contains a variably-sized buffer, we add a hard limit on the maximum buffer
	// to place back in the pool.

	// todo
	// See https://golang.org/issue/23199
	// const maxSize = 1 << 16 // 64KiB
	// if cap(e.buf) > maxSize {
	//	return
	// }

	eventPool.Put(e)
}

func (e *Event) addField(key string, val any) *Event {
	if e == nil {
		return e
	}

	e.data.fields.Add(key, val)

	return e
}

// Str adds the field key with val as a string to the *Event context.
func (e *Event) Str(key, val string) *Event {
	return e.addField(key, val)
}

func (e *Event) Int(key string, val int) *Event {
	return e.addField(key, val)
}

func (e *Event) Bool(key string, val bool) *Event {
	return e.addField(key, val)
}

func (e *Event) Bytes(key string, val []byte) *Event {
	return e.addField(key, val)
}

func (e *Event) Time(key string, val time.Time) *Event {
	return e.addField(key, val)
}

func (e *Event) Err(err error) *Event {
	if e == nil {
		return e
	}

	return e.AnErr(ErrorFieldName, err)
}

func (e *Event) IPAddr(key string, ip net.IP) *Event {
	return e.addField(key, ip)
}

func (e *Event) AnErr(key string, err error) *Event {
	if e == nil {
		return e
	}

	switch m := ErrorMarshalFunc(err).(type) {
	case nil:
		return e
	case error:
		if m == nil || isNilValue(m) {
			return e
		} else {
			// todo
			e.data.err = m

			return e
		}
	case string:
		e.data.err = errors.New(m)

		return e
	default:
		return e.Interface(key, m)
	}
}

func (e *Event) Interface(key string, i interface{}) *Event {
	if e == nil {
		return e
	}

	// todo interface to JSON
	return e
}
