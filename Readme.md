# Reggol

Reggol is a lightweight, zerolog‑inspired logger with a clear architecture “Logger → Transformer → Writer” and a focus on fast, pretty console output.

Based on ideas from [zerolog](https://github.com/rs/zerolog), but simpler and with a different output format. It supports Blocks, customizable transformers, and a global facade `log/`.

[![Go Coverage](https://github.com/efureev/reggol/wiki/coverage.svg)](https://raw.githack.com/wiki/efureev/reggol/coverage.html)

Languages: English | [Русский](./Readme.ru.md)

## Installation

```bash
go get github.com/efureev/reggol
# or only the global facade
go get github.com/efureev/reggol/log
```

Supported Go versions: 1.24+.

## Features

- Minimal allocations, simple chain‑style API: `logger.Info().Str("k","v").Msg("hi")`.
- Two transformers included:
  - ConsoleTransformer — pretty console output with colors.
  - TextTransformer — flat `key=value` format (great for logs/greppers).
- Blocks — short tags in front of the message.
- Global facade `github.com/efureev/reggol/log` — similar to the standard `log`, but leveled.
- Global log level (can raise the minimum for all loggers).
- Customizable formatting hooks at transformer level.
- Field sorting option (`SortFields`) for stable output (enabled by default).

Intentionally smaller than zerolog: no hooks/sampling/stacktrace/context/CBOR — kept compact on purpose.

## Quick start

The global facade is ready to use: it writes to stderr in pretty console format by default.

```go
package main

import "github.com/efureev/reggol/log"

func main() {
    log.Info().Msg("hello world")
}
```

Create your own logger:

```go
package main

import (
    "github.com/efureev/reggol"
)

func main() {
    // Console (colors by default; can be disabled):
    cw := reggol.NewConsoleWriter()
    logger := reggol.New(cw)
    logger.Info().Str("user", "bob").Msg("signed in")

    // Flat key=value text:
    tt := reggol.NewTextTransformer("")
    cw2 := reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = tt })
    reggol.New(cw2).Warn().Msg("disk almost full")
}
```

## Log levels

Levels: `Trace < Debug < Info < Warn < Error < Fatal < Panic`. Default global level is `Info`.

```go
import (
    "flag"
    "github.com/efureev/reggol"
    "github.com/efureev/reggol/log"
)

func main() {
    dbg := flag.Bool("debug", false, "enable debug")
    flag.Parse()

    if *dbg {
        reggol.SetGlobalLevel(reggol.DebugLevel)
    }

    log.Debug().Msg("only in debug")
    log.Info().Msg("always visible (>= info)")
}
```

Per‑logger minimum level:

```go
logger := reggol.New(reggol.NewConsoleWriter()).Level(reggol.WarnLevel)
logger.Info().Msg("hidden")   // below warn
logger.Error().Msg("visible") // >= warn
```

## Fields, messages, errors

```go
log.Info().Str("user", "alice").Int("age", 30).Msg("profile updated")

// Errors: Err adds an error field; exact formatting depends on transformer
log.Error().Err(fmt.Errorf("db down")).Msg("")

// Arbitrary objects
type payload struct{ A int }
log.Info().Interface("obj", payload{A: 1}).Msg("with object")
```

Error behavior:
- ConsoleTransformer shows the error text without a key (and colors it).
- TextTransformer prints `error=<text>`.

## Blocks

Blocks — short markers before the message (e.g., component name, request tag):

```go
log.Info().Blocks("API", "GET /users").Msg("ok")

// Custom formatting for a block value:
block := reggol.NewBlock("auth", func(s string) string { return "[" + s + "]" })
log.Info().Block(block).Msg("token verified")
```

In console, blocks are printed before the message separated by spaces; in text transformer a `blocks=[...]` form is available.

## Transformers and formatting

Available transformers:
- `ConsoleTransformer(noColor bool, timeFormat string)` — human‑readable format with short levels (`INF/WRN/ERR`) and colors.
- `TextTransformer(timeFormat string)` — `key=value` format that’s easy to parse.

Shared options (via `AbstractTransformer` methods/fields):
- `HideTimestamp()` / `HideLevel()` — hide timestamp/level.
- `SetSortFields(false)` or `DisableSort()` — disable field sorting (faster, but order not guaranteed).
- Formatting customization via function hooks:
  - `FormatLevelFn`
  - `FormatTimestampFn`
  - `FormatFieldFn`, `FormatFieldNameFn`, `FormatFieldValueFn`
  - `FormatMessageFn`, `FormatErrorFn`
  - `BeforeTransformFn` / `AfterTransformFn`

Disable field sorting example:

```go
tr := reggol.NewConsoleTransformer(false, "")
tr.DisableSort() // or tr.SetSortFields(false)
logger := reggol.New(reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = tr }))
```

## Writers

- `ConsoleWriter` — writes to an `io.Writer` (stdout by default). Appends exactly one newline.
- `TransformWriterAdapter` — adapts any `io.Writer` with a selected transformer, also appends one newline.
- Any `Logger` implements `io.Writer` (its `Write` formats via `Log().Msg(string(p))`).

The global `log` facade is preconfigured to `ConsoleWriter` with stderr output.

## Fatal and Panic behavior

- `logger.Fatal().Msg(...)` — attempts to close the writer to flush, then calls `os.Exit(ExitCode)` (default 1).
- `logger.Panic().Msg(...)` — calls `panic(msg)` after writing.

## Use cases

1) CLI tools and services with pretty console logs
- use `ConsoleTransformer` (default in the global facade), blocks for short tags, colored levels
```go
log.Info().Blocks("CLI").Msg("starting")
log.Warn().Blocks("db").Msg("reconnect")
```

2) Logs for parsing and metrics
- use `TextTransformer`, no colors, clean `key=value`
```go
tr := reggol.NewTextTransformer("")
cw := reggol.NewConsoleWriter(func(w *ConsoleWriter) { w.Trans = tr })
logger := reggol.New(cw)
logger.Info().Str("service", "auth").Int("status", 200).Msg("request")
```

3) Wrapping libraries that require an `io.Writer`
```go
logger := reggol.New(reggol.NewConsoleWriter()).Level(reggol.DebugLevel)
someLib.SetOutput(&logger) // Logger implements io.Writer
```

4) High‑throughput logging without stable field order
```go
tr := reggol.NewConsoleTransformer(true, "")
tr.DisableSort() // faster when there are many fields
logger := reggol.New(reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = tr }))
```

## Global settings

- `reggol.SetGlobalLevel(lvl)` — set the global minimum level (e.g., `Disabled` to silence).
- `reggol.GlobalLevel()` — get the current global level.
- `reggol.ExitCode` — the exit code used by `Fatal()` (default 1).

## Performance tips

- Disable field sorting (`DisableSort()`) if order doesn’t matter.
- Reuse one logger per component/module instead of creating one per log line.
- Avoid heavy formatting in `Msgf` if the message may be filtered by level; check `e.Enabled()`.

## Screenshot

![Pretty Console Image](.assets%2Fconsole_screen_1.png)

## License

MIT. See LICENSE.
