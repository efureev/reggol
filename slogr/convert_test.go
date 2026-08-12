package slogr

import (
	"log/slog"
	"testing"
	"time"

	"github.com/efureev/reggol"
)

// TestConvertValueCoversEveryKind pins the Kind parity that makes the bridge a
// copy rather than a round trip through any.
func TestConvertValueCoversEveryKind(t *testing.T) {
	ts := time.Date(2026, 8, 12, 13, 45, 30, 0, time.UTC)

	for _, tc := range []struct {
		name string
		in   slog.Value
		want reggol.Kind
		text string
	}{
		{"string", slog.StringValue("s"), reggol.KindString, "s"},
		{"int64", slog.Int64Value(-5), reggol.KindInt64, "-5"},
		{"uint64", slog.Uint64Value(5), reggol.KindUint64, "5"},
		{"float64", slog.Float64Value(1.5), reggol.KindFloat64, "1.5"},
		{"bool", slog.BoolValue(true), reggol.KindBool, "true"},
		{"duration", slog.DurationValue(time.Minute), reggol.KindDuration, "1m0s"},
		{"time", slog.TimeValue(ts), reggol.KindTime, "2026-08-12T13:45:30Z"},
		{"any", slog.AnyValue(struct{ A int }{1}), reggol.KindAny, "{1}"},
		{
			"group",
			slog.GroupValue(slog.String("a", "1"), slog.Int("b", 2)),
			reggol.KindGroup,
			"[a=1 b=2]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := convertValue(tc.in)

			if got.Kind() != tc.want {
				t.Fatalf("Kind = %v, want %v", got.Kind(), tc.want)
			}

			if s := string(got.AppendTo(nil)); s != tc.text {
				t.Fatalf("AppendTo = %q, want %q", s, tc.text)
			}
		})
	}
}

// logValuer resolves to a nested group, exercising the LogValuer path.
type logValuer struct{}

func (logValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.String("resolved", "yes"))
}

func TestConvertValueResolvesLogValuer(t *testing.T) {
	got := convertValue(slog.AnyValue(logValuer{}))

	if got.Kind() != reggol.KindGroup {
		t.Fatalf("Kind = %v, want group", got.Kind())
	}

	if s := string(got.AppendTo(nil)); s != "[resolved=yes]" {
		t.Fatalf("AppendTo = %q", s)
	}
}

func TestQualify(t *testing.T) {
	for _, tc := range []struct {
		groups []string
		key    string
		want   string
	}{
		{nil, "k", "k"},
		{[]string{"a"}, "k", "a.k"},
		{[]string{"a", "b"}, "k", "a.b.k"},
		{[]string{"a"}, "", "a."},
	} {
		if got := qualify(tc.groups, tc.key); got != tc.want {
			t.Errorf("qualify(%v, %q) = %q, want %q", tc.groups, tc.key, got, tc.want)
		}
	}
}

func TestToSlogLevelCoversEveryLevel(t *testing.T) {
	for _, tc := range []struct {
		in   reggol.Level
		want slog.Level
	}{
		{reggol.TraceLevel, LevelTrace},
		{reggol.DebugLevel, slog.LevelDebug},
		{reggol.InfoLevel, slog.LevelInfo},
		{reggol.NoLevel, slog.LevelInfo},
		{reggol.WarnLevel, slog.LevelWarn},
		{reggol.ErrorLevel, slog.LevelError},
		{reggol.FatalLevel, LevelFatal},
		{reggol.PanicLevel, LevelPanic},
		{reggol.Disabled, LevelPanic + 1},
		{reggol.Level(-42), slog.LevelInfo},
	} {
		if got := ToSlogLevel(tc.in); got != tc.want {
			t.Errorf("ToSlogLevel(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestAppendAttrDropsEmptyAndInlines(t *testing.T) {
	t.Run("empty attr is dropped", func(t *testing.T) {
		if got := appendAttr(nil, nil, slog.Attr{}); len(got) != 0 {
			t.Fatalf("got %v, want none", got)
		}
	})

	t.Run("empty group is dropped", func(t *testing.T) {
		if got := appendAttr(nil, nil, slog.Group("g")); len(got) != 0 {
			t.Fatalf("got %v, want none", got)
		}
	})

	t.Run("group with empty key is inlined", func(t *testing.T) {
		got := appendAttr(nil, nil, slog.Attr{
			Key:   "",
			Value: slog.GroupValue(slog.String("a", "1")),
		})

		if len(got) != 1 || got[0].Key != "a" {
			t.Fatalf("got %v, want a single inlined key", got)
		}
	})

	t.Run("nested groups are dotted", func(t *testing.T) {
		got := appendAttr(nil, []string{"root"},
			slog.Group("g", slog.String("a", "1"), slog.Int("b", 2)))

		if len(got) != 2 {
			t.Fatalf("got %d fields, want 2", len(got))
		}

		if got[0].Key != "root.g.a" || got[1].Key != "root.g.b" {
			t.Fatalf("keys = %q, %q", got[0].Key, got[1].Key)
		}
	})
}

func TestHandlerEdgeCases(t *testing.T) {
	l := reggol.Nop()
	h := NewHandler(l)

	if got := h.WithAttrs(nil); got != h {
		t.Error("WithAttrs(nil) should return the same handler")
	}

	if got := h.WithAttrs([]slog.Attr{{}}); got != h {
		t.Error("WithAttrs of only-empty attrs should return the same handler")
	}

	if got := h.WithGroup(""); got != h {
		t.Error(`WithGroup("") should return the same handler`)
	}
}
