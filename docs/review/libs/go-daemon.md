# go-daemon Adoption Report (NOT adopted — correctly, with one documentation note)

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/LarsArtmann/go-daemon` v0.4.0 · Checkout: `/home/lars/projects/go-daemon`

## Verdict

| Direct usage       | Transitive | Latest                       | Verdict                   |
| ------------------ | ---------- | ---------------------------- | ------------------------- |
| **0 import sites** | no         | v0.4.0 (verified 2026-11-05) | **Correctly not adopted** |

## What it is

Long-lived daemon building blocks over unix domain sockets: `SocketServer` lifecycle (stale-socket cleanup, sd_notify READY/STOPPING/WATCHDOG, graceful drain), socket/tCP HTTP clients, JSON/CBOR content negotiation, WHATWG-conformant SSE parsing (`ParseSSEData`), a flight recorder, liveness probes. Mechanism, not policy — extracted from project-discovery-daemon.

## Why cqrs-htmx should NOT adopt it

1. **Wrong layer**: cqrs-htmx serves HTTP over consumer-owned servers (`httputil.NewServer`, appkit, or the consumer's own); socket lifecycle and systemd integration are deployment concerns of applications, not a web library.
2. **Redundant overlaps**: SSE framing is served by go-sse (server-side, where cqrs-htmx lives — `ParseSSEData` is a client-side parser); flight recording already enters the graph via `go-flightrecorder` (go-cqrs-lite otel stack); content negotiation is handled per-contract in root handlers.
3. **Library principle**: pulling a unix-socket daemon framework into consumer graphs to use none of it is exactly the dependency tax cqrs-htmx refuses.

## The one worthwhile note (docs, not code)

`NotifyReady`/`NotifyStopping`/`Watchdog` are standalone sd_notify helpers for **any** `http.Server`. A short "running under systemd" note in [docs/guides/production-readiness.md](../../guides/production-readiness.md) — pointing consumers at go-daemon's sd_notify helpers (or their own `sd_notify` call) around `setup.Run` — would cost nothing and close a real operational gap for systemd deployments. No dependency required.

## How to verify

```bash
rg -n 'larsartmann/go-daemon' --type go -g '!scripts/**' . | wc -l   # 0
rg -n 'sd_notify|NotifyReady' docs/ .agents/ 2>/dev/null | head     # currently absent — the docs note's home
git -C ~/projects/go-daemon ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- None.
