package reggol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestLevelStringAndLabel(t *testing.T) {
	for _, tc := range []struct {
		lvl   Level
		name  string
		label string
	}{
		{TraceLevel, "trace", "TRC"},
		{DebugLevel, "debug", "DBG"},
		{InfoLevel, "info", "INF"},
		{WarnLevel, "warn", "WRN"},
		{ErrorLevel, "error", "ERR"},
		{FatalLevel, "fatal", "FTL"},
		{PanicLevel, "panic", "PNC"},
		{NoLevel, "", "???"},
		{Disabled, "disabled", "???"},
		{Level(-100), "-100", "???"},
		{Level(100), "100", "???"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.lvl.String(); got != tc.name {
				t.Errorf("String = %q, want %q", got, tc.name)
			}

			if got := tc.lvl.Label(); got != tc.label {
				t.Errorf("Label = %q, want %q", got, tc.label)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    Level
		wantErr error
	}{
		{"trace", TraceLevel, nil},
		{"TRACE", TraceLevel, nil},
		{"Debug", DebugLevel, nil},
		{"info", InfoLevel, nil},
		{"warn", WarnLevel, nil},
		{"error", ErrorLevel, nil},
		{"fatal", FatalLevel, nil},
		{"panic", PanicLevel, nil},
		{"disabled", Disabled, nil},
		{"", NoLevel, nil},
		{"-1", TraceLevel, nil},
		{"3", ErrorLevel, nil},
		{"-42", Level(-42), nil},
		{"nonsense", NoLevel, ErrUnknownLevel},
		{"999", NoLevel, ErrLevelOutOfRange},
		{"-999", NoLevel, ErrLevelOutOfRange},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseLevel(tc.in)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestParseLevelErrorsAreLowercase pins ST1005, which v0 violated.
func TestParseLevelErrorsAreLowercase(t *testing.T) {
	_, err := ParseLevel("nonsense")
	if err == nil {
		t.Fatal("expected an error")
	}

	if msg := err.Error(); msg == "" || strings.ToLower(msg[:1]) != msg[:1] {
		t.Fatalf("error string should start lowercase: %q", msg)
	}
}

func TestLevelTextMarshalling(t *testing.T) {
	for _, lvl := range []Level{TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel, PanicLevel} {
		b, err := lvl.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText(%v): %v", lvl, err)
		}

		var got Level
		if err := got.UnmarshalText(b); err != nil {
			t.Fatalf("UnmarshalText(%q): %v", b, err)
		}

		if got != lvl {
			t.Fatalf("round trip %v -> %q -> %v", lvl, b, got)
		}
	}
}

func TestLevelJSONRoundTrip(t *testing.T) {
	type config struct {
		Level Level `json:"level"`
	}

	var cfg config
	if err := json.Unmarshal([]byte(`{"level":"warn"}`), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cfg.Level != WarnLevel {
		t.Fatalf("level = %v, want warn", cfg.Level)
	}

	out, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(out) != `{"level":"warn"}` {
		t.Fatalf("marshaled to %s", out)
	}
}

func TestUnmarshalTextRejectsNilReceiver(t *testing.T) {
	var l *Level
	if err := l.UnmarshalText([]byte("info")); err == nil {
		t.Fatal("expected an error for a nil receiver")
	}

	var valid Level
	if err := valid.UnmarshalText([]byte("bogus")); err == nil {
		t.Fatal("expected an error for an unknown level")
	}
}

// TestPanicWritesThenPanics covers the terminal level that cannot be tested by
// simply letting it run.
func TestPanicWritesThenPanics(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp())), WithLevel(TraceLevel))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Panic() did not panic")
		}

		if r != "the sky is falling" {
			t.Fatalf("panic value = %v", r)
		}

		if !strings.Contains(buf.String(), "the sky is falling") {
			t.Fatalf("event was not written before panicking: %q", buf.String())
		}
	}()

	l.Panic().Msg("the sky is falling")
}

// TestPanicCarriesFormattedMessage covers the formatted path: the message lives
// in the event buffer rather than in a string, so the panic value has to be
// materialized from it.
func TestPanicCarriesFormattedMessage(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp())), WithLevel(TraceLevel))

	defer func() {
		r := recover()
		if r != "shard 7 is gone" {
			t.Fatalf("panic value = %v, want the formatted message", r)
		}

		if !strings.Contains(buf.String(), "shard 7 is gone") {
			t.Fatalf("event was not written: %q", buf.String())
		}
	}()

	l.Panic().Msgf("shard %d is gone", 7)
}

// TestFatalExits runs Fatal in a subprocess, the only way to observe os.Exit.
//
// It also pins a deliberate semantic: Fatal terminates even when the level
// filters the event out. Suppressing the record is a logging decision;
// suppressing the termination would turn a raised log level into a silent
// change of control flow.
func TestFatalExits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level string
		code  string
		want  int
	}{
		{"visible", "trace", "3", 3},
		{"filtered still exits", "disabled", "5", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestFatalExits") //nolint:gosec // re-executing the test binary
			cmd.Env = append(os.Environ(),
				"REGGOL_FATAL_SUBPROCESS=1",
				"REGGOL_FATAL_LEVEL="+tc.level,
				"REGGOL_FATAL_CODE="+tc.code,
			)

			err := cmd.Run()

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected the subprocess to exit non-zero, got %v", err)
			}

			if got := exitErr.ExitCode(); got != tc.want {
				t.Fatalf("exit code = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestMain hosts the Fatal subprocess.
func TestMain(m *testing.M) {
	if os.Getenv("REGGOL_FATAL_SUBPROCESS") != "1" {
		os.Exit(m.Run())
	}

	lvl, err := ParseLevel(os.Getenv("REGGOL_FATAL_LEVEL"))
	if err != nil {
		os.Exit(99)
	}

	code, err := strconv.Atoi(os.Getenv("REGGOL_FATAL_CODE"))
	if err != nil {
		os.Exit(98)
	}

	SetExitCode(code)

	New(io.Discard, WithEncoder(NewTextEncoder()), WithLevel(lvl)).
		Fatal().
		Msg("fatal from a subprocess")

	// Fatal must not return.
	os.Exit(97)
}

func TestMsgfFormats(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp(), WithoutLevel())), WithLevel(TraceLevel))
	l.Info().Msgf("n=%d s=%s", 42, "x")

	if got := strings.TrimSuffix(buf.String(), "\n"); got != "message=n=42 s=x" {
		t.Fatalf("got %q", got)
	}
}
