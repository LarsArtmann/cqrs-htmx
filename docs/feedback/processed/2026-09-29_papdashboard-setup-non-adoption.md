# Consumer Feedback — Why PapDashboard Does NOT Use `setup` (and What Would Change That)

> **PROCESSED** (2026-09-29): All 7 items triaged, every claim verified against source first. **Shipped:** #2 `Config.ExtraMiddleware` + `Config.DisableSecurityMiddleware` and #3 `Config.HealthChecks` (commit `a3699cd8`, 9 tests, zero-value = legacy chain); #6 codec/v4→go-codec migration in usermgmt (commit `d8684d5c`; setup's transitive indirect drops at the next usermgmt tag). **Documented:** #5 capability-floor table in `setup/README.md`; #7 decision doc `docs/guides/setup-vs-hand-wiring.md` (commit `4ef0f03e`). **ROADMAP:** #1 setup/core split (OQ 23, acceptance criteria recorded verbatim) and #4 pluggable SSE envelope (OQ 24). One unresolved reference: the cited upstream #67/#68 issue numbers resolve in neither go-cqrs-lite nor cqrs-htmx; the underlying codec work was confirmed real via go-cqrs-lite ADR-0128. Open owner calls surfaced: train policy for the usermgmt re-tag, and whether to build OQ 23/24 now or hold for a second consumer.

**From:** PapDashboard project (github.com/larsartmann/papdashboard — private, event-sourced notification hub)
**Date:** 2026-09-29
**Version evaluated:** cqrs-htmx/v4 v4.12.0 family train (2026-09-22); `setup/v4` read at local checkout HEAD
**Consumer:** Crush (AI assistant) + Lars
**Provenance:** full surface audit 2026-09-29 (`docs/status/2026-09-29_09-01_cqrs-htmx-surface-audit_status.md` on the PapDashboard side; decision gates AUTH-G1, HTMX-G1/G2 in its DECISION-QUEUE)

---

## Who we are (the fit context)

PapDashboard is a **one-operator-household** CQRS/ES notification hub: one shared API key (`PAP_API_KEY`, constant-time Bearer/`X-Api-Key`) plus a hand-rolled 7-day HttpOnly cookie session for the browser (EventSource cannot send headers, so the cookie is the only browser-usable SSE credential). No user accounts, no tenants, no roles — key possession IS authorization. Hosting is go-appkit (graceful drain, `/health` trio), DI is samber/do, the API surface is Huma v2 with a generated OpenAPI spec, the store is a hand-built SQLite event store bridged to go-cqrs-lite `event/v4` via an adapter that implements **only `event.Store`**.

Current cqrs-htmx usage: **exactly one symbol** — `HTMXScriptHandler()` at the `/htmx.js` route — now enforced by a source-scan pin test, because prose claims about dependency scope rot silently.

## What setup does well (credit where due)

- **`doc.go` is exemplary** — the one-call pitch, the capability list, the customization ladder (`Run` → `RunHandler` → `Mount`+`Middleware`), and the persistence defaults are all documented honestly ("in-memory by default, lost on restart").
- **Fail-fast config validation at `New`** (path collisions, reserved `/`, trailing-slash normalization) — misconfiguration surfaces before `Mount` can panic. We pin similar contracts by test on our side; doing it in the library is better.
- **Every sub-component exposed on the Bundle** for post-construction override; `Config.Service` adoption mode with explicit conflict rejection ("nothing is silently ignored") is the right escape-hatch design.
- **AsyncStartup + readiness gating** (503 while projections drain) is a genuinely good idea we recognize from our own drain work.

None of that is the problem. The problem is the shape of our app.

## Why we are not using setup (ranked by weight)

### 1. The identity service is unconditional — and identity is 100% dead weight for us

`setup.New` calls `usermgmt.NewService(...)` unconditionally (`setup/setup.go:123`). `DisableAdmin` / `DisableDashboard` / `DisableLogin` skip **panels**, never the service. For an app with zero users this means adopting setup runs a full event-sourced identity stack — its own event store, projections, checkpoints, eviction goroutines — as permanent idle ballast, alongside the real domain it would host. We are the wrong app for that today, and setup offers no lane for us.

### 2. setup is a composition root; we already deliberately built (and pinned) ours

Adopting `bundle.Run`/`RunWithAppkit` means replacing go-appkit's lifecycle (drain-aware readiness), our samber/do eager-materializer health graph, and our route registration with setup's serving path and its own `/health` semantics. These aren't hypothetical preferences — they are frozen by invariant and lifecycle test suites on our side. A composition root competes with an existing composition root; it does not compose with one.

### 3. Auth model mismatch is structural

usermgmt sessions (24h cookie default, WebAuthn/TOTP/OAuth2 ceremonies, Casbin) vs our one-key + cookie model whose exact attributes, constant-time compare, restart-revocation semantics, and rate limits are test-pinned. Bridging would mean either maintaining two auth truths or unpicking pins for no user-visible gain.

### 4. The SSE wire contracts are both frozen — and different

setup's shared `/sse` feed speaks the `transport/` envelope (`DomainEventToSSE`, Last-Event-ID replay, heartbeat frames — golden-pinned on your side). Our stream is an 11-event, capture-first, bare-`data:` huma envelope with the type inside the JSON payload — golden-pinned and compat-fixture-gated on ours. Two frozen contracts that disagree is a hard veto for either party, by design.

### 5. Interface floor is above what our store bridge exposes

Our adapter implements `event.Store` only. `/sse` replay expects `SeekableJournal` (with a `ReadAll`+filter degradation — appreciated, but under-documented as the support floor), and the observability panels light up per-interface (`Journal`, `StreamReader`, `projectionhost`, journals we deliberately do not run). Out of the box, most of what setup coordinates would sit dark.

### 6. Module graph and build-system cost

`setup` transitively wants `usermgmt` + `identity-model`, whose go.mods still carry the deleted `codec/v4` requires (upstream #67/#68 — already our recorded adoption prerequisite). And in our Nix flake every new `github.com/larsartmann/*` module path costs a `publicDeps` entry plus a `vendorHash` re-derivation. The dance is affordable for value; it is not affordable for ballast.

### 7. We already own the observability surface it would add

`/api/audit`, generated AsyncAPI 3.0 + EventCatalog artifacts, projection-lag health checks. setup's panels are nicer than our fragments — but nicer-for-duplicate is still duplicate.

## What setup could improve to change our mind

Ranked by how much each would move a consumer shaped like us. Note the honest headline first: **our non-adoption is mostly "we are not the target app yet"** — single-user apps were never setup's pitch. These improvements are what would make setup relevant to that class anyway.

### 1. Split `setup` into `setup/core` + identity wiring (the big one)

The one-call value minus the identity assumption: a core bundle that wires event infra, health, SSE hub, and chosen panels **without importing usermgmt**, with the current identity wiring layered on top as the full `setup`. Acceptance from our side: `go.mod` gains `setup/core` (and its panel deps) but NOT `usermgmt`/`identity-model`; `setup.New`-equivalent boots with zero identity goroutines. This single change converts "unconditional ballast" into "composable infrastructure", and it would also make #67/#68 residue irrelevant for non-identity consumers.

### 2. Consumer-owned middleware chain

Expose the assembled chain as a configurable list (per-built-in skip flags + `ExtraMiddleware` in a defined position), not just per-panel delegates to `RecommendedSecurityMiddleware`. Apps with an ordered security chain — ours is `CORS → BodyLimit → APIKeyAuth → OriginGuard → RateLimit → Metrics → CSP` — must inject precisely, in order, or not adopt.

### 3. Health as injectable checks alongside the owned path

`HealthPath: "-"` (off) already exists — good, and it is the right escape hatch (config.go:708–711, mount.go:95). The remaining gap: no way to feed extra checks INTO the mounted `/health` (our `/api/health` sweeps `NamedCheck`-style checks with per-check deadlines and documents 200-always-degraded semantics for external probers). A `Config.HealthChecks []NamedCheck` seam would let consumers keep one health surface instead of choosing between setup's 503-while-draining contract and their own on the same port.

### 4. Pluggable SSE envelope/encoder

Both sides froze different wire shapes. An envelope seam (or a documented bare-`data:` mode where the type rides inside the JSON) removes the wire-contract veto for consumers who already shipped a stream. We'd still have to migrate or dual-envelope — but it becomes a decision, not an incompatibility.

### 5. Document the capability floor per feature

A short table in the setup docs: which features degrade gracefully (`SeekableJournal` → `Journal` fallback) vs silently stay dark (panels behind unimplemented interfaces). Bridge authors currently discover this by reading `core_bridge.go`.

### 6. Ship #67/#68

The codec/v4 cleanup is table stakes for any adoption path that touches usermgmt — it is the recorded flip-trigger prerequisite in two of our gates.

### 7. A "setup vs hand-wiring" decision doc

The honest decision tree — including "if you use one symbol, vendor the embed and skip us" — would have saved our audit a session and would build trust precisely with the consumers least likely to adopt today.

## What would actually flip us

Recorded on our side (ADR 0001 / DECISION-QUEUE AUTH-G1, HTMX-G1): **SaaS/multi-user mode activating** — the day PapDashboard needs real accounts, tenants, or roles, usermgmt becomes a requirement instead of ballast, and setup's one-call wiring is exactly the right shape for that day. Improvement #1 (core/identity split) is what would make us adopt setup-shaped infrastructure *before* that day for the parts we already hand-roll (SSE hub, readiness coordination); #2 and #3 are what would let it live inside our existing stack rather than replacing it.

Until then: not setup's fault, not a quality verdict — a target-audience mismatch with a pin test on our side to keep the dependency honest.
