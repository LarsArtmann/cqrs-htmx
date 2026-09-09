# async-startup-demo

One-call `setup.New` with `AsyncStartup: true` — the HTTP server binds
immediately while projections replay the journal in the background.

- `GET /health` returns **503** while any projection is draining, **200**
  once every worker is live (readiness gate: `cqrshtmx.ProjectionReadinessCheck`,
  mounted automatically by the bundle).
- Point your reverse proxy's health check at `/health` so restarts of apps
  with large journals serve 503 (retry-later) instead of refusing connections.

Guide: `docs/guides/async-projection-startup.md`.

## Run

```bash
cd examples/async-startup-demo
go run .
```

Open http://localhost:8098/health and watch 503 → 200.
