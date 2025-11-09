# Plutonium T6 Log Reader (Go)

## Getting started

Use the tailer to read a file and receive parsed events via a channel.

```go
package main

import (
    "context"
    "fmt"
    "log"

    ev "github.com/Yallamaztar/events/events"
)

func main() {
    logger := log.Default()
    ctx := context.Background()

    ch := make(chan ev.Event, 128)

    // Start tailing in a goroutine.
    go func() {
        // startAtEnd=false => read from beginning so you see existing lines
        if err := ev.TailFileContext(ctx, "games_mp3.log", false, ch); err != nil {
            logger.Printf("tail error: %v", err)
        }
        close(ch)
    }()

    // Handle events as they arrive.
    for e := range ch {
        switch t := e.(type) {
        case *ev.PlayerEvent:
            // Chat commands appear as say/sayteam
            if t.Command == "say" || t.Command == "sayteam" {
                fmt.Printf("[%s] %s: %s\n", t.Command, t.Player, t.Message)
            } else {
                fmt.Printf("PLAYER cmd=%s guid=%s num=%d name=%s\n", t.Command, t.XUID, t.Flag, t.Player)
            }
        case *ev.ServerEvent:
            fmt.Printf("SERVER %s data=%+v\n", t.Command, t.Data)
        default:
            fmt.Printf("OTHER %s raw=%q\n", e.GetCommand(), e.GetRaw())
        }
    }
}
```

### Parse a single line

If you want to parse individual strings without tailing a file:

```go
e, err := ev.ParseEventLine("J;ABCDEF;7;PlayerOne")
if err != nil { /* handle */ }
if p, ok := e.(*ev.PlayerEvent); ok {
    fmt.Println("player:", p.Player)
}
```

## Event types

- `Event` (interface):
  - `GetCommand() string`
  - `GetTimestamp() *time.Duration` (optional; parsed if a time prefix like `1:23:45` exists)
  - `GetRaw() string`
- `PlayerEvent` fields: `XUID`, `Flag` (client num), `Player`, `Message`, plus embedded `BaseEvent`
- `ServerEvent` fields: `Data` (map of k/v from lines like `InitGame: \key\value...`), plus embedded `BaseEvent`

## API surface (current)

- Tail
  - `TailFileContext(ctx, path string, startAtEnd bool, eventsCh chan<- Event) error`
  - `TailFile(path string, startAtEnd bool, eventsCh chan<- Event) error`
- Parse
  - `ParseEventLine(line string) (Event, error)`
- Player cache (optional)
  - `type Player struct { ClientNum int; Name string; GUID string }`
  - `type PlayerSource interface { Status() ([]Player, error) }`
  - `NewPlayerDirectory(source PlayerSource, ttl time.Duration) *PlayerDirectory`
  - Lookups: `FindByName`, `FindByClientNum`, `FindByGUID`, `Snapshot`, `Invalidate`

## Plutonium T6 line examples

- `InitGame: \sv_hostname\MyServer\mapname\mp_citystreets`
- `ShutdownGame:`
- `J;ABCDEF;7;PlayerOne` (join)
- `say PlayerOne hello world`
- `sayteam PlayerTwo need backup`
- Optional time prefix: `1:02:03 say PlayerOne hello` (parsed into `Timestamp`)

## Notes about the `internal` path

- Packages under `internal/` cannot be imported from outside this module. If you want to use this from other repositories:
  1) Move the code from `internal/events/events/` to a public directory like `events/` at repo root.
  2) Update `module` in `go.mod` and fix imports accordingly.

## Next steps / ideas

- Add a small dispatcher utility (typed handlers) to avoid manual type switches
- Provide a tiny example app and a generator that appends synthetic log lines
- Optional error callback for parse failures
- Integration tests for tail + parse + rotation

---

If you’d like, I can add a dispatcher and a tiny `examples/quickstart` program next, and/or flatten the package for a cleaner import path.# Events Package

Lightweight log tailing and parsing with a friendly, pluggable event handling API (dispatcher + typed callbacks) for your game/server logs.

> IMPORTANT: This lives under `internal/events/` right now. External projects cannot import an `internal` path. To publish for others, move the contents to a non-internal directory (e.g. `events/` or `pkg/events/`) and adjust the module path in `go.mod`.

## Features

- File tailing with automatic reopen / rotation detection
- Event parsing (player events, server lifecycle/config, chat lines, generic commands)
- Unified `Event` interface plus concrete types: `PlayerEvent`, `ServerEvent`
- Simple high-level streaming helpers: `Tail`, `Stream`, `Watch`
- Dispatcher for zero-boilerplate typed & command-specific handlers
- Extensible player lookup cache (`PlayerDirectory` + `PlayerSource` interface)
- Options pattern for customizing behavior (start position, logger, buffering)

## Install (after moving out of `internal`)

```bash
go get github.com/Yallamaztar/events@latest
```

## Quick Start

```go
import (
    "context"
    "fmt"
    "log"
    events "github.com/Yallamaztar/events"
)

func main() {
    logger := log.Default()
    ctx := context.Background()

    // Simple: just handle every event.
    _ = events.Stream(ctx, "games_mp3.log", func(ev events.Event) {
        fmt.Printf("%s -> %T\n", ev.GetCommand(), ev)
    }, events.WithLogger(logger), events.WithStartAtEnd(false))
}
```

## Dispatcher Usage

The dispatcher makes handling different event kinds trivial:

```go
d := events.NewDispatcher().WithLogger(log.Default())

d.OnAny(func(e events.Event) {
    fmt.Printf("RAW: %s\n", e.GetRaw())
})

d.OnPlayer(func(p *events.PlayerEvent) {
    fmt.Printf("PLAYER[%d] %s: %s\n", p.Flag, p.Player, p.Message)
})

d.OnServer(func(s *events.ServerEvent) {
    fmt.Printf("SERVER %s => %+v\n", s.GetCommand(), s.Data)
})

d.OnCommand("J", func(e events.Event) { fmt.Println("Join event detected") })

ctx := context.Background()
if err := events.StreamDispatch(ctx, "games_mp3.log", d, events.WithStartAtEnd(false)); err != nil {
    log.Fatalf("stream failed: %v", err)
}
```

### Background Watching

```go
cancel := events.WatchDispatch("games_mp3.log", d, events.WithStartAtEnd(true))
// later
cancel()
```

## Event Types

| Type         | Description                                    |
|--------------|------------------------------------------------|
| `BaseEvent`  | Generic fallback: raw line & command           |
| `PlayerEvent`| Player-related lines (join/chat/custom semicolon format) |
| `ServerEvent`| Server lifecycle/config (InitGame/ShutdownGame) |

Each implements:

```go
type Event interface {
    GetCommand() string
    GetTimestamp() *time.Duration // optional parsed timestamp prefix
    GetRaw() string               // original line
}
```

## Options

| Option | Purpose |
|--------|---------|
| `WithStartAtEnd(bool)` | Start tailing at EOF (true) or from beginning (false) |
| `WithLogger(*log.Logger)` | Capture tail/parse errors & handler panics |
| `WithBufferSize(int)` | Channel buffer capacity for `Tail`/`Stream` |

## Player Directory

If you have a status/RCON source, implement `PlayerSource`:

```go
type PlayerSource interface { Status() ([]events.Player, error) }

src := &MyRCON{}
pd  := events.NewPlayerDirectory(src, 2*time.Second)
player, _ := pd.FindByName("nick")
```

`PlayerDirectory` caches snapshots briefly to reduce backend churn.

## Error Handling Strategy

Current design logs parse errors and continues. You can wrap `Stream` yourself to collect metrics or halt on certain patterns. Potential future enhancements:

- Custom error types (`ErrParse`, `ErrTail`) with line context
- Expose a callback for parse failures (`WithErrorHandler(func(error, string))`)

## Testing

Create a temp file, append lines, and tail it. (Integration tests can simulate rotation by renaming & rewriting.)

```go
ev, err := events.Parse("J;ABCDEF;7;PlayerOne")
// assert *events.PlayerEvent
```

## File Rotation Handling

`TailFileContext` checks inode/size changes and reopens automatically. Your handlers do not need special logic.

## Move Out of `internal`

1. Create `events/` at repo root.
2. Move all `.go` files from `internal/events/events/` & this README there.
3. Update import paths in your code.
4. Adjust module path in `go.mod` (e.g., `module github.com/youruser/events`).

## Design Choices

- Keep parsing fast & allocation-light (single pass classification)
- Use an interface + concrete types for ergonomic type assertions
- Dispatcher avoids repetitive `switch ev.(type)` in user code
- Options pattern keeps API surface stable as features grow

## Roadmap / Future Ideas

- Rich error callback hook
- Built-in synthetic event generation (heartbeats, player join abstractions)
- Pluggable parsers or rule-based pattern matching
- Metrics counters (events/sec, parse failures) via optional exporter
- Context-aware cancellation on backpressure

## Minimal Example (All In One)

```go
package main
import (
    "context"; "fmt"; "log"; events "github.com/youruser/events"
)
func main() {
    d := events.NewDispatcher().OnAny(func(e events.Event){fmt.Println(e.GetRaw())})
    if err := events.StreamDispatch(context.Background(), "games_mp3.log", d, events.WithStartAtEnd(false)); err != nil {
        log.Fatal(err)
    }
}
```

## License

Choose a license (MIT recommended). Add `LICENSE` file at repo root when publishing.
