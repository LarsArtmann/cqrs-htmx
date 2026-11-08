# Status — spine completion: heal, M15/M16/M20/M23/M24/M25/M02/M08 executed, M17 honestly refused (2026-11-08 12:24 CET)

## §a Context and summary

Resumed under the owner's "GET SHIT DONE — the whole TODO list" directive with the 06-35 plan's runnable-now spine already deep in flight (M00/M01/M03/M13/M14 + the 08-32 report landed earlier in this lineage, with CI red on three pushes). This session: healed every code-level CI red, completed the interrupted report ritual, then executed the remaining runnable milestones M15, M16, M20, M24, M25, M02, M08, verified M18/M23 as already landed, refused M17 on its own gate, and found M19 gate-closed. **Master is CI-green** (run 42471360496 on `de62722c`; the only reds in between were GitHub 504 infra flakes on the tool-download endpoints, cleared by re-run). One unpushed daemon commit (`9ccb18ea`, sibling lane work) sits on top.

## §b Done — fully, with receipts

1. **Report ritual completed** (the mid-drain step from the handoff): TODO row 46 items 5/6 struck with M14-F61/F62 receipts, then the matcher semantics read from source (`~/projects/docs-gates/internal/docsfreshness/`) after a blind strike made the count *rise* (269→270): a harvest needs ≥2 item tokens in living docs AND ≥1 distinctive (DF<8). Four report-side annotations (10-32 items 15/23/34, 11-00 item 35) + the `lint-sweep.sh` literal restored in the TODO receipt → battery 5/5 PASS at 265.
2. **CI heal batch** (`316932e2` + `fd86df7e`): golines wraps ×4 (`authz_contract_test.go:432`, `huma/validation_code_drift_test.go:33`, `dashboardui/auth_stream_preset_test.go:28`, `systemadapter/presets_test.go:84` — applied by the real pinned golangci-lint 2.15.0, not hand-approximated), errname rename `errorString`→`driftError`, `strings.Split`→`SplitSeq` ×2, root-size baseline 55 files/8881 LOC, systemadapter dep budget 18→21 with receipts (the real over-budget module was **systemadapter**, not huma as the handoff claimed — huma was 6/7; the M03 probe's test-file imports made command/query/snapshot direct requires).
3. **Env-leak gate fix**: the M01 version-drift self-test branched on `${CI}` without stripping it — on GitHub runners every hermetic case covered the wrong branch (`check-selftest-env-leaks` caught it on first runner contact). Both helpers now `env -u CI`; the downgrade case passes `CI=true` explicitly; 16/16 under `CI=true`.
4. **M14-F63 sibling-clone cache**: first shipped as a composite action — rejected by the fleet cue-vet policy (every `uses:` must be SHA-pinned; a local action cannot carry a pin) **before any push landed**. Re-shipped as six inline step triplets: rev probe → rev-keyed `actions/cache@55cc8345… # v6.1.0` (exact-key restore, so a stale upstream master can never feed the projectionadapter replace) → clone gated on cache miss.
5. **Canonical templ re-pins ×4**: the ambient v0.3.1114 generator re-stamped adminui's seven `_templ.go` files four times through the day (concurrent sessions + daemon snapshots). Each fight re-pinned via `nix run .#gen` (module-dir, bare FileName) with the pinned binary on PATH for the staged-templ gate; one root-mode `templ generate` accident produced *different code* in `examples/basic/demo_templ.go` and was restored (HEAD was canonical).
6. **M20 docs-health**: 4 settled 11-06 reports archived (09-38, 10-32, 10-54, 11-00 → 2026-11-07 successors; m2m 21-28 stays live), `TAIL_BASELINE` 15→11, corpus 57,937→57,957 with the ledger sentence; erraudit promotion window trued to restart-on-growth semantics (opened 2026-11-07 @ TOTAL=332, closes 2026-12-07); 4 fully-resolved TODO rows retired with a CHANGELOG Housekeeping entry (huma tag, cqrs-lint drift, gotcha write-ups, tooling micro-batch); TODO header trued past the stale v4.17.0 line; audit-all drained to 260 (margin 8) via six verified-done item strikes (train executed; check-modules/lint/test batteries; adminui routeSpec single-source; setup `RoutePostures`).
7. **M15 adminui CHANGELOG reconstruction**: tag↔section set-difference found 5 uncovered tags — one (`v4.11.0`) was a malformed single-`#` heading hiding a full section (fixed + repositioned), `v4.12.0` written from history (family train + minified CSS bundles), the `v4.12.1–4.12.3` dep-train riders and the pre-v4.8 era (`v4.3.0/v4.4.0/v4.6.1`) given honest grouped entries pointing at root's CHANGELOG; coverage check now residue-free outside groups (`1d2c7513`).
8. **M16 hermetic suites**: samber-do-demo, loginpage, auditlog, e2e/server (no tests), integration_test — all rc=0 under `GOWORK=off GOEXPERIMENT=jsonv2` (the "v0.12.3-swept modules" resolved to samber-do-auditlog v0.12.3 consumers).
9. **M24 gate-ticket batch** (`e1dde450`): wave-tag reordered **sweep-before-strict-train** — the old order ran `--strict-lag 0` before the sweep that fixes exactly the lag the fresh tag creates, so the first wave of a train refused on its own pending fix (self-test updated, 22/22); `check-release-train --consumers` leg added (hermetic `GOWORK=off` build of every go.work member, exit 5, `TRAIN_CONSUMERS_CMD` stub-able, self-test 5/5); the wave-row existence check verified already-present (documented in WT8, duplicate avoided); check-codegen grew the templ `fmt`→codegen chain leg (formatter drift on committed `.templ` now fails the gate; live run PASSED).
10. **M23 verified done**: the branch guard + /tmp sibling-sweep warnings landed in round-45 M17; `test-preflight-tree-check.sh` green with both m17 legs — the plan row was stale, no code needed.
11. **M18 verified done**: the e2e lane already shipped `loginpage-webauthn.spec.ts`, `loginpage-golden.spec.ts` (+ committed golden snapshot), and `loginpage-posture.spec.ts` (plan M11); CI's e2e job is green on the current tree — the three F73/F74/F75 tickets are covered.
12. **M25 watches sweep**: kv v4.5 still unpublished; go-cqrs-lite#117 + httputil#48 still open with zero responses; **BuildFlow#1 closed upstream** (tool-command test suite landed); templ-components still v1.27.0 (popover anchoring + SidebarNav drawer asks unchanged); /mnt/buildcache 73% (57G free); README↔FEATURES quarterly audit not due (2027-01-17). TODO watch sentence trued.
13. **M02 branching-flow re-pin**: the +2 new findings adjudicated — `systemadapter/presets.go:77` PHANTOM-PLUMBING **fixed at source** (`RecommendedSplitSQLiteDeployment` now takes `SplitSQLiteDSNs{Events, Projections}` — named fields make the DSN transposition unrepresentable; probe call site moved, suite green, the finding dissolves), `e2e/server/bridges.go:152` re-anchored under the frozen intended-wiring verdict (same finding as :105, adjudicated 11-07). Baseline regenerated, triage row written, gate green at +0.
14. **M08 webphone pin sweep** (sibling repo): 4 pins to current trains (root v4.16.0→v4.18.1, usermgmt v4.19.0→v4.20.0, loginpage v4.16.0→v4.17.0, identity-model v4.15.0→v4.17.0; webauthn already current), hermetic build + module suite green on the 1.27.3 floor, committed scoped to go.mod/go.sum (`aa8c482e` in webphone).
15. **Status-row gate fix**: the F83 strike left a PARTIAL table row (struck action cell beside clean number/proof cells) — the exact class the checks job exists to catch; all three cells struck per the house pattern, gate rc=0 (`92fe0b36`).

## §c Partially done

1. **CI green** — green as of `de62722c` (42471360496 success) after the 504 re-run; one daemon commit (`9ccb18ea`) sits unpushed on top, so the NEXT push re-enters the lottery until the sibling lane settles.
2. **M14 "run green ×2"** (F74): the loginpage e2e specs are green in CI (multiple pushes today exercised the e2e job ≥2× green when it wasn't 504-flaked), but I did not execute a deliberate local ×2 run of the loginpage specs myself.
3. **M20 margin**: audit-all at 260 vs pin 268 is green but the volume-vs-pin race continues — every new report adds §f items; the margin should be drained further or the pin formally revisited with the owner.
4. **TODO row 36** (blanket-directive tail): still carries open legs (coverage-gate re-run on the post-tag tree, C28 consumer dispatch-gate waves, v4.19.0 wave planning) — not touched this session.

## §d Not started (gated or tail work, honestly open)

1. **M19 a11y first-response** — the sibling has pinned no axe findings surface; the gate ("sibling pins findings") never opened. Nothing to triage.
2. **M17 quiet-window batch** — **refused, not forgotten**: load 21.6 on 32 cores at check time, far above the verified-quiet bar the twice-quiet rule demands; refusal receipt recorded in the TODO bench row.
3. **M04/M05** (M2M W1/W2 migration waves), **M06/M07** (browser-history postures) — owner-gated on the decision sheet's D1–D4 tick.
4. **M09/M10/M12** (DataStar lockstep waves) — owner-gated on the go-datastar v0.8.0 GO/NO-GO.
5. **M21** (cqrs-lint fleet swap chain, D4/D6), **M22** (CI hardening: gosec pin, architecture-gate secrets posture) — tail/owner ticks.
6. **M26/M27** (upstream asks batch; crush-config lessons) — tail, not reached.

## §e Totally fucked up (owned)

1. **Three CI-red pushes landed before the heal** (the 08-32 report's §d, extended): I built the lint-parity gate mid-session and then *didn't run it* before pushing M13 — the exact gotcha-50 pattern. This session's pushes added one more of the same family: the F83 strike broke `check-status-rows` and I pushed before running that gate locally (caught by CI, fixed in `92fe0b36`). The lesson is now mechanical: **the pre-push set is not "the gates I remember" — it is check-modules (or its failed-job legs) run locally, every time.**
2. **Wrong inherited diagnosis, nearly shipped**: the handoff said "huma never tidied; run `go mod tidy`" for the dependency-budget failure. Tidy was a no-op and huma was *within* budget (6/7) — the real over-budget module was systemadapter. I verified against CI logs before fixing; had I executed the handoff's remedy blindly, the red would have persisted through another cycle.
3. **The composite-action detour**: I wrote the F63 cache as a local composite action without checking the fleet cue-vet SHA-pin policy first. The BuildFlow run on an attempted commit caught it (and the daemon's race made the first commit attempt land as a confusing no-op). A 30-second grep of the policy repo would have saved a full rework cycle.
4. **Root-mode templ generate polluted `examples/basic`**: running `templ generate` from the repo root instead of the canonical module-dir form produced genuinely different generated code outside the three gated modules — caught by diff inspection and restored, but the accident was avoidable by using `nix run .#gen` from the start.
5. **The audit-all blind strike**: striking row-46 items 5/6 without first reading the matcher made the count go UP (the receipt text dropped the distinctive tokens). Wasted a battery cycle; the fix (reading `docsfreshness/check.go` + `tokens.go` first) should have been step one.

## §f What we should improve / do next (ranked, ≤50)

1. **Systemic templ-stamp fix**: the ambient-vs-pinned generator war cost four re-pin cycles today. Add a pre-commit (or BuildFlow) guard that fails any staged `_templ.go` whose stamp isn't the pinned version, so ambient regens can never be *committed*, only caught.
2. **Pre-push parity discipline**: extend the pre-push hook (currently release-train + version-drift only, per AGENTS' own gap note) to include the checks-job row gate + status-rows + docs battery — today's row-gate red would have been caught locally.
3. **Retire the 08-32/12-24 report pair at the next archive pass** (tail is 12 after this report; the settle window opens tomorrow for the 11-06/11-07 remainder).
4. **Drain audit-all below 250** while the matcher semantics are fresh (the archived 21-31/15-41 reports still carry 7+6 open items, several verifiable-done).
5. **Bench re-pin** at the next verified-quiet window (M17 refusal receipt names the bar).
6. **`buildflow format`** in the same quiet window (M17's other half).
7. **Coverage-gate re-run** on the post-tag tree (TODO row 36's open leg; last full run predates the QUERY waves).
8. **C28 dispatch-gate consumer waves**: browser-history + InboxClean have real command surfaces (github-local-sync vacuous by design).
9. **v4.19.0 wave planning** (QUERY W3/W5 remainder) — needs the owner's DataStar tick for the full shape.
10. **M21 cqrs-lint fleet swap** (D4/D6): tag the nested cmd/cqrs-lint module, bump the fleet pin, re-add B024 suppression, wire the CI gate.
11. **M22 CI hardening**: D17 gosec pin; architecture-gate secret-or-documented-skip; repo-protection wiring per decision sheet.
12. **M26 upstream batch**: erraudit asks (config file, `--modules` hint, generic_return exemption), E005 proposal, upstream drafts under docs/proposals/upstream/.
13. **Templ-components popover upstream filing** (the vendored-CSS finding is verified on Chromium 156; verify-before-filing gates apply).
14. **SidebarNav drawer ask** (v1.28 watch) — re-check templ-components tags next sweep.
15. **go-cqrs-lite kv#113 re-check** (the R9 adapter deletes when the interface-level constructor ships).
16. **Deduplicate the six inlined cache triplets** if the fleet policy ever allows local composites (or via a reusable workflow with pinned SHAs).
17. **Add `--consumers` to the release playbook** (§1) so trains run it once per wave — the leg exists but nothing invokes it by default.
18. **Wave-tag carrier cascade**: exercise the reordered rhythm end-to-end on the next real tag (the reorder is self-tested but not yet battle-run).
19. **True the AGENTS wave-tag row** — it still describes the OLD leg order (train before sweep); the reorder changed the documented rhythm.
20. **`e2e/probe-ds.mjs` hygiene**: the sibling's probe script rides daemon commits; when their lane settles, decide keep-or-delete.
21. **M18 F74 completion**: deliberate local ×2 run of the loginpage specs (not just CI-observed greens).
22. **The unarchived m2m 21-28 report**: still live as the owner-gated thread's surface; archive it when M2M decisions land.
23. **Audit-all pin policy**: margin 8 today; decide the standing rule (drain-after-every-report vs owner re-pin) — plan §f17 asks exactly this.
24. **Erraudit window watch**: closes 2026-12-07; promote the strict gate to blocking only if it closes stable-or-shrinking.
25. **check-selftest-env-leaks coverage debt**: 3 CI-branching checkers still have no self-test (dependabot-alerts, replace-directives, require-tags).
26. **Dependabot coverage**: module-inventory reports 26 go.work members without dependabot gomod entries (generation capped at 20; the cap is the blocker — worth an upstream ask or a generator change).
27. **File-size-check debt**: 96 findings (max 350 lines) — several test files (oauth2/provider_test 1252) would benefit from the split pass the S-rounds started.
28. **Vulture finding**: `check-pr-secrets.py:29` unused `STEP_USES_SECRET_RE` — one-line cleanup.
29. **`tests/dashboard-live.spec.ts` type errors** (2 × `HTMLElement.value` — sibling lane, coordinate).
30. **pytest-test findings** (5 warnings, 0 collected) — the python lane is vestigial here; document or remove.
31. **Loginpage README ↔ FEATURES quarterly audit** — not due until 2027-01-17, but the watch item is now trued with the date.
32. **The `scripts v1.0.0` wave-row confusion**: the plan's M23 row pointed at work already done — audit the 06-35 plan's other tickets against the tree before executing the tail (two of today's milestones were stale rows; a 15-minute Step-0 pass on the remaining tail would find more).
33. **CHECK-TEMPL-CODEGEN-STAGED vs ambient PATH**: the gate regenerates with whatever `templ` is on PATH — inside `nix develop` that's pinned, but bare-shell commits (daemon) get the ambient binary and can red spuriously. Consider resolving the pinned binary inside the gate itself.
34. **`docs-gates annotate` rule-F lint in CI**: the battery runs advisory in CI; promote to the checks job when the owner tick lands.
35. **Wave-2 browser-e2e bugs #6/#7** (06-14 report): the firefox promotion/demotion decision is still pending — it was open before this session and nothing here touched it.
36. **`usermgmt/snapshot_test.go`** (475 lines) + `verification_totp_http_test.go` (483): split candidates from the same file-size batch.
37. **`dashboardui/core/events_test.go`** (366 lines): same batch.
38. **kv #113**: if upstream ships the interface-level `kv.NewCache`, delete the R9 adapter (the delete-when trigger is documented in usermgmt/view_cache.go).
39. **systemadapter upstream ask**: scream-report symmetry for system v4.17.0's implicit-volatile projections fallback (the W1.2 starvation root cause; queued, never filed).
40. **`#117`/`#48` maintainer pings**: 2+ weeks silent; a polite status comment (github-voice) may surface whether they're alive.
41. **DataStar Tier-4 demand**: 1 open issue upstream — re-verify it's still the tracked demand and not noise.
42. **SplitSQLiteDSNs doc example**: verify the presets README/AGENTS examples still show the OLD two-string signature anywhere (grep found the doc comment fixed; external docs unchecked).
43. **CHANGELOG cut discipline**: [Unreleased] carries the M13/M20/M02 entries — sweep into a v4.19.0 section at the next train.
44. **README quickstart gate**: run `nix run .#check-readme-quickstart` once after the ADR-0062 repoint wave settles (not run this session).
45. **Erraudit strict TOTAL re-measure** after the day's code growth (last receipt 332; the presets/probe edits added code).
46. **The `--skip-sweep` wave-tag path** now skips the sweep but STILL runs the strict train — correct (post-tag world has no self-lag for a leaf), but a comment on the leg would prevent misreading.
47. **Canon-generator guard for examples/**: `examples/basic` has no codegen gate leg (only adminui/loginpage/dashboardui) — today's root-mode accident landed exactly there; either gate it or document the exclusion.
48. **Adminui golden regeneration ritual**: v1.21.0's nonce-rule precedent says family bumps drift page goldens — the next templ-components bump must re-diff loginpage-render goldens line by line.
49. **`normalize-status-rows`** exists as a tool — point the docs-health runbook at it for future strike-format mistakes (I hand-fixed today; the tool may have done it in one step).
50. **The daemon race protocol** worked but cost ~6 aborted commits: consider `git commit --no-verify` ONLY for docs-only batches (gotcha-50 carve-out already exists) to shrink the race window on report batches.

## §g Questions for the owner (cannot self-answer)

1. **M2M D1–D4 + migration defaults** (decision sheet §A): the tick unlocks M04/M05 — still the single highest-leverage decision on the sheet. GO, and on which migration default?
2. **go-datastar v0.8.0 lockstep cut** (decision sheet §B): GO/NO-GO unlocks M09/M10/M12 and the v4.19.0 wave shape. The upstream docserver-era constraint (catalog needs 1070-era, family needs 1100-era) still binds — does the cut wait for upstream, or do we cut v4.19.0 without the DataStar lane (plan K2)?
3. **M17 scheduling**: is there a daily window this machine is reliably idle (for the bench re-pin + buildflow format), or should the twice-quiet rule be relaxed for the *format* half (formatter runs are load-insensitive; only the bench pin truly needs quiet)?

## §h Verification receipts

- `nix run github:Artmann-Technologies/docs-gates -- run .` → 5/5 PASS (final tree; audit-all 260 vs pin 268)
- `bash scripts/selftests/test-wave-tag.sh` → 22/22; `test-check-release-train.sh` → 5/5; `test-check-version-drift.sh` → 16/16 under `CI=true`; `test-preflight-tree-check.sh` → green incl. m17 legs
- `nix run .#check-lint-sweep` → 16 modules clean; `check-codegen` (now with the fmt chain leg) → PASSED; `check-branching-flow` → OK +0; `check-status-rows` → rc 0; `check-todo-hygiene` → clean (76 rows); corpus 57,958 ratchet green; tail 11→12 with this report
- CI: run 42471360496 SUCCESS on `de62722c`; the four 504-flaked lint jobs cleared on re-run; pre-push hooks green on every push today
- Webphone (sibling repo): hermetic build+test rc=0 at `aa8c482e`
