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
        if err := ev.TailFileContext(ctx, "YOUR_LOG_FILE_HERE.log", false, ch); err != nil {
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

---

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
