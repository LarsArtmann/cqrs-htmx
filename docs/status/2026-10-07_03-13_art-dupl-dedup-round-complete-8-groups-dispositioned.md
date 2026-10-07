# Status — art-dupl dedup round: 8 clone groups → zero harmful duplication

> **Generated:** 2026-10-07 03:13 CEST · **Session scope:** the user-pasted `art-dupl --sort total-tokens -t 4` report (8 actionable groups, 20 clones / 109 tokens) executed per the `deduplicate-code` skill: per-group read → triage → extract / accept-with-rationale → verify.
> **Verdict up front:** all 8 groups dispositioned. 3 extracted, 5 accepted with written rationale. All three touched modules (root, adminui, dashboardui) build / vet / test (-race) / lint green in every file I own. Remaining dashboardui lint findings (5) are all in the concurrent session's files.

---

## a) FULLY DONE

### Group #4 — usermgmt marshal/rewrap twins (3 sites) ✅
- `marshalViewJSON` (usermgmt/sql_view_marshal.go) now returns `(string, *errorfamily.Error)`; the three Handle sites (membership / tenant / bot in `sql_readmodel_extra.go`) consume `marshalErr` and attach `WithContextAny("agg_id", aggID)` — the double-Infrastructure-wrap stacking is gone.
- Verified: build/vet rc=0, full usermgmt suite `-race` rc=0, golangci-lint 0 issues, erraudit scoped clean, `erraudit-inventory` TOTAL=0.
- The typed-nil-interface trap was hit and fixed live (a nil `*Error` returned as `error` is non-nil); `deleteViewOnTombstone` deliberately stayed `(bool, error)`.
- Committed (entangled in daemon heuristic commit `4e16a0e3` — not amendable, accepted).

### Group #2 — adminui/dashboardui asset-handler twins (~120 lines) ✅
- New root API `asset_serve.go`: `ServeAsset(name, contentType, data)` (immutable posture: FNV-1a content ETag, 1-year immutable Cache-Control, nosniff, zero modtime, stdlib ServeContent conditional/range), `AssetETag(name, data)`, and (second pass) `AssetFromFS(fsys, name)` — the read half with Infrastructure-family errors.
- Both modules reduced to their irreducible `embed.FS` decl + a 3-line wrapper (read → errConfig wrap → ServeAsset). `dashboardui/layout.go serveConstAsset` is a 1-line delegate; the const-asset route gained nosniff it lacked.
- Tests: root `asset_serve_test.go` (headers, If-None-Match round-trip contract reuse, ETag derivation, AssetFromFS success/missing) + both modules' suites green `-race`.
- ETag format changed (documented in all three CHANGELOGs): `"<name>-<unpadded-hex>"` replaces module-prefixed `%016x`; consumers get one revalidation, no action.

### Group #1 — dashboardui vanilla test prologue (5 flagged sites) ✅
- New `dashboardui/testsetup_test.go`: `newTestDashboardMux(t)` owns store+New+Mount(`/dashboard/`).
- Converted exactly the 5 art-dupl-flagged sites: a11y_test (via `a11yOverview`), assets_test 304 test, errors_test 404 shape, handlers_coverage toast-bridge + global-error tests. handlers_coverage_test.go no longer imports memorystorage at all.
- Full dashboardui suite green `-race` after the conversion.

### Accept dispositions, all with written rationale ✅
- **Group #8** (commands/queries twin handlers, ~85 lines): prose rationale (divergent journal interfaces `ReadFrom`/`ReadQueriesFrom`, persisted types, exporters, ID parsers — a template would need ~6 callbacks + generics over unrelated types) + `//nolint:cyclop,dupl` on both handlers. Discovered en route: dashboardui's `dupl` linter was firing on this pair at HEAD (pre-existing red) and is now silenced the right way.
- **Groups #3/#5/#7** (page-skeleton/section idioms): rationale comments on `emptyStatePanel` (dashboardui/components.templ), `listNote` (adminui/components.templ), `definitionList` (dashboardui/components.templ). Regenerated via the canonical `nix run .#gen`; `check-codegen` green for adminui + loginpage.
- **Group #6** (detector self-overlap, single component): no action needed — confirmed artifact.

### Session recovery (situational, but real work) ✅
- The uncommitted Group #2 work was swallowed by a concurrent session's **Git Town stash** ("Git Town WIP") together with 35 foreign in-flight files. Popped the stash, restored the tree faithfully, re-verified everything, and separated my 8 files from theirs before any commit.
- A later concurrent-session tree operation silently **reverted my comma-list nolint fix** on handlers_audit.go back to the first-attempt form (re-introducing 2 dupl + 2 nolintlint findings); detected on the final lint sweep and re-applied.

### Verification & receipts ✅
- Final battery on my surfaces: build=vet=test=lint green for root + adminui; dashboardui green except 5 findings **all in files I never touched** (handlers_events.go cyclop, systembridge/telemetry.go exhaustruct ×2, accent_color.go gochecknoglobals + mnd — none authored by this session).
- `nix run .#erraudit-inventory` TOTAL=0; status-doc gates (check-status-rows.py, check-status-annotations.sh) rc=0.
- CHANGELOG receipts written + amended: root (`ServeAsset`/`AssetETag`/`AssetFromFS`), adminui + dashboardui (delegation, ETag format change, nosniff gain, error-family note). TODO_LIST P2 chore entry for the follow-up sweep.
- art-dupl re-run: all 8 original groups resolved; 8 remaining shown groups are documented-accepts, detector artifacts, or foreign-owned (telemetry.templ).

---

## b) PARTIALLY DONE

1. **Full dashboardui lint green** — my files are clean, but the module gate stays red on the 5 foreign/pre-existing findings (above). Cannot close without touching the concurrent session's files.
2. **check-codegen dashboardui** — red solely on the foreign `telemetry.templ`/`telemetry_templ.go` pair (their committed generated file lags their committed .templ by one line). My pairs are canonical; verified by adminui/loginpage passing and the dashboardui failure naming only telemetry.
3. **The wider vanilla-fixture sweep** — `newTestDashboardMux` exists, but ~13 near-vanilla `NewMemoryStore` sites (sse_replay, handlers_security, dashboard_test — deviating only by EventBus/ReadOnly) still build their own configs. Harvested into TODO_LIST as a bounded chore; not in the flagged scope, so not done now.
4. **CHANGELOG receipts are uncommitted edits on top of landed code** (daemon had not swept them at report time — expected to land as its next heuristic commit).

---

## c) NOT STARTED

1. **Release-train alignment** — root gained new exported API (`ServeAsset`/`AssetETag`/`AssetFromFS`); adminui/dashboardui need their requires bumped + new tags in the next train (hermetic `GOWORK=off` builds of the two modules still fail against proxy root v4.13.x, as expected mid-train).
2. **AGENTS.md knowledge capture** — two durable lessons from this session are not yet in AGENTS.md gotchas: (a) two consecutive `//nolint:X` directives — only one applies; comma-list form is the reliable pattern; (b) bare `templ generate` with a v0.3.1020 CLI + no `gofmt -w` produces non-canonical output — `nix run .#gen` is the only canonical generator (partially covered by gotcha 10, but the version-mismatch trap is new).
3. **assets.go residual twin** — art-dupl still shows the ~17-line embed+wrapper pair (now flagged 12-28 vs 12-28). Judged irreducible (embed is compile-time per-module); no explicit "accepted" comment was added on top of the doc comments that already explain it. Optional micro-cleanup only.
4. **Telemetry.templ clone triage** — 2 residual groups involve the foreign session's telemetry.templ; routed to TODO_LIST, blocked on their session landing.

---

## d) TOTALLY FUCKED UP (own failures this session)

1. **Bare `templ generate` from my shell mutated the foreign `telemetry_templ.go`** — my CLI is v0.3.1020 while the module requires v0.3.1070-era generation, and I skipped the `gofmt -w` step. I caught it, restored the file, and re-ran the canonical app, but the correct move was to check the flake's generator contract BEFORE running any templ command in a module with foreign in-flight templates. Cost: one restore + one re-verify cycle, plus a foreign file briefly dirtied twice.
2. **The nolint two-directive mistake shipped to HEAD** — my first form (`//nolint:cyclop` + `//nolint:dupl` on consecutive lines) silently failed: dupl stayed reported AND nolintlint flagged both directives as unused. It got daemon-committed (`8c572ef5`) before my comma-list re-edit landed, and then the tree reset resurrected the broken version. Fixed finally, but the broken variant spent time at HEAD.
3. **Missed that handlers_coverage_test.go's memorystorage import became unused** — the compiler caught it, but I should have predicted the cascade from removing the only two uses.
4. **wrapcheck churn on `AssetFromFS`** — I designed the read half to return raw fs errors ("caller owns presentation"), which root's wrapcheck immediately rejected, then the module pass-through form was rejected too. Two design rounds wasted; the repo-blessed pattern (wrap at the boundary; errorfamily.NewInfrastructure per app.go) should have been checked first.
5. **Let the daemon commit my phases entangled with foreign files** (accepted repo behavior per gotcha 4, but my phase boundaries were still too wide — Group #2 landed as part of a 43-file heuristic commit rather than a scoped one).

---

## e) WHAT WE SHOULD IMPROVE

1. **Guard the templ generation contract mechanically** — a check that the CLI generating is the flake's templ (or just always `nix run .#gen`) should be the ONLY documented path; gotcha 10 says it, my fingers still typed the bare command.
2. **nolint discipline** — repo-wide, prefer single comma-list directives; a one-line addition to gotcha 7/8 would codify the two-directive failure mode.
3. **Foreign-session interlock** — today's stash-pop + tree-reset events make a stronger case for `nix run .#preflight-tree-check` before EVERY phase, not just batch steps; I skipped it after the pop and paid for it.
4. **art-dupl in CI-adjacent gates?** — the report is a good ratchet candidate (like branching-flow): pin the accepted-clone list, fail on NEW groups. Not asked for; noting as an idea.
5. **CHANGELOG receipt timing** — write receipts at phase commit time, not at the end; the daemon race (b99e2a32 landing code without receipts) creates exactly the mixed commits we then cannot amend.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

*Prioritized; items 1–12 are this round's direct follow-ups, 13–20 are adjacent quality debt observed this session, 21+ are repo-level ideas noted in passing (do NOT treat as scheduled work).*

**Dedup-round follow-ups**
1. Land the pending CHANGELOG/TODO_LIST amendments (daemon, or one scoped commit).
2. After the telemetry session lands: regenerate dashboardui (`nix run .#gen`) so check-codegen goes green again.
3. Same trigger: triage the 2 telemetry.templ clone groups (aggregates vs telemetry cell glue; telemetry self-overlap).
4. Wider fixture sweep: config-variant helper for the ~13 near-vanilla NewMemoryStore sites (TODO_LIST entry filed).
5. Release train: tag root with the new asset API → bump adminui/dashboardui requires → verify hermetic `GOWORK=off` builds per module (the standard wave order; `--refresh-cache` between tag and commit per gotcha 27).
6. Add the explicit "accepted residual" one-liner to both assets.go doc comments (art-dupl still shows the embed+wrapper pair).
7. AGENTS.md: capture the two-new nolint + templ-generator lessons (see c2).
8. Re-verify handlers_audit.go survived the foreign session's final tree operations (it was reverted once already).
9. Re-run full `nix run .#check-modules` on a quiet tree (my phases were verified per-module; the workspace gate hasn't seen the final state).
10. `nix run .#test-all` (includes e2e/examples the standard `.#test` skips) once the foreign session's systemadapter/integration_test work lands.
11. Re-pin `docs/benchmarks` ONLY if the asset path changed a bench-relevant route (it didn't — skip unless the gate fires).
12. Update the earlier session's status report (`2026-10-07_02-11_art-dupl-dedup-8-groups-status.md`) with a completion annotation pointing at this report.

**Adjacent debt noticed (not worked)**
13. `handlers_events.go` cyclop 16 > 12 — concurrent session's file; needs decomposition or a suppression decision by its owner.
14. `accent_color.go` gochecknoglobals (`cssNamedColors`) + mnd (`3`) — pre-existing; cheap wins (table + named const).
15. `systembridge/telemetry.go` exhaustruct_v5 ×2 — foreign; their session should add the missing fields or an ignore-pattern.
16. dashboardui `.golangci.yml` — the `erraudit` unknown-linter warning (runner/nolint_filter) appears on every run; the local binary predates the analyzer name.
17. The daemon's 43-file heuristic commits mix foreign + my work — consider a pre-commit daemon pause hook while a verified phase is in flight (gotcha 4 pain, recurring).
18. Git Town stash leaks: a `git stash list` check belongs in `preflight-tree-check` (a WIP stash sat unclaimed for ~40 min this session).
19. Root `CHANGELOG.md` Unreleased section is getting very long (10+ large bullets) — consider cutting `[Unreleased]` into the next version heading at train time.
20. `asset_serve_test.go` import block sorted by treefmt but hand-edited twice — rely on `nix run .#fmt` immediately after append-style edits (I did; keep doing it).

**Repo-level ideas (passing notes, unscheduled)**
21. Ratchet art-dupl like branching-flow (SARIF baseline, fail-on-new-groups).
22. `dupl` linter currently has NO suppressions repo-wide except mine — audit whether other modules carry latent twin classes art-dupl -t 4 didn't reach (it found 283 groups, 197 suppressed by filters).
23. Root module error idiom: `errConfig`-style helpers exist per UI module but root has none — the asset errors went through errorfamily directly; consider a tiny root `errInfra(code, msg)` helper if more wrapcheck sites appear.
24. The `nix run .#fmt --` app + `templ` formatter wrapper (gotcha 22a) solved the version trap for FORMATTING; a matching guard for GENERATION (warn when bare templ is older than go.mod's templ) would have prevented today's drift.
25. `handlers_coverage_test.go` still mixes `mustTestDashboardWithConfig` + `mustTestDashboard` + now `newTestDashboardMux` — three helper names; consolidate the doc map in testsetup_test.go.
26. stale `//nolint` audit: run the cqrs-lint/`--fail-on-stale-suppressions` equivalent for golangci nolintlint repo-wide (it caught my 2 unused directives; a repo sweep may find more).

*(Stopping at 26 — the remaining ideas belong to the other session's hardening plan, not this round.)*

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **The telemetry session's endgame:** their committed `telemetry.templ`/`telemetry_templ.go` pair is inconsistent (one line apart) and their `systembridge` + `accent_color.go` lint findings are red. Should I (a) wait and only re-verify after their session lands, or (b) at some point take over and fix their lint findings + regenerate? If (b), when is their session "done"?
2. **ETag format change acceptance:** the new asset ETags (`"<name>-<unpadded-hex>"`) replace the module-prefixed padded form; consumers get one revalidation. I documented it in three CHANGELOGs — is that sufficient, or do you want byte-identical ETags (I'd reintroduce the prefix into `AssetETag` and re-pin tests)?
3. **Group #1 sweep width:** convert ONLY the 5 flagged sites (done) or also fold the ~13 near-vanilla EventBus/ReadOnly sites into a config-variant helper now (the TODO_LIST chore)? The former is report-clean; the latter shrinks the fixture surface ~40% but touches files outside the original report scope.
