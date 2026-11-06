# Fleet-Plan Execution — Waves 1–2 Partial Landing, Concurrent-Session Discovery

> **ANNOTATED 2026-11-06** — ~~this report~~ **SUPERSEDED** by [`2026-11-06_22-03_fleet-plan-wave-5-partial-render-presets-charts-landed-a22-mid-flight_status.md`](../2026-11-06_22-03_fleet-plan-wave-5-partial-render-presets-charts-landed-a22-mid-flight_status.md), which waves-1-2 execution state carried forward by the wave-5 report. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-06 (round-23 (2026-11-06)).**

**Date:** 2026-11-06 02:44 · **Author:** Crush (plan-execution session) ·
**Plan of record:** [`docs/planning/2026-11-06_00-37_SUPERB-fleet-libs-audit-follow-through-pareto-plan.md`](../../planning/2026-11-06_00-37_SUPERB-fleet-libs-audit-follow-through-pareto-plan.md)
**Session span:** A1 → A2 → A3 → A11 (in flight), interrupted for this report.

## The headline: a SECOND session is executing the same plan in this tree

Within ~90 minutes of starting, commits that were not mine landed
(`7d3a47ca`, 14 files, 02:04; last own-commit ~02:14; daemon commits continuing).
Their lane: **session-lifecycle audit events (ADR-0060), R4 SQLite DLQ,
R7 commandlifecycle, checkpoint-batching docs, adminui superpowers work** —
i.e. tasks A5, A6, (most of) A10, and A27 of the very plan I was executing.
Consequences handled this session:

- I **did not duplicate** any of their landed work (verified per-task before starting: grep for `CommandLifecycle`, `durableDeadLetterStore`, `FailureLog`, `NewMiddlewareStack`, `LogErrorContext`, `Audience`).
- Their tree carried a **build-breaking typo** (`adminui/handler.go:94 icons.ChartBar` — no such symbol; the only chart icon is `icons.Chart`) that broke EVERY downstream module (setup, examples, e2e). I fixed the one-liner to unblock my verification (adminui/handler.go, uncommitted→daemon).
- Their session-audit feature references `identitymodel.Session*` symbols that exist only in-tree; **identity-model v4.16.0 is unpublished**, so `check-cqrs-lint` (GOWORK=off) is red on root+usermgmt+systemadapter — the gotcha-30 root-tag-gap class, PRE-EXISTING my changes (`es_constants.go:34` et al. are theirs). A1's lint leg cannot go green until that train ships. The train is theirs to cut.

## a) FULLY DONE (this session, verified)

### A1 — R2 derived event catalog (code + tests green; lint leg train-blocked)

- `usermgmt/es_event_catalog.go`: **single registration table** (`eventCatalogSpecs`, 27 rows) now feeds BOTH wire formats — the served `cqrshtmx.EventCatalog` (Go-vocabulary fields, wire shape UNCHANGED, existing tests untouched-green) and the new architecture format. Deleted the five `register*Events` functions.
- `usermgmt/es_architecture_catalog.go`: **`ArchitectureCatalog()`** — the 27-event published language as a go-cqrs-lite `*catalog.Catalog` (shared `catalog.SchemaFromType[T]` reflection, curated descriptions as `WithSummary`, payload-schema-version stamp in the fleet's derived-catalog vocabulary). Retires F032 in usermgmt with REAL usage.
- `usermgmt/es_architecture_catalog_test.go`: 4 gates — `catalog.AssertEventInventory` (upstream drift gate), parity with the served catalog (same events+summaries), field-parity on 4 pilot payloads (embedded-struct walking), `Validate()` clean.
- `systemadapter/architecture_catalog.go`: **`ArchitectureCatalog(ctx)`** = `systemcatalog.FromDomainConfig(DomainConfig())` (ephemeral in-memory boot; the real declared-truth derivation, 21 commands + 27 events). Bumped `systemcatalog/v4 v4.0.0` + `catalog/v4 v4.12.0`.
- `systemadapter/architecture_catalog_test.go`: command-inventory gate (21 ids), event-inventory gate (27 ids), **parity test usermgmt ↔ systemadapter derivations**.
- F032 disables REMOVED from both `.cqrs-lint.json` (with retirement provenance comments).
- `docs/guides/event-catalog-guide.md`: fixed the stale "21 events" table (now 27 incl. Auth aggregate + CredentialUsed), documented the two-catalog bridge.
- **Verified:** usermgmt suite green, systemadapter suite green, catalog tests green. **`check-cqrs-lint` RED — pre-existing root-tag-gap (unpublished identity-model v4.16.0), NOT caused by A1** (failing files are theirs; proven before I started).

### A2 — SSE drop observability (3 of 4 legs, all tested)

- **root** `sse_drop_stats.go`: `DropStats` (atomic counter honoring go-sse's in-lock callback contract), `NewBroadcaster` is now variadic over `sse.Option[sse.Event]` (the construction seam), `DropStats.Option()/Attach()/Total()/Snapshot()` + `SSEHubSnapshot`. 4 tests incl. per-subscriber-per-broadcast counting semantics.
- **dashboardui**: owns its broadcaster → wires `DropStats` at construction; `core.HubHealth.Drops`; overview card shows `N · M dropped` (amber tone) — drops stop being silent; existing goldens unaffected (suffix only renders when drops > 0). Suite green.
- **setup**: `Bundle.SSEDrops` on the shared hub + lifetime-drop WARN at Close. 2 tests. Suite green (after unblocking adminui).
- **adminui leg NOT started** (deliberate: active concurrent session in adminui files).

### A3 — health-dashboard CSP nonce (honest re-scope)

The audit's claim ("thread the nonce option through the bridge") was **imprecise**: `health.NewDashboard` already passes `opts ...healthdashboard.Option` through, and `WithNonce`/`WithNonceExtractor` exist upstream. What was genuinely missing: **proof + consumer doc**. Landed:
- `health/dashboard_test.go`: `TestNewDashboard_ForwardsCSPNonceOptions` — pins BOTH nonce options render `nonce="…"` on inline scripts, AND pins the host-side pairing contract (`healthdashboard.RecommendedCSP(nonce)` admits the same nonce). Upstream truth respected: the dashboard never sets its own CSP header (first test draft wrongly expected one — corrected against source).
- `health/README.md`: nonce-CSP section with the `httputil.NonceFromRequest` extractor snippet (symbol verified to exist) + pairing note. Suite green.

## b) PARTIALLY DONE

### A11 — audience-aware SafeDetail (code complete, VERIFICATION INTERRUPTED)

Landed (working tree, daemon-committed):
- `httperr/audience.go`: `Audience` (zero value = `AudiencePublic`, fail-safe), `AudienceOperator`, `SafeDetailFor(err, status, audience)`.
- `httperr/errors.go`: `SafeDetail` now delegates (`includeInternal` ⇄ operator); **`ProblemDetailsAudienceErrorHandler(resolver, loginRedirect)`** — per-request audience discrimination on the RFC 9457 body; nil resolver fails public.
- `httperr/structured.go`: audience-aware builders (`NewStructuredErrorForAudience[WithContext]`); operators get raw 5xx detail AND machine context (`ErrorContext()` map on 5xx — public never does).
- Root re-exports: type + consts + functions (`httperr_reexport.go`).
- `httperr/audience_test.go`: 5 tests (user-vs-operator detail, client-fault identical, delegation, RFC 9457 body public/operator/nil-resolver).

**NOT done:** the test suite has NOT been re-run after the final re-export fix (type alias moved from `var` block to `type` block — the first run failed only on that placement). ADR-0061 paragraph, CHANGELOG, guide mention all missing.

## c) NOT STARTED

A4 (go-etag bump) · A7 (FailureLog views) · A8 (partial-render showcase) · A9 (MiddlewareStack) · A12 (LogErrorContext) · A13 (L1 tail) · A14 (docs truth pass — scope now ALSO covers row 71, see below) · A15 verify · A16 (engine presets) · A17 (render lane) · A18 (AutoDetect decision) · A19 (Diagnose inspector) · A20 (pooling bench) · A21 (charts panel) · A22 (appkit hooks) · A23 (systemd docs) · A24 (atomic write) · A25 (catalog watch) · A26 (archive round).

## d) TOTALLY FUCKED UP / incidents (all recovered)

1. **Disk 100% full** on /mnt/buildcache (1.3 GB free / 220 GB) — go-cache-env failed fast (gotcha 12 class). Recovered: `go clean -cache` + pruned golangci-lint/goimports caches → 128 GB free. Cost: one cold-cache rebuild window.
2. **Two near-misses with the concurrent session** (gotcha 47): (i) feared my `health/dashboard_test.go` write clobbered their file — false alarm (matching test names lived in `probe_test.go`); (ii) their `icons.ChartBar` typo broke the whole tree — I fixed it (one line) rather than waiting.
3. **Test-draft bugs caught by verification** (the system working as designed): `WithBufferSize(0)` means "default" not "zero" (drop test); expected a CSP header the upstream deliberately never sets (nonce test); re-export type alias in a `var` block (build error, fixed, suite not yet re-run).
4. **Session-start debt:** ~90 min executed before noticing the second session. The overlap check should have been step zero.

## Discoveries that SHRANK the plan (verified, not remembered)

- **A27 is already done** (TODO row 71 is stale): `StructuredError.Context` carries `ErrorContext()` for client-faults (`httperr/structured.go`), and the upgrade CLI exits via `errorfamily.HandleError(err)` (`cmd/cqrs-htmx-upgrade/main.go:124`) which maps `ExitCode`. → A27 collapses into A14's annotation pass.
- **A5 fully landed by the concurrent session**: `durableDeadLetterStore` (SQLite-dialect → `NewSQLiteDeadLetterStore` on the same DB; postgres/mysql documented upstream gap), `usermgmt/dlq_durability_test.go`, `production-readiness.md` DLQ note.
- **A6 fully landed by them**: `ServiceConfig.CommandLifecycle` (`usermgmt/service_core.go:215`, factory wiring, guide § in leveraging-go-cqrs-lite.md).
- **A10 looks landed or nearly**: `WithCheckpointEvery` consumer seam documented at `setup/config.go:438` — needs one verify pass before striking.

## e) WHAT WE SHOULD IMPROVE

1. **Multi-agent coordination is the #1 process gap.** Two agents executing one plan in one tree without a claim protocol. Proposal: a `SESSION_CLAIMS.md` (or plan-file checkboxes checked at task START) so sessions claim tasks before editing.
2. **Audit reports over-claimed gaps** (A3 "thread through", A27 entire): plan items should get a 2-minute already-done grep BEFORE execution (I did this per-task after the discovery; it should be the standing first step of every B-task).
3. **The un-published-train red blocks the local lint gate** fleet-wide: consider making `check-cqrs-lint` skip symbol-resolution failures attributable to root-tag-gap with a loud SKIPPED (vs FAIL) verdict when `git ls-remote` shows no covering tag — keeps the gate actionable for other work.
4. **CHANGELOG discipline**: my A1/A2/A3/A11 work has zero CHANGELOG entries yet — must land with the verification commit, not after.

## f) NEXT (up to 50, roughly dependency-ordered)

1. Re-run `go test ./httperr/ .` — verify A11 after the re-export fix.
2. A11 finish: ADR-0061 (audience-aware error detail), root+`httperr` CHANGELOG, error-mapping guide note.
3. Adminui A2 leg: `Config.SSEHubStats func() cqrshtmx.SSEHubSnapshot` dashboard card (after their session quiets; re-read files first).
4. A4: `bump-dep go-etag v0.7.0` sweep + treadmill 0-lag verify + CHANGELOG.
5. A9: rebuild `RecommendedSecurityMiddleware` over `httputil.NewMiddlewareStack` + behavior-equivalence test.
6. A12: `errorfamily.LogErrorContext` at dispatch error paths + family→level mapping test + bench-suppression check.
7. A7: dashboardui command-fates tab (FailureLog/RetryCount views; depends A6 recorder — landed).
8. A8: examples/basic partial-render dogfood (RenderPartialOrFull + OOBHTML + RenderIf).
9. A13: B60 -race audit-log test; B61/62 A013 value-embeds (e2e/server `sync_types.go:19`, sync-demo `main.go:44`).
10. A14 (expanded): annotate TODO rows 27, 86, 88, 96 **and 71** (done-elsewhere/discovered-done).
11. A15 verify: TODO_LIST consolidated row exists (created prior session) — run `check-todo-hygiene`.
12. A16: systemadapter engine presets (pebble/bbolt/postgres/duckdb/turso).
13. A17: record the render-path decision (blocked on owner question g.1 — see questions).
14. A18: codec AutoDetect vs ADR-0056 opaque-stamp contract — adopt-or-N/A record.
15. A19: dashboardui payload inspector CBOR→diagnostic via codec.Diagnose.
16. A20: codec EncodePooled bench-spike (adopt only on a median win).
17. A21: dashboardui event-throughput charts panel (templ charts; goldens + CSS bundle rebuild).
18. A22: setup appkit lifecycle-hook forwarding + shutdown-order test.
19. A23: production-readiness sd_notify section (go-daemon docs note).
20. A24: atomic-write at `cmd/cqrs-htmx-upgrade/rewrite.go:91` + fingerprint-failure test.
21. A25: check go-cqrs-lite catalog release; bump integration_test + catalog-demo when shipped; CI un-red.
22. A26: archive round (2–3 oldest status reports via `.#archive-status --apply` + TAIL_BASELINE drop).
23. Watch/coalesce with the concurrent session: identity-model v4.16.0 train → then `check-cqrs-lint` 14/14 + `check-modules` full leg for A1's receipt.
24. `usermgmt/CHANGELOG.md` + `systemadapter` CHANGELOG: A1 entries (ArchitectureCatalog pair, F032 retirement).
25. dashboardui CHANGELOG: A2 drop-card entry; root CHANGELOG: DropStats + audience entries; health CHANGELOG: nonce test/README entry.
26. Corpus ledger row for this report + baseline re-true (done same-change as this file; re-verify after later doc edits).
27. `docs/guides/sse-and-datastar.md` or realtime reference: document DropStats + Bundle.SSEDrops.
28. AGENTS.md quick-reference: systemcatalog/catalog pins on the train line; F032-retired note (update the cqrs-lint row).
29. `docs/review/libs/go-etag.md` + index.html: mark lag CLOSED after A4.
30. `docs/review/libs/go-health-dashboard.md`: correct the A3 row (passthrough existed; test+doc landed) — the report's own verify block asks for it.
31. Consider `AssertInventory` gate wired into CI's usermgmt leg explicitly (it rides the suite, but the receipt should name it).
32. Root `api_surface_compat_test.go`: add compile-pins for the new re-exports (Audience, SafeDetailFor, ProblemDetailsAudienceErrorHandler, NewStructuredErrorForAudience*).
33. Examples: `examples/system-demo` could print the derived ArchitectureCatalog inventory as a smoke demo (nice-to-have).
34. Run `buildflow --fix` scoped to touched modules (format pass over new files).
35. Full `nix run .#test` battery before any wave commit (currently only per-module suites ran).
36. `nix run .#check-modules` aggregate once A11 finishes + adminui leg lands.
37. `nix run .#check-docs-links` after all guide edits (I edited three docs).
38. `nix run .#check-css-bundles`-class: not needed (no UI class changes) — skip note.
39. Bench canary: `nix run .#bench-spike` after A2 (broadcaster construction changed shape — should be neutral, verify).
40. B55 note: the A13 -race audit-log test may already exist via their session — grep before writing (same class as A5/A6 discovery).
41. Re-verify `go-sse v0.7.0` is latest before finishing A2 docs (treadmill).
42. TODO_LIST: strike/annotate the fleet-follow-through row sub-items as waves complete (keep hygiene gate green).
43. If the concurrent session stays quiet >1h: propose taking over their train (owner question).
44. Merge-conflict probe: `git log --oneline --since="02:00" -- adminui/` before the adminui A2 leg.
45. `docs/PERMISSIONS.md` — no posture changes made; verify gate still green post-A2 adminui leg.
46. Session-claims file (e) improvement — implement if owner approves.
47. Post-train: `nix run .#verify-remote-tags` health check.
48. Post-everything: `nix run .#train-preflight` ONLY if a release-bearing wave happens (none planned without owner).
49. Re-read `docs/planning/…pareto-plan.md` errata section and append the A5/A6/A10/A27 discovered-done notes (keeps the plan honest for the next session).
50. Final: docs-health `.#archive-status` eligibility check for THIS report at next archive round.

## g) Questions for the owner (cannot be resolved from the tree)

1. **The identity-model v4.16.0 train.** The second session's session-audit feature is complete in-tree but untagged, which keeps `check-cqrs-lint` (and strict hermetic legs touching usermgmt) red for EVERYONE. Their session has been quiet since ~02:14. Cut the train myself (wave-tag, strict release-train), or leave it to them?
2. **Render verification lane (A17; carried-over question g.1).** No chromium on this host; firefox/chrome are shell-banned. Sanction a browser/screenshot path (e.g. install chromium via nix into a profile), or waive the screenshot done-bar for HTML deliverables (structural verification only)?
3. **go-etag v0.7.0 (A4; carried-over question c).** Standalone chore bump now, or batch into the next templ-components family sweep? (11 chore-only commits upstream; either is safe.)

---

### Verification state at time of writing

| Leg | State |
| --- | --- |
| usermgmt suite | ✅ green (A1) |
| systemadapter suite | ✅ green (A1) |
| dashboardui suite | ✅ green (A2) |
| setup suite (SSE subset) | ✅ green (A2) |
| health suite | ✅ green (A3) |
| root suite (DropStats subset) | ✅ green (A2); A11 httperr tests ⏳ NOT re-run after final fix |
| `check-cqrs-lint` | 🔴 pre-existing root-tag-gap (identity-model tag pending) |
| `check-modules` / full battery | ⏳ not yet run this session |
| corpus/tail baselines | re-trued same-change as this report |

## h) A14 docs-truth-pass closures (2026-11-06, resuming session)

Rows removed as done (receipts: AGENTS gate inventory, round-45 M15/M16/M18; verified live before removal): **README badge resolver gate** (`check-readme-badges`, in check-modules + CI + fixture self-test), **"No permanently-red PR gate" guard** (`check-pr-secrets`, same wiring), **Dependency-CVE watch** (`check-dependabot-alerts`, advisory report, live-verified 0 open alerts 2026-10-31), **Error-context rendering row** (part a: `StructuredError.Context` carries `ErrorContext()` — concurrent session's A27 below 500 + ADR-0061 operator audience on 5xx; part b: upgrade CLI exits via `errorfamily.HandleError`, `cmd/cqrs-htmx-upgrade/main.go:124`). The **go-cqrs-lite leverage row** trued in place: R1/R2/R4/R7/R8 moved to landed (fleet-plan waves 1–2 receipts), F1/L1-tail/X1/X2/X3 stay open. TODO-hygiene green after: 69 rows, 0 violations. A15 verified: the consolidated fleet-plan row exists and routes to the plan of record (no new harvest needed).

A11 resumed and completed: `WithContextMap` + charset-prefix test fixes → httperr + root suites green; ADR-0061 written + INDEX; README § error-mapping row; root CHANGELOG entries (audience + SSE drop stats); compat pins for the audience surface.
