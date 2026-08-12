package slogr

import (
	"io"
	"log/slog"

	"github.com/efureev/reggol"
)

// FromHandler returns a reggol.Logger that feeds an existing slog.Handler.
//
// This is the reverse of NewHandler: use it when the surrounding application
// already owns a slog pipeline and you want reggol's chained API in front of
// it. Records are handed over structurally — no text is produced on the way.
//
// Two level thresholds apply, and a record has to clear both: reggol's own —
// the returned logger accepts everything, but the process-wide
// reggol.GlobalLevel still filters, and it defaults to InfoLevel — and then the
// handler's. Debug records therefore stay invisible until
// reggol.SetGlobalLevel(reggol.DebugLevel) is called, no matter how the handler
// is configured.
//
// Allocations: unlike the rest of reggol this path is not free. slog.Record
// copies attributes, and the message has to become a string, so expect a small
// constant cost per record. That is inherent to slog.Handler's contract, not an
// oversight.
func FromHandler(h slog.Handler, opts ...reggol.Option) reggol.Logger {
	if h == nil {
		return reggol.Nop()
	}

	opts = append([]reggol.Option{
		reggol.WithEncoder(&forwarder{h: h}),
		reggol.WithLevel(reggol.TraceLevel),
	}, opts...)

	// The writer is never reached: the forwarder consumes the event and emits
	// no bytes.
	return reggol.New(io.Discard, opts...)
}

// forwarder is an Encoder that hands events to a slog.Handler instead of
// rendering them.
//
// The encoder is the only place with structured access to a finished event, so
// it is the natural interception point. It deliberately does not implement
// PrefixEncoder: without a pre-encoded prefix, reggol keeps a child logger's
// bound fields in structural form, which is exactly what the handler needs.
type forwarder struct {
	h slog.Handler
}

// AppendEvent implements reggol.Encoder. It returns dst untouched — there is no
// text to emit.
func (f *forwarder) AppendEvent(dst []byte, d *reggol.EventData) []byte {
	lvl := ToSlogLevel(d.Level())

	ctx := d.Context()
	if !f.h.Enabled(ctx, lvl) {
		return dst
	}

	rec := slog.NewRecord(d.Time(), lvl, string(d.MessageBytes()), 0)

	if blocks := d.Blocks(); len(blocks) > 0 {
		texts := make([]string, len(blocks))
		for i := range blocks {
			texts[i] = blocks[i].Value()
		}

		rec.AddAttrs(slog.Any(reggol.BlocksFieldName, texts))
	}

	for _, field := range d.Fields() {
		rec.AddAttrs(slog.Attr{Key: field.Key, Value: toSlogValue(field.Val)})
	}

	// The handler owns its errors; reggol has no channel to surface them from
	// inside an encoder.
	_ = f.h.Handle(ctx, rec)

	return dst
}

// toSlogValue maps a reggol.Value onto a slog.Value.
//
// The Kind sets mirror each other, so this is a copy rather than a round trip
// through any.
//
//nolint:exhaustive // KindAny and unknown kinds share the default branch
func toSlogValue(v reggol.Value) slog.Value {
	switch v.Kind() {
	case reggol.KindString:
		return slog.StringValue(v.String())
	case reggol.KindInt64:
		return slog.Int64Value(v.Int64())
	case reggol.KindUint64:
		return slog.Uint64Value(v.Uint64())
	case reggol.KindFloat64:
		return slog.Float64Value(v.Float64())
	case reggol.KindBool:
		return slog.BoolValue(v.Bool())
	case reggol.KindDuration:
		return slog.DurationValue(v.Duration())
	case reggol.KindTime:
		return slog.TimeValue(v.Time())
	case reggol.KindBytes:
		return slog.StringValue(string(v.Bytes()))
	case reggol.KindError:
		if err := v.Err(); err != nil {
			return slog.StringValue(err.Error())
		}

		return slog.AnyValue(nil)
	case reggol.KindGroup:
		members := v.Group()

		attrs := make([]slog.Attr, 0, len(members))
		for _, m := range members {
			attrs = append(attrs, slog.Attr{Key: m.Key, Value: toSlogValue(m.Val)})
		}

		return slog.GroupValue(attrs...)
	default:
		return slog.AnyValue(v.Any())
	}
}
