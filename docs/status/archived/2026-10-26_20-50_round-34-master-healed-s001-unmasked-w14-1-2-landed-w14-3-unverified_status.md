# Round 34 — master healed (streamid re-pin), cqrs-lint S001 unmasked+fixed, W14.1/W14.2 landed, W14.3 committed-but-unverified

> **ANNOTATED 2026-10-27** — ~~this report~~ **SUPERSEDED** by [`2026-10-27_01-30_round-35-w14-w15-landed-ci-battery-of-record-8-verdicts-parked-rows-held-gates-named_status.md`](2026-10-27_01-30_round-35-w14-w15-landed-ci-battery-of-record-8-verdicts-parked-rows-held-gates-named_status.md), which superseded by round-35 (w14.3 verified+fixed, W14/W15 complete, W17 verdicts). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-27 (round-23 (2026-10-27)).**

> Session 2026-10-26 ~15:00–20:50 CET. Lane W01–W08/W10–W12/W14/W15/W17 (sibling owns W09/W13/W16).
> Supersedes the round-33 report (archived under this one); round-33's §g questions are carried
> unchanged in §g below. CI was RED on master at session start; it is GREEN now.

## a) FULLY DONE

1. **Todos restored** (12 items) and kept current through the session.
2. **W12 §b1 — master CI RED healed.** `check-streamid-identity.sh` allowlist re-pinned
   `e2e/server/main.go:1` → `e2e/server/dashboard.go:1`. Note: the summary said "dashboard.go:112"
   but the pin format is `file:COUNT`, not file:line — the :112 was the site's line; the correct
   pin is `:1`. Candidate total stayed 12 across 8 files (nothing weakened). Gate rc=0, fixture
   self-test 5/5, full check-modules battery 58/59 (sole red = the held branching-flow row).
   Landed as `d821be26` (daemon raced the commit; amended with proper message). Pushed.
   **CI run 40338624795 on the heal = success — master green again.**
3. **Gotcha 41 recorded** (SQLite "no such table: events" under parallel `.#test` load = known
   flake; rerun once before triage) — docs/gotchas.md + AGENTS.md index row 41; corpus baseline
   32665→32669; all three docs gates rc=0. Daemon-captured (8d1cfa34, title left heuristic).
4. **cqrs-lint CRITICAL (S001) root-caused and fixed** (`db00d773`). The 2026-10-25 auth-audit
   enum const `AuthFailureTokenExchange` trips the "hardcoded secret" heuristic on the word
   "Token". Suppression anchored at the const-block line (the tool's own stale-suppression hint
   names the anchor). Gate `.#check-cqrs-lint` rc=0 across all 13 modules; identity-model builds.
   **The red had been silently live since 2026-10-25**: every commit since was daemon-captured
   (`--no-verify`), so the first hook-run commit of this session surfaced it. Bonus finding:
   check-modules runs only the cqrs-lint SELF-TEST — the real walk is standalone local-only, so
   no gate had caught it.
5. **W14.1 — loginpage AutoRedirect follow-through (complete).**
   - `Config.NoWebAuthn` implemented: page-level counterpart of `NoOAuth2` (hides passkey form +
     registration on THIS page while the shared service keeps WebAuthn for other surfaces);
     unlocks the OAuth-only recipe; `NoWebAuthn`+`NoOAuth2` rejected at `New`.
   - `AuthPrefix` validated root-relative at `New` (empty or leading `/`, not `//`) — it feeds
     link targets and the AutoRedirect 302 Location.
   - README: OAuth-only recipe (`NoWebAuthn`+`AutoRedirect`), session-agnostic semantics note
     (gate with `setup.RequireSessionRedirect`); config table rows updated.
   - FEATURES.md rows for `AutoRedirect` + setup's `LoginAutoRedirect` passthrough;
     CHANGELOG Unreleased Added+Changed entries.
   - 6 new tests (incl. stub WebAuthn provider + combined service). Receipts:
     `.#test-loginpage` rc=0, module lint 0 issues. Committed `74a82dc6` (+ daemon captures).
6. **W14.2 — dashboardui rune-safety + path escaping (complete).**
   - `truncate` backs off mid-rune cuts (was byte-slicing → mojibake cells);
   - `initials` decodes runes (was byte-indexing → "Über GmbH" → "ÃG" garbage initials);
   - new `pathSeg` (url.PathEscape) applied to 9 name href/action sites in dlq.templ +
     projections.templ (names with spaces/slashes previously 404'd or mis-split routes);
   - templ regenerated module-canonical; 4 regression tests incl. a full cut-width rune sweep.
   - Suite green, lint 0 issues. Committed `c3152426` (+ daemon captures).

## b) PARTIALLY DONE

1. **W14.3 — data-base-path hook + audit export streaming: code complete, verification
   INCOMPLETE.** Implemented: `data-base-path` attribute on the dashboard.js script tag (JS reads
   `dataset.basePath` first, src-regex fallback); `loadAllCommands`/`loadAllQueries` paged-cursor
   loaders (500/batch) replacing the zero-cursor `ReadFrom(zero, exportLimit)` head-reads — the
   export is now the COMPLETE trail instead of the oldest 10,000; nil-journal guard added; 2
   completeness tests (10,005-command CSV, 503-query JSON); `TestLayout_ScriptsAreDeferred` pin
   updated for the new attribute. **Not verified:** the re-run hit a 611s suite timeout + a
   golangci-lint timeout — box load average spiked to 33 during the run and is **139.95 now**
   (this is environmental, same class as the session's earlier govulncheck spawn-timeouts).
   The owner interrupt then landed mid-verification. The daemon has already captured all w14.3
   files (commits 285ae4f7/245992f2/8c8b7ab5/8e76ce22) — **unverified code is committed on
   master** (CI state on those pushes unknown; see §d).
2. **W12 wrap-up**: heal watched to green, but CI on the later pushes (`db00d773`..HEAD) was
   in-progress/unwatched when work continued; `.#verify-remote-tags` not run; book
   reconciliation (`git log --stat` since `f6dec11d` vs receipts) not run.
3. **§g defaults applied silently** per resumption instruction (heal streamid / hold
   branching-flow / hold filings) — all three still owner-unratified (see §g).

## c) NOT STARTED

- w14.4 (FindEventNeighbors early-stop + tail-scan cap, snapshot-store 404/500 split, htmx
  boost-inheritance guard), w14.5 (biome.json prove-or-remove + root allowlist gate sketch),
  w14.6 (sweep-plan M18/M20/M21 + RequireSession export-or-document).
- All of W15: art-dupl r3 accept-baseline + 2 extractions, check-skill-claims.sh skeleton,
  snippet compile gate + README split-brain scan, tooling canaries a–g, upstream ask drafts
  (drafts HELD per §g3 default).
- All of W17: 8 parked-row verdicts, TODO promote/unpark sweep, final battery, final receipt,
  CURRENT_FOCUS rewrite, plan terminal DONE line (also blocked on sibling lane).
- Unscheduled tails from the dashboardui row (page-size off-menu option, browser e2e smoke).

## d) TOTALLY FUCKED UP (this session's honest ledger)

1. **Unverified w14.3 code is on master.** The daemon captured the in-flight state while
   verification was still running; the interrupted verification never completed (suite timeout
   under load). The W11 lesson ("gate-before-push") partially repeated — though the daemon, not
   I, pushed. First action on resume: verify dashboardui (isolated, `GOWORK=off`), fix fallout.
2. **Daemon-race near-miss on the S001 fix**: the daemon snapshotted the INTERMEDIATE wrong
   anchor (line 177 instead of 171); my first `--amend` only re-messaged without staging the
   corrected file — caught by the post-amend `git status` check, fixed by a second
   stage+amend (`db00d773`). Rule confirmed: after every amend, verify the diff is EMPTY and
   the committed content is the intended one.
3. **Pre-commit hook env-failures under load** (twice): govulncheck spawn-timeouts (load 33),
   then the cqrs-lint finding surfaced. Used the documented `--no-verify` fallback with direct
   independent verification each time — but the first failure was mis-attributed to "pure
   environment" when it ALSO contained the real cqrs-lint red (10 failed steps, only the
   govulncheck ones read).
4. **Test-code sloppiness, each caught by compile/lint**: an invalid Go construct
   (`utf8.ValidString(got := ...)`), a missing `query` import, Han-script literals tripping
   gosmopolitan (fixed via `\u` escapes), 3 leftover unused nolint directives (nolintlint),
   4 style findings (nlreturn/wsl). All fixed; none shipped.
5. **View-tool discipline violated twice more** (awk inspection ≠ View) — edit tool rejected
   both; corrected. Recurring session-over-session failure mode.
6. **Round-33's next-steps instruction was subtly wrong**: "change the allowlist entry to
   dashboard.go:112" — the pin format is file:COUNT. Caught before editing, but handoff
   summaries should quote pin FORMATS, not just targets.

## e) WHAT WE SHOULD IMPROVE

1. **Wire the real cqrs-lint walk into check-modules (or CI).** The S001 red lived 1+ day
   because the battery only runs its self-test. One battery row would have caught it at the
   2026-10-25 landing.
2. **Post-amend verification ritual**: `git status --short` + `git show --stat` after EVERY
   amend (message-only amends are silent no-ops on content).
3. **Load-aware verification**: when `uptime` >> nproc, defer heavy suites (or expect the
   611s-class timeout) and prefer isolated per-module runs; the box is shared and was at
   load 140 at session end.
4. **Daemon vs verification sequencing**: the daemon commits faster than any full suite runs;
   for code waves, commit AFTER per-module green (as done for w14.1/w14.2) and treat any
   daemon capture of RED state as a P1 to re-verify, not an accepted state.
5. **Test fixture sizing**: the 10,005-command export test is fine logically but amplifies
   suite time under load; consider `-short` gating for pathological-size proofs.

## f) NEXT (ordered; ~35 items)

1. Re-verify dashboardui w14.3 in isolation (suite + lint + goldens); fix any fallout; proper commit message if the daemon title stuck.
2. Watch CI on `db00d773`..HEAD to terminal green (expect green; branching-flow self-skips in CI).
3. `nix run .#verify-remote-tags`.
4. W12 book reconciliation: `git log --stat f6dec11d..HEAD` vs wave receipts.
5. w14.4: FindEventNeighbors early-stop + tail-scan O(journal) cap.
6. w14.4: snapshot-store 404/500 error split.
7. w14.4: htmx boost-inheritance guard (2026-10-18 gotcha class).
8. w14.5: biome.json prove-or-remove (verdict).
9. w14.5: root allowlist gate sketch.
10. w14.6: sweep-plan M18 / M20 / M21 rows.
11. w14.6: RequireSession export-or-document decision.
12. w15.1: art-dupl r3 accept-baseline decision + the 2 deferred extractions.
13. w15.2: check-skill-claims.sh skeleton + fixture.
14. w15.3: skill-snippet compile gate + module-README split-brain scan.
15. w15.4: tooling canaries a–c (tidy-sweep, verify-tag poison, skip_steps canary).
16. w15.5: tooling canaries d–g (coverage fail-loud, toolchain runbook, codespell, single-source thresholds).
17. w15.6: upstream ask drafts (HELD per §g3 default until owner answers).
18. w17.1: one dated verdict per parked row (8 rows: billing S00, DataStar Tier-4, ReadLastN, SidebarNav, BuildFlow BF1, cqrs-lint fleet swap, check-cqrs-lint CI, GCL).
19. w17.2: TODO sweep — remove/verdict rows 60 (loginpage AutoRedirect: DONE this session) and 61 (dashboardui tails: w14.2/14.3 done, w14.4 pending); run check-todo-hygiene.
20. w17.3: final battery 4× rc=0 (test, lint, coverage-gate, check-modules).
21. w17.4: push + CI watch + ahead 0.
22. w17.5: final receipt superseding THIS report (archive ritual; manual corpus bump; tail ≤2).
23. w17.6: CURRENT_FOCUS rewrite (Wave D done; next thread named).
24. Plan-of-record terminal DONE line once the sibling lane (W09/W13/W16) also closes.
25. Proposal: add the real cqrs-lint walk as a check-modules row (from §e1).
26. Deferred design: true streaming export writers (bounded memory) — current fix pages reads but still materializes the slice.
27. Unscheduled tail: dashboardui page-size selector off-menu option.
28. Unscheduled tail: browser-level e2e smoke for dashboardui.
29. Optional: extend e2e loginpage spec with a NoWebAuthn OAuth-only recipe case.
30. Tagging decision for loginpage (new feature: NoWebAuthn) + dashboardui (fixes) — see §g Q3.
31. If authorized (§g Q1): branching-flow triage ledger + baseline refresh for the +46/−40/~111 split drift.
32. Re-check `docs/agents-notes.md` needs for the S001 unmasking story (daemon-masked CI red class).

## g) QUESTIONS FOR THE OWNER (carried + new; defaults applied where silent)

1. **(carried §g1)** Authorize the branching-flow baseline refresh (triage the +46 added/−40
   removed/~111 modified as W11-split location drift, re-pin `docs/analysis/branching-flow-baseline.sarif`
   + triage ledger entry)? Default held: the local gate row stays red, CI unaffected.
2. **(carried §g2/§g3)** Ratify the silent defaults: streamid heal (landed — master was red),
   and the upstream filings (S49 StateShape + system.Execute asks) — file now or keep holding?
   Also carried: W04-defaults ratification from round 33.
3. **(new)** Release timing: this session shipped unreleased code in loginpage (a NEW config
   feature: `NoWebAuthn` + `AuthPrefix` validation) and dashboardui (rune-safety/escaping/export
   fixes). Cut a train now (loginpage minor + dashboardui patch) or batch everything into the
   W17 closure train? I cannot pick the release calendar myself.

---
*Verification receipts this session: check-streamid-identity rc=0 + selftest 5/5; check-modules
58/59 (held row only); check-cqrs-lint rc=0 (13 modules); test-loginpage rc=0; dashboardui
suite rc=0 (w14.2 state) + lint 0; docs gates 3× rc=0. Unverified at write time: the w14.3
dashboardui state (see §b1/§d1).*
