# Migrating within v3: v3.0 → v3.3

These versions are **backward compatible** — no import path changes, no breaking API changes. This guide documents the new **opt-in** features you can adopt incrementally.

---

## v3.3.0 — Checkpoint Replay, BasicCommand, Server-Timing

### 1. Checkpoint-Based Projection Replay (ADR-0031)

**Problem:** On restart, `StartProjections` replays ALL historical events via `journal.ReadAll()`. For large event stores, this causes slow startup.

**Fix:** Set `EventSourcedConfig.CheckpointStore` to resume from the last processed position:

```go
// Before (v3.0 — full replay on every restart)
setup, _ := usermgmt.DefaultEventSourcedSetup(usermgmt.EventSourcedConfig{
    EventStore: myStore,
    EventBus:   myBus,
})

// After (v3.3 — resume from checkpoint)
setup, _ := usermgmt.NewEventSourcedSetup(usermgmt.EventSourcedConfig{
    EventStore:      myStore,
    EventBus:        myBus,
    CheckpointStore: usermgmt.NewMemoryCheckpointStore(), // opt-in
})
```

**Backward compatible:** `CheckpointStore` defaults to `nil` → full replay (v3.0 behavior). No change required for existing consumers.

### 2. BasicCommand Embedding (ADR-0032)

All 20 usermgmt commands now embed `*command.BasicCommand`, which structurally guarantees every command has a unique ID. This is transparent to consumers — no API change.

**If you create custom commands** (outside usermgmt), embed `*command.BasicCommand` to get the same guarantee:

```go
type MyCustomCmd struct {
    *command.BasicCommand // provides ID(), Type(), Metadata()
    Data string
}

func NewMyCustomCmd(data string) *MyCustomCmd {
    return &MyCustomCmd{
        BasicCommand: command.New("MyCustom"),
        Data:         data,
    }
}
```

### 3. Server-Timing API (ADR-0033)

Opt-in W3C Server-Timing header for browser-visible request timing:

```go
app, _ := cqrshtmx.New(cqrshtmx.Config{
    Commands:    disp,
    ServerTiming: func(r *http.Request) bool {
        return r.URL.Query().Get("debug") == "1"
    },
})
```

**Backward compatible:** `ServerTiming` defaults to `nil` → disabled (zero overhead).

---

## v3.1.0 — SQL Read Models & Stack Presets

### One-Call SQLite Setup

```go
setup, err := usermgmt.NewSQLiteEventSourcedSetup(usermgmt.EventSourcedConfig{
    DatabaseDSN: "file:events.db?cache=shared&_journal_mode=WAL",
})
```

This replaces manual wiring of event store + bus + 4 SQL read models + projections. Existing manual setups continue to work — the presets are additive.

### Graceful Shutdown

```go
setup, _ := usermgmt.NewSQLiteEventSourcedSetup(cfg)
defer setup.Close() // closes bus + store if they implement io.Closer
```

---

## Quick Checklist

| Feature                | Version | Breaking?        | Opt-in?                         |
| ---------------------- | ------- | ---------------- | ------------------------------- |
| Checkpoint replay      | v3.3.0  | No               | `CheckpointStore` field         |
| BasicCommand embedding | v3.3.0  | No (transparent) | Automatic for usermgmt          |
| Server-Timing          | v3.3.0  | No               | `Config.ServerTiming` predicate |
| SQL read models        | v3.1.0  | No               | `ReadModelDB` field             |
| Stack presets          | v3.1.0  | No               | `NewSQLiteEventSourcedSetup`    |
| Graceful shutdown      | v3.1.0  | No               | `setup.Close()`                 |

All v3.x versions are backward compatible. You can upgrade to v3.3.0 without changing any existing code — just adopt new features when ready.
