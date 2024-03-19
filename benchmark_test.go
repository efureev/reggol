package reggol

import (
	"errors"
	"io"
	"testing"
	"time"
)

var (
	errExample  = errors.New("fail")
	fakeMessage = "Test logging, but use a somewhat realistic message length."
)

func BenchmarkLogEmpty(b *testing.B) {
	logger := New(io.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Log().Msg("")
		}
	})
}

func BenchmarkDisabled(b *testing.B) {
	logger := New(io.Discard).Level(Disabled)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkInfo(b *testing.B) {
	logger := New(io.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkLogFields(b *testing.B) {
	logger := New(io.Discard)
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
