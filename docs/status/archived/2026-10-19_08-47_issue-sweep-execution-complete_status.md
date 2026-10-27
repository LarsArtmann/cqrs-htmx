# Status — open-issue sweep EXECUTION COMPLETE (all closes posted, master green)

> **ANNOTATED 2026-10-24** — ~~this report~~ **SUPERSEDED** by [`2026-10-19_12-18_issue-sweep-self-review-and-next-50_status.md`](2026-10-19_12-18_issue-sweep-self-review-and-next-50_status.md) (same series, later in the
> session). The successor re-covers every section with the outcome, so nothing here is a
> unique open item. Strike depth: supersession verdict (round-18 precedent for
> series-internal reports). **ARCHIVED 2026-10-24 (round 22).**


> **Point in time:** 2026-10-19 08:47 CEST · **HEAD:** master `c4745247` = origin/master, **CI GREEN** (run [39211056076](https://github.com/LarsArtmann/cqrs-htmx/actions/runs/39211056076) — every job ✓: checks incl. Service-method + domain-count + codegen + link gates, 13 lint legs, test + all 10 coverage legs, build, security, module-architecture)
> **Prior:** `docs/status/2026-10-19_04-38_issue-sweep-execution-train-complete_status.md` (mini-train session) · Plan: `docs/planning/2026-10-19_03-00_SUPERB-open-issue-sweep-pareto-plan.md` (PROCESSED-annotated)

## a. Issue board (11 open at sweep start → 1 open by design)

| # | Outcome |
| --- | --- |
| #13, #21, #22, #27, #31 | CLOSED with evidence (verified already fixed; earlier session) |
| #20 | CLOSED — verdict "2 of 3 graduated" (earlier session) |
| #25 | OPEN by design — demand-gated, trigger condition + #23-precedent comment posted |
| #28 | CLOSED — `WithRequestFilter`/`RequestFilterFunc` `1909bda8` (comment posted after the auto-close) |
| #29 | CLOSED — `transport.TailReader` `4b679996` |
| #23 | CLOSED — auth-audit journal landed via identity-model v4.13.0 mini-train; design comment (Auth-stream placement, best-effort, known-users-only, zero-actor probes, bounded reasons, authAuditor/ADR-0038) |
| #18 | CLOSED — `TypedPayloadRenderer[P]` + `RegisterPayloadType[P]` registry, typed-first chain, decode-mismatch fallback |
| #19 | CLOSED — `Config.EmptyStateCopy`, 10 typed keys, field-wise merge, byte-identical defaults (goldens) |

`origin/wip/issue-23-auth-audit` deleted (local + remote).

## b. This session's delta (after the 04:38 report)

1. **Push + CI heal cascade (4 sequential unmaskings — gotcha 40 proven live again):** service-method ceiling (81>80) → domain-count drift (FEATURES.md 21→24) → markdown link (status report's `docs/planning` needed `../planning`) → codegen drift (ambient templ v0.3.1020 regen vs canonical `.#gen`). Each fix verified before the next push.
2. **ADR-0038 fix:** `authAuditor` collaborator extraction (usermgmt/auth_audit.go) — Service 81→77 methods, OAuth2Service loses store/bus fields for the auditor; `NewOAuth2Service` signature unchanged; all call sites + tests migrated; hermetic tests + lint green.
3. **C005 fix:** LoginFailed `Detail` decode moved to `event.DecodePayloadAuto[LoginFailedPayload]` (audit_log.go) — clears the cqrs-lint critical.
4. **#18 implemented** (dashboardui/core/payload.go + core_bridge aliases + Config.PayloadTypes seam): typed registry consulted before the untyped renderer; nil config = package registry; race-tested.
5. **#19 implemented** (dashboardui/empty_state.go + 6 templ files + pageData threading): 10 EmptyStateKey surfaces, validate/resolve in withDefaults, overrides merge field-wise; golden tests prove byte-identical defaults; e2e override test.
6. **Drift bills:** branching-flow +5 adjudicated (2 new payload-struct mixins under the standing reject verdict, 2 anchor shifts) + baseline re-pinned + triage amendment; 21→24 event-count sweep across FEATURES/README/ROADMAP/metaengine-guide/CONTRIBUTING/systemadapter-README; erraudit TOTAL=0; codegen drift fixed via `.#gen`.

## c. Honest tail / carry

- The #18/#19 feature commits ride master as 4 daemon heuristic chunks (titles accepted debt per gotcha 4; the HEAD of each wave carries the proper title). Tag debt: dashboardui + usermgmt (+ root) have untagged consumer-visible code (auth journal was already trained via identity-model v4.13.0 + consumers; dashboardui #18/#19 + usermgmt authAuditor ride the NEXT patch train — same posture as the documented untagged-debt rows).
- `ReadLastN` on the SQL event store routed to TODO_LIST as an upstream ask (SQLEventStore is an alias over go-cqrs-lite/storage/v4).
- Machine load 77+ starved 2 BuildFlow pre-commit runs (documented noise; content verified by the real gates + CI).
- 3 open owner questions stand in the 04:38 report §f (root-tag cadence; impersonated-session attribution; unknown-attempt knob).
