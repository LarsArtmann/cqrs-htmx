# Status Report — BuildFlow full-run red: release train + type-check + residue repairs

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-19_12-18_issue-sweep-self-review-and-next-50_status.md`](2026-10-19_12-18_issue-sweep-self-review-and-next-50_status.md), which successor's §f carries vendorHash, lychee excludes, dependabot cap, BuildFlow binary; systemadapter red closed 10-19; gomod-freshness is D15; nix-fmt loop is the BuildFlow filing TODO row. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-24)).**

> **Point-in-time snapshot: 2026-10-15 00:07 CEST.** Scope: THIS session only.
> Input was the failed `buildflow --fix --build-mode=full` run (21:50–21:55,
> exit 69, 3 failed steps). Shared-tree session: the auto-commit daemon
> raced me repeatedly (documented below); at snapshot HEAD is 8 commits ahead
> of origin/master, tree clean, nothing pushed by me.

---

## The three failed steps and their fixes (verified)

| BuildFlow step | Root cause | Fix | Verification |
| --- | --- | --- | --- |
| `type-check` | e2e dep refresh (pnpm-update churned `e2e/bun.lock`) made `page.evaluate(string)` return `unknown` → 8 TS errors in `e2e/tests/sync.spec.ts` (TS2571/TS18046/TS2339) | `SyncSnapshot` type via the evaluate generic at all GET_EVENTS sites; moved the direct `window.cqrsSync.pull()` call into the file's established string-expression idiom (`PULL`) | `tsc --noEmit` rc=0; the pre-commit hook's own `type-check` step ✔ (2.1s); daemon-captured as `c7ce6af7` |
| `go-mod-update` + `go-mod-update [setup]` | Gotcha-30 class: `setup.go:336` consumes `loginpage.Config.AutoRedirect`, which exists only in uncommitted-then-unreleased loginpage code; hermetic (GOWORK=off) builds resolve published loginpage/v4.12.3 → `unknown field AutoRedirect` | Full release train: loginpage CHANGELOG cut (`685b1c60`) → hermetic loginpage build+vet+test green → `verify-tag.sh loginpage v4.13.0 --push` → tag-cache refresh → `bump-dep.sh 'cqrs-htmx/loginpage/v4$' v4.13.0` sweep (setup, integration_test, 2 demos; all tidy/build/vet PASS) → `verify-tag.sh setup v4.15.0 --push` | Hermetic `go test` green on setup (9.4s) and integration_test (1.1s); strict release-train 0/0/0 at every gate |
| (consequent) CI `mod-tidy` job | go-mod-update's workspace-mode sweep wrote union-graph-only go.sum hashes into 23 modules — exactly what CI's per-module hermetic tidy -diff rejects (gotcha-27d direction) | Hermetic `go mod tidy` applied to the 23 drifted members (testdata untouched by design); −366 lines of residue (`1a484901`) | Hermetic `tidy -diff` now clean on all 28 members; spot hermetic builds green |

## a) FULLY DONE

1. **TS type-check fixed and verified** (`c7ce6af7`, daemon-captured; tsc rc=0).
2. **loginpage/v4.13.0 tagged + pushed** at the CHANGELOG-cut commit (content assertions + message guard + post-push ls-remote all green).
3. **Consumer sweep landed**: loginpage require v4.13.0 in setup, integration_test, examples/setup-demo, examples/async-startup-demo (`e1dee56b`, daemon-captured; my release message amended onto the follow-up CHANGELOG commit `b7a5e848`).
4. **setup/v4.15.0 tagged + pushed** (all gates green).
5. **Root CHANGELOG receipt updated** with the cut versions + sweep note.
6. **go.sum union residue stripped** across 23 modules (`1a484901`).
7. **mypy findings fixed**: `check_file` return annotation was lying (`tuple[int,int,int]` vs actual `tuple[int,int,list[str]]` — one root cause behind both mypy errors); dead `DEFAULT_DIR_ARG_HELP` removed from `scripts/lib/status_table.py` (`22633fa2`, daemon-captured + amended). Verified: mypy rc=0 on both files; equivalence + check-status-rows + normalize-status-rows self-tests all green; row gate green over 473 files.
8. **SECURITY.md aligned** with the securitymd contract (`# Security Policy` title, Supported Versions table reflecting the real latest-tag-only policy, Security Practices wrapper) — `buildflow -s securitymd` rc=0 (`7f4b6f01`).
9. **branching-flow baseline re-pinned 164 → 165** with a triage-decisions amendment: +1 legit new (loginpage Config 16 fields via AutoRedirect — the config/composition-root reject class), +3/−3 pure anchor-drift twins from the erraudit reflows. Gate at HEAD: **OK (+0 added)** (daemon-captured as `5425e73e`).
10. **check-modules re-run**: everything green except branching-flow (fixed above). module-isolation (GOWORK=off build+vet all modules) ✓ and dep-budgets ✓ — these are CI's `module-architecture` job legs, whose setup leg was the AutoRedirect victim.
11. **systemadapter red precisely diagnosed** (was masked inside the battery failure, see §d): workspace mode green (0.02s) vs hermetic red (deterministic 5.02s `system: not found`); pin-bisects prove NO single published-tag combo fixes it (system v4.13.0 AND v4.14.0 fail; metaengine v4.18.0 AND v4.19.0 fail) → the fix lives in go-cqrs-lite's UNPUBLISHED cross-module local state (system +10, metaengine +30, systemscenario +21 post-tag commits). Blocked on the next go-cqrs-lite train; foreign repo actively churning (100-file heuristic commits), so no tag cut from here (gotcha-36 doctrine).

## b) PARTIALLY DONE

1. **Full-workspace verification battery**: `nix run .#test` ran to completion — 27/28 modules green; systemadapter red (§d/pre-existing). NOT re-run after the residue-strip + sweeps (setup/integration_test/loginpage legs individually re-verified hermetically in isolation).
2. **`nix run .#lint` (full canonical lint) NOT run** — one full pre-commit hook run died in the golangci parallel-runner lock-contention class (5 modules "failed" with zero findings; all 5 independently re-verified clean by direct sequential `golangci-lint run`). Hook-scoped lint green on everything else.
3. **The 3 originally-failed BuildFlow steps NOT re-run as BuildFlow steps** (`buildflow -s go-mod-update`, `-s 'go-mod-update [setup]'`, `-s type-check`) — each root cause is verified fixed by equivalent-or-stronger evidence, but the receipt run is owed.
4. **Daemon commit `5425e73e` (baseline re-pin + triage) carries a heuristic title** — my chained commit lost the race; amend still possible (HEAD~0 is it? no — it IS HEAD, amendable).
5. **Push**: 8 commits ahead of origin/master, NOT pushed. Tags (loginpage/v4.13.0, setup/v4.15.0) ARE pushed.

## c) NOT STARTED

1. Post-push CI watch (`.#push-and-watch`) — nothing pushed.
2. `vendorHash.nix` staleness (go.sum churn tripped the FOD freshness warning) — sanctioned repair is `buildflow -s nix-hash-fix --fix`; not run. `nix build` users hit a hash mismatch until then.
3. TODO_LIST/AGENTS harvest of this session's findings (docs-health HARVEST; incl. the refined systemadapter diagnosis and the loginpage session's still-unharvested §f).
4. The cqrs-lint 50 / e2e-server golangci 4 / file-size 86 / jscpd 4 / art-dupl 546 detect-only findings — inspected, classified as known/baselined noise, untouched.
5. lychee archived-docs excludes (`docs/modularization/archived`, `docs/plans/archived`) — preflight suggestion, not acted on.
6. BuildFlow-repo issues observed from here (NOT filed, foreign repo): resume-checkpoint SQLITE_BUSY flood under concurrent instances; gomod-freshness check disagrees with CI's hermetic tidy (workspace vs hermetic mode — it re-creates the exact residue class its own go-mod-update step wrote); go-work-paths `/v4`-suffix warning false-positives on correct multi-module workspaces; type-check step classified a real tsc diagnostic as "no compiler diagnostics"; dependabot 29-modules > 20-entry cap info.

## d) TOTALLY FUCKED UP

1. **Nothing broken, but three races lost against the daemon**: my chained fmt→add→commit sequences were beaten three times (`c7ce6af7`, `e1dee56b`+`8b074422`, `5425e73e`) — the daemon captured the content mid-hook/mid-verify. Two were message-amended after the fact; `5425e73e` not yet. Consequence: train readability debt, zero content loss.
2. **One full pre-commit hook run (2m15s) wasted** dying in the golangci lock-contention class — I chose the full hook for the sweep commit instead of anticipating the contention that the SQLITE_BUSY flood had already announced. Recovered via documented `--no-verify` + independent verification + justification.
3. **Repeated my own shell-path mistake twice** (`cd systemadapter` while already inside it, `source scripts/lib/...` with a relative path from a module dir) — two wasted diagnostic calls, one of which printed a misleading `rc=1` that I correctly distrusted but still cost a round trip.
4. **The full battery surfaced systemadapter only at the very end** — I could have run it in parallel with the early recon (it was predictably the one module whose hermetic story was suspicious, per gotcha 40). The diagnosis then had to be squeezed in after.
5. **Assumed "0 unpushed" at 22:14 meant the daemon pushes** — I never re-checked `git log origin/master..HEAD` before building my commit plan; the pile grew to 8 through my own commits and I never reconciled against what CI would see per-commit.

## e) WHAT WE SHOULD IMPROVE (self-critique)

1. **The branching-flow gate prints counts but not the added findings** — I had to reverse-engineer the +4 via a manual analyzer run + SARIF parsing (and my first jq diff was garbage because `--baseline` writes the delta to stderr and an empty stdout). The gate should print (or keep) the added-findings list. Cheap script fix, high leverage.
2. **Tidy-mode split brain is now mechanized**: BuildFlow's go-mod-update writes workspace-union go.sum state; CI's mod-tidy + the repo's hermetic doctrine reject it; BuildFlow's own preflight then warns "31 modules need tidy". Until BuildFlow fixes the mode, every `--fix` full run re-creates this residue — the repo could add a hermetic-tidy sweep gate to catch it before push (or just accept re-running the loop after each full run).
3. **Daemon-race mitigation**: my chained commits still lose because `nix run .#fmt`'s multi-second window is the gap. The reliable pattern this session: let the daemon capture, then `git commit --amend` HEAD. I should START from that pattern (edit → wait for capture → amend) instead of racing.
4. **Working-directory discipline**: absolute paths (or `$REPO_ROOT` anchors) in every shell call that mixes `cd` with `source`/relative tooling. Two self-inflicted failures of exactly this.
5. **The vendorHash staleness was visible in BOTH preflights before I touched go.sums** and I deferred it until after the push-dependent work; now it's the last blocking-ish cleanup and touches nix hashing (gotcha-31 care). Should have run `nix-hash-fix` early when the tree was quiet.
6. **systemadapter's gotcha-40 entry is now stale**: it says the failures were "exposed when adminui healed" — the refined truth is "no published go-cqrs-lite pin combination passes; fixed only by unpublished local state; blocked on the foreign train". AGENTS.md should carry that so the next session doesn't re-bisect what I bisected.

## f) NEXT — up to 50 things (impact-ordered)

1. Amend daemon commit `5425e73e`'s message (baseline re-pin).
2. ~~`nix run .#lint` full canonical run.~~ done — lint rc=0 by round-33; the 2.14/2.15 split closed at bcc64baa (2.15.0 everywhere)
3. ~~Push master (8 commits; pre-push hook runs strict train + version-drift — expected green).~~ done — master pushed many times through the strict pre-push gates; v4.18.0 world tags verified on origin today
4. ~~`.#push-and-watch` + step-level triage of the resulting CI run (expect: build/test/mod-tidy/module-architecture/lint(setup) heal; `test` job may stay red on the systemadapter leg until the go-cqrs-lite train).~~ done — push-and-watch + step triage done — CI green receipts since (8101189f onward; round-36 wrap 179/179 tags)
5. Re-run the receipt: `buildflow -s go-mod-update`, `-s 'go-mod-update [setup]'`, `-s type-check`.
6. ~~`buildflow -s nix-hash-fix --fix` for vendorHash.nix (run the derivation; never trust the checker's hash alone — gotcha 31).~~ done — superseded — gotcha 43: the stale-vendorHash warning is a documented NON-FIX (the hash belongs to the benchstat derivation, not this repo)
7. ~~Re-run full `nix run .#test` at final HEAD (post-sweep state).~~ done — .#test rc=0 at final HEADs (round-33, race suite green across all modules)
8. ~~TODO_LIST: systemadapter hermetic red — record as blocked-on-go-cqrs-lite-train with this session's bisect matrix (system v4.13/v4.14 × metaengine v4.18/v4.19 all red; workspace green).~~ done — superseded — the family trains bumped the pins; systemadapter suites green at worst-lag 0 (2026-11-06 receipt battery); today's residual red is the unrelated catalog templ poison
9. ~~AGENTS.md gotcha-40 refinement (§e-6 above).~~ done — gotcha 40 carries the sequential-CI-red lesson (verified in the AGENTS table)
10. ~~docs-health HARVEST of this report + the unharvested 2026-10-14 21:46 loginpage report §f.~~ done — the 10-14 21:46 report was evicted with the corpus; its §f was re-listed here (items 16–33) and closed via round-34 W14 + this drain
11. Make `check-branching-flow.sh` print the added findings (or keep the delta log at a stable /tmp path).
12. ~~Consider a repo-side hermetic-tidy sweep gate (or a post-BuildFlow `--fix` checklist item) to mechanize the residue strip.~~ done — tidy-sweep.sh landed — per-module hermetic tidy, testdata excluded by construction (round-35 canary (a))
13. ~~Fix the 4 remaining golangci findings in e2e/server/main.go (varnamelen `f` + wrapcheck ×3 — likely wrap with event-family context; e2e module).~~ done — fixed in round-33 W11 — the e2e lint findings died with the file splits (varnamelen f→form, wrapcheck nolints); .#lint rc=0 since
14. ~~Fix the cqrs-lint "embeds BasicCommand by value" pair (e2e/server/main.go:453, examples/sync-demo/main.go:44).~~ done — 2026-11-06 — B58/B59 BasicCommand value-embeds, fleet-plan A13, receipt in root CHANGELOG [Unreleased] (TODO leverage row)
15. ~~cqrs-lint STRONG_ID trio on root `sync_pull.go` (SyncEvent.EventID/StreamID/BackendID) — adjudicate wire-DTO vs branded-ID (ADR-0056 wire contract says wire DTO; suppress or accept).~~ done — resolved — routed: TODO_LIST sync follow-through N5 streamId-form watch — the STRONG_ID adjudication rides the id-module branding publish
16. ~~loginpage: close the NoOAuth2/NoWebAuthn asymmetry (`Config.NoWebAuthn`, setup passthrough) — prior session's §f-8.~~ done — Config.NoWebAuthn shipped 2026-10-26 + setup passthrough (FEATURES.md:356 — verified in tree)
17. ~~loginpage: `AuthPrefix` validation at `New` (feeds a 302 Location now) — prior session's §f-10.~~ done — AuthPrefix validated root-relative at New (FEATURES.md:356; loginpage README:85; handler_test.go:1151)
18. ~~loginpage: AutoRedirect × existing-session semantics decision (§g-1 there).~~ done — decided + documented — session-agnostic by design, wrap with setup.RequireSessionRedirect (FEATURES.md:356)
19. loginpage: stale "Zero external asset requests / CSS inlined" bullet in doc.go — prior session's §e-2, still unfixed.
20. ~~FEATURES.md inventory for AutoRedirect + LoginAutoRedirect.~~ done — FEATURES.md:356 OAuth2 Auto-Redirect row + :432 LoginAutoRedirect (verified in tree)
21. OAuth-only recipe guide (docs/guides/) incl. first-login account-creation path.
22. Runnable OAuth-only example (setup-demo variant with LoginAutoRedirect).
23. ~~Behavioral setup e2e: OAuth-stubbed service → assert bundle.Login 302s.~~ done — TestNew_LoginAutoRedirect (setup_defaults_test.go:420) + the raw-302 assertion pattern (setup_authhandler_config_test.go:146-158)
24. HEAD-request 302 test + 302-fallback-body disambiguation test (prior §f-18/19).
25. ~~Test AutoRedirect with explicit single-provider OAuth2Buttons (not just auto-detect).~~ done — TestServeHTTP_AutoRedirect_SingleProvider at loginpage/handler_test.go:902 (verified in tree)
26. Test AutoRedirect + NoRegistration combination pin.
27. Remove `loginpage/coverage.out` if stale (prior §f-23).
28. Align loginpage `.golangci.yml` exhaustruct → exhaustruct_v5 (prior §f-24; mid-migration train).
29. ~~setup/config.go: migrate deprecated `usermgmt.TenantID`/`User` aliases to identity-model imports (prior §f-25).~~ done — setup/config.go carries no usermgmt.TenantID/User alias uses today (verified by rg); residual alias debt rides the v5 re-export row
30. ~~Re-verify loginpage gates after the concurrent SSE-flake/exhaustruct session's work lands (prior §f-26; its status doc was untracked at 21:44).~~ done — loginpage gates green — suites + lint through the W14 close and every train since
31. ~~Post-train sanity: `.#check-css-bundles` + `.#check-codegen` (prior §f-48; no CSS/templ changed, expect green).~~ done — css-bundle + codegen gates green through the battery rounds (33–36)
32. ~~Root CHANGELOG: split the one-paragraph Unreleased blob per-module at the next root cut (prior §f-16).~~ done — root CHANGELOG per-module sections — the v4.18.0 section cut at 6b804a01
33. examples README sweep for AutoRedirect mention post-release (prior §f-46).
34. ~~Watch go-cqrs-lite for the family train; when it lands: bump pins here, re-run hermetic systemadapter tests, expect the battery fully green.~~ done — the GCL trains landed; pins swept to worst-lag 0; battery green (2026-11-06 receipt)
35. ~~Consider pinning the metaengine/projectionadapter sibling-replace story: CI clones go-cqrs-lite for it — the ci.yml comment's removal condition should be re-checked (is the replace still in go.mod?).~~ done — ci.yml:26-33 documents the clone step + the removal condition (verified in tree)
36. ~~BuildFlow repo: file gomod-freshness mode-mismatch issue (workspace tidy check vs hermetic consumers).~~ done — resolved — routed: S55 packet §T01 (gomod-freshness pre-commit asks) + TODO "Upstream ask drafts batch" row — BuildFlow asks owner-gated
37. ~~BuildFlow repo: resume-checkpoint SQLITE_BUSY under concurrent instances (WAL/timeout).~~ done — resolved — routed: TODO "Upstream ask drafts batch" row names BuildFlow resume-checkpoint SQLITE_BUSY
38. BuildFlow repo: go-work-paths `/v4`-suffix false positive for multi-module workspaces.
39. ~~BuildFlow repo: type-check "no compiler diagnostics" misclassification of pretty-format tsc output.~~ done — resolved — routed: same row names the type-check no-diagnostics misclassification
40. BuildFlow repo: binary-freshness — the installed binary is behind its repo HEAD (7ccd809); rebuild/reinstall.
41. ~~lychee.toml: add `exclude_path` for the two archived doc dirs.~~ done — superseded — no lychee.toml exists; check-docs-links.sh owns link checking today
42. ~~dependabot 29-module > 20-entry cap: document or split config.~~ done — dependabot.yml posture block documents the cap decision (file head, reviewed 2026-09-20 / accepted 2026-10-07)
43. Baseline ratchet-down candidates (165 pin): the three re-attributed phantom findings could be genuinely fixed (brand the ActorID/NewActorID raw slots) as a ratchet-down exercise.
44. ~~file-size-check 86 findings: biggest offenders (>450 lines) triage for splits.~~ done — the five biggest offenders split in round-33 W11 (setup/config, declarations, service_core, layout, e2e main); check-large-files green
45. ~~jscpd 4 script-duplication findings: extract shared fixture-harness helpers in scripts/selftests.~~ done — scripts/selftests/lib/ carries the shared fixture-harness helpers (verified in tree)
46. ~~The go-licenses "cannot find license" noise for examples/* (devShell shim covers usermgmt/webauthn; examples lack LICENSE files — add or exclude).~~ done — superseded — no go-licenses step exists in CI or scripts today; the noise class is gone
47. ~~SECURITY.md: consider adding the actual disclosure contact link to the repo's GitHub Security Advisories if enabled.~~ done — SECURITY.md:147-149 reports via GitHub security advisories + security@lars.software (verified in tree)
48. vulnix 23 nix-store drv advisories: mostly stdenv bootstrap; re-scan after next nixpkgs bump.
49. Interrogate/ruff/shellcheck/mypy not in devShell (BuildFlow ran them via nix-run without project deps): add to devShells.default or accept.
50. ~~Reconcile `docs/status/` filename convention: two `_status.md`-suffixed files from different sessions within minutes (2026-10-14 21:44/21:46) — session-slug prefix norm (prior §f-44).~~ done — the _status.md session-slug convention is the norm — all living and archived files conform (verified)

## g) Questions I cannot answer myself (decisions, not lookups)

1. **Push now or hold?** The 8-commit pile heals 5 of the 5 known-red CI jobs except the `test` job's systemadapter leg, which stays red until go-cqrs-lite's next train (their tree is churning; doctrine forbids cutting their tags from here). Push and accept a partially-red CI, or hold the push until the foreign train lands?
2. **systemadapter depth:** is the recorded bisect + blocked-on-foreign-train disposition the right depth, or do you want a full old-combination bisect (the entire pre-reentrancy-wave pin set) to identify the exact guilty pair — and would you ever authorize cutting a minimal go-cqrs-lite tag from this side if their churn stalls?
3. **The residue class:** BuildFlow's go-mod-update will re-write this exact go.sum residue on its next `--fix` full run. Do you want a repo-side hermetic-tidy gate (item 12) so pushes are protected regardless, or is that BuildFlow's bug to fix upstream (items 36) with the repo just re-running the strip loop after each full run?

---

**Status: report written. WAITING FOR INSTRUCTIONS.**
