# Roadmap — cqrs-htmx

> Long-term direction and raw ideas not yet refined into actionable tasks.
> For short-term work, see [TODO_LIST.md](TODO_LIST.md).
> For what exists today, see [FEATURES.md](FEATURES.md).
> For completed work, see [CHANGELOG.md](CHANGELOG.md).

**Updated:** 2026-07-22 | **Version:** v4.3.0+unreleased (go-cqrs-lite v4.0.x; see AGENTS.md for per-sub-module versions)

## Current State

- **Version:** v4.3.0+unreleased (12 modules: root + usermgmt + 3 auth sub-modules + adminui + loginpage + integration_test + 4 examples)
- **Coverage:** 93.8% root, 80.2% usermgmt, 88.2% totp, 89.2% webauthn, 88.3% oauth2, 69.0% adminui, 80.1% loginpage (~920 tests), race-safe. CI gates: root 90%, usermgmt 74%, auth 80%, adminui 66%, loginpage 80% (see `nix run .#coverage-gate`)
- **Lint:** 0 issues across all linted modules
- **ErrorFamily:** 0 violations across all modules (sub-modules adopted go-error-family directly in v4.2.0)
- **Dependencies:** go-cqrs-lite v4.0.x (sub-modules v4.0.0–v4.0.2), go-error-family v0.7.0, go-branded-id v0.3.2, httputil v0.5.0, templ-components v0.16.0. Auth deps (go-webauthn, oauth2, oidc, pquerna/otp) are in optional sub-modules — core usermgmt has ZERO auth deps
- **Architecture:** Fully event-sourced usermgmt (12 events, 20 commands, Decider pattern, WebAuthn passwordless, OAuth2/OIDC, multi-tenancy, bot accounts, membership RBAC, impersonation, checkpoint-based projection replay). Auth strategies extracted behind interfaces (ADR-0035). loginpage module (passwordless login UI). adminui module (templ+HTMX dashboard).
- **Modules:** 12 Go modules in go.work (root, usermgmt, usermgmt/totp, usermgmt/webauthn, usermgmt/oauth2, adminui, loginpage, integration_test, 4 examples)

---

## Upstream Adoption & Scale

_Focus: Adopting go-cqrs-lite capabilities to reduce hand-rolled code._

| Area | Item                                                               | Priority | Status                                                          |
| ---- | ------------------------------------------------------------------ | -------- | --------------------------------------------------------------- |
| ES   | Adopt `projectionhost/v3` — replace hand-rolled `StartProjections` | High     | Planned (checkpoint replay shipped in v3.3.0 as interim fix)    |
| ES   | Adopt `CatchUpSubscriber` — ordered durable projections            | Medium   | Planned (ADR-0031 Accepted; deferred — needs sync-wait wrapper) |
| Perf | Profile and optimize hot paths (dispatch, decode)                  | Low      | Planned                                                         |
| Perf | Benchmark projection replay with large stores (10K+ events)        | Low      | Planned                                                         |

---

## Not Planned

These are explicitly out of scope for this library:

- **WebSocket upgrade logic** — Consumers should use dedicated libraries (gorilla/websocket, coder/websocket, etc.). The library provides protocol helpers (`WSMessage`, `WSOOBHTML`) only.
- **ORM integration** — Store interfaces are intentionally simple; consumers provide their own implementations.
- **Template engine support beyond templ** — The `TemplComponent` duck-typing pattern covers any `Render(ctx, w) error` interface.
- **Built-in HTTP router** — Framework-agnostic: works with `net/http`, Gin, Chi, etc. — no router dependency.
- **TOTP management views in adminui** — This library is passwordless-first: WebAuthn passkeys + OAuth2 only. TOTP remains available as an optional sub-module (`usermgmt/totp/v4`) for consumers who genuinely want it, but the admin UI will not ship TOTP enable/disable/QR-code views. We are not building for the old-school TOTP world.
- **Redis adapters (SessionStore / OAuth2StateStore / IdempotencyStore)** — Multi-instance ephemeral-store adapters belong in go-cqrs-lite (or consumer code), not cqrs-htmx. Low consumer demand, Redis is overrated, and the existing in-memory + SQL stores cover the documented use cases. Re-open upstream if a real consumer needs it.
- **Consumer-facing v3→v4 codemod** — Automated migration tool. All known consumers are already on v4; the one-time migration is documented in `docs/migrations/v3-to-v4.md`. Building a codemod now would be cost without an audience.
- **Root module: extract SSE/WS/ratelimit into optional sub-packages** — 16 of 46 root files have zero logic coupling to the core, but they share the same go.mod = same dep tree = zero consumer benefit. Only a separate Go module would reduce transitive deps, and that is not justified by current demand.
- **Shared types module (`usermgmt/types/`)** — A cross-module types boundary would add a JSON serialization round-trip (~400ns–1.2µs per ceremony). The cost is negligible, the conceptual smell is real, but the extraction has no consumer benefit until dep-tree reduction is needed.
- **`broadcaster.ServeSSE()` high-level helper** — Crosses the "building blocks, not a server" design line. Consumers compose `Broadcaster` + `SSEStream` themselves; a one-call server helper would impose opinionated routing/response semantics this library deliberately avoids.
- **usermgmt god-package split** (domain layer extraction, SQL infrastructure extraction, Service struct split, cross-module Service-layer integration test) — Sub-package extraction within the same Go module provides zero consumer benefit: same `go.mod` = same dep tree. Clean seams are identified (20 pure fold/decide files with zero I/O, 9 SQL infrastructure files) but only separate Go modules would reduce transitive deps, and that is not justified by current consumer demand. Re-open when a consumer specifically requests a reduced dep tree.
- **`TypedRepository` / `TypedDecider` adoption across usermgmt** — Premise invalid: (1) zero command type assertions exist — `command.RegisterTyped[Cmd]` already gives fully-typed handlers (see `es_dispatch.go`); (2) `TypedDecider` binds ONE command type per repository, incompatible with usermgmt's multi-command aggregates (User has Register/ChangeEmail/AddRole/Suspend/...); (3) the current `repo.Execute(ctx, aggID, aggType, decideFn)` + per-command closure pattern is the correct, already-type-safe design for multi-command aggregates.
- **Integration test importing the published version (not local replace)** — Blocked, not rejected: the `go.work` local replaces exist precisely because published go-cqrs-lite tags carry broken zero pseudo-versions. An integration test against the published version would fail until upstream cuts a clean consolidated release (v4.0.3+ or v4.1.0). Re-open once the publishing bug is resolved.
- **Standardize import grouping** — Cosmetic defer. gofmt + goimports already enforce a consistent style; further normalization has no functional impact.
- **Automate GitHub Release creation via CI on tag push** — Manual `gh release create` is sufficient for the current release cadence; automating adds CI complexity without near-term payoff.
