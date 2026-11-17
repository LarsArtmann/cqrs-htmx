# Status — M01 aggregate closure: branching-flow re-pin, final 104/104, docs-gates green

> **ANNOTATED 2026-11-16** — ~~this report~~ **SUPERSEDED** by [`2026-11-15_14-37_comprehensive-status-update-plan-tail-execution_status.md`](2026-11-15_14-37_comprehensive-status-update-plan-tail-execution_status.md), which M01 closure + branching-flow re-pin + docs-gates green all landed and extended by the 11-16 re-pin (212). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-16 (round-23 (2026-11-16)).**

**Scope of this report.** VERIFIED this session (2026-11-14 ~16:25–18:30, the continuation session after the 16-21 exit report landed): the final `check-modules` aggregate on the repaired tree (**104/104 GREEN, rc=0**, `/var/tmp/check-modules-final3.log`), the `coverage-gate` verdict (**PASSED, 16/16 modules**, `/var/tmp/coverage-gate-final.log`), the `check-branching-flow` re-pin (**+0 added**, run post-commit), the docs-gates battery (**freshness/annotate/routing/cadence PASS, audit-all the standing owner red**), and the corpus/tail/conflict-marker gates post-true-up (corpus 59,301 = baseline). NOT verified: a full `check-modules` rerun AFTER the final docs true-ups (plan §9/exit-report/TODO_LIST edits — only the specific affected gates were re-run; the 104/104 verdict binds the pre-true-up tree, and the true-ups touch no code), the e2e browser battery (untouched this session), push/CI (75+ commits ahead of origin, unpushed), and the audit-all itemization (the 267→277 drift is attributed by class, not itemized).

The 16-21 exit report carried one open technical question: the final aggregate had failed **2 of 104 checks** — corpus (repaired by that report's commit 82876e9f) and ONE check whose name was unknown (its first log was lost to /tmp cleanup). This session resolved the second, closed M01, and re-greened the docs gates.

## a) FULLY DONE

| Item | Receipt | Command/artifact |
| --- | --- | --- |
| **Second failing check identified** | `branching-flow` — the rerun log `/var/tmp/check-modules-final2.log` (recovered from /var/tmp; 103/104 PASS, corpus already healed) named it: +6 added, −6 removed, ~167 modified, =30 unchanged | read of `/var/tmp/check-modules-final2.log` |
| **The +6 decomposed to zero new defects** | SARIF-level diff of current findings vs the committed baseline (ad-hoc python over the tool's own output): per-rule counts IDENTICAL 211 = 211, same tool 0.6.7, exactly ONE message-text change — `usersListData`↔`tenantsListData` (:113) flipped to `usersListData`↔`auditData` (:168), the SAME 4 shared fields (Total, Search, BasePath, embedded listPage) with the analyzer's reported counterpart flipped because M09–M12 grew the file. The other five gate-adds are fix-snippet re-anchors (identical messages; the differ counts the changed `afterCode` snippets as add+remove pairs — this pairing rule was reverse-engineered, see §e) | python diff of `docs/analysis/branching-flow-baseline.sarif` vs regenerated SARIF |
| **branching-flow re-pinned + adjudicated** | baseline regenerated in place (211 → 211 net-zero, minified, committed ffb607d2 by the daemon); amendment row appended to `docs/analysis/triage-decisions.md` naming all six pairs (all already adjudicated 2026-10-07 T15 / 2026-11-14 M02-M03); gate re-run **+0 added — GREEN** | `nix run .#check-branching-flow` → rc=0 |
| **Final aggregate: 104/104 GREEN** | `nix run .#check-modules` rc=0 on the repaired tree — the M01 receipt the plan's partial row (§9 line 433) waited on ("recorded when done") | `/var/tmp/check-modules-final3.log` |
| **coverage-gate: PASSED 16/16** | root 90.6/90, identity-model 75.0/70, usermgmt 83.0/74, totp 88.2/80, webauthn 88.4/80, oauth2 88.4/80, **adminui 69.7/66 (UP from 68.2 pre-wave with the M09–M12 files)**, loginpage 81.1/79, dashboardui 73.6/60 + core 89.2/80, datastar 100/90, huma 95.6/90, setup 93.7/80, systemadapter 91.4/70, health 100/90, auditlog 100/90 | `nix run .#coverage-gate` rc=0 |
| **Plan §9 trued** | final M01 row appended (aggregate 104/104 + branching-flow decomposition + coverage table); the stale predecessor-session tail row `| - | M01 \| not started |` DELETED (0 occurrences verified in HEAD) | `docs/planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md` |
| **Exit report §a/§f resolved inline** | §a1 rewritten NOT-yet-green → RESOLVED (names both checks, the decomposition, the final verdict); §f34 branching-flow → RESOLVED; §f35 coverage → RESOLVED | `docs/status/2026-11-14_16-21_...md` |
| **CURRENT_FOCUS trued** | "waits on … the exit report" → aggregate GREEN (104/104); freshness removed from standing reds | `CURRENT_FOCUS.md` |
| **docs-gates freshness: DRIFT → PASS** | the exit report's 4 unharvested §f items resolved: §f23 (stale-proxy gotcha 62 candidate) + §f19 (axe page sweeps) harvested as a new TODO_LIST P2 sub-row batch; §f26/§f28 (tenant rename UI, sessions admin surface) `fresh:skip`'d as owner-gated panel surfaces of (h)(3)/(h)(4); §f19 additionally needed a reasoned fresh:skip — every ≥4-char token in it is generic across TODO_LIST (DF ≥ 8) and "axe" sits below the matcher's token floor, so a mechanically-provable harvest is impossible (the archived 2026-11-03 report carries the identical precedent) | `nix run github:Artmann-Technologies/docs-gates -- freshness .` → PASS |
| **Corpus ledger + re-true, same change** | triage row +1 → ledger sentence in `docs/status/README.md`, `CORPUS_BASELINE` 59,299 → 59,301, gate green; the second true-up batch (plan row, report resolutions, TODO rows) measured NET-ZERO — every edit was a same-line replacement, no ledger owed | `nix run .#check-docs-corpus-budget` → 59,301 |
| **Docs-gates battery 4/5 PASS** | freshness PASS (newly green), annotate CLEAN (1498 struck, 0 findings), routing CLEAN, cadence ARCHIVE-HEALTHY; audit-all remains RED = the standing owner-gated red (see §b) | `docs-gates run .` matrix |
| **Tree state** | clean in HEAD; session content landed via daemon commits ffb607d2, 12ba3e47, f72c55a1, e4bb1465 (content spot-checked in HEAD per gotcha 51: M01 row present, stale row gone, 3× fresh:skip, gotcha-62 TODO row present) | `git status`, `git show --stat` |

## b) PARTIALLY DONE

1. **docs-gates audit-all — still RED, honestly unchanged.** 277 unharvested open items vs the pinned `audit_max: 268` (was 267 ≤ 268 at the M01-partial run; the +10 is the 16-21 exit report's open §f volume — attributed by class, NOT itemized per item). Policy forbids raising the pin without an owner tick; the drain-or-re-pin decision sits in the TODO_LIST P2 top row. Nothing this session could do owner-independently.
2. **The 104/104 verdict's strictness boundary.** The aggregate ran BEFORE the final docs true-ups; after them only the affected gates re-ran (corpus/tail/conflict-markers/freshness battery). Defensible (docs-only delta, zero code), but a pedantic full-aggregate rerun would be the only way to claim 104/104 on the exact HEAD commit.
3. **Tail budget:** 5 live reports after this file lands (budget 3, `TAIL_BASELINE` 4 → bumped to 5 same-change below, the 16-21 precedent). The next docs-health round owes the ARCHIVE pass (the settled 11-29 + 08-41 predecessors are the candidates the ledger already names).
4. **Session ledger style:** my 16-4x ledger sentence in `docs/status/README.md` used ASCII (`->`, `-`) where the file's established style is unicode (`→`, `—`). Cosmetic drift in a file that future sessions pattern-match against.

## c) NOT STARTED (all owner-gated or out of this session's directive)

1. **Push + CI watch** — 75+ commits ahead of origin, strict train gate 0 lag, aggregate green; `train-preflight` → push → `push-and-watch` awaits authorization (the 16-21 report §h2 asked; still unanswered).
2. **D1–D3 adminui packet ticks** (impersonation panel / ONE usermgmt tag train for the five (h) asks / wave ordering) — recommendations pre-staged, one tick per row converts to GO.
3. **M2M D1–D4 ticks** (separate packet, unchanged) and **usermgmt→metaengine D1–D5** (W1/W2 probes already GO).
4. **Firefox advisory promotion** — the 66 `setDownloadOptions` protocol errors need a browser-stack decision, not test fixes.
5. **The audit-all drain-or-re-pin** (§b1).
6. Every §f item below marked "(rides …)" — panel surfaces gated on the (h) asks.

## d) TOTALLY FUCKED UP (this session; nothing destroyed, all repaired in-session)

1. **The branching-flow red existed for the entire M09–M12 stretch and nobody looked.** The ratchet went un-pinned through four feature waves — the SAME batch-the-ritual-to-session-end shape as the corpus lesson recorded in §d7 of the 16-21 report, repeated one gate over. A per-wave `check-branching-flow` (it is fast) would have made this session's entire triage unnecessary: the pairing flip would have been a same-change re-pin riding M09. Process debt, not code debt — but it cost a reverse-engineering session to close.
2. **The gate script discards the findings it fails on.** `check-branching-flow.sh` runs the tool with stdout to /dev/null and the failure message names only the counts — the actual +6 findings were recoverable ONLY by re-running the tool manually and hand-diffing two SARIF files with an ad-hoc python script. Worse, the tool's own differ counts (+6/−6/~167) do NOT match a naive (rule, file, message) diff (+1/−1): five of the six "added" findings have byte-identical messages and differ only in their `afterCode` fix snippets — a pairing rule I had to reverse-engineer from the output. Two hours of avoidable triage per future incident until the gate echoes its added findings.
3. **Two wasted explicit-commit attempts racing the daemon.** Each failed `git commit` on an already-moved HEAD still ran the full ~40 s BuildFlow pre-commit hook before reporting "nothing to commit". The daemon owns doc-only commits in this repo; checking `git status` before reaching for `git commit` is the cheap fix I skipped twice.
4. **The freshness matcher was fought by guess-and-retry instead of read-first.** I restructured the TODO harvest row twice (combined → split sub-rows) on an untested theory of the matching semantics before actually reading the matcher's output carefully enough to learn the real rules: one-to-one row-to-item assignment, generic-token DF ≥ 8 rejection, and a ≥4-char token floor that makes "axe" unmatchable in principle. The verify-before-building lesson (§e1 of the 16-21 report, learned the hard way with T104) applied to a gate, not a service — and I still reached for iteration before reading.

## e) WHAT WE SHOULD IMPROVE

1. **Make the ratchet re-pin a per-wave step, not a session-end step.** Rule candidate for the runbook: any commit that moves Go code re-runs `check-branching-flow` in its own verification batch (it is seconds-fast), and a red is re-pinned in the SAME commit with its amendment row — exactly the corpus-ledger ritual, extended to the SARIF ratchet.
2. **Echo added findings from the gate.** `check-branching-flow.sh` rc=1 should print the added findings (the tool already classifies them internally; surfacing them is a plumbing change). Until then, the manual recipe (regenerate SARIF, diff by rule+file+message, then by fix-snippet for the residue) belongs in `docs/analysis/README.md` — it took real work to reconstruct and is now only in this session's history.
3. **Record the differ semantics.** "Identical message + changed fix snippet = added+removed pair" is now tribal knowledge from this session; one sentence in the analysis README prevents the next person from distrusting the tool or their own diff.
4. **Doc-only commits belong to the daemon.** House rule: explicit commits for content that must ride a specific message (reports, code); everything else, write files and let the daemon sweep — after verifying the previous sweep landed.
5. **Read a matcher/gate's rules before restructuring artifacts to satisfy it.** The freshness fight cost two TODO_LIST reshapes; one careful read of one DRIFT output would have cost none.
6. **Freshness token floor:** items whose only distinctive noun is <4 chars ("axe") can never harvest mechanically — the `fresh:skip: routed` pattern is the documented workaround, but the docs-gates tool could lower the floor or whitelist short distinctive tokens. Upstream-ask candidate (owner-gated filing).
7. **Ledger sentences should copy the file's unicode style** (`→`, `—`) — pattern-match the previous entry before writing the next one.
8. **Itemize audit-all drift when it moves.** "+10 = report volume" is an attribution, not a proof; a one-line per-item provenance in the ledger sentence would keep the owner-gated decision honest.

## f) Up to 50 things we should get done next

Tiered: **T1 = actionable owner-independent** (start here), **T2 = owner-gated** (tick first, then go), **T3 = roadmap fuel / polish** (not commitments). Items carried from the 16-21 queue keep their original numbers for traceability.

**T1 — owner-independent, high impact**

1. Run one full `check-modules` on the current HEAD so the 104/104 claim binds the exact final commit (the §b2 boundary) — minutes-fast, closes the pedantic gap.
2. Make `check-branching-flow.sh` echo its added findings on rc=1 (§e2) + fixture self-test case.
3. Document the manual SARIF-diff recipe + the differ's fix-snippet pairing rule in `docs/analysis/README.md` (§e3).
4. Add the per-wave ratchet re-pin rule to `docs/runbooks/session-start.md` (§e1) — the same update the 16-21 §f50 already owes (corpus-ledger-with-report pairing; both ritual gaps in one runbook edit).
5. Triage the erraudit +2 drift (330→332; `loginpage resolveOAuth2Buttons` extraction is the likely source) at the next advisory sweep — carried 16-21 §f33.
6. Verify the route-posture manifest golden covers the four import/export rows + switcher (16-21 §f32; CI hermetic leg spot-check post-push).
7. Copy the T132–T134 plan erratum sentence into the adminui CHANGELOG archaeology note (16-21 §f41) — one line, consumer-facing truth.
8. Extend axe sweeps to the remaining adminui pages (audit, dashboard, import form) — harvested this session as TODO P2 sub-row (16-21 §f19).
9. Keyboard-path test for the tenant switcher itself — the library Dropdown ships keyboard nav, pin it once (16-21 §f44).
10. Keyboard-path tests for the remaining forms: import upload, new-user, new-tenant, member add (16-21 §f21).
11. `aria-current` audit across other chip/link toggles the sweep did not cover: loginpage, dashboardui (16-21 §f18).
12. Admin-demo env knob for the 60-filler pagination trap (or a spec helper that always searches) — it bit two specs (16-21 §f46).
13. Add the import page + switcher open-state to the screenshots harness — the two most distinctive new UIs (16-21 §f42).
14. Mobile-viewport admin screenshots (16-21 §f20).
15. Consider axe on the error/404 pages — the sweep skips them (16-21 §f43).
16. Surface "added by" on the user-detail access rows too — M08 data exists, only the roster renders it (16-21 §f45).
17. Candidate gotchas 61–63 (reuseExistingServer stale-server trap; stale-proxy `go version -m`; networkidle-never-settles-on-SSE) — 62 is already harvested to TODO P2; batch all three in one gotchas.md edit with AGENTS.md index rows.
18. The `go.work`-vs-published drift gate idea (16-21 §f40) — assert hermetic demo builds serve the same JS the workspace serves; the durable fix for the §d2 class.
19. Pin the e2e battery's server build against stale-proxy resolution (16-21 §f36 — a `go.work` presence assertion in the bootstrap).
20. The docs-health ARCHIVE pass: settle the 11-29 + 08-41 predecessors (the tail ledger names them; drops live tail 5 → 3 and retires the TAIL_BASELINE bump chain).
21. Update the session-start runbook with the corpus-ledger + baseline-bump + tail-bump report pairing (16-21 §f50; re-derived twice across two sessions).
22. SKILL.md adminui section: add the switcher/import surfaces to the consumer-facing truth (16-21 §f48; `check-skill-claims`/`check-readme-splitbrain` watch).
23. Evaluate cutting an adminui tag train — the CHANGELOG [Unreleased] carries four substantial entries (16-21 §f49; the release playbook once owner ticks land).
24. Itemize the audit-all +10 drift per-item in the next ledger sentence (§e8).
25. Upstream ask: `httputil/server_timing` v1.9.0 tag — the family split that aborts unanchored bump sweeps (16-21 §f38).
26. Upstream ask: a dashboardui family tag carrying the delegated-slider fix — no tag contains it, every hermetic consumer build of the demo serves stale JS (16-21 §f39).
27. Document the anchored-pattern bump recipe (`'larsartmann/httputil$'` + server_timing publication check) as a bump-dep pre-flight step in the runbook (16-21 §f37).
28. Upstream ask (docs-gates): freshness token floor <4-char handling — or bless the `fresh:skip: routed` pattern as the official answer (§e6).
29. Ledger-style sweep: normalize my 16-4x ASCII ledger sentence to the file's unicode style (§b4, one-line cosmetic).

**T2 — owner-gated (tick converts to work)**

30. Push authorization → `train-preflight` → push → `push-and-watch`; first CI verdict on the loginpage/admin-demo lint legs, the e2e chromium job, and the docs gates (16-21 §f5/§f6) — question (g2).
31. D1 tick → impersonation panel surface behind opt-in config (no usermgmt change; the erratum stands) (16-21 §f8).
32. D2 tick → ONE usermgmt tag train: bot listing, batched membership counts, tenant rename, session listing/revocation, audit tenant attribution (16-21 §f9).
33. D3 tick → wave ordering: sessions+impersonation before bots (16-21 §f10).
34. (h)(5) lands → rebuild the T104 per-tenant audit shortcut against real entries (16-21 §f11).
35. Adminui audit-log attribution wave — surface tenant + membership actions in the audit trail (16-21 §f17; rides (h)(5)).
36. Tenant rename UI (rides (h)(3)) (16-21 §f26).
37. Sessions admin surface "sign out everywhere" (rides (h)(4)) (16-21 §f28).
38. Bot management surface (rides (h)(1) + D3 ordering) (16-21 §f27).
39. Impersonation forensics view (rides ADR-0060's journal, post-D1) (16-21 §f29).
40. Batched membership counts adoption (rides (h)(2)) — retires the N+1 in `tenantsIndex` AND `tenantSwitchOptions` (16-21 §f30).
41. M2M D1–D4 ticks (16-21 §f12).
42. usermgmt→metaengine D1–D5: the W1/W2 probes are already GO without further input — scheduling is the only open bit (16-21 §f13).
43. audit-all drain-or-re-pin tick (§b1; TODO P2 top row) — question (g1).
44. Firefox browser-stack decision: fix or accept the 66 `setDownloadOptions` mismatches, then the 3-clean-runs promotion clock (16-21 §f4/§f16).
45. Import/export depth: dry-run mode, per-row error export, batch guidance (16-21 §f25; D2-adjacent).

**T3 — roadmap fuel / design passes**

46. Color-contrast design pass over adminui chrome (accent brand palette + metadata gray ramp; ~115 accepted findings) then shrink the ACCEPTED entry to per-node scoping (16-21 §f14/§f15; question (g3) territory).
47. `tenantSwitchOptions` cost note: `AllTenants()` on every super-admin render — fine at demo scale, batched API at fleet scale (16-21 §f31).
48. M12 T123 member-row renderer dedupe — contract divergence, revisit only if the surfaces converge (16-21 §f5).
49. `networkidle`-on-SSE candidate gotcha 63 write-up if the batch in item 17 does not absorb it (16-21 §f24).
50. The stale-server `reuseExistingServer` candidate gotcha 61 write-up if item 17's batch does not absorb it (16-21 §f22).

## g) Three questions I cannot answer myself

1. **audit-all:** drain the 277 (a docs-health HARVEST round over the archived reports) or re-pin `audit_max` to the measured count? Policy forbids the re-pin without your tick, and the battery stays 4/5 either way until you call it.
2. **Push:** the tree is 75+ commits ahead, strict train gate 0 lag, aggregate 104/104 — is it cleared for `train-preflight` → push → CI watch now, or does it wait for the D1–D3 ticks to ride the same window?
3. **The next wave's shape:** do the D1–D3 adminui ticks land first (unblocking impersonation/sessions/bots/rename), or does the T1 polish tier (items 1–29: gate ergonomics, a11y depth, gotchas 61–63, archive round) run while the packets wait?

## h) Files touched this session

`docs/analysis/branching-flow-baseline.sarif` (re-pinned 211→211), `docs/analysis/triage-decisions.md` (amendment row), `docs/status/README.md` (ledger sentence), `docs/CORPUS_BASELINE` (59,301), `docs/planning/2026-11-14_08-45_...pareto-plan.md` (§9 final M01 row, stale row deleted), `docs/status/2026-11-14_16-21_...md` (§a1/§f34/§f35 resolved, §f19/§f26/§f28 fresh:skip), `TODO_LIST.md` (P2 §f-tails harvest batch), `CURRENT_FOCUS.md` (standing reds trued), this report. Daemon commits: ffb607d2, 12ba3e47, f72c55a1, e4bb1465 + this report's own true-ups.
