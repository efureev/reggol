package reggol

import (
	"bytes"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestConcurrentWritesAreSerialised is the permanent guard for the v0 defect
// where logging from several goroutines into one writer was a data race with no
// way to fix it from the outside.
//
// Run it under -race; without SyncWriter the same shape reproduces the race.
func TestConcurrentWritesAreSerialised(t *testing.T) {
	const (
		goroutines = 50
		perRoutine = 200
	)

	var buf bytes.Buffer

	l := New(
		SyncWriter(&buf),
		WithEncoder(NewTextEncoder(WithoutTimestamp())),
		WithLevel(TraceLevel),
	)

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for g := range goroutines {
		go func() {
			defer wg.Done()

			for i := range perRoutine {
				l.Info().Int("g", g).Int("i", i).Msg("concurrent")
			}
		}()
	}

	wg.Wait()

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != goroutines*perRoutine {
		t.Fatalf("lines=%d, want %d", len(lines), goroutines*perRoutine)
	}

	// Every line must be intact: interleaving would corrupt the structure.
	for i, line := range lines {
		if !strings.HasPrefix(line, "level=info, message=concurrent, g=") {
			t.Fatalf("line %d is torn: %q", i, line)
		}
	}
}

// TestConcurrentChildLoggers exercises the pre-encoded prefix from several
// goroutines: a shared backing array would surface here.
func TestConcurrentChildLoggers(t *testing.T) {
	var buf bytes.Buffer

	root := New(
		SyncWriter(&buf),
		WithEncoder(NewTextEncoder(WithoutTimestamp())),
		WithLevel(TraceLevel),
	)

	var wg sync.WaitGroup

	wg.Add(20)

	for g := range 20 {
		go func() {
			defer wg.Done()

			child := root.With().Str("worker", strconv.Itoa(g)).Logger()
			for range 50 {
				child.Info().Msg("tick")
			}
		}()
	}

	wg.Wait()

	for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		if strings.Count(line, "worker=") != 1 {
			t.Fatalf("prefix corrupted: %q", line)
		}
	}
}

// TestSiblingLoggersDoNotShareBacking covers the copy-on-write requirement:
// two children of one parent must not overwrite each other's bound fields.
func TestSiblingLoggersDoNotShareBacking(t *testing.T) {
	var buf bytes.Buffer

	root := New(&buf, WithEncoder(NewTextEncoder(WithoutTimestamp())), WithLevel(TraceLevel))
	base := root.With().Str("app", "demo").Logger()

	a := base.With().Str("role", "reader").Logger()
	b := base.With().Str("role", "writer").Logger()

	a.Info().Send()
	b.Info().Send()

	out := buf.String()
	if !strings.Contains(out, "role=reader") || !strings.Contains(out, "role=writer") {
		t.Fatalf("sibling loggers clobbered each other: %q", out)
	}

	if strings.Count(out, "app=demo") != 2 {
		t.Fatalf("parent field lost: %q", out)
	}
}

// TestSetGlobalLevelIsRaceFree exercises the atomic global state.
func TestSetGlobalLevelIsRaceFree(t *testing.T) {
	prev := GlobalLevel()
	t.Cleanup(func() { SetGlobalLevel(prev) })

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for range 1000 {
			SetGlobalLevel(DebugLevel)
		}
	}()

	go func() {
		defer wg.Done()

		for range 1000 {
			_ = GlobalLevel()
		}
	}()

	wg.Wait()
}
