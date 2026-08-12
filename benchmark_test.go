package reggol

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

var (
	errExample  = errors.New("fail")
	fakeMessage = "Test logging, but use a somewhat realistic message length."
)

// benchLogger builds a logger over io.Discard with the given encoder.
func benchLogger(enc Encoder) Logger {
	return New(io.Discard, WithEncoder(enc), WithLevel(TraceLevel))
}

// textLogger matches the shape the v0 benchmarks measured: New(io.Discard) there
// resolved to the text transformer, so these numbers are the comparable ones.
func textLogger() Logger { return benchLogger(NewTextEncoder()) }

func consoleLogger() Logger {
	return benchLogger(NewConsoleEncoder(WithColorMode(ColorNever, nil)))
}

func BenchmarkLogEmpty(b *testing.B) {
	logger := textLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Log().Msg("")
		}
	})
}

func BenchmarkDisabled(b *testing.B) {
	logger := textLogger().Level(Disabled)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkInfo(b *testing.B) {
	logger := textLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkLogFields(b *testing.B) {
	logger := textLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().
				Str("string", "four!").
				Time("time", time.Time{}).
				Int("int", 123).
				Msg(fakeMessage)
		}
	})
}

func Benchmark10Fields(b *testing.B) {
	logger := textLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().
				Str("s1", "a").Str("s2", "b").Str("s3", "c").
				Int("i1", 1).Int("i2", 2).Int("i3", 3).
				Bool("b1", true).Bool("b2", false).
				Float64("f1", 3.14).
				Dur("d1", time.Second).
				Msg(fakeMessage)
		}
	})
}

func BenchmarkErrField(b *testing.B) {
	logger := textLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Error().Err(errExample).Msg(fakeMessage)
		}
	})
}

// BenchmarkConsoleInfo measures the encoder the facade actually uses. v0 never
// benchmarked this path at all.
func BenchmarkConsoleInfo(b *testing.B) {
	logger := consoleLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkConsoleFields(b *testing.B) {
	logger := consoleLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().
				Str("string", "four!").
				Time("time", time.Time{}).
				Int("int", 123).
				Msg(fakeMessage)
		}
	})
}

func BenchmarkConsoleColor(b *testing.B) {
	logger := benchLogger(NewConsoleEncoder(WithColorMode(ColorAlways, nil)))

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Str("k", "v").Msg(fakeMessage)
		}
	})
}

func BenchmarkJSONFields(b *testing.B) {
	logger := benchLogger(NewJSONEncoder())

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().
				Str("string", "four!").
				Time("time", time.Time{}).
				Int("int", 123).
				Msg(fakeMessage)
		}
	})
}

// BenchmarkChild measures the pre-encoded prefix: a child logger with bound
// fields should cost a memmove, not a re-encode.
func BenchmarkChild(b *testing.B) {
	logger := textLogger().With().
		Str("service", "auth").
		Str("region", "eu-central-1").
		Int("shard", 7).
		Logger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkSortDisabled(b *testing.B) {
	logger := benchLogger(NewTextEncoder(WithoutSort()))

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().
				Str("string", "four!").
				Time("time", time.Time{}).
				Int("int", 123).
				Msg(fakeMessage)
		}
	})
}

// BenchmarkStdlibSlogText is an external reference point, measured in the same
// process so the numbers are directly comparable.
func BenchmarkStdlibSlogText(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(fakeMessage,
				slog.String("string", "four!"),
				slog.Time("time", time.Time{}),
				slog.Int("int", 123),
			)
		}
	})
}
