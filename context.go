package reggol

import (
	"context"
	"fmt"
	"time"
)

// PrefixEncoder is an Encoder that can pre-encode a child logger's fields.
//
// Implementing it is optional. When an encoder does, a child logger's bound
// fields are rendered once, at With().Logger() time, and every later event pays
// only a memmove of the resulting bytes. When it does not, the fields are
// carried structurally and re-encoded per event — correct, just slower.
type PrefixEncoder interface {
	Encoder
	AppendPrefix(dst []byte, fields []Field) []byte
}

// Context accumulates fields to bind to a child logger.
type Context struct {
	l      Logger
	fields []Field
}

// With starts building a child logger with bound fields.
func (l Logger) With() Context {
	return Context{l: l}
}

// Fields appends pre-built fields.
func (c Context) Fields(fields ...Field) Context {
	c.fields = append(c.fields, fields...)

	return c
}

// Field appends a pre-built field.
func (c Context) Field(f Field) Context {
	c.fields = append(c.fields, f)

	return c
}

// Str binds a string field.
func (c Context) Str(key, val string) Context { return c.Field(String(key, val)) }

// Int binds an int field.
func (c Context) Int(key string, val int) Context { return c.Field(Int(key, val)) }

// Int64 binds an int64 field.
func (c Context) Int64(key string, val int64) Context { return c.Field(Int64(key, val)) }

// Uint64 binds a uint64 field.
func (c Context) Uint64(key string, val uint64) Context { return c.Field(Uint64(key, val)) }

// Float64 binds a float64 field.
func (c Context) Float64(key string, val float64) Context { return c.Field(Float64(key, val)) }

// Bool binds a bool field.
func (c Context) Bool(key string, val bool) Context { return c.Field(Bool(key, val)) }

// Dur binds a duration field.
func (c Context) Dur(key string, val time.Duration) Context { return c.Field(Dur(key, val)) }

// Time binds a time field.
func (c Context) Time(key string, val time.Time) Context { return c.Field(Time(key, val)) }

// Bytes binds a byte-slice field.
func (c Context) Bytes(key string, val []byte) Context { return c.Field(Bytes(key, val)) }

// Err binds an error under the conventional error key.
func (c Context) Err(err error) Context { return c.Field(Err(err)) }

// AnErr binds an error under an explicit key.
func (c Context) AnErr(key string, err error) Context { return c.Field(AnErr(key, err)) }

// Stringer binds a fmt.Stringer field.
func (c Context) Stringer(key string, val fmt.Stringer) Context {
	return c.Field(Stringer(key, val))
}

// Any binds a field holding an arbitrary value.
func (c Context) Any(key string, val any) Context { return c.Field(Any(key, val)) }

// Interface binds a field holding an arbitrary value.
func (c Context) Interface(key string, val any) Context { return c.Field(Any(key, val)) }

// Logger returns the child logger.
//
// Bound fields are sorted once here, then pre-encoded when the encoder supports
// it. Ordering is deliberate and documented: bound fields precede event fields,
// and sorting applies within each group rather than across them — a global sort
// would require decoding the prefix on every line, defeating the point.
func (c Context) Logger() Logger {
	l := c.l

	if len(c.fields) == 0 {
		return l
	}

	// Copy the parent's bound fields so that two children of one parent cannot
	// write into a shared backing array.
	fields := make([]Field, 0, len(l.ctxFields)+len(c.fields))
	fields = append(fields, l.ctxFields...)
	fields = append(fields, c.fields...)

	sortFields(fields)

	l.ctxFields = fields
	l.ctx = nil

	if pe, ok := l.enc.(PrefixEncoder); ok {
		l.ctx = pe.AppendPrefix(nil, fields)
	}

	return l
}

// contextKey is the private key under which a Logger is stored in a
// context.Context.
type contextKey struct{}

// WithContext returns a copy of ctx carrying the logger.
func (l Logger) WithContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the logger stored in ctx.
//
// The second result reports whether one was present; when it is false the
// returned logger is a no-op, so callers may ignore it.
func FromContext(ctx context.Context) (Logger, bool) {
	if ctx == nil {
		return Nop(), false
	}

	l, ok := ctx.Value(contextKey{}).(Logger)
	if !ok {
		return Nop(), false
	}

	return l, true
}
