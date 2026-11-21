# Plan-tail completion — B9 battery, B13 QUERY panel, B14 OpenAPI consumer, B15 FileInput, B16 PR prep

> **ANNOTATED 2026-11-21** — ~~this report~~ **SUPERSEDED** by [`2026-11-21_14-12_ci-heal-toolchain-split-brain-round48-docs-round_status.md`](../2026-11-21_14-12_ci-heal-toolchain-split-brain-round48-docs-round_status.md), which its-plan-tail-was-fully-harvested-into-the-round-48-drain-row-this-report-receipts. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-21 (round-23 (2026-11-21)).**

**Date:** 2026-11-15 14:2x · **Session:** the final continuation of the [2026-11-15 07-21 v2 Pareto plan](../../planning/2026-11-15_07-21_SUPERB-fleet-audit-follow-through-v2-pareto-plan.md) (resumed from the 12-53 handoff).
**Mission:** execute the ENTIRE remaining plan tail (B9, B11-residual, B13–B17-residual), the plan execution-log rows, TODO strikes, final trues, second push. **Verification bar:** every receipt below was run THIS session on the shared tree.

## a) Context (what this session inherited)

7 commits ahead of the pushed origin (`44f29274`), the plan log ending at B1, and a LIVE sibling session (gfl-usermgmt + systemadapter tenant-deletion work, authored commit `549e9363` at 12:47, in-flight files until its own `daf8fb5b` at 14:17). The owner's standing directive: "GET SHIT DONE! The WHOLE TODO LIST!" — work in parallel under the pathspec-commit ritual, never touch sibling files.

## b) FULLY DONE (the whole executable plan)

| Item | What landed | Receipt |
| --- | --- | --- |
| Treadmill first-hand | `20 repos · 81 upstream prefixes · worst lag 0 · 0 disagreement(s)` — the 12-53 session's second-hand "80 prefixes" citation was off by one and is now trued from the live run | commit `d1a47e71` |
| **B9** full-battery receipt | `check-modules` 101/106 legs GREEN; the 5 red legs ALL trace to the sibling's in-flight modules (its dep-budget overrun, its event-catalog artifact drift, its streamid/branching findings — line numbers MOVED between runs, proof of live editing). My lineage's one debt (datastar dep budget 6→7 for B3's datastartest) fixed same-session. `coverage-gate` 15/15 gated modules PASS; adminui re-verified post-B13 at 67.3% ≥ 66. Full workspace `go test ./... -count=1 -race` **GREEN rc=0** | `406ffba2` + plan log |
| **B13** QUERY panel | adminui user search rides the RFC 10008 ladder: `QUERY /users/search` + POST fallback twin (ONE handler, byte-identical fragments), `MethodOverrideMiddleware` inside `routes()`, the embedded `cqrs-query` ext served at `/-/ext/cqrs-query.js` and loaded in the layout; Enter upgrades to native QUERY, typing fires the twin, GET stays bookmarkable; one shared data path (`usersSearchData`) so rungs cannot drift | ladder-parity + criteria + 400s + 401 + asset-posture + rendered-wiring tests (`adminui/user_search_test.go`); **e2e chromium 13/13** (method assertions: Enter → QUERY, typing → POST); route-postures + CSS class gates green; CHANGELOG; row 13 trued |
| **B14** OpenAPI consumer | admin-demo's first App dispatch surface: `SearchDirectory` over the QUERY ladder (JSON body, native + POST twin), session-gated via the demo's own `requireSession` (the session middleware only ENRICHES — the API surface answers 401 itself), `/openapi.json` built from the handler's own `WithOpenAPI` metadata via `App.OpenAPIRoutes()` (the per-route option form's first shipped consumer; OpenAPI 3.2 via the query op; ETag/304); sidebar link via `ExtraNav` | examples/admin-demo suite GREEN (parity + gate + spec-shape tests); README section; row 9 partial→full, **score re-derived 83 → 87** |
| **B15** FileInput | the import form's hand-rolled file input → `forms.FileInput` (ID hook preserved); canonical `templGen` regen; CSS bundle rebuilt same-change | `check-css-bundles` + `check-css-bundle-classes` GREEN (1056 → 1077 tokens); `TestImportPage_Renders` holds unchanged |
| **B16** popover PR prep | Evidence pack: Chromium **152.0.7977.82** computed-style probes — the library's `span-inline-*` bands are REJECTED in position-area (var-substitution failure → INITIAL value, the corner-park mechanism); `span-x-start`/`span-x-end` + single-keyword bands all compute; `anchor()` cannot see the implicit invoker (`0px`) but resolves with explicit `anchor-name`/`position-anchor` (`129px` exact). Fix branch `fix/popover-position-area-chromium` on the templ-components checkout (keyword swap + engine-verification comment, both source CSS files); their display + cmd/tc suites GREEN; PR text per github-voice (checker 0 FAIL 0 WARN) | [docs/drafts/templ-components-popover-position-area-pr.md](../../drafts/templ-components-popover-position-area-pr.md); their commit `0f977eaf`; **SEND stays owner-gated (K3)** |
| Plan log + TODO | execution-log rows for B9/B11/B13–B16 + the B2–B20 window receipt; fleet item (1) treadmill residual struck; plan-execution status row added | plan § execution log; `TODO_LIST.md` (hygiene gate clean: 79 rows, 0 dead) |

## c) PARTIALLY DONE

1. **The second push** — staged as this report's commit train (release-train refresh → push). Everything else that blocked it is green.
   > **DONE 2026-11-16:** the train landed as `1d8df13b..9d159fdc master -> master` (pre-push release-train strict + version-drift strict green) after the settled-tree gate-heal pass; the CI watch rides `.#push-and-watch`.
2. **The settled-tree composite re-run** (§f.50 ritual) — deferred until the sibling's session ends; its module state rode the B9 run (documented, not hidden).
   > **DONE 2026-11-16:** the composite re-ran on the settled tree at 101/106 with the GFL wave's five reds triaged and healed in the same pass (dep-budget raise, streamid identity fix, branching-flow re-pin, usermgmt lint nolint).

## d) NOT STARTED

1. **B18** — trigger-gated (next family train). Correctly untouched.
2. K2/K3/K4 — owner gates (bot idempotency default-on; popover PR send; M2M/migration packets).

## e) OWNED FAILURES

1. **Daemon-shredding continued**: my B13 Go files landed inside daemon chore commits (`bcbb8f2b`/`d7705949`) before my pathspec commit ran — content safe (verified by grep), history shredded. The 13:45–13:55 window was too long between edit-finish and commit-start.
2. **A changelog edit created a duplicate `### Added` header** in adminui/CHANGELOG.md (fixed in the next edit; the docs-freshness/link gates were not yet run — caught by rereading before commit).
3. **Two pre-commit hook blocks on the templ-components checkout** — the `settle-staged-go-format` self-test failed while MY hook raced THEIR auto-commit daemon for the git index; passes standalone every time. The fix landed via their daemon commit (`0f977eaf`) with a heuristic message instead of my descriptive one.
4. **A repro CSS bug cost one iteration** (B16 probe: `inset: auto` declared AFTER `top` resets it — my own stylesheet, not the platform).

## f) Next items (ranked)

1. The second push + `.#push-and-watch` CI triage (this report's commit train).
2. K3: send the popover PR (draft ready, checker-clean).
3. When the sibling session ends: settled-tree `check-modules` + hermetic gfl-usermgmt build (§f.50).
4. B18 on the next family train (e2e/examples pin sweep).
5. The five deterministic firefox e2e divergences (TODO-tracked).
6. docs-gates `audit-all` RED (393 vs pin 322) — owner drain-or-re-pin decision.
7. Watch go-cqrs-lite for the `storage` tag carrying go-cqrs-lite#117 (flip `TestSnapshotShape_SQLStoreDropsStampToday` + close TODO row 93).
8. Watch go-datastar for the QUERY-lane release (W3.2 sweep).
9. `FEATURES.md` rows for the QUERY search panel + JSON directory search (user-visible demo features).
10. The admin-demo e2e battery could assert the versioned script URL (§f.26 of the 11-52 report) — Go smoke covers the contract.

## g) Owner questions

1. **K2** — default `CommandIdempotency` ON for token-authenticated dispatch? (recipe shipped; default owner-gated per plan).
2. **K3** — send the prepared popover PR? (branch + draft ready on the templ-components checkout).
3. **Push timing** — the sibling authored `daf8fb5b` at 14:17 and may keep committing; push now (its module rides CI as it has all day) or hold for a quieter window?
   > **RESOLVED 2026-11-16:** the owner's standing GO settled it — pushed mid-morning with the sibling's committed wave aboard (`1d8df13b..9d159fdc`), CI watch riding.

## h) Verification receipts (this session)

- `nix run .#check-upstream-treadmill` → GREEN `20 repos · 81 prefixes · worst lag 0`.
- `nix run .#check-modules` → 101/106 (5 red = sibling in-flight, receipted); `.#coverage-gate` → 15/15 PASS; workspace race battery → rc=0.
- adminui: full suite GREEN (7.9s); coverage 67.3% ≥ 66; route-postures GREEN; CSS gates GREEN; e2e chromium admin-behavior **13/13** (31.2s).
- examples/admin-demo: build + suite GREEN (incl. 3 new B14 tests).
- templ-components checkout: `go test ./display/... ./cmd/tc/...` GREEN; headless computed-style probes (the evidence pack); fixed CSS computes `span-x-start bottom`.
- github-voice checker: 0 FAIL, 0 WARN (--ai-drafted).
- `check-todo-hygiene` → clean (79 rows, 0 dead); conflict-marker gate green on every commit (pre-commit ran throughout; no `--no-verify` on any cqrs-htmx commit).
