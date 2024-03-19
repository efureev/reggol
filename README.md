# Reggol

Fork from [zerolog](https://github.com/rs/zerolog).

## Install

## Features, cutting from original

- Hooks
- Sample
- Stacktrace
- Context
- CBOR

## Features, missing in `zerolog`:

- Blocks
- new Arch of writer: `logger -> Formatter -> Writer`
- You can realize your own specify `Formatter`

```bash
go get -u github.com/efureev/reggol/log
```

## Getting Started

### Simple Logging Example

For simple logging, import the global logger package `github.com/efureev/reggol/log`

```go
package main

import (
	"github.com/efureev/reggol"
	"github.com/efureev/reggol/log"
)

func main() {
	log.Print("hello world")
}
```

### Advanced Logging Example: defining Transformer and Writer

For simple logging, import the global logger package `github.com/efureev/reggol`

```go
package main

import "github.com/efureev/reggol"

func main() {
	trans := reggol.NewTextTransformer(``)
	//trans.HideTimestamp()
	//trans.HideLevel()

	//writer := reggol.NewConsoleWriter().WithTransformer(trans)
	writer := reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = trans })
	logger := reggol.New(writer)

	logger.Warn().Msg(`test`)

	// or

	logger2 := reggol.New(reggol.NewConsoleWriter())
	logger2.Info().Msg(`test`)
}

```

### Leveled Logging

**reggol** allows for logging at the following levels (from highest to lowest):

* Panic (`zerolog.PanicLevel`, 5)
* Fatal (`zerolog.FatalLevel`, 4)
* Error (`zerolog.ErrorLevel`, 3)
* Warn (`zerolog.WarnLevel`, 2)
* Info (`zerolog.InfoLevel`, 1)
* Debug (`zerolog.DebugLevel`, 0)
* Trace (`zerolog.TraceLevel`, -1)

#### Setting Global Log Level

This example uses command-line flags to demonstrate various outputs depending on the chosen log level.

```go
package main

import (
    "flag"

    "github.com/efureev/reggol"
    "github.com/efureev/reggol/log"
)

func main() {
    debug := flag.Bool("debug", false, "sets log level to debug")

    flag.Parse()

    // Default level for this example is info, unless debug flag is present
	reggol.SetGlobalLevel(reggol.InfoLevel)
    if *debug {
		reggol.SetGlobalLevel(reggol.DebugLevel)
    }

    log.Debug().Msg("This message appears only when log level set to Debug")
    log.Info().Msg("This message appears when log level set to Debug or Info")

    if e := log.Debug(); e.Enabled() {
        // Compute log output only if enabled.
        value := "bar"
        e.Str("foo", value).Msg("some debug message")
    }
}
```

#### Logging without Level or Message

You may choose to log without a specific level by using the `Log` method. You may also write without a message by setting an empty string in the `msg string` parameter of the `Msg` method. Both are demonstrated in the example below.

```go
package main

import (
    "github.com/efureev/reggol"
    "github.com/efureev/reggol/log"
)

func main() {
    log.Log().
        Str("foo", "bar").
        Msg("")
}

// Output: time=1494567715, foo=bar
```

### Error Logging

You can log errors using the `Err` method

```go
package main

import (
	"errors"

	"github.com/efureev/reggol"
	"github.com/efureev/reggol/log"
)

func main() {
	err := errors.New("seems we have an error here")
	log.Error().Err(err).Msg("")
}

// Output: level=error, error=seems we have an error here, time=1609085256}
```

## Global Settings

Some settings can be changed and will be applied to all loggers:

- `reggol.SetGlobalLevel`: : Can raise the minimum level of all loggers. Call this with `reggol.Disabled` to disable logging altogether (quiet mode).
