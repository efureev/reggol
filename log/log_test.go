package log_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/efureev/reggol"
	"github.com/efureev/reggol/log"
)

// capture redirects the package logger into a buffer for the duration of a test.
func capture(t *testing.T) *bytes.Buffer {
	t.Helper()

	prev := log.L()
	t.Cleanup(func() { log.SetLogger(prev) })

	prevGlobal := reggol.GlobalLevel()
	t.Cleanup(func() { reggol.SetGlobalLevel(prevGlobal) })
	reggol.SetGlobalLevel(reggol.TraceLevel)

	buf := &bytes.Buffer{}
	log.SetLogger(reggol.New(buf,
		reggol.WithEncoder(reggol.NewTextEncoder(reggol.WithoutTimestamp())),
		reggol.WithLevel(reggol.TraceLevel),
	))

	return buf
}

func TestEveryLevelWrapper(t *testing.T) {
	for _, tc := range []struct {
		name  string
		emit  func()
		level string
	}{
		{"trace", func() { log.Trace().Msg("m") }, "trace"},
		{"debug", func() { log.Debug().Msg("m") }, "debug"},
		{"info", func() { log.Info().Msg("m") }, "info"},
		{"warn", func() { log.Warn().Msg("m") }, "warn"},
		{"error", func() { log.Error().Msg("m") }, "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := capture(t)
			tc.emit()

			got := buf.String()
			if !strings.Contains(got, "level="+tc.level) {
				t.Fatalf("expected level=%s in %q", tc.level, got)
			}

			if !strings.Contains(got, "message=m") {
				t.Fatalf("missing message in %q", got)
			}
		})
	}
}

func TestLogAndWithLevel(t *testing.T) {
	buf := capture(t)

	log.Log().Msg("no level")

	if strings.Contains(buf.String(), "level=") {
		t.Fatalf("Log should not emit a level: %q", buf.String())
	}

	buf.Reset()
	log.WithLevel(reggol.WarnLevel).Msg("m")

	if !strings.Contains(buf.String(), "level=warn") {
		t.Fatalf("WithLevel ignored: %q", buf.String())
	}
}

func TestErrWrapper(t *testing.T) {
	buf := capture(t)

	log.Err(errors.New("boom")).Msg("context")

	got := buf.String()
	if !strings.Contains(got, "level=error") || !strings.Contains(got, "error=boom") {
		t.Fatalf("unexpected output: %q", got)
	}

	if !strings.Contains(got, "message=context") {
		t.Fatalf("message dropped alongside the error: %q", got)
	}

	buf.Reset()
	log.Err(nil).Msg("fine")

	if !strings.Contains(buf.String(), "level=info") {
		t.Fatalf("nil error should log at info: %q", buf.String())
	}
}

func TestPrintWrappers(t *testing.T) {
	buf := capture(t)

	log.Print("a", "b")
	log.Printf("n=%d", 7)
	log.Println("line")

	got := buf.String()
	for _, want := range []string{"ab", "n=7", "line"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestWriteWrapper(t *testing.T) {
	buf := capture(t)

	n, err := log.Write([]byte("payload\n"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if n != len("payload\n") {
		t.Fatalf("n = %d, want %d", n, len("payload\n"))
	}

	if !strings.Contains(buf.String(), "payload") {
		t.Fatalf("missing payload: %q", buf.String())
	}
}

func TestLevelAndGetLevel(t *testing.T) {
	buf := capture(t)

	if got := log.GetLevel(); got != reggol.TraceLevel {
		t.Fatalf("GetLevel = %v, want trace", got)
	}

	quiet := log.Level(reggol.ErrorLevel)
	quiet.Info().Msg("dropped")

	if buf.Len() != 0 {
		t.Fatalf("child level ignored: %q", buf.String())
	}

	quiet.Error().Msg("kept")

	if !strings.Contains(buf.String(), "kept") {
		t.Fatalf("error should pass: %q", buf.String())
	}
}

func TestWithWrapper(t *testing.T) {
	buf := capture(t)

	log.With().Str("service", "auth").Logger().Info().Msg("m")

	if !strings.Contains(buf.String(), "service=auth") {
		t.Fatalf("bound field missing: %q", buf.String())
	}
}

func TestContextWrappers(t *testing.T) {
	buf := capture(t)

	ctx := log.WithContext(context.Background())

	got, ok := reggol.FromContext(ctx)
	if !ok {
		t.Fatal("logger not stored in context")
	}

	got.Info().Msg("from ctx")

	if !strings.Contains(buf.String(), "from ctx") {
		t.Fatalf("context logger writes elsewhere: %q", buf.String())
	}

	buf.Reset()
	log.Ctx(context.Background(), reggol.WarnLevel).Msg("ctx level")

	if !strings.Contains(buf.String(), "level=warn") {
		t.Fatalf("Ctx ignored the level: %q", buf.String())
	}
}

func TestOutputWrapper(t *testing.T) {
	capture(t)

	var other bytes.Buffer

	log.Output(&other).Info().Msg("redirected")

	if !strings.Contains(other.String(), "redirected") {
		t.Fatalf("Output did not redirect: %q", other.String())
	}
}

// TestSetLoggerIsRaceFree covers the v0 defect where the package logger was an
// exported variable, so replacing it raced with every read.
func TestSetLoggerIsRaceFree(t *testing.T) {
	prev := log.L()
	t.Cleanup(func() { log.SetLogger(prev) })

	quiet := reggol.Nop()

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for range 500 {
			log.SetLogger(quiet)
		}
	}()

	go func() {
		defer wg.Done()

		for range 500 {
			log.Info().Msg("concurrent")
		}
	}()

	wg.Wait()
}

// TestDefaultLoggerIsSynchronised documents that the out-of-the-box facade is
// safe to use from several goroutines.
func TestDefaultLoggerIsSynchronised(t *testing.T) {
	prev := log.L()
	t.Cleanup(func() { log.SetLogger(prev) })

	var buf bytes.Buffer

	log.SetLogger(reggol.New(reggol.SyncWriter(&buf),
		reggol.WithEncoder(reggol.NewTextEncoder(reggol.WithoutTimestamp())),
		reggol.WithLevel(reggol.TraceLevel),
	))

	var wg sync.WaitGroup

	wg.Add(20)

	for range 20 {
		go func() {
			defer wg.Done()

			for range 50 {
				log.Info().Msg("x")
			}
		}()
	}

	wg.Wait()

	if got := strings.Count(buf.String(), "\n"); got != 1000 {
		t.Fatalf("lines = %d, want 1000", got)
	}
}
