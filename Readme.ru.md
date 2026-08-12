# Reggol

Reggol — логгер без аллокаций с ясной архитектурой `Logger → Encoder → Writer` и упором на
быстрый, красивый консольный вывод.

Вдохновлён [zerolog](https://github.com/rs/zerolog), но компактнее, с другим форматом вывода,
блоками и полноценным мостом к `log/slog`.

Языки: [English](./Readme.md) | Русский

> **v1 — чистый разрыв.** Общего API с веткой 0.x нет, миграции тоже нет: ядро переписано, чтобы
> убрать аллокации, закрыть гонку данных при конкурентном логировании и добавить дочерние
> логгеры, контексты и машиночитаемый вывод. Нужен старый API — фиксируйте `v0.4.1`.

## Установка

```bash
go get github.com/efureev/reggol
# или только глобальный фасад
go get github.com/efureev/reggol/log
```

Поддерживаемые версии Go: 1.25+.

## Возможности

- **Ноль аллокаций** на всех встроенных путях — сообщение, типизированные поля, ошибки, блоки,
  цвет, дочерние логгеры. Это проверяется в CI, а не просто измеряется.
- Три энкодера: `ConsoleEncoder` (человекочитаемый, цветной), `TextEncoder` (плоский `key=value`),
  `JSONEncoder` (по объекту на строку).
- Дочерние логгеры с привязанными полями: `logger.With().Str("service", "auth").Logger()`.
- Поддержка `context.Context`, включая подключаемые экстракторы идентификаторов трассировки.
- Мост к `log/slog` в `slogr/`, проверенный официальным conformance-набором стандартной библиотеки.
- Блоки — короткие метки перед сообщением.
- Потокобезопасность по построению через `SyncWriter`.
- Типизированные хуки форматирования без аллокаций.
- Автоопределение цвета с учётом `NO_COLOR` и `FORCE_COLOR`.

Намеренно меньше zerolog: без сэмплирования, стектрейсов и CBOR.

## Быстрый старт

Глобальный фасад готов к работе: пишет человекочитаемый вывод в stderr, с синхронизацией.

```go
package main

import "github.com/efureev/reggol/log"

func main() {
    log.Info().Msg("hello world")
}
```

Свой логгер:

```go
package main

import (
    "os"

    "github.com/efureev/reggol"
)

func main() {
    logger := reggol.New(os.Stdout)
    logger.Info().Str("user", "bob").Msg("signed in")

    // Плоский текст key=value:
    text := reggol.New(os.Stdout, reggol.WithEncoder(reggol.NewTextEncoder()))
    text.Warn().Msg("disk almost full")
}
```

`New` возвращает значение, и все методы объявлены на значении, поэтому работает и запись в одну
строку:

```go
reggol.New(os.Stdout, reggol.WithEncoder(reggol.NewTextEncoder())).Warn().Msg("disk almost full")
```

## Уровни логирования

Уровни: `Trace < Debug < Info < Warn < Error < Fatal < Panic`. Порогов два, и событие обязано
пройти оба: собственный минимум логгера и глобальный. Глобальный по умолчанию — `Info`, поэтому
`Debug` не виден, пока его не понизить.

```go
import (
    "flag"

    "github.com/efureev/reggol"
    "github.com/efureev/reggol/log"
)

func main() {
    debug := flag.Bool("debug", false, "enable debug")
    flag.Parse()

    if *debug {
        reggol.SetGlobalLevel(reggol.DebugLevel)
    }

    log.Debug().Msg("only in debug")
    log.Info().Msg("always visible (>= info)")
}
```

Минимум конкретного логгера:

```go
logger := reggol.New(os.Stdout).Level(reggol.WarnLevel)
logger.Info().Msg("hidden")   // ниже warn
logger.Error().Msg("visible") // >= warn
```

## Поля, сообщения, ошибки

```go
log.Info().Str("user", "alice").Int("age", 30).Msg("profile updated")

// Ошибка не вытесняет сообщение — выводится и то, и другое.
log.Error().Err(errors.New("db down")).Str("host", "db-1").Msg("query failed")
// ERR query failed db down host=db-1

// Явный ключ учитывается.
log.Error().AnErr("cause", err).Msg("failed")
```

Типизированные сеттеры — `Str`, `Int`, `Int64`, `Uint64`, `Float64`, `Bool`, `Dur`, `Time`,
`Bytes`, `IPAddr`, `Any` — сохраняют значение без боксинга, и именно отсюда берётся отсутствие
аллокаций. `Any` выбирает максимально конкретное представление.

Поля сортируются по ключу ради стабильного вывода; `WithoutSort()` оставляет порядок вставки.
Повторный ключ выводится дважды, а не заменяет первый, — так же поступает `log/slog`.

## Дочерние логгеры

Привяжите поля один раз и платите копированием памяти на строку вместо повторного кодирования:

```go
requestLog := logger.With().
    Str("service", "auth").
    Int("shard", 7).
    Logger()

requestLog.Info().Str("user", "alice").Msg("token issued")
// INF token issued service=auth shard=7 user=alice
```

Привязанные поля идут перед полями события, а сортировка применяется внутри каждой группы, а не
сквозь обе: сквозная потребовала бы декодировать привязанный префикс на каждой строке.

Настраивайте энкодер до создания дочерних логгеров: привязанные поля кодируются в момент вызова
`Logger()`, поэтому более поздние изменения хуков на них не распространяются.

## Контексты

```go
ctx := logger.WithContext(context.Background())
if l, ok := reggol.FromContext(ctx); ok {
    l.Info().Msg("found in context")
}
```

Экстракторы переносят значения из контекста в каждое событие. Если их нет, путь бесплатен:

```go
logger := reggol.New(os.Stdout,
    reggol.WithContextExtractor(func(ctx context.Context, e *reggol.Event) {
        if id, ok := ctx.Value(traceKey{}).(string); ok {
            e.Str("trace_id", id)
        }
    }),
)

logger.Ctx(ctx, reggol.InfoLevel).Msg("handled")
```

## Блоки (Blocks)

Блоки — короткие маркеры перед сообщением: имя компонента, метка запроса.

```go
log.Info().Blocks("API", "GET /users").Msg("ok")

block := reggol.NewBlock("auth", func(s string) string { return "[" + s + "]" })
log.Info().Block(block).Msg("token verified")
```

## Энкодеры

| Энкодер | Вывод |
|---|---|
| `NewConsoleEncoder` | `2:09PM INF hello int=123 string=four!` |
| `NewTextEncoder` | `ts=…, level=info, message=hello, int=123` |
| `NewJSONEncoder` | `{"ts":"…","level":"info","message":"hello","int":123}` |

Общие опции: `WithTimeFormat`, `WithoutTimestamp`, `WithoutLevel`, `WithoutSort`, `WithKeyNames`,
плюс хуки форматирования `WithLevelFormatter`, `WithTimeFormatter`, `WithFieldFormatter`,
`WithKeyFormatter`, `WithValueFormatter`, `WithMessageFormatter`, `WithBlocksFormatter`,
`WithBeforeEncode`, `WithAfterEncode`.

Хуки дописывают в буфер вызывающей стороны, а не возвращают строку, поэтому настройка вывода
ничего не стоит:

```go
enc := reggol.NewTextEncoder(
    reggol.WithLevelFormatter(func(dst []byte, l reggol.Level) []byte {
        return append(dst, strings.ToUpper(l.String())...)
    }),
)
```

Опции консоли задаются через `WithColorMode` и `WithConsoleOptions`:

```go
enc := reggol.NewConsoleEncoder(
    reggol.WithColorMode(reggol.ColorAuto, os.Stderr),
    reggol.WithConsoleOptions(reggol.WithTimeFormat(time.RFC3339)),
)
```

`ColorAuto` включает цвет, только если получатель — терминал, и учитывает `NO_COLOR`,
`FORCE_COLOR` и `TERM=dumb`. `ColorAlways` и `ColorNever` решают за вас.

## Writers и конкурентность

Логгер настолько же потокобезопасен, насколько потокобезопасен writer под ним, а большинство
writer'ов — включая `bytes.Buffer` и любые обёртки над ним — небезопасны вовсе. Оборачивайте:

```go
logger := reggol.New(reggol.SyncWriter(out))
```

Фасад в `log/` уже это делает. `MultiWriter` рассылает одно событие в несколько мест.

Энкодеры завершают запись ровно одним `\n`; writer'ы передают байты без изменений.

## Мост к log/slog

`slogr` делает reggol бэкендом для структурного логгера стандартной библиотеки. Пакет проходит
`testing/slogtest` — conformance-набор, который поставляется вместе с `log/slog`.

```go
import (
    "log/slog"

    "github.com/efureev/reggol"
    "github.com/efureev/reggol/slogr"
)

logger := slogr.New(reggol.New(os.Stdout, reggol.WithEncoder(reggol.NewJSONEncoder())))
logger.Info("handled", slog.String("user", "alice"))
```

Атрибуты, привязанные через `WithAttrs`, используют то же предкодирование, что и `With()`. Группы
разворачиваются в ключи через точку — так они сохраняют смысл и в консольной строке `key=value`.

Для slog-совместимых имён ключей:

```go
reggol.NewJSONEncoder(reggol.WithKeyNames(slog.TimeKey, slog.LevelKey, slog.MessageKey))
```

## Поведение Fatal и Panic

- `logger.Fatal().Msg(…)` закрывает writer для сброса буферов, затем вызывает
  `os.Exit(ExitCode())`, по умолчанию 1. Процесс завершается, **даже если уровень отфильтровал
  событие**: подавление записи — решение логирования, а подавление завершения превратило бы
  поднятие уровня в тихое изменение потока управления.
- `logger.Panic().Msg(…)` записывает событие, затем паникует с этим сообщением.

## Глобальные настройки

- `reggol.SetGlobalLevel(lvl)` / `reggol.GlobalLevel()` — минимум для всего процесса.
- `reggol.SetExitCode(code)` / `reggol.ExitCode()` — код возврата для `Fatal`.
- `reggol.SetErrorHandler(fn)` — вызывается при ошибке записи события; без него ошибки печатаются
  в stderr.

Всё перечисленное защищено атомиками и безопасно меняется в рантайме.

## Советы по производительности

Замерено на go1.26, darwin/arm64, Apple M5 Pro, через `go test -bench=. -benchmem`:

| Бенчмарк | ns/op | allocs/op |
|---|---:|---:|
| `Disabled` | ~0.4 | **0** |
| `LogFields` (3 типизированных поля) | ~32–40 | **0** |
| `Info` (только сообщение) | ~58–66 | **0** |
| `Child` (3 привязанных поля) | ~65 | **0** |
| `slog TextHandler` из stdlib (те же поля) | ~227 | 3 |

Время зависит от загрузки машины, число аллокаций — нет; поэтому CI блокирует сборку по
аллокациям, а тайминги считает справочными.

Рекомендации:

- Держите один логгер на компонент вместо создания логгера на строку.
- Повторяющиеся поля привязывайте через `With()`, а не дублируйте на каждом вызове.
- `WithoutSort()`, если порядок ключей не важен.
- Дорогие аргументы `Msgf` закрывайте проверкой `e.Enabled()`.

## Скриншоты

![Pretty Console Image](.assets%2Fconsole_screen_1.png)

## Лицензия

MIT. См. LICENSE.
