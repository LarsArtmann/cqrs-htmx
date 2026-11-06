# Fleet-plan wave-5 completion pass: A22 verified + B93/B94, A23, A24, CHANGELOG debt cleared, SA1019 sweep workspace-wide green, A2-adminui mid-flight

**Date:** 2026-11-06 22:42 CET · **Session:** resumption of the fleet-plan executor (wave 5 tail) · **Plan of record:** `docs/planning/2026-11-06_00-37_SUPERB-fleet-libs-audit-follow-through-pareto-plan.md` · **Mode:** execute + verify one item at a time.

## a) What was done (verified, with receipts)

1. **A22 appkit lifecycle hooks — COMPLETED (was landed-unverified at 22:03).** Setup build + vet + full `-race` suite green FIRST (12.7–15.4 s), then the remaining deliverables:
   - **B93** `setup/run_appkit_hooks_test.go` — `TestRunWithAppkit_HookForwardingAndLifecycleOrder` pins all four contracts: (1) Config's three `Appkit*Hooks` fields forward and fire (two start hooks prove in-order execution), (2) both start hooks ran before the first accepted HTTP response (post-listen/pre-serve), (3) exact `start:1 → start:2 → drain → shutdown` lifecycle order, (4) shutdown hooks run BEFORE `Bundle.Close()` — proven by a pre-subscribed SSE hub channel still being open inside the shutdown hook (hub Close closes subscriber channels). Passes in 0.16 s under `-race`.
   - **B94** docs: `setup/README.md` gains the three config-table rows + a lifecycle-hooks paragraph in § Drain posture; `docs/guides/production-readiness.md` gains § 9 "systemd Readiness Signals" (the A23 deliverable — `Type=notify` + `daemon.SdNotify` READY/STOPPING examples + the ordering guarantee). Cross-linked both ways.
   - Root `CHANGELOG.md` [Unreleased]/Added receipt. Setup lint driven to **0**: the A22 forwarding had pushed `runWithAppkit` over the funlen limit (62>60 — extracted `appkitServiceConfig` helper), `Bundle.Close` tripped nestif (extracted `drainAndCloseSSE` + `logger()`), and setup's two SA1019 sites fell in passing (below).

2. **A23 systemd docs note — COMPLETED** (rode A22's fields; see §a1 B94). The production-readiness checklist now has the full RunWithAppkit → sd_notify recipe.

3. **A24 upgrade-CLI atomic write — COMPLETED.** `cmd/cqrs-htmx-upgrade/rewrite.go` no longer writes rewritten consumer files with a bare in-place `os.WriteFile` (a crash mid-write left a TRUNCATED consumer file — the worst failure mode for a tool whose recovery story is "re-run me"). New `writeFileAtomic`: same-directory temp file (rename never crosses filesystems), original permissions restored (`CreateTemp` forces 0600), temp removed on every failure path, rename last. **B98 fingerprint-failure test:** `TestAtomicWrite_FailureLeavesOriginalIntact` (unwritable directory → loud error, original byte-identical, zero temp litter; skips under root) + `TestAtomicWrite_SuccessLeavesNoTempAndKeepsMode` (rewrite lands, mode 0644 preserved, no litter). Root build + vet + full cmd test package green; receipt in root CHANGELOG [Unreleased]/Fixed.

4. **CHANGELOG debt — CLEARED.**
   - **dashboardui A2 leg:** the drop-stats hub card (landed by the sibling DropStats wave, pinned by `overview_sse_hub_test.go` "1 · 1 dropped") had NO receipt in `dashboardui/CHANGELOG.md` → added ([Unreleased]/Added).
   - **health A3:** `NewDashboard` nonce forwarding (code + `TestNewDashboard_ForwardsCSPNonceOptions` + README § nonce-CSP all pre-existing) had NO receipt anywhere (health has no module CHANGELOG by repo structure) → added to root CHANGELOG [Unreleased]/Added.
   - **A27:** verified BOTH receipts already exist in root CHANGELOG (the round-45 M13 entry covers ErrorContext-in-body AND the family-derived upgrade-CLI exit codes). No action needed — the debt item was a false alarm.

5. **SA1019 sweep — COMPLETE, workspace-wide zero.** Real `golangci-lint run` per module (LSP diagnostics ignored all session — they still show stale dashboardui findings that the compiler-verified runs refute):
   - Already zero: root `.` (the W0 wave), identity-model, loginpage, datastar, huma, health, auditlog, dashboardui (re-verified live), usermgmt + systemadapter (prior sessions).
   - **setup (2 sites):** `machine.go` `cqrshtmx.EventCatalogHandler` → `eventcatalog.Handler`; `mount.go` `HTMXExtensionHandler`/`HTMXExtSSE` → `htmx.ExtensionHandler`/`htmx.ExtSSE`.
   - **adminui (4 sites, lane verified quiet first):** `assets.go` `HTMXScriptHandler` → `htmx.ScriptHandler`; `errorpage.go` `IsHTMXRequest` → `htmx.IsRequest`; `handler_audit.go` + `handler_users.go` `RenderPartial` → `htmx.RenderPartial`. Three files dropped the root import entirely; adminui build + lint 0 + tests green (7.3 s).

6. **A2 adminui SSEHubStats leg — ~70% LANDED (verified green so far):**
   - Landed and compiling: `SSEHubStatsSource` interface (`SubscriberCount() int`, `DroppedEvents() int64`) + `Config.SSEHubStats` field with honest docs (nil = no cards; setup wires automatically when its hub exists); `icons.Signal`/`icons.Bolt` icon keys; `handler_dashboard.go` appends two cards to the polled stats region — "SSE live" (green) and "Dropped events" (yellow tone when nonzero — drops are silent data loss and deserve attention).
   - NOT yet landed: the adminui test, the setup adapter + `attachAdmin` wiring (gated on `SSEPath`/`DataStarPath` — `attachPanels` runs BEFORE `attachSSE`, so the adapter must read the bundle's hub lazily; root surface verified: `Broadcaster.Health().SubscriberCount` + `DropStats.Total() int64`), setup adapter test, adminui README row, adminui CHANGELOG receipt, root CHANGELOG setup-wiring receipt, adminui+setup lint/test re-verify.

## b) What I fucked up (own mistakes, no excuses)

1. **The B93 test hang (20 s + two debug cycles) was self-inflicted:** I wrote `defer stop()` AND an explicit `stop()` — serveInBackground's done channel is one-shot, so the deferred second call blocked the full 20 s timeout while everything underneath had already shut down cleanly. Every existing test uses `defer stop()` alone; I should have read the helper's contract before composing a new call pattern. Fixed with a `sync.Once` guard (explicit stop before the post-shutdown asserts + leak-safe defer).
2. **A malformed multiedit batch on `errorpage.go`:** I planned import+symbol edits but submitted only the import edit — the build caught `undefined: cqrshtmx` immediately. Sloppy tool-call construction, one wasted cycle.
3. **Corpus ledger debt left open:** the sibling's +30 drift (53637 → 53667, already written into `docs/CORPUS_BASELINE` in the working tree) still has NO ledger sentence in `docs/status/README.md`, and my own session's doc growth (production-readiness §9, setup README rows, three CHANGELOG entries) hasn't been re-trued either. Deliberately deferred to the end-of-session baseline pass — which has NOT run yet (see §d).

## c) Session environment notes (for the next session)

- **Sibling lanes:** the 21-28 (m2m-W0) and 21-31 (root-package gates) sessions' work is all daemon-committed; a LATER sibling pass archived six reports (`TAIL_BASELINE` 20 → 14, ratchet green) and re-trued the corpus file to 53667 (uncommitted at session start; no ledger row). adminui had NO commits after 20:00 — lane confirmed quiet before my edits.
- The daemon committed everything I touched within minutes (verified via `git log` after each wave).
- templ was NOT needed this session (no .templ edits landed yet — A2's dashboard.templ was untouched; cards ride the existing stat grid).

## d) Remaining work (ordered)

1. **Finish A2 adminui leg** (§a6 remainder — the exact file list is there; ~30 min).
2. **Final battery:** full `nix run .#check-modules` (incl. the --examples leg, tidy-sweep, dep budgets, reexport policy, route postures, session-route wrappers); `docs/review/libs/index.html` go-etag row correction (erratum 5); plan Table A trues (A2/A3/A8/A13/A16/A18–A24 → done; A25 upstream-gated, A17 owner-gated); TODO_LIST trues; `nix run .#check-readme-badges`.
3. **Baseline pass LAST:** corpus re-true + ONE ledger sentence covering the sibling's +30 AND this session's growth; tail ratchet re-check; post-report survival check (`git ls-files docs/status/ | grep 22-42`).
4. **A25** stays parked on the go-cqrs-lite catalog release (external gate). **A17** stays parked on the owner answer.

## e) Verification receipts (commands run, all green)

- `setup`: `go build ./... && go vet ./...` OK; `go test ./... -count=1 -race` ok (12.7 s); `golangci-lint run` **0 issues**; focused B93 run `-race -v` PASS 0.16 s.
- `cmd/cqrs-htmx-upgrade`: `go test -count=1 ./...` ok (both new atomic-write tests PASS); root `go build ./...` + `go vet ./cmd/...` OK.
- `adminui`: build OK, `golangci-lint run` **0 issues**, `go test -count=1` ok (7.3 s).
- SA1019 scans (real golangci runs): root 0, identity-model 0, loginpage 0, datastar 0, huma 0, health 0, auditlog 0, dashboardui 0, setup 0 (fixed), adminui 0 (fixed).
- `check-docs-tail-budget.sh --ratchet` green (target 3, baseline 14).

## f) Reads that shaped decisions (don't re-read)

- `setup/run_appkit.go` + `run_appkit_test.go` (serveInBackground one-shot contract, testDrainDelay, existing defer-stop pattern), `setup/sse.go` (attachSSE ordering vs attachPanels, closeSSEDone/Broadcasters), `setup/bundle.go` Close (nestif site), `setup/setup.go` attachAdmin Config literal (setup.go:281).
- appkit v0.10.0 `config.go`/`service.go` (Hook alias, hooks run once in order, start post-listen/pre-serve, drain at drain start, shutdown post-connections — verified at the module cache, not assumed).
- go-sse v0.7.0 `fanout.go` `BroadcasterHealth` + root `sse_broadcaster.go` (`Broadcaster.Health()` wrapper), root `readiness.go` HubReadinessCheck (the existing live-count precedent).
- templ-components v1.27.0: `icons` is a SEPARATE module (`templ-components/icons@v1.27.0` — not a subdir), `StatToneYellow` = `"yellow"` → `bg-yellow-50 …` tile class; adminui `stats_partial_test.go`/`handler_test.go` harness (`newTestPanel(t, user, config...)` — a passed Config REPLACES the default, so it must carry Service).
- `cmd/cqrs-htmx-upgrade/rewrite.go` run() + `upgrade_test.go` (writeFile/fixtureTable helpers).

## g) Questions (max 3, owner-gated; work continued past them per the standing directive)

1. **identity-model v4.16.0** (carried): cut the tag myself or owner's call? Standing rule says never tag/push without instruction — it stays parked until answered.
2. **A17 render verification** (carried): sanction nix-profile chromium screenshots, or waive to structural evidence? Either answer closes the item; it is the last non-upstream-gated plan row I cannot finish alone.
3. **A22 hook ordering** (carried, now PINNED): consumer `ShutdownHooks` currently run BEFORE `Bundle.Close()` (telemetry flushes while stores/projections/SSE hub are still live — documented + test-pinned this session). Confirm the order or ask for the flip; a flip is a docs+test change, not a redesign.

**Next report's archive candidate:** the 05-24 wave-4 report (per the wave-5 session's queue).
