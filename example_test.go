package reggol_test

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/efureev/reggol"
)

// Every example here is also a snippet in Readme.md and Readme.ru.md. Keeping
// them compiled is the check that was missing in v0, where the Quick start
// snippet did not build at all.

// stdout keeps example output deterministic: no timestamp, no color.
func stdout() reggol.Logger {
	enc := reggol.NewConsoleEncoder(
		reggol.WithColorMode(reggol.ColorNever, nil),
		reggol.WithConsoleOptions(reggol.WithoutTimestamp()),
	)

	return reggol.New(os.Stdout, reggol.WithEncoder(enc), reggol.WithLevel(reggol.TraceLevel))
}

func ExampleNew() {
	logger := stdout()
	logger.Info().Str("user", "bob").Msg("signed in")

	// Output: INF signed in user=bob
}

// ExampleNew_inline is the shape that failed to compile in v0: New returns a
// value and the level methods are declared on the value receiver.
func ExampleNew_inline() {
	enc := reggol.NewTextEncoder(reggol.WithoutTimestamp())

	reggol.New(os.Stdout, reggol.WithEncoder(enc), reggol.WithLevel(reggol.TraceLevel)).
		Warn().
		Msg("disk almost full")

	// Output: level=warn, message=disk almost full
}

func ExampleLogger_Err() {
	logger := stdout()

	// The message survives alongside the error; in v0 it was silently dropped.
	logger.Error().Err(errors.New("db down")).Str("host", "db-1").Msg("query failed")

	// Output: ERR query failed db down host=db-1
}

func ExampleLogger_With() {
	logger := stdout().With().
		Str("service", "auth").
		Int("shard", 7).
		Logger()

	logger.Info().Str("user", "alice").Msg("token issued")

	// Output: INF token issued service=auth shard=7 user=alice
}

func ExampleEvent_Blocks() {
	logger := stdout()
	logger.Info().Blocks("API", "GET /users").Msg("ok")

	// Output: INF API GET /users ok
}

func ExampleNewBlock() {
	logger := stdout()
	block := reggol.NewBlock("auth", func(s string) string { return "[" + s + "]" })
	logger.Info().Block(block).Msg("token verified")

	// Output: INF [auth] token verified
}

func ExampleNewTextEncoder() {
	logger := reggol.New(os.Stdout,
		reggol.WithEncoder(reggol.NewTextEncoder(reggol.WithoutTimestamp())),
		reggol.WithLevel(reggol.TraceLevel),
	)

	logger.Info().Str("service", "auth").Int("status", 200).Msg("request")

	// Output: level=info, message=request, service=auth, status=200
}

func ExampleNewJSONEncoder() {
	logger := reggol.New(os.Stdout,
		reggol.WithEncoder(reggol.NewJSONEncoder(reggol.WithoutTimestamp())),
		reggol.WithLevel(reggol.TraceLevel),
	)

	logger.Info().Str("service", "auth").Int("status", 200).Msg("request")

	// Output: {"level":"info","message":"request","service":"auth","status":200}
}

func ExampleSyncWriter() {
	// Wrap any writer that is not itself concurrency-safe.
	logger := reggol.New(reggol.SyncWriter(os.Stdout),
		reggol.WithEncoder(reggol.NewTextEncoder(reggol.WithoutTimestamp())),
		reggol.WithLevel(reggol.TraceLevel),
	)

	logger.Info().Msg("safe from many goroutines")

	// Output: level=info, message=safe from many goroutines
}

func ExampleLogger_Level() {
	logger := stdout().Level(reggol.WarnLevel)

	logger.Info().Msg("hidden")
	logger.Error().Msg("visible")

	// Output: ERR visible
}

func ExampleSetGlobalLevel() {
	prev := reggol.GlobalLevel()
	defer reggol.SetGlobalLevel(prev)

	reggol.SetGlobalLevel(reggol.ErrorLevel)

	logger := stdout()
	logger.Info().Msg("filtered by the global level")
	logger.Error().Msg("passes")

	// Output: ERR passes
}

func ExampleLogger_Ctx() {
	type traceKey struct{}

	enc := reggol.NewConsoleEncoder(
		reggol.WithColorMode(reggol.ColorNever, nil),
		reggol.WithConsoleOptions(reggol.WithoutTimestamp()),
	)

	logger := reggol.New(os.Stdout,
		reggol.WithEncoder(enc),
		reggol.WithLevel(reggol.TraceLevel),
		reggol.WithContextExtractor(func(ctx context.Context, e *reggol.Event) {
			if id, ok := ctx.Value(traceKey{}).(string); ok {
				e.Str("trace_id", id)
			}
		}),
	)

	ctx := context.WithValue(context.Background(), traceKey{}, "abc123")
	logger.Ctx(ctx, reggol.InfoLevel).Msg("handled")

	// Output: INF handled trace_id=abc123
}

func ExampleEvent_Dur() {
	logger := stdout()
	logger.Info().Dur("elapsed", 1500*time.Millisecond).Msg("done")

	// Output: INF done elapsed=1.5s
}
