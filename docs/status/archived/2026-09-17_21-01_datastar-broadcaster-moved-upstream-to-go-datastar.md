# Status Report — 2026-09-17 21:01

## Session scope

**Question 1 (answered):** "What does our datastar do more than go-datastar?" — evidence-backed delta analysis (code-level, both repos read symbol-by-symbol).

**Task 2 (executed, ~90% done):** "Move Broadcaster upstream into go-datastar as a submodule and deprecate it over here." Cross-repo refactor spanning go-datastar + cqrs-htmx (datastar, setup, integration_test modules).

_(Format note: user explicitly requested `.md`; the status-report skill's HTML default was overridden per its own spec.)_

> **ANNOTATED 2026-09-20** (docs-health sweep): the cross-repo Broadcaster move is fully closed. The 2026-09-18 follow-up block below finished the docs+gates half; the 2026-09-20 train-lag sweep pushed the go-datastar lockstep tags and stripped the 3 TEMPORARY replaces (zero replaces remain repo-wide).
> - **§b:** b16/b17/b18 DONE.
> - **§c:** c19–c23 DONE (docs, gates, tags, replaces stripped).
> - **§f:** struck rows confirmed done; the only open item is 22 (upstream generic `EventBridge` — future boundary move, demand-gated).
> - **§g:** Q1 answered (tags cut + pushed; replaces stripped); Q2 (setup lint) resolved by the exhaustruct_v5 migration; Q3 (`Raw()` early removal) stands.

---

## a) FULLY DONE

### go-datastar side (new `broadcast/` submodule)

1. **`broadcast/` module created** — `github.com/larsartmann/go-datastar/broadcast`:
   - `broadcaster.go`: the moved implementation (Broadcaster struct, `NewBroadcaster`, `NewBroadcasterWithBufferSize`, `NewBroadcasterWithReplay`, `NewBroadcasterFromHub`, `Hub`, `Broadcast`, `BroadcastMany`, `BroadcastEvent`, `SubscriberCount`, `ServeHTTP` with subscribe-before-replay ordering + 15s heartbeat). Deprecated baggage (`Raw()`, `NewBroadcasterFromRaw`, `LastEventID` re-export) deliberately NOT carried into the fresh module.
   - `doc.go`: package docs stating the layering (protocol root vs connection-lifecycle submodule; EventBridge stays consumer-side per the root non-goals).
   - `broadcaster_test.go`: full behavior suite ported to the repo's **stdlib-only test style** (no testify — matches go-datastar/datastartest/static convention; cqrs-htmx's testify tests were rewritten, not copied): subscriber lifecycle, patch delivery via httptest, replay-on-reconnect, hub sharing, promoted SubscribeFilter, OnSubscribe, close/shutdown/health.
   - `go.mod` mirroring the datastartest replace pattern (`go-datastar => ..`, `static => ../static`), tidied.
   - `README.md` (install, quick start, API table).
2. **Hermetic verification green**: `GOWORK=off go build/vet` + `go test -race -count=1` all rc=0.
3. **Workspace verification green**: go.work-mode build/vet/test (now 4 modules).
4. **go.work** updated (+`./broadcast`).
5. **CI ci.yml** updated in all 8 module-list locations (paths ×2, matrix, workspace job, `go work use` check, replace audit, lint, erraudit loop, govulncheck).
6. **flake.nix** wired: `broadcastVendorHash` computed empirically (`sha256-He3vRF41044AdDz+LS0Sq4iF2n9MTeRvMj2DfKWPjO4=`), `broadcastSrc` minimal-fileset (ADR-004 pattern), `hermeticCheckBroadcast`, `checks.buildBroadcast`, and all module-list apps (test/test-race/build/vet/lint/lint-ci/erraudit/govulncheck/coverage).
7. **Nix FOD build green**: `nix build .#checks.x86_64-linux.buildBroadcast` rc=0.
8. **Docs**: CHANGELOG `[Unreleased]` entry; README (install line, comparison-table rows, new "Fan out patches" section); AGENTS.md (4-module table, replace-directives note, command lists).

### cqrs-htmx side

9. **datastar module → deprecated facade**: `broadcaster.go` is now `type Broadcaster = broadcast.Broadcaster` (transparent alias) + deprecated constructor var aliases + deprecated `NewBroadcasterFromRaw` function. `LastEventID` re-export relocated to `patch.go`.
10. **datastar go.mod**: require `go-datastar/broadcast v0.6.0` + TEMPORARY sibling replace with removal-condition comment.
11. **datastar tests migrated**: new slim facade test (alias compile-proof via helper param — ST1023-clean, deprecated pins, FromRaw pin, broadcast-typed connectSubscriber); `event_bridge_test.go` + `coverage_test.go` constructor sites moved to `broadcast.*`.
12. **datastar module fully green**: tidy/build/vet/`test -race`/**golangci-lint 0 issues** (under the drifted 2.13.2 binary, even).
13. **setup migrated**: `sse.go` uses `broadcast.NewBroadcasterFromHub`; `bundle.go` field `DataStarBroadcaster *broadcast.Broadcaster` (type-identical alias — zero consumer break); go.mod require + TEMPORARY replace. **tidy/build/vet/`test -race` (9.9s suite) green.**
14. **integration_test migrated**: both test files off deprecated constructors; go.mod require + TEMPORARY broadcast replace + **TEMPORARY family dev-replace `datastar/v4 => ../datastar`** (required: hermetic mode resolves published datastar v4.9.0 whose `Broadcaster` is still the concrete struct — without it, `broadcast.NewBroadcaster()` values don't type-check against `ds.NewEventBridge`). **tidy/build/vet/`test -race`/lint 0 issues — all green.**
15. **datastar module docs**: CHANGELOG `[Unreleased]` (move + deprecations + early `Raw()` removal documented), `doc.go` rewritten (layering + broadcast-anchored Quick Start), README.md updated (Quick Start, API table, pointer to broadcast).

---

## b) PARTIALLY DONE

16. ~~**Guides not yet updated** — `docs/guides/datastar-integration.md` (§2 still shows `ds.NewBroadcaster()`/`NewBroadcasterWithReplay`/dead `NewBroadcasterWithHeartbeat` — the heartbeat constructor was removed back in v4.1.0, pre-existing staleness noticed during this session) and `docs/guides/sse-and-datastar.md` (5+ `ds.NewBroadcaster*` references incl. the deprecated-API migration table). Section was located and read; edits not applied.~~ done (guides updated 2026-09-18)
17. ~~**cqrs-htmx AGENTS.md not yet updated** — the datastar module bullet still describes the pre-move Broadcaster; the "Module-level replaces" gotcha doesn't list the 3 new TEMPORARY replaces.~~ done (AGENTS.md updated 2026-09-18)
18. ~~**Format pass not run** — `nix fmt` (treefmt: golines/gci/dprint) not run on either repo's changed files; `gofmt -l` was checked for go-datastar/broadcast only, not for the edited cqrs-htmx files.~~ done (nix fmt clean)

---

## c) NOT STARTED

19. ~~go-datastar ROADMAP touch (theme 2 broadcaster example items now have a library home) + `docs/architecture.md` layer note.~~ done (go-datastar ROADMAP/docs updated)
20. ~~setup CHANGELOG entry (bundle field type spelling change + broadcast import).~~ done (CHANGELOG entry landed)
21. ~~Repo-level gates not run: `nix run .#check-modules` (dead-replace guard + strict drift + release-train for the 3 new replaces), `nix run .#coverage-gate` (datastar threshold 90% — suite shrank since behavior tests moved upstream), `check-cqrs-lint` (datastar has `.cqrs-lint.json`).~~ done (check-modules green (2026-09-20))
22. ~~go-datastar: `nix flake check` full run, docspec test (README snippets), actionlint on the ci.yml edits.~~ done (go-datastar flake check 8/8 + docspec + actionlint)
23. ~~Tag/push choreography (deliberately not started — never push without explicit instruction): `broadcast/v0.6.0` (+ root v0.6.0 lockstep), then cqrs-htmx `datastar/v4.9.1+`, `setup/v4.9.1+`, then strip the 3 TEMPORARY replaces.~~ done (tags pushed; replaces stripped 2026-09-20)

---

## d) TOTALLY FUCKED UP!

Nothing unrecoverable. Honest damage list:

24. **integration_test hermetic type mismatch** — I migrated tests to `broadcast.NewBroadcaster()` without anticipating that hermetic (GOWORK=off) mode resolves **published** `datastar/v4 v4.9.0` (concrete struct), breaking `ds.NewEventBridge(broadcast.NewBroadcaster())` outside the workspace. Cost one build round-trip; fixed with the documented family dev-replace pattern. I _knew_ this class of workspace-vs-hermetic resolution difference (it's in AGENTS) and should have predicted it before editing.
25. **Sloppy edit sequence**: hand-rolled `contains`/`stringsIndex` in the ported test (replaced with `strings.Contains`); a failed multiedit on setup/go.mod (bogus old_string); an import-order slip + temporarily-unused import in setup/sse.go (caught by build). All caught by verification, none shipped.
26. **Wrong nix attribute path first** (`.#buildBroadcast` — derivations live under `checks.`), one wasted background job.

---

## e) WHAT WE SHOULD IMPROVE

27. **Predict hermetic consequences before cross-module type edits** — when mixing a locally-replaced module's types with family-sibling types, the published-tag graph is what compiles in GOWORK=off. Check `go list -m` resolution first, not after the vet failure.
28. **Batch the doc pass with the API change** — I located stale guide content (dead `NewBroadcasterWithHeartbeat` since v4.1.0!) only mid-migration; a pre-change grep of guides would have surfaced the whole doc surface up front.
29. **Run the canonical formatter early** (`nix fmt` per repo) instead of deferring to the end (where it still sits, unstarted).
30. **Deprecation doc comments**: the alias's `// Deprecated:` block is long (it renders verbatim into every SA1019 message). Next time: one-line pointer + godoc link.
31. **The deprecation-message divergence**: go-datastar's facade-alias deprecation text (in bq's AGENTS/CHANGELOG) vs cqrs-htmx's — fine, but the alias doc mentions `SignalPatch/ElementsPatch` methods in bundle.go's field comment that don't exist by that name on the broadcaster — pre-existing doc inaccuracy I noticed and left (not my line, not this task's scope).

---

## f) Up to 50 things to do next

**Close out this refactor (P0):**

1. ~~Update `docs/guides/datastar-integration.md` §2 (broadcast constructors; remove dead `NewBroadcasterWithHeartbeat`; fix heartbeat prose to match 15s built-in + `sse.Stream.Heartbeat`).~~ done (closed 2026-09-18 (docs pass))
2. ~~Update `docs/guides/sse-and-datastar.md` (adapter table row, diagram label, snippets, deprecated-API table rows → `broadcast.*`).~~ done (closed 2026-09-18)
3. ~~Update cqrs-htmx `AGENTS.md` datastar bullet + Gotchas (3 TEMPORARY replaces, removal conditions, upstream move date).~~ done (AGENTS.md datastar bullet + replaces gotcha updated)
4. ~~Add setup CHANGELOG entry (DataStarBroadcaster field type spelling → `*broadcast.Broadcaster`).~~ done (CHANGELOG entry landed)
5. ~~`nix fmt` both repos; `gofmt -l` sweep of edited files.~~ done (nix fmt clean)
6. ~~Run `nix run .#check-modules` (dead-replace guard, strict drift, release-train advisory) — first gate the 3 new replaces pass through.~~ done (check-modules green (2026-09-20))
7. ~~Run `nix run .#coverage-gate`; if datastar coverage dipped below 90% (suite moved upstream), extend facade tests or adjust the documented number.~~ done (coverage-gate PASSED (datastar 100%/90))
8. ~~Run `nix run .#check-cqrs-lint` (datastar module).~~ done (cqrs-lint PASSED 13/13)
9. ~~go-datastar: `nix flake check`, docspec test, actionlint on ci.yml.~~ done (go-datastar flake check 8/8)
10. ~~Re-run full cqrs-htmx hermetic `nix run .#build` + `.#test` once the toolchain tug-of-war settles (root go.mod 1.27.1 vs go.work 1.26.7 — live concurrent-session issue, not mine).~~ done (full gate ladder green)

**Release choreography (P0, needs owner):**

11. ~~go-datastar lockstep cut: root + broadcast + static + datastartest → v0.6.0 tags (verify-tag.sh discipline), push.~~ done (go-datastar lockstep tags pushed)
12. ~~cqrs-htmx train: `datastar/v4.9.1` (facade), `setup/v4.9.1` (broadcast import), then strip the 3 TEMPORARY replaces (broadcast ×3 + family dev-replace) and re-verify hermetic.~~ done (datastar/setup train tagged; replaces stripped 2026-09-20)
13. ~~integration_test requires stay at published tags after strip; confirm `check-release-train` 0 unpublished / 0 lag.~~ done (check-release-train 0 unpublished / 0 lag)
14. ~~Post-tag: `examples/datastar-demo` README/docs sanity (code unaffected — no Broadcaster usage, verified by grep).~~ done (datastar-demo sanity verified)

**Follow-ups surfaced by this session (P1–P2):**

15. ~~go-datastar ROADMAP: mark theme-2 broadcaster example items superseded by the broadcast submodule; note in `docs/architecture.md` (fourth layer).~~ done (go-datastar ROADMAP/docs/architecture updated)
16. ~~Pre-existing setup lint findings under golangci-lint 2.13.2 (exhaustruct deprecated → old `//nolint:exhaustruct` on setup.go:103 dead; gochecknoglobals on `sseDrainTimeout`; nolintlint unused-directive) — repo-wide linter-version drift, needs a dedicated sweep (exhaustruct_v5 migration or config pin).~~ done (exhaustruct_v5 migration done (AGENTS gotcha))
17. ~~The live toolchain tug-of-war (sibling session bumping root go.mod to 1.27.1): lasting fix = `GOTOOLCHAIN=go1.26.7` for root-module commands OR a coordinated 28-module bump to 1.27.1 (policy decision).~~ done (coordinated 1.27.1 bump landed 2026-09-19)
18. ~~go-datastar `example/domain-adapter/main.go` now duplicates concepts the broadcast submodule ships — consider pointing it at broadcast or keeping as the dependency-free miniature (decision).~~ done (closed 2026-09-18)
19. ~~Root `sse_broadcaster.go:74` comment says "[NewBroadcasterFromHub] or datastar's equivalent" — reword to point at broadcast.~~ done (sse_broadcaster.go doc reworded)
20. ~~setup `bundle.go` field comment mentions nonexistent `SignalPatch/ElementsPatch` broadcaster methods — fix wording.~~ done (setup/bundle.go field comment fixed)
21. ~~datastar README "For SSE keep-alive..." paragraph + guide: document the built-in 15s heartbeat explicitly (currently only in godoc).~~ done (heartbeat documented)
22. Consider upstreaming `EventBridge` in GENERIC form (no go-cqrs dep) to go-datastar later — recorded as the natural next boundary move if non-cqrs users ask (this session's original recommendation).
23. ~~e2e sanity: run `nix run .#e2e` with the documented `/tmp` Playwright cache after the train (SSE/offline-sync paths touch the hub vocabulary).~~ done (e2e green)
24. ~~`docs/guides/v5-removal-inventory.md`: add the datastar Broadcaster facade + NewBroadcasterFromRaw + Raw()-already-gone to the v5 removal bundle list.~~ done (v5-removal-inventory updated)
25. ~~Re-verify `docs freshness gate` / uniform-at check after the tag train (templ-components-style drift gates read go.mod requires).~~ done (freshness gate green)

---

## g) Questions I can NOT figure out myself

1. **Tag & push authorization + order:** may a follow-up session cut and push `broadcast/v0.6.0` (+ go-datastar root lockstep v0.6.0) and then the cqrs-htmx `datastar/v4.9.1` + `setup/v4.9.1` train so the 3 TEMPORARY replaces can be stripped — or do you want to review the facade diff first? (I cannot push without explicit approval.)
2. **Pre-existing lint drift (exhaustruct_v5 / nolintlint in setup):** fix now in a dedicated sweep, or leave for the lint-gate owner given the toolchain tug-of-war is also live?
3. **`Raw()` early removal:** I dropped the deprecated method with the alias move (zero in-repo consumers, v5-removal already scheduled, CHANGELOG documents it). Acceptable, or do you want a wrapper-struct variant that preserves `Raw()` until v5 at the cost of breaking the forward `broadcast.Broadcaster` → `NewEventBridge` path?

---

_Report written from session memory per instruction (no fresh repo-wide research). Auto-commit daemon will pick this file up._

---

## ANNOTATED — 2026-09-18 follow-up session (docs + gates finished)

Everything in section c) NOT STARTED plus follow-ups 19-21 and 24 is now DONE:

- **Docs:** `datastar-integration.md` (broadcast imports, dead `NewBroadcasterWithHeartbeat` block replaced with the real 15s comment-frame heartbeat semantics — it never sent `event: ping`), `sse-and-datastar.md` (adapter table, diagram, snippets, deprecated-API table), `fullstack-wiring.md` route-split snippet, root `CHANGELOG.md` [Unreleased]/Changed entry (incl. the setup field spelling note), `AGENTS.md` (module bullet, key-deps, hub-vocabulary + replaces gotchas), `v5-removal-inventory.md` §2 (facade joins the removal class, dep budget back to 5 at v5), comment fixes 19/20/21 (`sse_broadcaster.go` Hub() doc, `setup/bundle.go` field doc, datastar README heartbeat). go-datastar: `docs/architecture.md` (broadcast node in the mermaid + protocol table + "four Go modules") and `ROADMAP.md` theme-2 refresh.
- **Gates:** coverage-gate PASSED (datastar 100%/90, setup 88.6%/80). cqrs-lint PASSED 13/13 modules. check-modules --report: 7/8 green (isolation, toolchain, release-train [0 unpublished, my 3 replaces + family dev-replace exempt], replace-directives, docs-freshness, docs-links, dep-budgets after justifying datastar 6 and documenting dashboardui 21). Remaining red: version-drift — the CONCURRENT session's active dependency sweep (grew 4→9 drift families while this session ran); left to them.
- **Discoveries fixed en route (cqrs-lint 4.6.0→4.8.1 tool drift):** V006 now anchors at the require block's first family line (suppression moved); C026 idempotency directive needed first-non-blank-line adjacency (reordered under the nolint); C035 sql_hydrate map is construction-time immutable (suppressed with justification); B024 es_setup bus recovery is the deliberate library-principle seam (suppressed); samber-do-demo `fmt.Errorf` → `errorfamily.NewRejection` (real fix, demo tests green); stale `examples/middleware-showcase/vendor/` re-synced (`go mod vendor` — the httputil v1.2.0 sweep missed it; this alone failed the root cqrs-lint gate via load errors). D005 doc-truth fixed in README ("v4.6.0" token dropped) + AGENTS ("v4.10.0+" token dropped).
- **go-datastar verified:** `nix flake check` 8/8 (hermetic FOD builds of all 4 modules at the committed tree), docspec compile-checked snippets green, actionlint clean, `nix fmt` clean.
- **Questions (g) status:** (1) tag/push authorization STILL OPEN — nothing tagged or pushed. (2) setup's 3 golangci findings are exhaustruct_v5-deprecation-era tool drift (golangci 2.13.2), untouched — separate sweep. (3) `Raw()` early removal stands as implemented (documented in CHANGELOG + v5 inventory).
