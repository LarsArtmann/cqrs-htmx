# go-daemon Adoption Report (NOT adopted — correctly; systemd note delivered)

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/LarsArtmann/go-daemon` v0.4.0 · Checkout: `/home/lars/projects/go-daemon`

## Verdict

| Direct usage       | Transitive | Latest                       | Verdict                   |
| ------------------ | ---------- | ---------------------------- | ------------------------- |
| **0 import sites** | no         | v0.4.0 (verified 2026-11-14) | **Correctly not adopted** |

## What it is

Long-lived daemon building blocks over unix domain sockets: `SocketServer` lifecycle (stale-socket cleanup, sd_notify READY/STOPPING/WATCHDOG, graceful drain), socket/tCP HTTP clients, JSON/CBOR content negotiation, WHATWG-conformant SSE parsing (`ParseSSEData`), a flight recorder, liveness probes. Mechanism, not policy — extracted from project-discovery-daemon.

## Why cqrs-htmx should NOT adopt it

1. **Wrong layer**: cqrs-htmx serves HTTP over consumer-owned servers (`httputil.NewServer`, appkit, or the consumer's own); socket lifecycle and systemd integration are deployment concerns of applications, not a web library.
2. **Redundant overlaps**: SSE framing is served by go-sse (server-side, where cqrs-htmx lives — `ParseSSEData` is a client-side parser); flight recording already enters the graph via `go-flightrecorder` (go-cqrs-lite otel stack); content negotiation is handled per-contract in root handlers.
3. **Library principle**: pulling a unix-socket daemon framework into consumer graphs to use none of it is exactly the dependency tax cqrs-htmx refuses.

## The one worthwhile note — delivered since the last audit

The 2026-11-05 report recommended a "running under systemd" note pointing consumers at sd_notify helpers around `setup.Run`. It landed — better than proposed: [docs/guides/production-readiness.md](../../guides/production-readiness.md) §9 now wires `daemon.SdNotifyReady`/`SdNotifyStopping` through `Config.AppkitStartHooks`/`AppkitDrainHooks` (setup's own hook seams) via `github.com/coreos/go-systemd/v22/daemon` directly — no go-daemon import needed, ever. Nothing outstanding here.

## How to verify

```bash
rg -n 'larsartmann/go-daemon' --type go -g '!scripts/**' . | wc -l   # 0
rg -n 'SdNotify' docs/guides/production-readiness.md | head -2        # the delivered §9 note (coreos/go-systemd + appkit hooks)
git -C ~/projects/go-daemon for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -1
```

## Prior reports

- 2026-11-05 (superseded by this refresh) — same verdict; its open item (systemd docs note) shipped 2026-11-06.
