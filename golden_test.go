package reggol

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// update rewrites the golden files instead of comparing against them.
//
//	go test -run TestEncoderGolden -update
//
// The inline tables in encoder_test.go pin a handful of cases precisely and
// read as documentation. This file does the opposite job: it sweeps a wide
// matrix so that an unintended change anywhere in the output shows up as a
// reviewable diff, and one flag regenerates every expectation when a format
// change is deliberate.
var update = flag.Bool("update", false, "rewrite testdata golden files")

// goldenTime keeps the matrix deterministic.
var goldenTime = time.Date(2026, 8, 12, 13, 45, 30, 123456789, time.UTC)

// goldenCase is one row of the matrix.
type goldenCase struct {
	name  string
	build func(*EventData)
}

// goldenCases sweeps level, timestamp, blocks, message, error, field count and
// field kind. Each encoder renders all of them into one golden file.
func goldenCases() []goldenCase {
	err := errors.New("connection refused")

	cases := []goldenCase{
		{"empty", func(d *EventData) {}},
		{"message only", func(d *EventData) { d.message = []byte("hello world") }},
		{"no level", func(d *EventData) { d.level = NoLevel; d.message = []byte("plain") }},
		{"zero time", func(d *EventData) { d.ts = time.Time{}; d.message = []byte("no timestamp") }},
		{"one field", func(d *EventData) {
			d.message = []byte("m")
			d.fields = []Field{String("k", "v")}
		}},
		{"three fields sorted", func(d *EventData) {
			d.message = []byte("m")
			d.fields = []Field{Int("z", 1), String("a", "x"), Bool("m", true)}
		}},
		{"duplicate keys", func(d *EventData) {
			d.message = []byte("m")
			d.fields = []Field{String("k", "first"), String("k", "second")}
		}},
		{"blocks", func(d *EventData) {
			d.message = []byte("ok")
			d.blocks = Blocks{{Text: "API"}, {Text: "GET /users"}}
		}},
		{"decorated block", func(d *EventData) {
			d.message = []byte("verified")
			d.blocks = Blocks{NewBlock("auth", func(s string) string { return "[" + s + "]" })}
		}},
		{"error with message", func(d *EventData) {
			d.level = ErrorLevel
			d.message = []byte("query failed")
			d.fields = []Field{Err(err), String("host", "db-1")}
		}},
		{"error under a custom key", func(d *EventData) {
			d.level = ErrorLevel
			d.fields = []Field{AnErr("cause", err)}
		}},
		{"nil error", func(d *EventData) { d.fields = []Field{AnErr("e", nil)} }},
		{"bound prefix", func(d *EventData) {
			d.message = []byte("handled")
			d.fields = []Field{String("req", "x1")}
		}},
		{"group", func(d *EventData) {
			d.fields = []Field{Group("g", String("a", "1"), Int("b", 2))}
		}},
		{"every scalar kind", func(d *EventData) {
			d.message = []byte("kinds")
			d.fields = []Field{
				String("str", "s"),
				Int("int", -7),
				Uint64("uint", 42),
				Float64("float", 1.5),
				Bool("bool", true),
				Dur("dur", 90*time.Second),
				Time("time", goldenTime),
			}
		}},
		{"escaping", func(d *EventData) {
			d.message = []byte("quote\" backslash\\ tab\t")
			d.fields = []Field{String("nl", "a\nb"), String("ctrl", "\x00\x1f")}
		}},
		{"unicode", func(d *EventData) {
			d.message = []byte("emoji 🙂 combining é")
			d.fields = []Field{String("ключ", "значение")}
		}},
		{"long message", func(d *EventData) {
			d.message = []byte(strings.Repeat("long ", 12))
		}},
		{"caller", func(d *EventData) {
			d.message = []byte("with a call site")
			d.pc = goldenPC
			d.fields = []Field{String("k", "v")}
		}},
		{"caller and blocks", func(d *EventData) {
			d.message = []byte("m")
			d.pc = goldenPC
			d.blocks = Blocks{{Text: "API"}}
		}},
	}

	// Sweep every level over a fixed body.
	for _, lvl := range []Level{TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel, PanicLevel} {
		cases = append(cases, goldenCase{
			name: "level " + lvl.String(),
			build: func(d *EventData) {
				d.level = lvl
				d.message = []byte("leveled")
				d.fields = []Field{Int("n", 1)}
			},
		})
	}

	return cases
}

// goldenEncoders are the encoder configurations covered by the matrix.
func goldenEncoders() []struct {
	name string
	enc  Encoder
} {
	return []struct {
		name string
		enc  Encoder
	}{
		{"console", NewConsoleEncoder(
			WithColorMode(ColorNever, nil),
			WithConsoleOptions(WithTimeFormat(time.RFC3339)),
		)},
		{"console-color", NewConsoleEncoder(
			WithColorMode(ColorAlways, nil),
			WithConsoleOptions(WithTimeFormat(time.RFC3339)),
		)},
		{"console-nosort", NewConsoleEncoder(
			WithColorMode(ColorNever, nil),
			WithConsoleOptions(WithTimeFormat(time.RFC3339), WithoutSort()),
		)},
		{"text", NewTextEncoder(WithTimeFormat(time.RFC3339))},
		{"text-bare", NewTextEncoder(WithoutTimestamp(), WithoutLevel())},
		{"json", NewJSONEncoder(WithTimeFormat(time.RFC3339))},
	}
}

// TestEncoderGolden renders the whole matrix through every encoder and compares
// it with testdata. Run with -update to regenerate.
func TestEncoderGolden(t *testing.T) {
	for _, e := range goldenEncoders() {
		t.Run(e.name, func(t *testing.T) {
			var b strings.Builder

			for _, tc := range goldenCases() {
				d := &EventData{ts: goldenTime, level: InfoLevel}
				tc.build(d)

				// One case exercises the pre-encoded prefix a child logger
				// carries; the rest leave it empty.
				if tc.name == "bound prefix" {
					if pe, ok := e.enc.(PrefixEncoder); ok {
						d.prefix = pe.AppendPrefix(nil, []Field{String("service", "auth"), Int("shard", 7)})
					}
				}

				fmt.Fprintf(&b, "%-26s | %s", tc.name, e.enc.AppendEvent(nil, d))
			}

			compareGolden(t, e.name, b.String())
		})
	}
}

// compareGolden checks got against testdata/<name>.golden, or rewrites it.
func compareGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name+".golden")

	if *update {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("create testdata: %v", err)
		}

		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}

		t.Logf("updated %s", path)

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v\nrun `go test -run TestEncoderGolden -update` to create it", path, err)
	}

	if got == string(want) {
		return
	}

	t.Errorf("output differs from %s\nrun `go test -run TestEncoderGolden -update` if the change is intended\n%s",
		path, diffLines(string(want), got))
}

// diffLines renders the first differing lines, which is enough to see what moved.
func diffLines(want, got string) string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")

	var b strings.Builder

	const maxReported = 10

	shown := 0

	for i := range max(len(w), len(g)) {
		var wl, gl string

		if i < len(w) {
			wl = w[i]
		}

		if i < len(g) {
			gl = g[i]
		}

		if wl == gl {
			continue
		}

		fmt.Fprintf(&b, "\nline %d:\n  want: %q\n   got: %q", i+1, wl, gl)

		if shown++; shown >= maxReported {
			b.WriteString("\n  …")

			break
		}
	}

	return b.String()
}
