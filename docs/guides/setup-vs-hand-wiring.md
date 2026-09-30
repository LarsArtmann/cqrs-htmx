# setup vs Hand-Wiring: A Decision Guide

> Which integration lane fits your app: the one-call `setup/v4` bundle, manual
> wiring of the individual modules, or skipping the library almost entirely.
> Written to be honest about when `setup` is the WRONG answer — including for
> apps that are otherwise happy cqrs-htmx consumers.

Origin: consumer feedback from the PapDashboard project (2026-09-29), whose
audit concluded "not setup's fault, not a quality verdict — a target-audience
mismatch". This guide is the decision tree they asked for, so the next
consumer does not need a full audit session to reach the same answer.

## The decision tree

```
How much of cqrs-htmx does your app actually use?
|
+- One or two symbols (e.g. just HTMXScriptHandler)
|    -> VENDOR THE ASSET AND SKIP THE LIBRARY. htmx.js is a standalone
|       script; embedding it yourself is ~4 lines and zero module-graph
|       cost. A dependency for one embed is ballast, not value.
|
+- Middleware / SSE building blocks / error mapping only (no App)
|    -> PATH 0: root module, no setup. You get Chain, Broadcaster,
|       SSEStream, MapError, RecommendedSecurityMiddleware — the same
|       primitives setup itself composes.
|
+- CQRS command/query endpoints, no user accounts
|    -> PATH A: root App + your dispatchers. setup would run a full
|       identity stack you never use (see below).
|
+- Accounts/auth + your own route layout, admin optional
|    -> PATH B: root App + usermgmt.Service + AuthHandler.
|       setup is fine here too IF its route map and serving fit.
|
+- Accounts + admin panel + dashboard, greenfield or setup-shaped
|    -> setup.New: this is the target app. One call, every panel,
|       readiness gating, sane middleware.
```

## The truths to weigh before adopting setup

### 1. The identity service is conditional since ADR-0054

> **ANNOTATED 2026-09-30:** `Config.DisableService` now builds the
> identity-external shell (no usermgmt.Service, no auth handler, no
> panels; stores from `Config.EventStore`/`EventBus`), and
> `Config.DisableAuth` drops the `/auth/*` routes while the service
> stays (own-login-endpoint mode). The session-gated surfaces (feeds,
> machine endpoints) are rejected in shell mode until an injectable
> gate lands — apps needing gated feeds still weigh the original
> section below.

`setup.New` constructs (or adopts) a `usermgmt.Service` — its event store,
projections, checkpoints, and eviction goroutines. `DisableAdmin`,
`DisableDashboard`, and `DisableLogin` skip **panels**, never the service —
unless `DisableService` opts out of the identity assumption entirely
(ADR-0054, `docs/adr/0054-identity-external-shell-mode.md`). For an app
with zero users (single-operator tools, key-authenticated services) that
still needs the session-gated feeds or machine endpoints, that stack is
permanent idle ballast running beside your real domain, and those apps
should stay on Path 0/A.

### 2. A composition root competes with yours; it does not compose with one

`bundle.Run` / `bundle.RunWithAppkit` own serving, shutdown, and the
`/health` contract. If your app already has a frozen lifecycle (appkit drain
semantics, DI-container health graphs, route registration suites), adopting
the Run paths means replacing it — usually a veto. The compatible seam is
`bundle.Mount(mux)` + `bundle.Handler(mux)`: setup mounts routes, you keep
your server and lifecycle. The HTTP chain is no longer all-or-nothing
either:

- `Config.ExtraMiddleware` — your middleware, composed inside the built-in
  security stack (defined position, listed order).
- `Config.DisableSecurityMiddleware` — remove the built-in layer entirely;
  you rebuild the chain you want in `ExtraMiddleware`.
- `Config.HealthChecks` — feed your dependency checks into the mounted
  `/health` instead of running a second probe on the same port.

If your chain must live OUTSIDE everything setup builds and health must
stay on your own endpoint, `HealthPath: "-"` plus `Mount` into your own
mux keeps setup out of your request path entirely.

### 3. Auth model: usermgmt sessions or yours

setup's panels and API assume usermgmt sessions (cookie, 24h default,
WebAuthn/TOTP/OAuth2 ceremonies, Casbin roles). Apps with a different
pinned auth truth (API keys, hand-rolled cookies, mTLS) face either two
auth realities or unpicking their own invariant tests. There is no
bridge layer today. This is the second-strongest non-adoption reason
after the identity ballast.

### 4. The SSE wire contract is frozen on both sides

setup's shared `/sse` speaks the `transport/` envelope (`DomainEventToSSE`,
Last-Event-ID replay, heartbeats) — golden-pinned in this repo. If your
existing stream shipped a different envelope (bare `data:` frames, type
inside the JSON payload), both contracts are frozen and adoption means a
client migration or a dual-envelope period. A pluggable envelope/encoder
seam is on the ROADMAP; until then, consumers with a shipped stream
should keep serving it themselves (Path 0 building blocks compose fine
next to setup — see `sse-and-datastar.md` for hub sharing).

### 5. Your store's capability floor decides what lights up

`setup.New` requires `event.SeekableJournal` from a custom `EventStore`.
Everything else degrades per-interface (paginated event log, aggregate
browser, audit panels). The full table — what each interface unlocks and
what stays dark — lives in `setup/README.md` § Capability floor. Bridge
authors: read that table BEFORE adopting; it is the support floor, not a
detail.

## When to move UP the ladder

The recorded flip trigger from the PapDashboard audit generalizes: **the
day your app needs real accounts, tenants, or roles, usermgmt stops being
ballast and becomes the requirement** — and setup's one-call wiring is the
right shape for exactly that day. Moving up later is cheap: the modules
compose (your Path 0/A code keeps working inside a setup app), so the
decision is reversible in the direction that matters.

## Quick reference

| Lane        | Import                            | Identity stack | You own                                   |
| ----------- | --------------------------------- | -------------- | ----------------------------------------- |
| Vendor-only | (none — embed htmx.js yourself)   | no             | everything                                |
| Path 0      | root                              | no             | routes, chain, SSE handlers               |
| Path A      | root                              | no             | routes, chain, dispatchers                |
| Path B      | root + usermgmt (+ strategy mods) | yes, yours     | routes, chain, serving                    |
| setup       | setup (+ strategy mods)           | yes, bundle's  | business routes; chain & health via seams |
