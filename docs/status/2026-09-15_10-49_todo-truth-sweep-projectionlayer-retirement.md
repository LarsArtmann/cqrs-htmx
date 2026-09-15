# Status Report — TODO Truth Sweep + ProjectionLayer Retirement

**Date:** 2026-09-15 10:49 CEST | **Repo:** cqrs-htmx @ master | **Branch state:** clean except AGENTS.md/CHANGELOG.md final edits (daemon picks up)
**Session scope:** execute the pasted TODO list. Outcome: most of it was already done silently; the rest was landed this session.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | **TODO truth audit** — pasted TODO_LIST vs tree diffed item-by-item; ~12 items found already completed in prior sessions but never removed from the list (CI blocking flip `--strict-lag 0`, systemadapter unlink bug + red-first regression test, next-train alignment sweep, setup `Config.CSRF` knob + 403 rejection test, tooling debt a-d incl. `scripts/lib/replace-exemption.sh` + drift self-test, docs debt bundle incl. actor-audit guide + async-startup-demo + health/auditlog wiring tests, micro-debt b/c + loginpage polish, SSEMaxReplay validation) | every claim re-verified against code/CHANGELOG before accepting |
| 2 | **systemadapter unlink regression test verified green** | `go test ./...` 0.7s, exit 0 |
| 3 | **`TestDeclarative_SQLite_UserLifecycle`** — register→credential→link→lookup→unlink→audit→delete over the SQLite engine (first persistent-engine declarative coverage; memory never exercised SQL layout/scan) | PASS 0.02s |
| 4 | **`TestDeclarative_SQLite_AuthzPolicies`** — role policy + `Enforce` + `FindPolicyByStreamID` on SQLite | PASS 0.01s |
| 5 | **`TestDeclarative_MissingLookups`** — negative paths: every keyed query returns `system.ErrNotFound`, scan-style queries return empty, never zero-value rows (the unlink-bug regression class, now pinned) | PASS 0.01s |
| 6 | **Phase 5 migration guide** `docs/guides/declarative-projections.md` — two-path comparison, full ProjectionLayer→declarative API mapping, drain semantics + `waitForView` polling replacement, honest checkpoint/DLQ caveat | 121 lines, links gate OK |
| 7 | **`examples/system-demo` migrated off `NewProjectionLayer`** — now `sys.Start(ctx)` + internal declarative host + typed queries; found & fixed a latent demo bug en route: it NEVER called `sys.Start` (ProjectionLayer was its only host); fixed `AuditEntryView.Email` compile error (field is `AggregateID`) | build+vet green, 0 `NewProjectionLayer` refs |
| 8 | **TODO_LIST.md rewritten to truth** (59→37 lines): ~12 done items removed, post-bump item collapsed to its bench-spike remainder, hardware item corrected, ProjectionLayer item is now "v5 removal itself" | re-dated 2026-09-15 |
| 9 | **AGENTS.md corrected where it lied:** `/mnt/buildcache` mount RESTORED (was "FULLY DEAD since 2026-08-22" — stale for ~3 weeks; would have pushed future sessions onto unneeded /tmp workarounds); guides list 22→24; templ-components claim rewritten to gate-verifiable truth | freshness gate PASS |
| 10 | **CHANGELOG entries added:** Phase 5 completion, system-demo migration, docs truth sweep | `[Unreleased]` Added+Changed |
| 11 | **Gates:** systemadapter tests + golangci (0 issues), system-demo build+vet, integration_test hermetic tidy/build/vet (no-op tidy — root-caused why: indirects track PUBLISHED adminui/dashboardui tags), docs-links 252 OK, docs-freshness PASS | all exit 0 |

## b) PARTIALLY DONE

- **bench-spike idle re-run** — sole remainder of the 2026-09-10 post-bump pass; refused again this session (load **825**/32 cores from a concurrent session; `BENCH_MAX_LOAD` guard enforces). Everything else in that pass was already green.
- **systemadapter first tag** — repo-side ready (replace-strip + verify-tag path documented); hard-blocked on upstream `metaengine/projectionadapter/v4 v4.5.0` (needs `OccurredAt` on `EventWithID`).
- **CI check-\* wiring** — everything except `check-cqrs-lint` (blocked: Nix-only binary, needs Go-installable distribution).
- **templ-components uniformity** — direct consumers v1.17.0; integration_test indirects v1.16.0 (correct: published-tag graph). Catches up at the next family train; hand-bumping would break hermetic graphs.
- **appkit adoption (ADR-001)** — `RunWithAppkit` stable since 2026-09-07; the default-flip fold-in (b)-(f) remains, gated on the DataStar ADR sequencing decision.
- **DataStar Tier 4** — Tiers 1-3 shipped; M11-M16 demand-gated.

## c) NOT STARTED

- **V007 migration (usermgmt, 39 findings):** `stack.Materialize` ×3, `stack.Bundle` ×2, `storage.SQLViewStore` ×34 — v5-prep, compiles green today.
- **Upstream asks:** decouple `stack/v4` root from metaengine; teach `cqrs-upgrade` workspace mode (22 manual runs last sweep).
- **Declarative checkpoint/DLQ injection** — no `system.New` option exists; ProjectionLayer is still the only path with `WithCheckpointStore`.
- **v4-branch history rewrite** (~27.7 MB blobs) + **setup-demo purge** (27 MB) — both need force-push (user gate).
- **`examples/datastar-demo` rebrand-or-remove** — owner call.
- **SSE hardening backlog remainder** (serve_test gaps, `WithSSEMaxReplay` option, fuzz, e2e reconnect scenario, …) — explicitly optional.
- **ROADMAP residuals:** loginpage cosmetic polish, `setup.NewFromSystem` bridge decision (OQ12).

## d) TOTALLY FUCKED UP

Nothing destroyed this session — but three real process failures surfaced (two inherited, one mine):

1. **The TODO_LIST split brain lived ~6 days (inherited).** Work was landing (CHANGELOG documented it, 2026-09-09/10/14) while the TODO still showed P1 items OPEN — including items marked "OVERDUE". A session reading only TODO_LIST would have re-implemented finished work. My audit caught it, but only because the user pasted the list and said "execute"; a routine session would have trusted it.
2. **AGENTS.md lied about hardware for ~3 weeks (inherited).** "FULLY DEAD, NOT fixable" while the mount was repaired and healthy. Nobody re-verifies "impossible" claims; the stale gotcha actively wasted future-session effort (unnecessary /tmp cache workarounds).
3. **My near-miss: I almost re-implemented the CSRF knob.** My first plan (built from the pasted TODO) included it; only a pre-work grep (`rg CSRF setup/`) revealed `Config.CSRF` + tests already existed. Had I executed top-down without verification, I'd have shipped a duplicate knob and a split API.
4. **Minor (mine):** write/edit tool read-tracking cost 2 wasted round trips (bash reads don't register as file reads); the daemon committed my work in 4+ heuristic fragments (content verified intact at HEAD, but history is noisy); LSP spammed ~65 stale `infertypeargs` infos that the CLI lint contradicts (known-unreliable, correctly ignored).

## e) WHAT WE SHOULD IMPROVE

1. **Verify-then-execute, always:** every TODO item must be grepped against the tree before implementation (this session: 10/12 "open" P1/P2 items were already done).
2. **Close the loop at landing time:** CHANGELOG entry + TODO removal must happen in the SAME commit as the work — the 2026-09-09/10 sessions documented but never removed.
3. **Periodic claim re-verification:** "X is broken/unfixable" facts (hardware, blocked gates) need a re-check date, not a permanent stamp.
4. **Trust the gates:** the freshness gate caught a claim I would have shipped as truth (v1.16.0 uniformity) — its "uniform-at" strictness (counts indirects) is a feature; phrase claims to what the gate can verify.
5. **Session hygiene:** register file reads via the view tool before writes (avoid read-tracking round trips); run repo gates before the daemon sweeps intermediate states.
6. **Load awareness:** check `/proc/loadavg` before any measurement/bench work; today's 825 would have poisoned any benchmark (the guard caught it, but only because I checked first).

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

*(brainstorm, not commitments — most belong in TODO_LIST via HARVEST or ROADMAP; sorted roughly by impact)*

**Release & trains**
1. Cut the next family train (publishes `Config.CSRF`, DataStar setup API, OTel `Observability` seam; aligns integration_test indirects to v1.17.0).
2. Request/file `metaengine/projectionadapter/v4 v4.5.0` upstream → unblocks systemadapter first tag.
3. After (2): strip the last replace (systemadapter + system-demo), tag systemadapter via `verify-tag.sh`, drift exemptions drop to zero.
4. Decide systemadapter first-version number (family-train version vs independent v4.8.0 — ROADMAP OQ12).
5. Re-run `bench-spike` on an idle machine; re-pin `setup-baseline.raw.txt` if appkit-service alone trips the 10% gate.
6. CHANGELOG straddle hygiene: move leftover `[Unreleased]` bullets under version headings at the train cut.

**Decisions awaiting the user**
7. `/sse` endpoint posture (A-D one-pager ready: `docs/planning/2026-08-30_sse-endpoint-shape-decision.md`).
8. ADR-001 appkit default-flip + fold-in items (b)-(f) (health dedup, chain dedup, logging posture, `Addr()`).
9. DataStar Tier 4 go/no-go per milestone (M11-M16), each demand-gated.
10. `examples/datastar-demo` rebrand-or-remove.
11. `setup.NewFromSystem()` bridge — build or reject (ROADMAP OQ12).

**Upstream asks (go-cqrs-lite)**
12. Teach `cqrs-upgrade` multi-module/workspace mode (22 manual runs per sweep).
13. Decouple `stack/v4` root package from `metaengine/v4`.
14. Expose checkpoint/DLQ store injection on `system.New` (closes the declarative path's one honest gap before ProjectionLayer removal).

**v5 prep (usermgmt, 39 V007 findings)**
15. Migrate `stack.Materialize` ×3 → metaengine auto-projection (ADR-0123).
16. Migrate `stack.Bundle` ×2 → `system.New` composition (ADR-0123; overlaps #2/#3).
17. Migrate `storage.SQLViewStore` ×34 → metaengine engines + layout planning (ADR-0126; largest, wait for API maturity).
18. Remove `ProjectionLayer` itself in the v5 bundle (prep is now DONE).
19. Re-run `cqrs-upgrade -dry-run` after the next train to refresh the V007 list.

**Tooling & gates**
20. Extend `check-docs-freshness` to pin the NEW templ-components phrasing (direct-consumers vs indirect split is currently prose-only).
21. Add `check-cqrs-lint` to CI once a Go-installable distribution exists (P3 blocked item).
22. Self-test for the release-train `--strict-lag` mode (the drift script has one; the train script's lag counting doesn't).
23. Add a gate that fails when TODO_LIST claims (file:line) no longer resolve — "TODO drift" detector, generalizing today's audit.
24. Seed the tag-cache TTL gotcha into `check-release-train --help` (hit again 2026-09-07; documented only in AGENTS.md).
25. Consider `BENCH_MAX_LOAD` advisory output naming the blocking process (today's 825 was opaque).

**Tests & hardening**
26. SSE backlog: `transport/serve_test.go` gaps (replay-before-subscribe, heartbeat join-on-exit, close-mid-stream, concurrent clients).
27. SSE backlog: `WithSSEMaxReplay` handler option + shared `SSEOptions` struct.
28. SSE backlog: fuzz `DomainEventToSSE`; e2e `/sse` reconnect-with-replay scenario.
29. SSE backlog: cross-module identical-wire-format contract test; `Retry` field on replayed events.
30. `JournalSSEStore.EventsAfter` bench on large journals.
31. Parametrize MORE declarative tests over the engine matrix (today only lifecycle+authz run on SQLite; tenant/bot/membership suites are memory-only).
32. Add a declarative-test for `Enforce` role inheritance edges (super_admin implies all) — currently only admin/viewer covered.
33. ProjectionLayer-negative tests for its own error paths (`no_event_store`, `journal_unsupported`, `no_event_bus`).

**Docs**
34. `docs/guides/declarative-projections.md`: add a checkpoint-store section update once upstream ships the injection option (link from #14).
35. Link the new guide from `leveraging-system-metaengine.md` + FEATURES.md systemadapter row (cross-link sweep).
36. setup/README: add `Config.CSRF` row adjacent to the security note (knob landed 2026-09-10; verify table completeness).
37. ROADMAP: record the direct-vs-indirect templ-components rationale so the next train sweep doesn't re-derive it.
38. Annotate this report + the 2026-09-09 status refs in TODO_LIST sources (they cite pre-sweep state).

**Repo hygiene**
39. v4-branch filter-repo + force-push (27.7 MB binary blobs) — user gate.
40. setup-demo blob purge from pushed master — user gate, low priority.
41. loginpage cosmetic residuals (favicon edge glyphs, error-copy tone).
42. Prune `docs/status/` archive README index as reports accumulate.

**Next-session process fixes**
43. Start every TODO-execution session with the audit script pattern: for each item, `rg` the claimed file:line before believing it.
44. When a TODO item is done elsewhere, remove it in the same session that discovers it (today's move — make it the rule).
45. Add "re-verify dead/broken hardware claims older than 2 weeks" to the docs-health sweep checklist.
46. Keep the daemon race mitigation: verify HEAD content after daemon commits (done today — keep doing it).
47. Batch new test additions with an immediate per-module lint run (golines caught one overlong line today — cheap to catch at write time).
48. Record `AuditEntryView` field naming (AggregateID, no Email) in the systemadapter README if one exists, else the guide.
49. Evaluate whether `waitForView`-style polling deserves a small exported helper in systemadapter (consumers will all hand-roll it; v5 candidate, needs an owner call).
50. After the train (#1): re-run the full gate battery (test-race/fuzz/flake/e2e/coverage) — the proven post-bump pass recipe.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Train timing:** should the next family train be cut NOW (publishes `Config.CSRF` + DataStar setup API + OTel seam, aligns templ-components tree-wide), or do you want more feature accumulation first? This gates items 1, 5, 36, 37, 50.
2. **`/sse` endpoint posture:** A (ship as-is), B (scoped-feed option), C (per-user streams), or D (drop the shared endpoint)? The one-pager is ready; it is purely a product/security call.
3. **Upstream tag request:** do you want me to file the `metaengine/projectionadapter v4.5.0` request in go-cqrs-lite now (it's the ONLY blocker for the systemadapter first tag), and should the ask include the `system.New` checkpoint/DLQ injection option in the same round?

---

*Per the status-report skill, section (f) is HARVEST input for TODO_LIST/ROADMAP — awaiting your instructions before running it. No commits made (harness rule); the daemon owns the working tree.*
