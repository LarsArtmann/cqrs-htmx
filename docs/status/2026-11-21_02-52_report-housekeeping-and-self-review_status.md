# Status-report housekeeping pass + self-review (billing lane)

**Report:** 2026-11-21 02:52 CET · lane: the housekeeping pass behind the 02:26 billing M01+M02 status report (TAIL bump, ledger, corpus re-true, gates, commit `139c8e3e`, push) — plus the owner-ordered self-review of that pass.
**Scope of verification:** every claim below was executed THIS pass with a command — `git status`/`git log`/`git branch -r --contains` (index clean, HEAD on origin before starting), `od -c` (TAIL_BASELINE byte format), `rg -c` (ledger anchor uniqueness), the two gate scripts' enumeration modes read (`find`-based working-tree counting for corpus; README-excluded non-recursive count for tail), the corpus `--update-baseline`/`--ratchet` output, five repo gates, the BuildFlow pre-commit output (107 s, staged-set printed in-invocation), `git show --stat HEAD` post-commit, the pre-push CI-parity output, and `git diff go.sum` at 02:52. **NOT verified:** who/what touched `go.sum` at 02:44 (uncommitted drift, gone by 02:52; three daemon heuristic commits `9470a606`/`310551a8`/`50fc81fb` landed after my push — almost certainly the sibling lane's tidy; attribution beyond that is unknowable from git), the sibling lanes' internals, and anything from the 02:26 report's not-verified list (carried, not re-researched per owner scope).

---

## a) FULLY DONE

| # | Item | Receipt |
| --- | --- | --- |
| 1 | Tree state re-verified BEFORE acting: index clean, HEAD (`add7d694`) already on origin, TAIL_BASELINE 16, CORPUS_BASELINE 64107, **17 live reports on disk** (16 committed + my untracked one) — which corrected the handoff draft's "eighteenth live" to **seventeenth** before it landed in any file | `git status/log/diff --cached`, `ls docs/status/*.md` |
| 2 | Gate counting modes verified before measuring: corpus gate counts the working tree via `find` (untracked included → measure-once order safe); tail gate counts non-recursive `docs/status/*.md` excluding README (17 reports ⇒ baseline 17) | script source read (lines 51/59) |
| 3 | `TAIL_BASELINE` 16 → 17, byte format preserved exactly (`od`: `17`, 2 bytes, no trailing newline) | `od -c` before/after |
| 4 | Ledger sentence appended to `docs/status/README.md` after a uniqueness-checked anchor (`rg -c` = 1), naming the prior sentence's unnamed ~+120 sibling share per house style | `rg -c 'an ADR pair is not a status report'` = 1 |
| 5 | `CORPUS_BASELINE` re-trued 64107 → **64231** = +124, EXACTLY the ledger's "~+124" (report 123 lines + ledger line 1) — measure-once held | `--update-baseline` output: "64231 lines across 438 file(s)" |
| 6 | Full docs gate battery green: links, todo-hygiene (87 rows / 0 dead), conflict-markers (2039 files), freshness, tail-budget `--ratchet` @ 17 | five rc=0 outputs |
| 7 | Commit `139c8e3e` via the 02:26 §e1 remedy — ONE invocation: `git add` (my 4 files only) + staged-set print + commit. Staged-set showed exactly 4 files; post-commit `git show --stat` verified 4 files, 126+/2−, **zero sibling sweep-in** — the §d1 race class did not recur | staged-set + POST-COMMIT-STAT output |
| 8 | Push green: pre-push CI-parity (release-train strict: 1063 requires, 0 unpublished / 0 lag; version-drift strict; 1061 requires resolve to published tags) → `add7d694..139c8e3e`, HEAD contained in `origin/master` | pre-push hook output + `git branch -r --contains` |
| 9 | Stray ` M go.sum` observed post-push — named, left untouched per gotcha 47; by 02:52 the tree is fully clean (drift committed/vanished via the daemon's heuristic commits) | `git status` 02:44 vs 02:52 |
| 10 | The 02:26 report presented to the owner with the three §g questions verbatim; the owner then ordered this second report | this file |

## b) PARTIALLY DONE

1. **The race fix is a shrink, not a kill.** `add`→`commit` now share one shell invocation (window ~milliseconds), and the post-commit stat is a detector — but a sibling staging inside that window is still theoretically possible, and detection ≠ prevention. Fully closing it needs a commit-time guard (pre-commit hook comparing the staged set before/after its own run). Not built.
2. **Handoff-draft trust.** Two stale draft numbers were caught ("eighteenth live", "+~155" vs measured +124) — but only because I re-derived them manually. The draft's structure was reused wholesale; nothing mechanizes the re-derivation.
3. **`go.sum` blip: half-diagnosed by nature.** Observed modified at 02:44, clean at 02:52; the diff I ran at 02:52 found nothing to diff (too late), and WHO is unknowable from git. Best evidence: three daemon heuristic commits landed post-push, consistent with a sibling tidy. Named, not chased.
4. **Commit-message footer divergence.** The harness mandates the Crush attribution heredoc; repo precedent (`9e0b16b7`) carries none. I followed the harness — the divergence from house style is documented, unresolved.
5. **The 02:26 §g questions: STILL OPEN.** Presented verbatim; the owner has not answered them yet (disclosure, ssh-key-monitor's paid shape, the K1 bar).

## c) NOT STARTED

1. **The docs-health archive round** — now **18 live reports vs budget 3** once this report lands. My own 02:26 ledger warned "the archive round MUST land BEFORE any next report"; the owner's order overrode it, and the debt is mine to name: the cadence gate is red and getting redder.
2. **All billing code M03–M26.** M03 (`billing-model` scaffold) remains the next executable; nothing in this pass touched it.
3. **docs-gates `audit-all` drain** vs its pin (pre-existing backlog, red in the hook output again).
4. **Housekeeping mechanization** — the bump-tail + ledger + corpus + gates sequence has now been executed correctly twice and mis-drafted once (stale handoff numbers). No tool exists yet.
5. ssh-key-monitor catalog rows (gated on owner q2) and Issue Shield's founder real-card smoke (their lane).

## d) TOTALLY FUCKED UP

1. **I wrote report #2 against my own archive-first warning without flagging it first.** The ledger I wrote an hour ago says the archive round MUST land before the next report; the owner's order wins, but the professional move was a one-line heads-up ("this deepens the archive debt — archive first?") before executing. Tail debt is now 18 vs budget 3 and I caused the last two increments.
2. **A `read-before-edit` miss:** the first `edit` on `TAIL_BASELINE` was rejected because I'd read the file via bash (`cat`/`od`), not the View tool the rule names. One wasted round trip, zero damage, but the rule is explicit and I knew it.
3. **Nothing else landed red:** no gate failed on this pass's changes, the commit is clean (4 files exactly), the push is green, no sweep-in occurred. The self-review found no data loss, no drift, no broken contract this pass — the failures above are process, not product.

## e) WHAT WE SHOULD IMPROVE

1. **Archive BEFORE the next report, unprompted.** `nix run .#archive-status -- <spec.tsv>` exists precisely for this; the settled 2026-11-15 → 11-21 set is archive-eligible now.
2. **Mechanize the housekeeping sequence** into `scripts/tools/` (bump-tail + ledger + corpus + gates, with the count re-derived from the tree, never from prose) — kills both the stale-draft class and the manual-step drift class in one move.
3. **Never lift numbers from a handoff draft.** Both stale numbers were draft artifacts; the tree never lied. Re-derive every count at write time.
4. **Read-before-edit means View-the-tool** — comply with the letter, not the spirit.
5. **Settle the commit-footer convention once** (harness attribution vs house style) and record the verdict in AGENTS.md so future commits don't re-litigate it.
6. **Characterize stray drift immediately:** when `go.sum`-class noise appears mid-pass, snapshot `git diff <file>` at detection time (the 02:52 diff was six minutes too late), even if authorship stays unknowable.

## f) Up to 50 things we should get done next

**A) This pass's own hygiene (my lane):**

1. Docs-health **archive round**: move the settled 2026-11-15→11-21 reports to `docs/status/archived/` via `nix run .#archive-status` (18 live vs budget 3; my ledger promised it twice)
2. `TAIL_BASELINE` re-true after the archive pass (ratchet must land at ≤ 3-class count)
3. `go.sum`-blip guard: if stray drift recurs, snapshot `git diff` at detection time (02:52's diff was too late)
4. Root + `examples/setup-demo` `go mod tidy` (BuildFlow preflight warning, 02:44 hook output)
5. `buildflow format` after the formatter-fingerprint drift (nix 2.34.9→2.35.6, templ-fmt 1123→1118) so cached verdicts re-verify
6. BuildFlow binary rebuild + reinstall (built at `cb5471e`, repo HEAD `ceb25ca` — advisory)
7. BuildFlow state DB VACUUM (1.07 GB advisory)
8. Extract the housekeeping sequence into `scripts/tools/status-report-housekeeping.sh` + fixture self-test (§e2)
9. Decide + record the commit-footer convention (§e5)
10. Commit-time staged-snapshot guard in the pre-commit hook (§b1's full closure)

**B) Billing lane — milestone-level (granular 39-item list lives in the 02:26 report §f 1–39; owned by the v3 plan, zero duplication here):**

11. M03 `billing-model` scaffold (go.mod + go.work + README + lint configs + coverage row + vocabulary pre-check)
12. M04 domain types: `ProcessorRef`/`BillingPeriod`/seen-refs, branded IDs, `PlanKey`/`PriceRef` + **named limit set** (D18), event payloads ×9, commands ×7, errorfamily rejections
13. M05 `PlanCatalog` (Issue Shield's three live plans as defaults) + `Entitlement` + plan→roles + tax-fact fields (D16)
14. M06 decider (all command paths, unknown/stale-ref rejection, named-limit seat clamping, checkout session-match)
15. M07 fold transitions + monotonic guards + seen-refs compaction invariants
16. M08 `systemscenario` suite under `-race` (happy/duplicate/out-of-order/unknown-refs/grace/seats/replay-idempotence)
17. M09 `billing/stripe` scaffold + `ProcessorGateway` + fake
18. M10 webhook ACL (signature verify, translation table, `ungated` posture per ADR 0063)
19. M11 idempotent dispatch (durable dedup, 30d TTL, `ErrDuplicate`→2xx)
20. M12 adversarial webhook suite incl. the CBOR/JSON decode switch (gotcha 55)
21. M13 `systemadapter.WithBilling` + M14 entitlement read model (no gateway on read paths — grep proof)
22. M15 guide `docs/guides/billing.md` + event-catalog entries
23. M16 `examples/billing-demo`; M17–M21 metrics/observability/adminui panel/OpenAPI goldens
24. M22 release train (`wave-tag` both modules), M23 vocabulary gate v2, M24 docs inventory, M25 e2e, M26 park-or-continue vs K2–K4

**C) Tooling advisories noticed in THIS pass's hook output (pre-existing, other lanes' or tooling):**

25. docs-gates battery reds drain: annotate / audit-all / cadence / freshness (reported again at 02:44)
26. `file-size-check`: 99 findings over the 350-line budget (advisory)
27. `module-inventory`: 28 dependabot-gomod-coverage infos
28. pytest "no tests ran" findings (5)
29. vulture: unused `STEP_USES_SECRET_RE` in `scripts/checks/check-pr-secrets.py`
30. `govalid-generate` timing regressions (+81k% aggregate) — review `buildflow timings --regressions`; re-pin baselines only on an idle machine
31. `govulncheck` timing regression (advisory)
32. `vendorHash.nix` stale-FOD warning — gotcha 43 documents this as a non-fix; leave unless a real nix build fails

**Routing note (HARVEST):** A-items are this lane's, executable on instruction (1–2 are the loud ones); B-items are owned by the v3 plan's task tables; C-items are other lanes'/tooling advisories recorded by their existing gates. Nothing new harvested into TODO_LIST this pass.

## g) Questions I cannot figure out myself

1. **Disclosure** *(carried, STILL OPEN from 02:26)*: Issue Shield's live Stripe price IDs and prices are now quoted in this (public) repo's decision record as evidence. Price IDs are non-secret by design (they ride client checkout URLs), but they are your product's commercial data. Keep as-is, or redact to plan keys + amounts only?
2. **ssh-key-monitor's paid shape** *(carried, STILL OPEN)*: is my "hosted monitoring tier" inference the actual plan, or is monetization something else (license, managed scans for specific orgs, support)? This pins its future catalog rows and whether D18's `monitored_hosts` limit name survives.
3. **The K1 bar going forward** *(carried, STILL OPEN)*: does "payment infrastructure live, zero revenue yet" satisfy the demand gate's spirit for you, or should the lane's standing receipt require the first real charge (Issue Shield's founder card smoke) before M02+ milestones count as demand-backed?

*(Fresh this pass, not a question — a default: the 02:44 `go.sum` blip's authorship is unknowable from git; I left it untouched and will snapshot diffs at detection time if it recurs. Say the word if you want it chased.)*

*Then wait for instructions.*
