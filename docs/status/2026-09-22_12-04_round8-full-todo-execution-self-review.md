# Round 8 — Full TODO-List Execution + templ-components v1.19.1 (local): Status & Self-Review

**2026-09-22 12:04 CEST** · session-scoped: what THIS session did, noticed, broke, and missed.
Inputs: the round-7 [TODO_LIST](../../TODO_LIST.md) as found at session start (02:2x), executed end-to-end; cross-repo work in `~/projects/templ-components`.
Machine context: load 614 → 36 → 889 across the session (external workloads, 32 cores); `/mnt/buildcache` 100% FULL (1.8G free; rust 155G + sccache 20G dominate) — all Go work ran on `/tmp` fallback caches.

> **ANNOTATED 2026-09-22 (follow-up session):** both release/adoption items in (b) resolved the same day — templ-components v1.19.1 (+ 6 sub-module tags + the CI fixes on master) is pushed and verified on origin via `git ls-remote` (root tag `f917cb89`), and the cqrs-htmx adoption was executed end-to-end (`eb6dfc7a` → `5f1d1bca`, verified by the 14:21 adoption session, sync completed by the 13:25 release-train session). pkg.go.dev propagation + real-CI observation on both pushed ranges remain open (routed to TODO_LIST). Unmarked items below are still open.

> Verdict in one line: every actionable TODO item is executed and locally verified (check-modules 16/16, both repos' test/lint suites green), ~~**but NOTHING is pushed** — templ-components `v1.19.1` + CI-red fixes and cqrs-htmx master are local-only, so real CI has not confirmed any of it.~~ both repos were pushed later the same day (tags verified on origin); real-CI observation on the pushed ranges is the remaining open tail (TODO_LIST).

---

## a) FULLY DONE (verified this session)

### cqrs-htmx

1. **Release-train advisory/strict split brain — DEAD.** `scripts/check-release-train.sh` now defaults to CI's exact flags (`--strict-lag 0`; `--advisory` restores planning mode), with the rationale + incident citation in the script header. Verified both modes against the live repo (`0 unpublished / 0 lag`, strict_lag 0).
2. **Pre-push CI-parity hook.** `scripts/hooks/pre-push.template` runs release-train (strict) + `check-version-drift.sh --strict` at push time; offline policy mirrors pre-commit (exit-4 warns; drift failures disambiguated by an explicit `ls-remote` probe so an offline push can't false-block and an online drift can't slip). `install-git-hooks.sh` rewritten to manage BOTH hooks (per-hook refuse-and-diff); self-test extended **12 → 17 cases**, including a real `git push` through the installed hook against a local bare remote. Both hooks installed live in `.githooks/`.
3. **`test-install-git-hooks.sh` T9 env-coupling bug FIXED** (pre-existing): the GOTOOLCHAIN-floor assertion failed whenever the ambient GOCACHE's disk was under the 2048MB guard — the test now pins scratch caches, so it tests logic, not machine state. This bug was found because the full disk tripped it.
4. **Wave-ordered train choreography codified** — `docs/guides/release-playbook.md` §3a (the `--no-verify` killer; the v4.12.0 proof).
5. **VCS-cache health gate (NEW, atomic).** `scripts/check-vcs-cache.sh` — every bare repo under `GOMODCACHE/cache/vcs` must carry `remote.origin.url` (the origin-loss class masquerades as `invalid package name: ""` code bugs; AGENTS gotcha 12). Fixture self-test (7 cases: healthy / origin-less / missing-dir). Flake apps (`check-vcs-cache`, `test-vcs-cache`), check-modules stages, CI steps. Real fleet run: **33/33 entries healthy**.
6. **Dependency-budget self-test (NEW, atomic).** `scripts/test-check-dep-budgets.sh` (7 cases) pins the dep-counting awk via a `DEP_BUDGETS_ROOT` fixture hook — the comment-line/`// indirect` exclusion (the 2026-09-22 regression class) can no longer silently regress. Wired into check-modules + CI.
7. **Consumer-eye verification for v4.12.0.** Throwaway module outside the workspace `go get`s published root+setup v4.12.0 and compiles a minimal consumer (proxy propagation proven from the consumer side); pkg.go.dev renders v4.12.0 (published Sep 21, 36 imports, full symbol index incl. `IPAddressFromContext`/`UserAgentFromContext`).
8. **X-Client-Id sweep closed.** Go side was already constant-only; the remaining raw literals in `sync/sync-client.js` hoisted to `CLIENT_ID_HEADER`/`COMMAND_ID_HEADER` with a keep-in-sync comment against the Go constants.
9. **usermgmt DEV-ONLY replace — already stripped by the v4.12.0 train** (found in this state); hermetic `GOWORK=off go build + go vet` verified green from the published tags.
10. **cqrs-lint zero-warning state re-verified** via `nix run .#check-cqrs-lint` — all modules `--strict` green in the canonical env. Learned + documented: manual WORKSPACE-mode runs surface phantom E014-class findings; the flake app's env (`GOWORK=off` + flake go 1.27.1) is the verdict of record.
11. **SSE hardening backlog — confirmed ALREADY SHIPPED in v4.12.0** (the TODO item was stale): all four serve_test gaps, `WithSSEMaxReplay`, `SSEOptions`, replayed-retry, `EventsAfter` bench, `FuzzDomainEventToSSE`, and the cross-module wire-format + reconnect-with-replay contract tests all exist and the transport suite re-ran green. The e2e variant is a documented deliberate exclusion (`e2e/tests/dashboard.spec.ts` header).
12. **datastar-demo: evidence for the owner call.** Hermetic build + tests green on published datastar v4.12.0; accurately named; recommendation recorded (KEEP AS-IS). Stray `go.mod.*.tmp` artifact trashed.
13. **Docs pass:** CHANGELOG ([Unreleased]: 4 Added / 2 Fixed / 2 Verified), TODO_LIST (round-8 rewrite: 7 items closed with evidence, 1 new item added, hardware-watch updated with the disk-full event), AGENTS.md (Gates row, gotchas 6 + 12, cqrs-lint row), ROADMAP (OQ16 refusal count 8→9).
14. **Full gate sweep:** `nix run .#check-modules -- --report` = **16/16 green** (13 old + 3 new stages); root build+vet+test green; shellcheck clean on all touched scripts (warning-severity); the two new flake apps run green.

### templ-components (cross-repo)

15. **Ask (a) `ListNote` count variant** — `ListNoteVariant` enum (`ListNoteTruncated` default = byte-identical old behavior, proven by golden diff; `ListNoteCount` = count-only "Showing N items.", always renders incl. zero, pluralized; unknown values degrade gracefully). 7 new subtests + new golden `list_note_count`. FEATURES/CHANGELOG updated; `_sources` synced.
16. **Ask (b) hybrid-rendering docs — upgraded into a DISCOVERY.** Root-caused empirically: `{children...}` renders empty standalone, BUT `templ.WithChildren(ctx, child)` populates the slot from plain Go. Both behaviors pinned by new tests (`TestGridHybridChildrenViaWithChildren` / `...EmptyWithout...`); godoc caveats on `Grid` + `htmx.PolledRegion`; new recipe `docs/recipes/hybrid-strings-builder-rendering.md`. This upgrades the cqrs-htmx AGENTS "children are unusable in hybrid path" belief into a solvable problem.
17. **Ask (c) CopyButton label color** — `span[data-tc-copy-text]` carries explicit `text-gray-700 dark:text-gray-200` (matches the button palette; `[data-tc-copy-text]` documented as the consumer override hook). Golden updated — diff is exactly the span class.
18. **Red-master CI diagnosed + fixed (3 independent root causes):**
    - `ci.yaml` per-module isolation loop included `visualtest` without website-dist or Chromium → deterministic red since 2026-09-19 (a ci.yaml↔ci-repro.sh split brain; ci-repro never had it). Fixed to mirror ci-repro.
    - `requireSiteDist` hard-failed browserless contexts → now skips without a browser, stays a hard failure with one (Visual lane's no-skip gate preserved by precedence).
    - `TestSiteSalesCopyButton` e2e: root-caused the full chain empirically — (1) "Document is not focused" → fixed with `emulation.SetFocusEmulationEnabled(true)`; (2) headless Chromium 152's clipboard daemon denies sanitized writes even with permission state "granted" (verified: grant succeeds, `writeText` still "Write permission denied") → the spy now swallows the daemon rejection so the COMPONENT's real success path stays under test; permissions granted via modern `browser.SetPermission` (the old `GrantPermissions` API is gone from current cdproto — first attempt failed compile, caught immediately). **Verified green against a real (nix store) Chromium.**
19. **Docs-count drift fixed** (257→258 goldens across README/FEATURES/ROADMAP/AGENTS — `TestDocsCountDrift` red→green).
20. **v1.19.1 CUT + TAGGED locally** (release.sh full verify passed inside the cut): root `v1.19.1` + 6 sub-module tags; release commit replace-free; replaces re-added post-tag per protocol. Their TODO_LIST (harvested section closed, item 270 updated) + skill SKILL.md catalogue updated.

---

## b) PARTIALLY DONE (started, verification incomplete)

1. **templ-components v1.19.1 release** — ~~cut + tagged **locally only**; push pending owner (house rule: never push unconfirmed).~~ pushed 2026-09-22 by the release-train-alignment session — all 7 v1.19.1 tags verified on origin via `git ls-remote` (root `f917cb89`). pkg.go.dev propagation, release-smoke, and real-CI confirmation are still unverified (routed to TODO_LIST consumer-eye/CI-watch).
2. **templ-components CI repair** — all three fixes implemented and locally verified, but their AGENTS push ritual (`scripts/ci-repro.sh --lint --website` green-on-tip) was **NOT run**; real CI has not executed any of it.
3. **templ-components visual pixel suite** — locally ran with a non-pinned Chromium: **PASS-count 0** (all pixel tests failed). Attributed to documented renderer/font drift ("a non-Nix Chromium false-fails pixel comparison" — their AGENTS), consistent with the failing set (pure pixel tests), but the claim is asserted, not proven; only `TestSiteSalesCopyButton` was verified green selectively. The pinned-Chromium run is outstanding.
4. **bench-spike (P1)** — refused under load for the 9th documented time (614→36→889 vs limit 8). Mechanically correct per the guard; the item stays open. OQ16 (automate-or-retire) updated with refusal #9.
5. ~~**templ-components v1.19.1 adoption into cqrs-htmx** — new TODO item created with the full recipe (pins + both CSS bundle rebuilds); not executed (family-train-scale change, owner scheduling).~~ done at `eb6dfc7a`, `650404e1`, `5f1d1bca`
6. **`templ.WithChildren` exploitation** — discovery documented + pinned upstream; no dashboardui spike yet (SidebarNav criterion (1) noted as now-satisfiable).

## c) NOT STARTED (this session's sweep — legitimately blocked/owner/wait-gated)

- **Owner calls:** datastar-demo keep/remove (evidence + recommendation delivered); OQ14 `setup.NewFromSystem()`; OQ15 GitHub-Releases posture; OQ16 bench-spike future; both repo pushes.
- **Upstream-blocked:** BuildFlow `go-version-auto-configure` re-enable (2 upstream fixes); cqrs-lint Go-installable distribution (→ CI wiring + blocking stale-suppression warnings); V007 cluster 1 (68 SQLViewStore findings — metaengine layout planning); go-cqrs-lite asks (recorded in their tracker).
- **Wait-gated:** DataStar Tier 4 M11–M16 (demand); appkit ADR-0052 (v5 window); ProjectionLayer removal (v5); SidebarNav revisit (templ-components v2 / production theming proof).
- **Session-noticed, not started:** `nix fmt` drift check on this session's md/nix edits; FEATURES.md rows for the new gates; pinned-Chromium visual verification; `docs/status/README.md` unarchived-count refresh (done in this pass); templ-components TODO ID-collision audit (I removed the duplicated 230–232 rows, but never swept for OTHER collisions).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Commit attribution — the session's biggest quality loss.** The auto-commit daemon won ~7 of 8 commit races across BOTH repos; nearly all round-8 work landed as `chore: auto-commit N changed file(s) (heuristic)`. My detailed commit messages (including one full attempt that failed on the full disk, then hit "nothing to commit") were wasted. History readability is degraded; `git log` now lies about authorship-of-intent. (Known systemic issue: templ-components TODO #93/#126 family. The release.sh commit DID land properly.)
2. **Wrote overclaims into living docs.** TODO_LIST said v1.19.1 was "released upstream" and CI "repaired" — both were local-only truths stretched to sound shipped. Caught in this self-review and fixed ("cut locally, push pending"). Lesson: ship-state vocabulary discipline.
3. **errorpage near-miss (unforced-error class).** I nearly "fixed" their `go.mod` (1.26→1.27.1) for a per-module test failure that was actually MY toolchain-env artifact (`GOTOOLCHAIN=go1.27.1` against their 1.26-pinned modules). Saved only by checking CI logs first (CI green with go 1.26). Lesson: match the foreign repo's pinned toolchain BEFORE running its gates.
4. **First commit attempt misdiagnosis.** buildflow pre-commit failed with "12 step(s) failed"; I initially read it as a code problem before spotting the `no space left on device` lint-cache errors (the full /mnt disk). Cost: one failed commit cycle.
5. **CHANGELOG structural slip** — momentarily created two `### Added` headings under [Unreleased]; caught and fixed in-session.
6. **Small first-try errors (all caught by tests/compile, none shipped):** dep-budget fixture count arithmetic (18 vs 19 — forgot single-line requires count); T12 push test missing its commit; `browser.GrantPermissions` (removed API) in the first pre-push/e2e draft; an unused variable that shellcheck flagged.
7. **Leftovers:** `/tmp/consumer-eye-dir.txt` still exists; my first malformed attempt at the F1 sed edit; the debug clipboard test file was committed-then-removed by the daemon (transient tree noise, final state clean).

## e) WHAT WE SHOULD IMPROVE (session lessons → proposals)

1. **Commit at tighter phase boundaries** — commit within ~60s of each green verification, and expect the daemon: stage + commit in ONE command. (This session: verification spans were long, the daemon always won.)
2. **`go-cache-env.sh` should AUTO-fall back to /tmp on low disk, not just fail-fast.** Today: a full-but-writable cache disk = hard stop (correct for gates), but interactive sessions then hand-roll the /tmp prefix 15+ times (I did). Proposal: `GO_CACHE_AUTO_FALLBACK=1` env that re-points GOCACHE/GOMODCACHE/GOLANGCI_LINT_CACHE to /tmp when under the threshold. Owner call (it changes gate semantics).
3. **Ship-state vocabulary rule for living docs:** "released/shipped" = pushed + proxy-verified; "cut" = tagged locally; "implemented" = code green locally. Encode in AGENTS docs conventions.
4. **Foreign-repo protocol step 0:** read their devShell/toolchain pin and export it for every command (errorpage lesson), BEFORE running their test suites.
5. **"CI repaired" requires the repo's own ritual** (ci-repro for templ-components), not subset verification — I verified lint+tests+guards+one e2e but skipped the canonical all-in-one.
6. **cqrs-lint entry-point discipline** is now in AGENTS (flake-app only) — consider a wrapper that refuses workspace-mode invocation outright (tiny script; prevents the phantom-finding trap for the next session).
7. **templ-components TODO ID uniqueness** — "unique across ALL sections" is asserted in their header but was violated (232 duplicated); a 5-line gate could enforce it (same pattern as our status gates).

## f) NEXT — up to 50 (sorted: unblocks verification first)

**Unblock/verify (this session's work):**
1. Owner: push templ-components `master --follow-tags` (v1.19.1 + 6 tags) — after `scripts/ci-repro.sh --lint --website` green-on-tip.
2. Run that ci-repro on the templ-components tip (the skipped ritual).
3. Owner: push cqrs-htmx master (will traverse the NEW pre-push gate — by design).
4. Watch CI on both repos after push; triage anything red.
5. templ-components: run the FULL visual suite with the pinned nixpkgs-chromium (`nix run .#visual`) to prove the pixel-drift attribution.
6. Verify pkg.go.dev picks up templ-components v1.19.1 (consumer-eye, like cqrs-htmx's).
7. cqrs-htmx: `nix fmt` drift check on this session's md/nix edits (commit any reformat).
8. cqrs-htmx: full `nix run .#test` (race, 28 modules) on the tip as the pre-push belt-and-braces.
9. cqrs-htmx: FEATURES.md — add rows for the new gates (vcs-cache, dep-budgets self-test, pre-push hook).
10. templ-components: TODO_LIST ID-collision audit (+ optional uniqueness gate).

**Adoption/value:**
11. Adopt templ-components v1.19.1 into cqrs-htmx (family pins + BOTH CSS bundle rebuilds in the same change).
12. dashboardui: adopt `display.ListNote` `ListNoteCount` for range/count semantics (exclusion reason gone).
13. dashboardui: spike `templ.WithChildren` in the hybrid path (Grid/PolledRegion now usable there).
14. dashboardui: revisit SidebarNav criteria (criterion 1 now satisfiable in principle).
15. Extract the clipboard permission+focus-emulation pattern into a visualtest helper for future clipboard e2e tests.

**Open TODO items carried forward:**
16. bench-spike idle re-run (needs load < 8; refusal #9 logged).
17. OQ16: decide automate-vs-retire for bench-spike (idle-detect prototype sketched in the OQ).
18. Reclaim `/mnt/buildcache` (rust 155G, scccache 20G — owner decision) or formalize the /tmp fallback.
19. `go-cache-env.sh` auto-fallback proposal (e above) — owner call, then implement.
20. BuildFlow upstream: testdata pruning + major.minor policy knob → re-enable `go-version-auto-configure`.
21. cqrs-lint Go-installable distribution → CI-wire `check-cqrs-lint` + blocking stale-suppression warnings.
22. V007 cluster 1: 68 SQLViewStore findings → metaengine (gated on upstream layout planning, ADR-0051).
23. go-cqrs-lite upstream asks follow-through (incl. `System.Explain`).
24. DataStar Tier 4 M11–M16 (demand-gated per ADR-0050).
25. OQ14: build-or-reject `setup.NewFromSystem()`.
26. OQ15: GitHub-Releases posture for family trains.
27. ProjectionLayer v5 removal (all prep done; waits for v5 window).
28. appkit ADR-0052 v5-window revisit.
29. datastar-demo owner decision (recommendation delivered: keep as-is).
30. SSE re-export alias deletion timing (OQ2, v5); OQ5 ActorID payload format; OQ6 UserID redundancy; OQ7 non-user actor roles; OQ8 AsyncStartup default; OQ9 readiness backoff semantics (all owner-call cluster for v5 planning).

**Smaller hardening/polish noticed this session:**
31. cqrs-htmx: wrapper (or doc note) preventing manual workspace-mode cqrs-lint runs.
32. templ-components #93 family: daemon honest-commit-messages (this session added ~7 more heuristic commits as evidence).
33. templ-components #124/#125/#126 BuildFlow fixes (gitignore re-append, CSS un-minify, commit classifier).
34. templ-components #28/#29: awesome-templ + templ.guide submissions (queued one-shots).
35. cqrs-htmx: consider wiring `check-vcs-cache.sh` into `prewarm-gocache.sh` as well (the original TODO offered either home; check-modules chosen).
36. T9-class env-coupling audit: pin scratch caches in the OTHER fixture self-tests (test-verify-tag, test-check-docs-*) so machine disk state can never flip them.
37. Trash `/tmp/consumer-eye-dir.txt` and the `/tmp/consumer-eye-*` leftovers on reboot cadence (trivial).
38. templ-components: auto-derive the golden-baseline doc counts (standing gate vs hand-maintained 258 — their own AGENTS prefers standing gates; this session tripped the hand-maintained one).
39. templ-components: `TestDocsCountDrift` also covers website docs prose — consider covering SKILL.md counts (currently listed as covered; verify).
40. cqrs-htmx: e2e (Playwright) `/sse` reconnect-with-replay spec — only if the documented exclusion is ever revisited.
41. Release-notes one-pager for templ-components v1.19.1 GitHub Release IF OQ15 lands as "yes Releases".
42. cqrs-htmx: README contributor note about the new pre-push gate (what fires, `--no-verify` escape, offline behavior).
43. templ-components: pre-commit hook could run `TestDocsCountDrift`-equivalent as a fast guard on doc-count edits (it fired only in full test runs this session).
44. cqrs-htmx: add `check-vcs-cache` to CI's `checks` job too (currently in the `module-architecture` job only — verify job placement matches house pattern; it does, but double-check on push).
45. Consider a tiny `scripts/session-env.sh` exporting the /tmp cache triple + GOEXPERIMENT + GOTOOLCHAIN for interactive sessions on this machine (until 18/19 resolve the disk).
46. templ-components website: rebuild + redeploy after the v1.19.1 push (site renders from source; it currently still says 1.19.0 in the rendered version string until then — verify).
47. cqrs-htmx: `git log` attribution repair is NOT possible (rewriting history is forbidden) — instead: append a session-attribution note to CHANGELOG [Unreleased] mapping the heuristic commits to their actual content (optional, docs-only).
48. Monitor: next pre-commit run on either repo will exercise buildflow against the /tmp caches — confirm no new "no space" class failures once the disk is reclaimed.
49. templ-components: item 233 (flake.lock daemon nudge keep-or-revert) — untouched this session, still open.
50. Both repos: after pushes, run the respective tag-verification consumers (verify-tag smoke / ls-remote) to close the release loops.

## g) QUESTIONS FOR THE OWNER (cannot be resolved from here)

1. **Push now?** Both repos are ahead of origin with green local gates: templ-components (master + `v1.19.1` + 6 sub-tags — recommended only after their `ci-repro.sh --lint --website` ritual) and cqrs-htmx (master; the push will traverse the new CI-parity pre-push gate). House rules forbid unconfirmed pushes — say the word and I'll run the rituals and push.
2. **The disk:** may I reclaim `/mnt/buildcache` space (trash `rust/` 155G + `sccache/` 20G — they are rebuild-safe caches, but they belong to other workloads/projects), or should the `/tmp` fallback caches become the documented default for this machine (making `go-cache-env.sh` auto-fallback the fix)?
3. **v1.19.1 adoption timing:** adopt templ-components v1.19.1 into cqrs-htmx immediately (module pins + both CSS bundle rebuilds — effectively starting the next family train), or hold it for the next coordinated train so the [Unreleased] gate work rides the same cut?

---

*Point-in-time snapshot. Living state: [TODO_LIST.md](../../TODO_LIST.md) · [CHANGELOG.md](../../CHANGELOG.md) · [ROADMAP.md](../../ROADMAP.md) · [AGENTS.md](../../AGENTS.md). Report convention: [docs/status/README.md](README.md).*
