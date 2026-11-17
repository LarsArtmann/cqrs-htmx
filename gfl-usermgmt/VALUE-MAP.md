# GFL value map across cqrs-htmx

Where GFL (the [graph-features-language](https://github.com/LarsArtmann/graph-features-language)
toolchain behind this folder) pays off across this repository, and — just as
importantly — where it deliberately provides nothing. Source: the
2026-11-15 survey plan (`graph-features-language/docs/planning/2026-11-15_08-10_SUPERB-gfl-cqrs-htmx-value-map.md`),
kept here as the durable per-repo view. Status column reflects execution
through 2026-11-15 (V01–V04 of that plan landed).

| Module / surface                                                          | Journal surface?                   | Verdict               | Why / candidate                                                                                                                                                      | Status          |
| ------------------------------------------------------------------------- | ---------------------------------- | --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------- |
| usermgmt deciders (4)                                                     | yes                                | **DONE**              | this folder: 32/32 scenarios green (23/23 parity + 9 read-model/audit extras)                                                                                        | ✅              |
| usermgmt error paths (12 ThenError tests)                                 | yes                                | **DONE**              | `error`/`errorFamily` bindings → `.ThenError(...)`                                                                                                                   | ✅ 2026-11-15   |
| systemadapter read models                                                 | views                              | **DONE**              | AuditEntriesFor (incl. tenant-events-out asymmetry pinned), cross-tenant memberships, FindBotByTokenHash, FindUserByEmail — all asserted in this folder (2026-11-15) | ✅ 2026-11-15   |
| DeleteTenant cascade                                                      | yes (shaped)                       | **DONE**              | ADR-0154 deriver in systemadapter + cascade scenario (ThenCommands); stale-view race A/B-proven and pinned                                                           | ✅ 2026-11-15   |
| DeleteUser cascade (memberships+bots+Casbin)                              | yes (shaped)                       | **HIGH/L**            | anchors: service_delete_cascade_test.go                                                                                                                              | open            |
| root eventcatalog/ package                                                | is a universe                      | **DONE**              | `GET /events/catalog` serves the generated artifact byte-identically (T03)                                                                                           | ✅ 2026-11-15   |
| catalog-demo (examples)                                                   | schema-only → executable           | **DONE**              | order-saga fixture + generated suite green (`-race`); committed MDX tree is now the GFL export incl. scenario flows (T07)                                            | ✅ 2026-11-15   |
| AuthAudit flows                                                           | service-layer                      | **MED/M**             | needs an AuthAudit command surface (register→LoginSucceeded+SessionStarted, ...)                                                                                     | open            |
| sync push/pull journal half                                               | yes                                | **MED/M**             | idempotent commandId, cursor ordering (sync_push.go)                                                                                                                 | open            |
| authz dispatch postures                                                   | no (policy)                        | **MED/M**             | catalog-shaped table; needs a GFL posture-metadata extension (GFL 2.0 gate)                                                                                          | design-first    |
| datastar-demo Todo                                                        | hand-rolled                        | **MED/M**             | tutorial-grade; needs decider rebuild first                                                                                                                          | open            |
| system-demo deltas                                                        | subsumed                           | **MED/S**             | audit projection step + engine presets only                                                                                                                          | open            |
| adminui                                                                   | no                                 | **MED doc-only**      | admin-scope feature graph, nothing executable                                                                                                                        | open            |
| dashboardui                                                               | introspection                      | **integration point** | renders event catalogs — future GFL-catalog feed, NOT a BDD target                                                                                                   | spike note open |
| bdd_test.go / bdd_realtime (root)                                         | no (HTTP/SSE)                      | **LOW**               | assertions are headers/frames — GFL gap by design; story prose → flows doc value                                                                                     | negative space  |
| transport/, integration_test/                                             | journal-backed, wire-asserted      | **LOW**               | SSE/HTML assertions                                                                                                                                                  | negative space  |
| totp/ webauthn/ oauth2/                                                   | none (verified: zero journal deps) | **NONE**              | honest negative space                                                                                                                                                | —               |
| e2e/ (Playwright), middleware, observability, huma, health, auditlog, ... | none                               | **NONE**              | negative space                                                                                                                                                       | —               |

## Negative space (a deliverable, not a backlog)

GFL deliberately asserts journals, read models, and derived commands
through `system.New`. It provides nothing for: browser behavior (e2e/),
SSE wire bytes (bdd_realtime, transport/), HTTP headers/bodies (bdd_test.go,
huma), health probes, DI wiring, UI rendering. Where GFL prose still helps
those: user-story and scenario narratives → EventCatalog flows
(documentation, not execution) — see `gfl export eventcatalog --bindings`.

Nothing in the NONE rows should be "GFL-ified" for the sake of it.
