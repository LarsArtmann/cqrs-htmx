# cqrs-htmx Datastar Todo Demo

An event-sourced Todo app showing CQRS command/query dispatch with [Datastar](https://data-star.dev/) for reactive, server-driven UI updates.

## What it demonstrates

- **CQRS separation** — commands (`CreateTodo`, `ToggleTodo`, `DeleteTodo`, `UpdateTodo`) vs queries (`ListTodos`) in `domain_commands.go` / `domain_cqrs.go`
- **Event sourcing** — every mutation produces a domain event stored in `domain_store.go`; the read model is a fold over the event log
- **SSE event stream** — `GET /api/events` fans events to every connected tab for live multi-tab updates
- **Event replay** — `GET /api/events/replay` re-streams the full event history (catch-up on connect)
- **Datastar reactivity** — the frontend merges server-sent fragments via Datastar rather than HTMX

## Run

```bash
cd examples/datastar-demo
go run .
```

Open http://localhost:8095 — open a second tab to see live sync via the event stream.

## Endpoints

| Method | Path                 | Purpose                              |
| ------ | -------------------- | ------------------------------------ |
| GET    | `/`                  | Index page (Datastar + Todo UI)      |
| POST   | `/api/todos`         | Create a todo (command)              |
| POST   | `/api/todos/toggle`  | Toggle done (command)                |
| POST   | `/api/todos/delete`  | Delete a todo (command)              |
| POST   | `/api/todos/update`  | Update text (command)                |
| GET    | `/api/todos`         | List todos (query)                   |
| GET    | `/api/events`        | SSE stream of domain events (live)   |
| GET    | `/api/events/replay` | Replay full event history over SSE   |
| POST   | `/api/simulate`      | Simulate concurrent mutations (demo) |

> Unlike the `basic` example (HTMX) this demo uses vanilla net/http handlers
> and Datastar fragments, showing that cqrs-htmx's CQRS/event patterns are
> framework-agnostic. The domain layer (`domain_*`) is identical in shape to
> what you would build on top of `cqrshtmx.App`.
