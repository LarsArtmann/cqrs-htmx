# system-demo

go-cqrs-lite `system.New()` integration via
[systemadapter](../../systemadapter/README.md): composes the identity-model
domain (4 deciders, 20 commands) onto the metaengine storage planner with
introspection, safety checks, and the driver registry. A console demo — no
HTTP server; watch the event log and view materialization on stdout.

```bash
GOEXPERIMENT=jsonv2 go run .
```

Guide: [Leveraging system/metaengine](../../docs/guides/leveraging-system-metaengine.md).
