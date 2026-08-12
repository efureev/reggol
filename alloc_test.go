//go:build !race

package reggol

import (
	"io"
	"testing"
	"time"
)

// Allocation gates.
//
// These are deterministic where the wall-clock benchmarks are not, which is why
// they — and not ns/op — are the blocking CI signal. The file is excluded under
// -race because the detector allocates on its own and makes the counts
// meaningless.
func TestZeroAllocations(t *testing.T) {
	withGlobalLevel(t, TraceLevel)

	cases := []struct {
		name string
		emit func(Logger)
		enc  Encoder
	}{
		{
			name: "empty",
			enc:  NewTextEncoder(),
			emit: func(l Logger) { l.Log().Msg("") },
		},
		{
			name: "message only",
			enc:  NewTextEncoder(),
			emit: func(l Logger) { l.Info().Msg("a realistic message") },
		},
		{
			name: "scalar fields",
			enc:  NewTextEncoder(),
			emit: func(l Logger) {
				l.Info().Str("s", "v").Int("i", 123).Bool("b", true).Msg("m")
			},
		},
		{
			name: "zero time field",
			enc:  NewTextEncoder(),
			emit: func(l Logger) { l.Info().Time("t", time.Time{}).Msg("m") },
		},
		{
			name: "current time field",
			enc:  NewTextEncoder(),
			emit: func(l Logger) { l.Info().Time("t", time.Now()).Msg("m") },
		},
		{
			name: "duration and float",
			enc:  NewTextEncoder(),
			emit: func(l Logger) {
				l.Info().Dur("d", time.Second).Float64("f", 3.14).Msg("m")
			},
		},
		{
			name: "error field",
			enc:  NewTextEncoder(),
			emit: func(l Logger) { l.Error().Err(errExample).Msg("m") },
		},
		{
			name: "blocks",
			enc:  NewTextEncoder(),
			emit: func(l Logger) { l.Info().Blocks("API", "GET").Msg("m") },
		},
		{
			name: "console",
			enc:  NewConsoleEncoder(WithColorMode(ColorNever, nil)),
			emit: func(l Logger) { l.Info().Str("s", "v").Int("i", 1).Msg("m") },
		},
		{
			name: "console with color",
			enc:  NewConsoleEncoder(WithColorMode(ColorAlways, nil)),
			emit: func(l Logger) { l.Info().Str("s", "v").Msg("m") },
		},
		{
			name: "json",
			enc:  NewJSONEncoder(),
			emit: func(l Logger) { l.Info().Str("s", "v").Int("i", 1).Msg("m") },
		},
		{
			name: "json escaping",
			enc:  NewJSONEncoder(),
			emit: func(l Logger) { l.Info().Str("s", "quote\" tab\t nl\n").Msg("m") },
		},
		{
			name: "disabled",
			enc:  NewTextEncoder(),
			emit: func(l Logger) { l.Trace().Str("s", "v").Msg("m") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := New(io.Discard, WithEncoder(tc.enc), WithLevel(DebugLevel))

			if got := testing.AllocsPerRun(200, func() { tc.emit(l) }); got != 0 {
				t.Fatalf("allocations = %v, want 0", got)
			}
		})
	}
}

// TestChildLoggerZeroAllocations covers the pre-encoded prefix: emitting through
// a child must not re-encode its bound fields.
func TestChildLoggerZeroAllocations(t *testing.T) {
	child := New(io.Discard, WithEncoder(NewTextEncoder()), WithLevel(TraceLevel)).
		With().
		Str("service", "auth").
		Str("region", "eu-central-1").
		Int("shard", 7).
		Logger()

	if got := testing.AllocsPerRun(200, func() {
		child.Info().Str("req", "x1").Msg("m")
	}); got != 0 {
		t.Fatalf("allocations = %v, want 0", got)
	}
}

// TestPoolCeiling checks that an oversized event is dropped rather than
// returned to the pool, so one huge record cannot inflate every buffer.
func TestPoolCeiling(t *testing.T) {
	l := New(io.Discard, WithEncoder(NewTextEncoder()), WithLevel(TraceLevel))

	huge := make([]byte, maxPooledBuf+1)
	for i := range huge {
		huge[i] = 'x'
	}

	l.Info().Str("big", string(huge)).Msg("m")

	// A normal event afterwards must still be allocation-free, which it cannot
	// be if the oversized buffer came back into circulation and kept growing.
	if got := testing.AllocsPerRun(200, func() {
		l.Info().Str("s", "v").Msg("m")
	}); got != 0 {
		t.Fatalf("allocations after an oversized event = %v, want 0", got)
	}
}
