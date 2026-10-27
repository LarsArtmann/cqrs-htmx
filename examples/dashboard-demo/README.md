# dashboard-demo

Runnable showcase of the [dashboardui](../../dashboardui/README.md)
event-store observability dashboard: projection health, event catalog, live
SSE updates, and time-travel over an in-memory event store, mounted at
`/dashboard`.

```bash
GOEXPERIMENT=jsonv2 go run .   # http://localhost:8098
```
