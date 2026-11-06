# system-demo

go-cqrs-lite `system.New()` integration via
[systemadapter](../../systemadapter/README.md): composes the identity-model
domain (4 deciders, 20 commands) onto the metaengine storage planner with
introspection, safety checks, and the driver registry. A console demo — no
HTTP server; watch the event log and view materialization on stdout.

```bash
GOEXPERIMENT=jsonv2 go run .                          # memory engine (dev default)
GOEXPERIMENT=jsonv2 go run . -engine sqlite -dsn file:app.db
GOEXPERIMENT=jsonv2 go run . -engine bbolt -dsn app.db
```

The `-engine` flag selects a `systemadapter.Recommended*Deployment` preset
(memory/sqlite/bbolt are linked here; the breadth presets for
pebble/postgres/duckdb/turso need their engine's blank import added first —
see the [systemadapter README](../../systemadapter/README.md)).

Guide: [Leveraging system/metaengine](../../docs/guides/leveraging-system-metaengine.md).
