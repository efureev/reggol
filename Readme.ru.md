# Reggol

Reggol — лёгковесный логгер в духе zerolog c понятной архитектурой «Logger → Transformer → Writer» и акцентом на быстрый консольный вывод.

Основан на идеях [zerolog](https://github.com/rs/zerolog), но проще и с другим форматом вывода. Поддерживает «блоки» (Blocks), настраиваемые трансформеры и работу через глобальный фасад `log/`.

[![Go Coverage](https://github.com/efureev/reggol/wiki/coverage.svg)](https://raw.githack.com/wiki/efureev/reggol/coverage.html)

Языки: [English](./Readme.md) | Русский

## Установка

```bash
go get github.com/efureev/reggol
# или только глобальный фасад
go get github.com/efureev/reggol/log
```

Поддерживаются версии Go 1.24+.

## Возможности

- Минимальные аллокации, простой API в стиле chain: `logger.Info().Str("k","v").Msg("hi")`.
- Два трансформера из коробки:
  - ConsoleTransformer — красивый консольный вывод с цветами.
  - TextTransformer — плоский текст `key=value` (удобно для лог-файлов/груберов).
- Блоки (Blocks) — короткие пометки/тэги перед сообщением.
- Глобальный фасад `github.com/efureev/reggol/log` — как стандартный `log`, но с уровнями.
- Глобальный уровень логирования (можно «поднять» минимум для всех логгеров).
- Настраиваемые форматтеры (хуки) на уровне трансформеров.
- Опция сортировки полей (`SortFields`) для стабильного вывода (по умолчанию включена).

Срезано по сравнению с zerolog: hooks, sampling, stacktrace, контекст и CBOR — пакет намеренно компактный.

## Быстрый старт

Глобальный фасад уже готов к использованию и пишет в stderr с «красивым» консольным форматом:

```go
package main

import "github.com/efureev/reggol/log"

func main() {
    log.Info().Msg("hello world")
}
```

Создание собственного логгера:

```go
package main

import (
    "github.com/efureev/reggol"
)

func main() {
    // Console (по умолчанию с цветами, можно отключить):
    cw := reggol.NewConsoleWriter()
    logger := reggol.New(cw)
    logger.Info().Str("user", "bob").Msg("signed in")

    // Плоский текст (key=value):
    tt := reggol.NewTextTransformer("")
    cw2 := reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = tt })
    reggol.New(cw2).Warn().Msg("disk almost full")
}
```

## Уровни логирования

Уровни: `Trace < Debug < Info < Warn < Error < Fatal < Panic`. По умолчанию глобально — `Info`.

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

Локальный уровень у конкретного логгера:

```go
logger := reggol.New(reggol.NewConsoleWriter()).Level(reggol.WarnLevel)
logger.Info().Msg("hidden")   // ниже warn
logger.Error().Msg("visible") // >= warn
```

## Поля, сообщения, ошибки

```go
log.Info().Str("user", "alice").Int("age", 30).Msg("profile updated")

// Ошибки: Err добавляет поле ошибки; для глобального фасада формат зависит от трансформера
log.Error().Err(fmt.Errorf("db down")).Msg("")

// Произвольные объекты
type payload struct{ A int }
log.Info().Interface("obj", payload{A: 1}).Msg("with object")
```

Поведение ошибок:
- ConsoleTransformer показывает текст ошибки без ключа (выделяя цветом).
- TextTransformer выводит `error=<text>`.

## Блоки (Blocks)

Блоки — короткие маркеры перед сообщением (например, имя компонента, тэг запроса):

```go
log.Info().Blocks("API", "GET /users").Msg("ok")

// Тонкая настройка:
block := reggol.NewBlock("auth", func(s string) string { return "[" + s + "]" })
log.Info().Block(block).Msg("token verified")
```

В консоли блоки выводятся перед сообщением через пробел; в текстовом трансформере доступен формат `blocks=[...]`.

## Трансформеры и настройка формата

Доступные трансформеры:
- `ConsoleTransformer(noColor bool, timeFormat string)` — человекочитаемый формат, короткие уровни (`INF/WRN/ERR`), цвета.
- `TextTransformer(timeFormat string)` — `key=value` формат, удобен для парсинга.

Общие опции (через методы и поля `AbstractTransformer`):
- `HideTimestamp()` / `HideLevel()` — скрыть метку времени/уровень.
- `SetSortFields(false)` или `DisableSort()` — отключить сортировку полей (быстрее, но порядок не гарантируется).
- Кастомизация форматирования (устанавливаются функциями):
  - `FormatLevelFn`
  - `FormatTimestampFn`
  - `FormatFieldFn`, `FormatFieldNameFn`, `FormatFieldValueFn`
  - `FormatMessageFn`, `FormatErrorFn`
  - `BeforeTransformFn` / `AfterTransformFn`

Пример отключения сортировки полей:

```go
tr := reggol.NewConsoleTransformer(false, "")
tr.DisableSort() // или tr.SetSortFields(false)
logger := reggol.New(reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = tr }))
```

## Writers

- `ConsoleWriter` — пишет в `io.Writer` (stdout по умолчанию). Добавляет завершающий перевод строки.
- `TransformWriterAdapter` — адаптер для произвольного `io.Writer` + выбранный трансформер. Тоже гарантирует ровно один перевод строки.
- Любой `Logger` можно использовать как `io.Writer` (метод `Write` форматирует как `Log().Msg(string(p))`).

Глобальный фасад `log` уже сконфигурирован на `ConsoleWriter` с выводом в stderr.

## Поведение Fatal и Panic

- `logger.Fatal().Msg(...)` — перед выходом пытается закрыть writer, затем вызывает `os.Exit(ExitCode)` (по умолчанию 1).
- `logger.Panic().Msg(...)` — вызывает `panic(msg)` после записи.

## Юзкейсы

Вот несколько практических сценариев использования пакета.

1) CLI-инструменты и сервисы со «вкусным» консольным логом
- используйте `ConsoleTransformer` (по умолчанию в глобальном фасаде), блоки для кратких тэгов, цветные уровни
```go
log.Info().Blocks("CLI").Msg("starting")
log.Warn().Blocks("db").Msg("reconnect")
```

2) Логи для парсинга и сбора метрик
- используйте `TextTransformer`, отключите цвета, оставьте `key=value`
```go
tr := reggol.NewTextTransformer("")
cw := reggol.NewConsoleWriter(func(w *ConsoleWriter) { w.Trans = tr })
logger := reggol.New(cw)
logger.Info().Str("service", "auth").Int("status", 200).Msg("request")
```

3) Оборачивание сторонних библиотек, которые требуют `io.Writer`
```go
logger := reggol.New(reggol.NewConsoleWriter()).Level(reggol.DebugLevel)
someLib.SetOutput(&logger) // Logger реализует io.Writer
```

4) Производительная запись без стабильного порядка полей
```go
tr := reggol.NewConsoleTransformer(true, "")
tr.DisableSort() // быстрее при очень большом числе полей
logger := reggol.New(reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = tr }))
```

## Глобальные настройки

- `reggol.SetGlobalLevel(lvl)` — установить минимальный глобальный уровень (например, `Disabled` для полного молчания).
- `reggol.GlobalLevel()` — получить текущий глобальный уровень.
- `reggol.ExitCode` — код выхода при `Fatal()` (по умолчанию 1).

## Советы по производительности

- Отключайте сортировку полей (`DisableSort()`), если порядок не важен.
- Переиспользуйте один логгер на компонент/модуль вместо создания нового на каждый лог.
- Избегайте тяжёлых форматирований в `Msgf`, если сообщение может быть отфильтровано по уровню; проверяйте `e.Enabled()`.

## Скриншоты

![Pretty Console Image](.assets%2Fconsole_screen_1.png)

## Лицензия

MIT. См. LICENSE.
