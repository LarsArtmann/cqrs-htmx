# go-daemon Adoption Report (NOT adopted — correctly; systemd note delivered)

> Audit date: 2026-11-22 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/LarsArtmann/go-daemon` v0.4.0 · Checkout: `/home/lars/projects/go-daemon`

## Verdict

| Direct usage       | Transitive | Latest                       | Verdict                   |
| ------------------ | ---------- | ---------------------------- | ------------------------- |
| **0 import sites** | no         | v0.4.0 (verified 2026-11-22) | **Correctly not adopted** |

## What it is

Long-lived daemon building blocks over unix domain sockets: `SocketServer` lifecycle (stale-socket cleanup, sd_notify READY/STOPPING/WATCHDOG, graceful drain), socket/tCP HTTP clients, JSON/CBOR content negotiation, WHATWG-conformant SSE parsing (`ParseSSEData`), a flight recorder, liveness probes. Mechanism, not policy — extracted from project-discovery-daemon.

## Why cqrs-htmx should NOT adopt it

1. **Wrong layer**: cqrs-htmx serves HTTP over consumer-owned servers (`httputil.NewServer`, appkit, or the consumer's own); socket lifecycle and systemd integration are deployment concerns of applications, not a web library.
2. **Redundant overlaps**: SSE framing is served by go-sse (server-side, where cqrs-htmx lives — `ParseSSEData` is a client-side parser); flight recording already enters the graph via `go-flightrecorder` (go-cqrs-lite otel stack); content negotiation is handled per-contract in root handlers.
3. **Library principle**: pulling a unix-socket daemon framework into consumer graphs to use none of it is exactly the dependency tax cqrs-htmx refuses.

## The one worthwhile note — delivered and grown a seam since

The 2026-11-05 report recommended a "running under systemd" note pointing consumers at sd_notify helpers around `setup.Run`. It landed — and grew: [docs/guides/production-readiness.md](../../guides/production-readiness.md) §9 "systemd Readiness Signals" (`:108`) now wires `daemon.SdNotify(false, daemon.SdNotifyReady)` (`:119`) and `SdNotifyStopping` (`:123`) through **three** hook seams via `github.com/coreos/go-systemd/v22/daemon` (`:114`) — `Config.AppkitStartHooks` (`setup/config.go:364`), `Config.AppkitDrainHooks` (`:372`), and `Config.AppkitShutdownHooks` (`:381`, a seam added since the 2026-11-14 audit). Still no go-daemon import needed, ever. Nothing outstanding here.

Upstream drift since the tag (HEAD `b6af9f1`, 2026-11-18): the CHANGELOG `[Unreleased]` section adds `JSONv2Codec` and injectable negotiation codecs (`WithJSONCodec`/`WithCBORCodec`, CHANGELOG.md:12-17) — content-negotiation depth that does not touch the non-adoption logic (cqrs-htmx negotiates per-contract in root handlers). And v0.4.0's standalone sd_notify API (`NotifyReady`/`NotifyStopping` without a `SocketServer`) would technically replace the guide's two-line coreos wrappers — but importing a unix-socket daemon framework for two no-op-aware constants remains worse than the two lines; the flip trigger is untouched.

## How to verify

```bash
rg -n 'larsartmann/go-daemon' --type go -g '!scripts/**' . | wc -l   # 0
rg -l 'larsartmann/go-daemon' -g 'go.mod' -g 'go.sum' . | wc -l    # 0 — absent from the whole module graph
rg -n 'SdNotify' docs/guides/production-readiness.md | head -2        # :119 SdNotifyReady, :123 SdNotifyStopping (coreos/go-systemd + appkit hooks, §9 at :108)
rg -n 'AppkitStartHooks|AppkitDrainHooks|AppkitShutdownHooks' setup/config.go | grep -v '//' | head -3   # :364, :372, :381 — the three hook seams
git -C ~/projects/go-daemon for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -1   # v0.4.0
rg -n 'JSONv2Codec' ~/projects/go-daemon/CHANGELOG.md | head -1      # :12 — [Unreleased] negotiation codecs, no bearing on the verdict
```

## Prior reports

- 2026-11-14 (prior state of this file): correctly-not-adopted, no score (this report's convention) · flip triggers re-hold — §9 now wires three seams (`AppkitShutdownHooks` at `setup/config.go:381` is new); upstream `[Unreleased]` at HEAD `b6af9f1` (2026-11-18) adds negotiation codecs only.
- 2026-11-05 (superseded by this refresh) — same verdict; its open item (systemd docs note) shipped 2026-11-06.
