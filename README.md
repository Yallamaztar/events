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
