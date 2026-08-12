# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] — 2026-08-12

### Added

- **Call sites.** `WithCaller` records where each record was produced;
  `Event.Caller` opts in for a single event, and `AddCallerSkip` lets wrapping
  code report its own callers rather than itself. The position is rendered after
  the level in the console and as a `caller` field in the text and JSON
  encoders, shortened to the last two path segments.
- `EventData.PC`, `EventData.Caller` and `EventData.CallerFunction` expose the
  call site to encoders and formatting hooks; `WithCallerFormatter` overrides how
  it is rendered.
- `slogr` now carries call sites in both directions: `FromHandler` fills the
  program counter in `slog.Record`, so a downstream handler with `AddSource: true`
  finally reports a source, and `NewHandler` uses the counter slog captured
  instead of a position inside the bridge.
- **256-color and 24-bit output.** `Color256`, `ColorRGB`, the `Style` type and
  `ConsoleEncoder.SetLevelStyle` allow colors a `TextStyle` bitmask cannot hold.
  `WithColorDepth` sets how many colors the destination supports; `DepthAuto`, the
  default, reads `COLORTERM` and `TERM`. Anything the terminal cannot express is
  reduced to the nearest color it can, rather than emitted as escape codes.

### Changed

- **No dependencies at all.** `gh.tarampamp.am/colors` was archived by its author on
  10 April 2026, so the part of it reggol actually used — a bitmask and one method
  mapping it to ECMA-48 SGR codes — now lives here. `golang.org/x/sys` went with it:
  that arrived only through the archived package's terminal detection, which reggol
  never called. `go.mod` now requires nothing, and `go.sum` is gone.

  `TextStyle` was an alias for the archived package's type and is now reggol's own.
  Every constant keeps its name and its numeric value, and the escape sequences are
  byte-for-byte identical: `TestColorCodesMatchReference` pins the old implementation's
  own output, captured before the dependency was removed, and the golden files did not
  move.
- Every public event constructor now calls one internal funnel directly. `Err` no
  longer routes through `Error`, `WithLevel` no longer through `Fatal`/`Panic`,
  `Ctx` no longer through `WithLevel`, and `Print`/`Printf`/`Println` no longer
  through `Debug`. Behaviour is unchanged; the uniform call depth is what makes a
  single caller-skip constant correct.

### Removed

- `TextStyle.Wrap`, `TextStyle.Start`, `TextStyle.Reset` and `TextStyle.String`. These
  were never reggol's own API — they were reachable only because `TextStyle` aliased the
  archived package's type — and each consulted a package-level on/off flag that reggol
  deliberately ignored in favour of deciding color per encoder. `Has`, `Add`, `Remove`
  and `ColorCodes` are kept.

## [1.0.0] — 2026-08-12

A complete rewrite of the core. **There is no API in common with the 0.x line and no migration path**; pin `v0.4.1` if
you need the old interface. The rewrite exists to make the library deliver what its README always claimed — near-zero
allocation logging — and to fix a data race that made concurrent logging unsafe.

The *Fixed* section below is the full list of what was wrong with 0.4.1; the *Performance*
table is why a refactor could not have delivered it.

### Performance

The point of the release. Measured on go1.26, darwin/arm64, Apple M5 Pro:

| Benchmark                    | 0.4.1                         | 1.0.0                       |
|------------------------------|-------------------------------|-----------------------------|
| `LogEmpty`                   | 87.9 ns · 200 B · 8 allocs    | ~60 ns · 0 B · **0 allocs** |
| `Info`                       | 195.2 ns · 545 B · 14 allocs  | ~58 ns · 0 B · **0 allocs** |
| `LogFields` (3 typed fields) | 489.7 ns · 1204 B · 29 allocs | ~22 ns · 0 B · **0 allocs** |
| `Msgf`                       | not measured                  | ~65 ns · 0 B · **0 allocs** |
| `Child` (3 bound fields)     | feature absent                | ~65 ns · 0 B · **0 allocs** |

Zero allocations is a contract, not an observation: `scripts/check-allocs.sh` fails CI if any benchmark allocates, and
`TestZeroAllocations` covers every built-in path. Wall-clock numbers are informational — they swing threefold on a
loaded machine, allocation counts do not.

How it was achieved:

- Values are stored in a typed `Value` (kind + `uint64` + `string` + `any`) instead of being boxed into
  `map[string]any`. Integers, booleans, durations and timestamps never touch an interface.
- Encoders append into one pooled buffer. The three intermediate representations of 0.x —
  `fmt.Sprintf` per field, a `[]string`, then a `bytes.Buffer` — are gone, and with them 95% of all allocations.
- Numbers go through `strconv.Append*`, timestamps through `AppendFormat`, ANSI sequences are precomputed once per
  encoder rather than rebuilt per line.
- Child loggers pre-encode their bound fields, so each record pays a memory copy instead of a re-encode.
- The event pool has a capacity ceiling, so one oversized record no longer inflates every pooled buffer — a TODO that
  had been in the code since the beginning.

### Added

- **`JSONEncoder`** — one JSON object per line, escaping per RFC 8259, without `encoding/json`
  on the hot path. Invalid UTF-8 and non-finite floats still produce a valid document.
- **Child loggers** — `Logger.With()` returns a `Context` builder; `Logger()` binds the fields.
- **`context.Context` support** — `Logger.WithContext`, `FromContext`, `Logger.Ctx`,
  `Event.Ctx`, `Event.GetCtx`, and pluggable `WithContextExtractor` for trace identifiers.
- **`slogr` subpackage** — a `log/slog` handler backed by reggol, verified against the standard library's own
  `testing/slogtest` suite (all 17 cases). Includes `NewHandler`, `New`,
  `FromSlogLevel`, `ToSlogLevel` and the `LevelTrace`/`LevelFatal`/`LevelPanic` extensions.
- **`SyncWriter`** — serializes concurrent writes; the `log/` facade uses it by default.
- **`MultiWriter`** — fans one event out to several destinations.
- **Typed value system** — `Kind`, `Value`, the `*Value` constructors, `Value.AppendTo`,
  `Field` and its constructors (`String`, `Int`, `Dur`, `Time`, `Group`, `Any`, …).
- **`EventData` accessors** — `Level`, `Time`, `Message`, `MessageBytes`, `Fields`, `Blocks`,
  `Prefix`. In 0.x the struct had no accessors at all, which made implementing the encoder interface impossible from
  outside the package.
- **Color policy** — `ColorMode` (`ColorAuto`, `ColorAlways`, `ColorNever`) with terminal detection against the writer
  actually in use, honoring `NO_COLOR`, `FORCE_COLOR` and
  `TERM=dumb`.
- **Encoder options** — `WithTimeFormat`, `WithoutTimestamp`, `WithoutLevel`, `WithoutSort`,
  `WithKeyNames`, `WithBeforeEncode`, `WithAfterEncode`, plus console-specific
  `WithColorMode` and `WithConsoleOptions`.
- **Logger options** — `WithEncoder`, `WithLevel`, `WithContextExtractor`; plus `Logger.Output`
  and `Logger.Encoder`.
- **Event methods** — `Send`, `Timestamp`, `Field`, `Fields`, `Int64`, `Uint64`, `Float64`,
  `Dur`, `Any`.
- **Facade completeness** — `log.L`, `log.SetLogger`, `log.GetLevel`, `log.Println`,
  `log.Write`, `log.With`, `log.Ctx`, `log.WithContext`, `log.Output`.
- **Level helpers** — `Level.Label`, and the `ErrUnknownLevel` / `ErrLevelOutOfRange` sentinels.
- **Global configuration accessors** — `SetExitCode`, `ExitCode`, `SetErrorHandler`.
- **`LICENSE`** — the MIT file both READMEs had always referenced but that never existed.
- **Generated screenshot** — `.assets/console.svg` is rendered from the library itself by
  `make screenshot`, so the picture cannot drift from the format the way the previous hand-taken PNG did.
- **Test infrastructure** — allocation gates, a concurrency guard, four fuzz targets, runnable examples for every README
  snippet, an AST-based facade parity test, and `slogtest`.
- **CI** — separate `alloc`, `fuzz` and `bench` jobs; the benchmark job blocks on allocations and reports `benchstat`
  against the base branch for information only.

### Changed

Everything in this section is breaking.

- **Minimum Go is 1.25** (was 1.24).
- **`Logger` methods take value receivers.** In 0.x `New` returned a value while the level methods were declared on
  `*Logger`, so `reggol.New(w).Info()` — the form printed in the README — did not compile.
- **`New(w io.Writer, opts ...Option)`** replaces `New(w io.Writer)`; the encoder is configured on the logger rather
  than on the writer.
- **`Transformer` → `Encoder`**, and `Transform(EventData) []byte` → `AppendEvent(dst []byte,
  *EventData) []byte`. Encoders append rather than allocate and return.
- **Formatting hooks are typed and append-based.** `Formatter func(interface{}) string` is replaced by `LevelFormatter`,
  `TimeFormatter`, `FieldFormatter`, `KeyFormatter`,
  `ValueFormatter`, `MessageFormatter` and `BlocksFormatter`, all of the shape
  `func(dst []byte, …) []byte`.
- **Fields are an ordered slice, not a map.** A repeated key now emits both entries instead of replacing the first,
  matching `log/slog`.
- **The trailing newline is appended by the encoder**, not the writer. Exactly one newline per record still reaches the
  output.
- **Errors are ordinary fields.** `Err` and `AnErr` add a field like any other, so an event may carry both an error and
  a message.
- **`Event.Push` → `Event.Send`.**
- **`Msg` and `Msgf` store the message as bytes** in the pooled event, which is what makes
  `Msgf` allocation-free; `MessageFormatter` therefore receives `[]byte`.
- **`NewConsoleTransformer(noColor bool, timeFormat string)` → `NewConsoleEncoder(opts ...)`.**
  The old first argument read as the opposite of its name.
- **`log.Logger` variable → `log.L()` / `log.SetLogger()`** over an `atomic.Pointer`; replacing the package logger used
  to race with every read of it.
- **`ExitCode` and `ErrorHandler` variables → functions** guarded by atomics.
- **Level lookup tables are arrays indexed by level**, not maps: no hashing per record and no concurrent map access.
- **`ParseLevel` errors are lowercase and wrap sentinels** (`ErrUnknownLevel`,
  `ErrLevelOutOfRange`), so callers can inspect the cause without matching text.
- **Default time layouts are explicit per encoder** — `DefaultConsoleTimeFormat`,
  `DefaultTextTimeFormat`, `DefaultJSONTimeFormat` — and are actually in effect.
- **Color constants alias the upstream `colors` package** instead of being re-declared with
  `1 << iota`, which could have drifted silently into wrong escape sequences.
- **Linting** — `golangci-lint` now runs with `default: standard`, enabling `staticcheck`,
  `unused` and `revive`. Their absence in 0.x is why dead exported symbols and style violations survived a 45-linter
  configuration.

### Removed

- `Transformer`, `AbstractTransformer`, `ConsoleTransformer`, `TextTransformer`.
- `ConsoleWriter`, `NewConsoleWriter`, `ConsoleBufInitCap`.
- `TransformWriter`, `TransformWriterAdapter`, `LevelWriterAdapter`.
- `Fields` (the `map[string]any` type) and `Formatter`.
- `LevelColors`, `FormattedLevels`, `LevelFieldMarshalFunc`, `ErrorMarshalFunc`, and the
  `ErrorHandler` and `ExitCode` variables.
- `SetColor`.
- `utils.go` and `utils_go21.go`, whose `go1.21` build tag was dead under a `go 1.24` directive.

### Fixed

- **Data race in concurrent logging.** Nothing in the 0.x chain was synchronized, and no way to fix it from the outside
  existed. Guarded by `TestConcurrentWritesAreSerialised`.
- **The README quick-start example did not compile** (see *Changed*, value receivers).
- **`.Err(err).Msg("text")` silently dropped the text.** Both encoders rendered either the error or the message, never
  both.
- **`AnErr(key, err)` discarded the key** unless `ErrorMarshalFunc` returned a non-error, non-string value.
- **`Discard()` leaked events**, never returning them to the pool.
- **The `log/` facade was missing `GetLevel`, `Println` and `Write`.** Parity is now enforced by a test that parses both
  files and fails on drift.
- **No terminal detection**, so redirecting output to a file wrote ANSI escape sequences into it.
- **`TimeFieldFormat` was exported but unused**, so setting it had no effect. It now genuinely controls the rendering of
  time-valued fields.
- **`ParseLevel` returned capitalized error strings** (ST1005).
- **The event pool had no capacity ceiling**, so a single large record permanently inflated it.

### Documentation

- Both READMEs rewritten for the new API and kept section-for-section identical.
- Every README snippet is a runnable example in `example_test.go` whose output is asserted — the check whose absence
  produced the non-compiling quick start.
- `UPGRADE.md` tracks what remains before the tag and the directions worth exploring after it.

## [0.4.1] — 2025-12-07

### Fixed

- CI workflow corrections.

## [0.4.0] — 2025-12-07

### Changed

- Core functionality reworked, documentation revised, outdated README and unused files removed.

## [0.3.1] — 2025-12-06

### Changed

- Dependency updates.

## [0.3.0] — 2024-03-22

### Changed

- Vendored color helpers removed from the public API.

### Added

- Initial test coverage.

## [0.2.1] — 2024-03-20

### Fixed

- Color helpers made public again, reverting the change in 0.2.0.

## [0.2.0] — 2024-03-20

### Changed

- Color dependency made private.

## [0.1.0] — 2024-03-19

### Added

- Initial release: `Logger → Transformer → Writer` architecture, console and text transformers, blocks, levels, and the
  global `log/` facade.

[Unreleased]: https://github.com/efureev/reggol/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/efureev/reggol/compare/v0.4.1...v1.0.0
[0.4.1]: https://github.com/efureev/reggol/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/efureev/reggol/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/efureev/reggol/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/efureev/reggol/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/efureev/reggol/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/efureev/reggol/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/efureev/reggol/releases/tag/v0.1.0
