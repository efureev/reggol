// Package slogr bridges reggol and log/slog.
//
// It lives in its own package on purpose: the root package stays free of a
// log/slog import, and its public surface — every symbol of which is API under
// SemVer — does not grow.
package slogr

import (
	"context"
	"log/slog"

	"github.com/efureev/reggol"
)

// Handler adapts a reggol.Logger to slog.Handler.
type Handler struct {
	l      reggol.Logger
	groups []string
}

// NewHandler returns a slog.Handler writing through l.
func NewHandler(l reggol.Logger) *Handler {
	return &Handler{l: l}
}

// New returns a *slog.Logger writing through l.
func New(l reggol.Logger) *slog.Logger {
	return slog.New(NewHandler(l))
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(_ context.Context, lvl slog.Level) bool {
	return FromSlogLevel(lvl) >= h.l.GetLevel() && FromSlogLevel(lvl) >= reggol.GlobalLevel()
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Ctx rather than WithLevel: it attaches the context and runs any context
	// extractors the logger was built with.
	e := h.l.Ctx(ctx, FromSlogLevel(r.Level))
	if e == nil {
		return nil
	}

	// A zero Record.Time must not be emitted; reggol suppresses the timestamp
	// for the zero time, so this covers both cases in one assignment.
	// slog captured the call site already; capturing our own here would point
	// at this bridge. A zero PC clears whatever the logger may have recorded,
	// so no bogus position is printed.
	e.Timestamp(r.Time).CallerPC(r.PC)

	r.Attrs(func(a slog.Attr) bool {
		for _, f := range appendAttr(nil, h.groups, a) {
			e.Field(f)
		}

		return true
	})

	e.Msg(r.Message)

	return nil
}

// WithAttrs implements slog.Handler.
//
// Bound attributes go through reggol's own child-logger machinery, so they are
// pre-encoded once rather than re-rendered per record.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	var fields []reggol.Field
	for _, a := range attrs {
		fields = appendAttr(fields, h.groups, a)
	}

	if len(fields) == 0 {
		return h
	}

	return &Handler{l: h.l.With().Fields(fields...).Logger(), groups: h.groups}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	groups := make([]string, len(h.groups), len(h.groups)+1)
	copy(groups, h.groups)
	groups = append(groups, name)

	return &Handler{l: h.l, groups: groups}
}

// appendAttr flattens a slog.Attr into dst as one or more reggol fields.
//
// Groups are flattened into dotted keys rather than nested. That keeps the
// bridge honest for every encoder — nesting means nothing in a key=value console
// line — and it sidesteps the merge problem a nesting handler hits when
// WithAttrs and record attributes land in the same open group.
//
// The three rules slog requires are implemented here: an empty Attr is dropped,
// a group with an empty key is inlined into its parent, and a group that ends up
// with no members disappears entirely.
func appendAttr(dst []reggol.Field, groups []string, a slog.Attr) []reggol.Field {
	if a.Equal(slog.Attr{}) {
		return dst
	}

	a.Value = a.Value.Resolve()

	if a.Value.Kind() != slog.KindGroup {
		return append(dst, reggol.Field{
			Key: qualify(groups, a.Key),
			Val: convertValue(a.Value),
		})
	}

	members := a.Value.Group()
	if len(members) == 0 {
		return dst
	}

	inner := groups
	if a.Key != "" {
		inner = append(append(make([]string, 0, len(groups)+1), groups...), a.Key)
	}

	for _, m := range members {
		dst = appendAttr(dst, inner, m)
	}

	return dst
}

// groupValue turns nested slog attributes into a reggol group value.
func groupValue(attrs []slog.Attr) reggol.Value {
	fields := make([]reggol.Field, 0, len(attrs))

	for _, a := range attrs {
		fields = appendAttr(fields, nil, a)
	}

	return reggol.GroupValue(fields...)
}

// qualify prefixes a key with the open group names.
func qualify(groups []string, key string) string {
	if len(groups) == 0 {
		return key
	}

	n := len(key)
	for _, g := range groups {
		n += len(g) + 1
	}

	b := make([]byte, 0, n)
	for _, g := range groups {
		b = append(b, g...)
		b = append(b, '.')
	}

	return string(append(b, key...))
}

// convertValue maps a slog.Value onto a reggol.Value.
//
// The Kind sets are deliberate mirrors of each other, so this is a copy rather
// than a round trip through any.
func convertValue(v slog.Value) reggol.Value {
	switch v.Kind() {
	case slog.KindString:
		return reggol.StringValue(v.String())
	case slog.KindInt64:
		return reggol.Int64Value(v.Int64())
	case slog.KindUint64:
		return reggol.Uint64Value(v.Uint64())
	case slog.KindFloat64:
		return reggol.Float64Value(v.Float64())
	case slog.KindBool:
		return reggol.BoolValue(v.Bool())
	case slog.KindDuration:
		return reggol.DurationValue(v.Duration())
	case slog.KindTime:
		return reggol.TimeValue(v.Time())
	case slog.KindGroup:
		// appendAttr flattens groups before they reach here; this covers a
		// group value arriving on its own, e.g. nested inside an Any.
		return groupValue(v.Group())
	case slog.KindLogValuer:
		return convertValue(v.Resolve())
	case slog.KindAny:
		return reggol.AnyValue(v.Any())
	default:
		return reggol.AnyValue(v.Any())
	}
}

// slog level constants that reggol extends beyond.
const (
	// LevelTrace is the slog level corresponding to reggol.TraceLevel.
	LevelTrace = slog.LevelDebug - 4
	// LevelFatal is the slog level corresponding to reggol.FatalLevel.
	LevelFatal = slog.LevelError + 4
	// LevelPanic is the slog level corresponding to reggol.PanicLevel.
	LevelPanic = slog.LevelError + 8
)

// FromSlogLevel maps a slog.Level onto a reggol.Level.
func FromSlogLevel(l slog.Level) reggol.Level {
	switch {
	case l <= LevelTrace:
		return reggol.TraceLevel
	case l <= slog.LevelDebug:
		return reggol.DebugLevel
	case l <= slog.LevelInfo:
		return reggol.InfoLevel
	case l <= slog.LevelWarn:
		return reggol.WarnLevel
	case l <= slog.LevelError:
		return reggol.ErrorLevel
	case l <= LevelFatal:
		return reggol.FatalLevel
	default:
		return reggol.PanicLevel
	}
}

// ToSlogLevel maps a reggol.Level onto a slog.Level.
func ToSlogLevel(l reggol.Level) slog.Level {
	switch l {
	case reggol.TraceLevel:
		return LevelTrace
	case reggol.DebugLevel:
		return slog.LevelDebug
	case reggol.InfoLevel, reggol.NoLevel:
		return slog.LevelInfo
	case reggol.WarnLevel:
		return slog.LevelWarn
	case reggol.ErrorLevel:
		return slog.LevelError
	case reggol.FatalLevel:
		return LevelFatal
	case reggol.PanicLevel:
		return LevelPanic
	case reggol.Disabled:
		return LevelPanic + 1
	default:
		return slog.LevelInfo
	}
}
