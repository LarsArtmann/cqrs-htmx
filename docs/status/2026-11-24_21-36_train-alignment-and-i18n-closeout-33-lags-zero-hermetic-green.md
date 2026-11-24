# Train Alignment (33 → 0 Lags), Hermetic Green, i18n Batch-3 Close-out — Status

**Date:** 2026-11-24 21:36 CET
**Scope of this report:** this GET-SHIT-DONE session's run only (resumed ~20:05 after the 19-25 adminui-wave report; the concurrent identity-merge session finished ~19:55 and a follow-on RELEASE-TRAIN session started mid-session — its commits `619ddb57`/`3ff468d6`/`4cc14c45` are theirs, witnessed not executed here).
**Tree at report time:** master, clean; adminui build + full suite + lint 0 + surface gate green in BOTH workspace and hermetic mode (hermetic green for the first time since the merge wave landed — the train healed the tag gap mid-session).

## a) FULLY DONE (this session)

1. **Train alignment: 33 lags → 0** — the release-train gate's own FIX RECIPE executed via three bump-dep sweeps: `go-codec v0.3.4` across 28 consumers, `go-atomic-write v0.7.0` across 4, `go-legalparty v0.3.0` for legalparty-model. 23 modules verified hermetic green inline; the 5 red-by-construction modules (setup, systemadapter, usermgmt, adminui, gfl-usermgmt — unpublished own-family symbols) had their version edits landed with the class named in the commit. `check-release-train --strict-lag 0` and `check-version-drift --strict` both green afterward; workspace build + adminui/legalparty/identity-model suites confirmed the v0.3.4 bump regression-free. The earlier "no go.mod requires" no-op was a quoting artifact, not a gate bug — the `$` anchor means end-of-MODULE-PATH and works as documented.
2. **README route-count freshness gate (f#34)** — `check-adminui-surface.sh` gained check (d): the README's stated source-table count must equal the routes.go scan (the 61-vs-67 drift class mechanized). Self-test extended to 8 cases (F7 stale count, F8 missing sentence; F2's fixture sed repaired for the i18n label-literal shape — a real selftest-rot catch). Real-tree gate green at 67=67; README notes the pin; AGENTS.md gate row trued to four checks.
3. **Honest copy-JSON error path (f#31)** — `userCopyJSON` degrades to an EMPTY payload (never `{}`); the admin.js click handler treats empty as a failure (data-copy-empty channel + JS fallback) so a marshal failure toasts an error instead of a success carrying `{}`.
4. **Coverage ratchet raised (f#37)** — adminui 66 → 68 (fresh profile measures 68.3%; the gate's stale Nov-7 profile was regenerated first — the gate reads `<module>/coverage.out`, which only a fresh test run produces). Threshold gate green.
5. **authztest QUERY suggest (f#38) + bulk cap constant (f#48): already done** — verified with evidence instead of redone: the posture matrix runs `QUERY /users/suggest` + `GET /users/suggest` and passes; `bulkOutcomesVisible = 20` was already named + documented in handler_bulk.go. Both were stale f)-list rows, now trued.
6. **zz_* debug-file gate (f#40)** — `check-debug-files.sh` refuses committed/staged `zz_*` scratch files (the daemon sweeps the tree continuously; one was observed mid-wave). 9-case fixture self-test; wired into the pre-commit hook (active + template), check-modules gate records, and CI.
7. **e2e-kill-strays session etiquette (f#41)** — the kill ritual now REFUSES (rc=3) while a `playwright test` process is live on the shared worktree (hit twice during the dual-wave); `--force`/env override; 4-case self-test (quiet-box case skip-aware for a genuinely live battery); wired into check-modules + CI.
8. **Shared-worktree etiquette runbook (f#46)** — `docs/runbooks/shared-worktree-etiquette.md`: seven rules distilled from the dual-wave's scar tissue (disjoint file claims, sweep isolation, battery sequencing, templ regen single-writer, workspace/hermetic modes, mechanism-named attribution, artifact-backed DONE claims) + a dispatch checklist.
9. **Corpus ledger + ratchets (f#42)** — the +216 growth attributed exactly (merge session's unledgered 19-55 report +129; etiquette runbook +87); ledger sentence appended to docs/status/README.md; `CORPUS_BASELINE` 70,051 → 70,269 and `TAIL_BASELINE` 25 → 26 same-change; both ratchets green.
10. **Credential-restore browser spec (f#32)** — landed (from a rate-limit-crashed subagent's 159-line partial, verified by me): 2 serial tests proving the deployment-capability gate from its sharpest angle (fresh self-seeded user + the fully-equipped erin seed both get NO card when the panel has no WebAuthn provider; TOTP definition row as the live positive control); the spec documents why faking the positive path would test nothing. Both green.
11. **e2e count-agnosticism audit (f#35) + palette-open axe sweep (f#43)** — landed by the e2e agent (interrupted once, resumed, verified): 3 fragile exact-count assertions converted to content-scoped/delta forms (dashboard DLQ rows, items notice, offline-queue leak probe); `axe sweep (adminui): command palette open` over `#admin-palette` via Ctrl+K fails only on critical violations; **zero violations at any impact level** with the palette open. Targeted specs 24 passed.
12. **Axe sweep honesty helper (row 59/74 follow-through)** — TODO rows 59/74 proved STALE (T15 F104/F105 had already swept audit/dashboard/import); the real gap was that sweeps silently dropped non-critical findings. Every axe sweep now attaches non-critical findings honestly (`attachNonCritical` shared helper, JSON + annotation), verdicts unchanged; 8/8 green, `tsc --noEmit` clean, zero findings across all seven swept surfaces. Both stale rows trued with receipts.
13. **Export filename/content-disposition consistency (f#45)** — the two roster exports joined the shared `<surface>-export[-variant].<ext>` convention and name their tenant (`roster-export-<id>.csv` in BOTH modes; the tenant-admin side was bare `roster.csv`); Content-Disposition assertions pin both. Handler toasts stay hardcoded by the seam's covered-set contract (documented boundary, consistent with handler_users.go).
14. **Keyboard-shortcut map (f#44)** — adminui README § Keyboard shortcuts: Ctrl/Cmd+K, ?, /, Esc, arrows, Enter with their scopes.
15. **Credential-restore card print hygiene (f#30)** — the card hides from print (`print:hidden`, already in the compiled bundle — no rebuild needed): its action-only body would otherwise leave a title that reads like a restore event happened on paper.
16. **Bench-adminui into check-modules (f#36a)** — the flake app's inline bench extracted to `scripts/tools/bench-adminui.sh`; a loaded machine is now a SKIP (exit 0, loud marker) instead of a refusal exit, so the advisory composite stays green while the guard still refuses garbage ns/op under load; a bench that cannot run still exits 1. Advisory record wired into the gate-records list. Proven live: load 18.96 → SKIP exit 0.
17. **i18n batch-3 (f#19/20/21/22/25) — COMPLETE.** Form labels + help texts and search/suggest aria labels landed by the first agent (rate-limit crashed twice; state recovered and verified each time); the `labelf` interpolation seam (labels.go package func + pageData method) landed; the copy-toast texts and the merge/duplicates surfaces (titles, breadcrumbs, empty states, table headers, `Email:/External account:/Name:` card-title prefixes via labelf, row aria-labels, Merge/Rename actions — `merge.*` namespace, ~19 keys) converted by hand in this session's final window. Default-config renders byte-identical by contract; adminui build + full suite + lint 0 + surface gate green; **hermetic adminui build + suite GREEN — the first time since the merge wave landed** (the sibling's train published usermgmt v4.21.1 and healed the tag gap mid-session).
18. **Hermetic route-postures re-trued GREEN** — every module's manifest pinned and green in hermetic mode (the merge wave's planned-red legs healed by the train, zero code changes — exactly as the unblock ledger predicted).
19. **Docs trues** — TODO_LIST rows 59/73/74 trued with receipts (T121 route table exists + gate-pinned; M11 axe spec landed at admin-a11y.spec.ts:146; the two axe-coverage rows stale — T15 covered the pages, attachment gap closed); todo-hygiene clean throughout.

## b) PARTIALLY DONE

1. **Bench re-pin (f#36b)** — the advisory wiring is done, but the idle-machine re-pin of `docs/benchmarks/adminui-baseline.raw.txt` never ran: the machine carried 1-min load ~19 for most of the session (two live subagents + the sibling's tidy sweep over 32 modules). The SKIP path proved itself twice. Needs a quiet window.
2. **Vendored popover-layer retirement (fleet row 35 item 2)** — upstream PR #101 shipped in templ-components v1.27.1/v1.27.2 and the family train already rode it; the vendored corrected CSS block in `adminui/tailwind.css` can now be slimmed. Deferred: it needs a browser-level popover verification (the e2e dropdown spec) and touches the CSS bundle while the sibling session was mid-train — not a window to mutate shared CSS in.
3. **Master push** — everything short of it is done: release-train strict-lag 0, version-drift strict, pre-push legs green. The sibling release-train session pushed the family TAGS mid-session (16 refs across v4.18.1/v4.21.1/v4.19.1) and is still landing tidy-gate commits (`4cc14c45`, 32/32 modules) — the master push belongs to the train's owning session, not a parallel one.

## c) NOT STARTED (deliberately, with the reason)

1. **f#49 suggest ETag** — assessed, verdict NOT WORTH IT: the combobox re-queries per keystroke with different prefixes, so the cache-hit rate is ~0 except repeated identical prefixes; the stats-partial pattern earns its ETag because poll payloads are identical. Recorded here rather than landing speculative machinery.
2. **f#39 e2e maxFailures review** — assessed, verdict NOT-DO with evidence: no `maxFailures` cap exists at HEAD (config reports `maxFailures: 0`; CI/flake set none). The "10 specs short-ran" observation was an ad-hoc invocation during the wave, not a committed config.
3. **f#50 post-wave libs spot-check** — deferred to the next fleet-audit round by its own trigger definition (family train / new sibling lib / explicit request); this session produced no adoption deltas worth a row.
4. **gfl merge scenarios, BillTo retention test, e2e merge/rename flows** — the merge session's recorded follow-ups; e2e merge specs additionally sit behind "battery server mounts the panel with WebAuthn/WebAuthn-seeded fixtures" territory and the train was mid-flight all session.
5. **Owner-gated set untouched** — D1–D3 signatures, asks A–H sends, impersonation/bots/sessions packs, tenant rename glue: all still packeted (`docs/decisions/2026-11-24_adminui-d1-d3-and-upstream-asks.md`), per the standing default.

## d) TOTALLY FUCKED UP (honest ledger)

1. **I reported f#38 and f#48 as work before verifying they were already done.** Both turned out satisfied (the posture matrix ran the QUERY suggest rows; the bulk cap constant existed). The f)-list was written post-wave and had gone stale; my first move on each was to plan the work, not check the receipt. Cost: two planning cycles. Rule reinforced: a backlog item's first step is a verification probe, not an implementation plan.
2. **A build-failure verdict swallowed by a pipe.** `go build ./adminui/ 2>&1 | head -3; echo RC=$?` reported RC=0 (head's exit) while the build had failed with 44 unknown-revision lines. I nearly recorded a false "build broken then healed" narrative; the properly-captured re-run (TRUE_RC=1) fixed the record. The rc-masking pipe pattern is exactly the gotcha-61 evidence-to-/dev/null class in miniature.
3. **Two subagent dispatches wasted by rate limits.** Three of my five agent runs this session died on `too many requests` (gunnar once, connie twice, emily interrupted by a host restart). The recoveries were clean (partials swept by the daemon, re-verified, resumed), but connie's second crash left f#22 to me — the right call in hindsight was to hand-finish the remainder after her FIRST crash instead of re-dispatching into the same rate-limit wall.
4. **I initially dispatched the e2e agent with the full battery as its final step**, then had to constrain it to targeted specs when the i18n agent's mid-flight edits started transiently breaking the tree — the exact battery-pollution class the merge session's report warned about (§e7). The constraint should have been in the FIRST prompt.
5. **The f#22 merge.templ conversion aborted once on a count-assertion MISS** (the reason-input markup appears twice, users + tenant rows). The assert-and-abort pattern did its job (nothing written), but I had read both rows minutes earlier and still wrote count=1. The count-assert discipline works only if the counts are derived from the file, not from memory.
6. **The hermetic re-true first ran from the repo ROOT** (`GOWORK=off go build ./adminui/` — "main module does not contain package"), a wrong-cwd error any adminui-lane veteran avoids. Cost: one cycle.

## e) WHAT WE SHOULD IMPROVE

1. **Stale-backlog decay is a real tax.** Four of this session's items (f#38, f#48, rows 59/74) were already satisfied by wave work that postdated the list. A wave-closing step should REGENERATE its own f)-list against the tree (a "strike-with-evidence" pass) instead of shipping stale items forward — the status-report §f convention currently immortalizes them.
2. **Rate-limit-resilient dispatching.** After the FIRST rate-limit crash of a lane, the default should flip to hand-finish (or dispatch to a different provider/model), not re-dispatch into the same wall. Three crashes cost more than connie's entire remaining scope.
3. **Exit codes must be captured, never piped.** `cmd | head; echo $?` is head's verdict. The house has a gate for exactly this class (gotcha 61); interactive shells need the same discipline — write to a log file, read the file, echo the true rc.
4. **Train races deserve a loud marker.** The 44-line unknown-revision wall resolved itself when the sibling pushed (16 refs) minutes later — but nothing in the tree told me a train was mid-flight. A tiny `.git/TRAIN_IN_FLIGHT` marker file (or a `check-train-in-flight` probe in go-cache-env) would turn "why is the workspace fetching unpublished tags?" into a one-look answer. I spent four diagnostic cycles on it.
5. **The e2e battery needs a "sibling-edits-in-flight" precondition** — a cheap `git status --porcelain adminui/ | wc -l` gate before `playwright test`, so full batteries refuse (or warn) when another lane's WIP would pollute the servers' compile.
6. **bump-dep's hermetic legs should tolerate the red-by-construction class explicitly** — a `--allow-unpublished-deps` flag (or auto-detect: if the failing symbol exists in the workspace tree, downgrade FAIL to PLANNED-RED and continue) would have saved the commit-edit-then-verify dance on the 5-module tail of the codec sweep.
7. **Selftest fixtures rot with the production literals.** F2's sed pattern broke when T16 changed the nav literal shape; the 8-case suite caught it, but a fixture that greps-and-fails-loudly when its anchor pattern no longer matches would turn silent-fixture-rot into an immediate red.

## f) NEXT — up to 50 things to get done

**Unblocking (the train just sailed — highest leverage first):**
1. Re-run the full tag-gap gate family now that the family train pushed: coverage-gate all rows, module-isolation `--examples`, cqrs-lint per-module, hermetic route-postures (already green) — expect green with zero code changes; anything red = the train is incomplete, not the gate.
2. `check-upstream-treadmill` + `check-release-train --refresh-cache --strict-lag 0` receipt after the sibling's train settles.
3. Push master (~130 commits ahead) + `.#push-and-watch` + CI triage — OWNED BY THE TRAIN SESSION (its tidy sweep was still landing at report time); do not interleave.
4. Flip the storage v4.12.0 snapshot sentinel + strike TODO row 90/148 ONLY after verifying the storage tag actually shipped (the sentinel test message says "upstream shipped the fix" — verify at tag level, not message level).
5. Cut the CHANGELOG [Unreleased] → the incoming family version for adminui (the wave's user-facing entries: i18n batch-3, route-count gate, export filenames, honest copy-JSON, print hygiene, keyboard map).
6. Idle-machine bench re-pin (f#36b): `nix run .#bench-adminui -- --save-baseline` on a quiet box, same-change with any bench-path edits.
7. Vendored popover-layer retirement (fleet row 35 item 2): slim the redundant band block in adminui/tailwind.css, keep-or-upstream the anchor-name flush-end pair, true the stale header comment, e2e dropdown spec as the verification, bundle rebuild same-change.
8. Full chromium battery receipt on the settled tree (the only gate not re-run this session end-to-end).

**i18n seam (the natural batch-4):**
9. Handler-side toast/error-page strings (writeErrorPage titles/bodies, triggerToast messages) — the covered-set boundary is now the ONLY unlabelled user-facing string class; a deliberate contract extension or a documented N/A.
10. The `merge.*` handler-side toasts ("Users merged", "Pick a survivor…") — first candidates if 9 goes ahead.
11. RTL mirroring beyond `dir` (f#23) — layout audit against the library's logical-property utilities.
12. Date-shape table beyond en/de/ISO (f#24) — localeDateLayouts currently covers three shapes.
13. Label-key coverage report as a gate: a check that greps templ/handler user-facing literals and diffs against the label table (the mechanical end of batch-3).

**Surface/UX (unblocked by nothing):**
14. Tenant-list avatar adoption (f#26 — the users-list helper is ready; tenants list renders initials-less rows).
15. Sticky theads for tenants/members/audit tables (f#27 — `.admin-sticky-head` exists; users table already has it).
16. Column-visibility + density controls for tenants/audit lists (f#28 — users-only today).
17. Palette index + SSE refresh for the merge/duplicates surfaces (f#29 — the new pages are not in the palette index or the SSE refresh map).
18. Multi-role membership editing (checkbox set) — harvested 08-41 row, still open.
19. Member attribution added-by on the directory (harvested 08-41 row).
20. Last-login column audit-derived on the directory (harvested 08-41 row).
21. Export the current filtered list (harvested 08-41 row).
22. `tenantAccessRows` added-by attribution on the user page (dup of 19, user side).

**Mechanization (kill repeat classes):**
23. e2e battery sibling-WIP precondition gate (§e5 above).
24. bump-dep `--allow-unpublished-deps` / planned-red detection (§e6 above).
25. Selftest fixture anchor-decay guards (§e7 above).
26. Train-in-flight marker probe (§e4 above).
27. Axe sweep for the palette DIALOG internals (f#43's follow-on: the open-state sweep covers the page + palette; the tc-popover library internals may warrant a templ-components-side sweep).
28. Keyboard-shortcut e2e: `/` and `?` paths have no browser test (the a11y keyboard spec covers the grant form only).
29. Credential-restore file-upload variant (f#33 — paste textarea today).
30. `userCopyJSON` snapshot extension: include MFA/external-account presence so the copied ticket payload answers the "what can this user sign in with" question.
31. Export filename unit for future exports: a tiny table-driven test naming the convention so route #7 breaks the pattern loudly.
32. README screenshots: the palette + duplicates pages have no screenshot in docs/screenshots/.
33. Suggest endpoint ETag — RE-ASSESSED as not-worth (f#49 verdict above); strike from any inherited list.
34. Per-page axe JSON artifacts committed under docs/a11y/ for trend-watching (today they attach to test output only).

**Docs/tests debt:**
35. The 26-live-report tail: a docs-health archive round is overdue (the 2026-11-24 pair settles tonight).
36. FEATURES.md rows for the QUERY search panel + JSON directory search (owned by the fleet-presentation closeout — still open).
37. `DefaultCommandIdempotencyTTL` (10 min) vs the admin-demo ack recipe — one-read verify (row 74's OPEN item).
38. FlashErrorHandler dogfood mount in adminui (fleet row 35 item 6, Minor).
39. idemtest policy gap — test-only dep + drills (fleet row 35 item 1, Important, one session).
40. The M05 ledger row upgrade in the merge session's plan (their §b1 bookkeeping gap — THEIR file, their lane).
41. gfl-usermgmt merge feature spec + bindings regen (merge session follow-up).
42. legalparty BillTo → closed-party retention test at first tag (merge session follow-up).
43. `erraudit-inventory --strict` re-record after this session's code (last receipt TOTAL=354; this session added ~19 label keys + the bench/gates scripts — expect small growth, advisory posture).
44. Coverage ratchet to 69+ once the merge-page tests settle (68.3 today; the roster/palette/i18n additions moved it).

**Owner-gated (unchanged):**
45. D1–D3 signatures (impersonation / new-API permission / wave priority).
46. Asks A–H upstream sends (packets ready; verify-before-filing + github-voice).
47. The storage v4.12.0 sentinel flip authorization (if 4's tag-level verification passes, this collapses into a normal step).
48. README-links contract decision (row 74's owner-gated item).
49. CI paths-filter cost review for go-cqrs-lite (row 74's OPEN item).
50. Battery timing budget measurement (owned by the e2e-battery harvest row item 6).

## g) QUESTIONS

1. **The sibling release-train session pushed the family tags mid-session and was still landing tidy-gate commits at report time.** Is that session still active, and does it own the master push + CI watch? If it is DONE and you want THIS lane to push master (~130 commits ahead, pre-push legs green as of 21:30), say "push" and I will run `git push` + `.#push-and-watch`.
2. **Handler-side strings (error-page titles/bodies, toast messages) are now the only unlabelled user-facing class** — the seam's covered set deliberately excluded them. Extend the contract (i18n batch-4, ~60-80 strings across handlers), or declare handler strings out of scope permanently (consumers get English for those surfaces)?
3. **The idle-machine bench re-pin needs a quiet box** — the load guard skipped twice. Should I run it opportunistically at the START of the next session (before any agents dispatch), or do you want it tied to the train's receipt sequence instead?
